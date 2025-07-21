// Copyright (c) HashiCorp, Inc.
// SPDX-License-Identifier: MPL-2.0

package arm

import (
	"context"
	"errors"
	"fmt"
	"log"
	"net/url"
	"strings"
	"sync"
	"time"

	"github.com/hashicorp/go-azure-helpers/resourcemanager/commonids"
	"github.com/hashicorp/go-azure-sdk/resource-manager/compute/2022-03-01/virtualmachines"
	"github.com/hashicorp/go-azure-sdk/resource-manager/network/2023-09-01/networksecuritygroups"
	"github.com/hashicorp/go-azure-sdk/resource-manager/resources/2022-09-01/deploymentoperations"
	"github.com/hashicorp/go-azure-sdk/resource-manager/resources/2022-09-01/deployments"
	"github.com/hashicorp/packer-plugin-azure/builder/azure/common/client"
	"github.com/hashicorp/packer-plugin-azure/builder/azure/common/constants"
	"github.com/hashicorp/packer-plugin-sdk/multistep"
	packersdk "github.com/hashicorp/packer-plugin-sdk/packer"
	"github.com/hashicorp/packer-plugin-sdk/retry"
	giovanniBlobStorageSDK "github.com/tombuildsstuff/giovanni/storage/2020-08-04/blob/blobs"
)

type DeploymentTemplateType int

const (
	VirtualMachineTemplate DeploymentTemplateType = iota
	KeyVaultTemplate
	VMResourceType               = "Microsoft.Compute/virtualMachines"
	NetworkInterfaceResourceType = "Microsoft.Network/networkInterfaces"
	KeyVaultResourceType         = "Microsoft.KeyVault/vaults"
)

type StepDeployTemplate struct {
	client                  *AzureClient
	deploy                  func(ctx context.Context, subscriptionId string, resourceGroupName string, deploymentName string) error
	deleteDetachedResources func(ctx context.Context, subscriptionId string, resourceGroupName string, resources map[string]string)
	getDisk                 func(ctx context.Context, subscriptionId string, resourceGroupName string, computeName string) (string, string, error)
	deleteDisk              func(ctx context.Context, imageName string, resourceGroupName string, isManagedDisk bool, subscriptionId string, storageAccountName string) error
	deleteVM                func(ctx context.Context, virtualMachineId virtualmachines.VirtualMachineId) error
	deleteNic               func(ctx context.Context, networkInterfacesId commonids.NetworkInterfaceId) error
	deleteDeployment        func(ctx context.Context, state multistep.StateBag) error
	deleteKV                func(ctx context.Context, id commonids.KeyVaultId) error
	listDeploymentOps       func(ctx context.Context, id deploymentoperations.ResourceGroupDeploymentId) ([]deploymentoperations.DeploymentOperation, error)
	say                     func(message string)
	error                   func(e error)
	config                  *Config
	factory                 templateFactoryFunc
	name                    string
	templateType            DeploymentTemplateType
}

func NewStepDeployTemplate(client *AzureClient, ui packersdk.Ui, config *Config, deploymentName string, factory templateFactoryFunc, templateType DeploymentTemplateType) *StepDeployTemplate {
	var step = &StepDeployTemplate{
		client:       client,
		say:          func(message string) { ui.Say(message) },
		error:        func(e error) { ui.Error(e.Error()) },
		config:       config,
		factory:      factory,
		name:         deploymentName,
		templateType: templateType,
	}

	step.deploy = step.deployTemplate
	step.getDisk = step.getImageDetails
	step.deleteDisk = step.deleteImage
	step.deleteDetachedResources = step.deleteDetachedResourcesWithQueue
	step.deleteDeployment = step.deleteDeploymentObject
	step.deleteNic = step.deleteNetworkInterface
	step.deleteVM = step.deleteVirtualMachine
	step.deleteKV = step.deleteKeyVault
	step.listDeploymentOps = step.listDeploymentOperations
	return step
}

func (s *StepDeployTemplate) Run(ctx context.Context, state multistep.StateBag) multistep.StepAction {
	s.say("Deploying deployment template ...")

	var resourceGroupName = state.Get(constants.ArmResourceGroupName).(string)
	var subscriptionId = state.Get(constants.ArmSubscription).(string)
	s.say(fmt.Sprintf(" -> ResourceGroupName : '%s'", resourceGroupName))
	s.say(fmt.Sprintf(" -> DeploymentName    : '%s'", s.name))

	return processStepResult(
		s.deploy(ctx, subscriptionId, resourceGroupName, s.name),
		s.error, state)
}

func (s *StepDeployTemplate) Cleanup(state multistep.StateBag) {
	ctx, cancel := context.WithTimeout(context.Background(), time.Minute*10)
	defer func() {
		err := s.deleteDeployment(ctx, state)
		if err != nil {
			s.say(err.Error())
		}
		cancel()
	}()

	ui := state.Get("ui").(packersdk.Ui)
	deploymentName := s.name
	resourceGroupName := state.Get(constants.ArmResourceGroupName).(string)
	subscriptionId := state.Get(constants.ArmSubscription).(string)
	config := state.Get("config").(*Config)
	deploymentOpsId := deploymentoperations.ResourceGroupDeploymentId{
		DeploymentName:    deploymentName,
		ResourceGroupName: resourceGroupName,
		SubscriptionId:    subscriptionId,
	}
	// If this is the KeyVault Deployment, delete the KeyVault
	if s.templateType == KeyVaultTemplate {
		ui.Say("\nDeleting KeyVault created during build")
		deploymentOperations, err := s.listDeploymentOps(ctx, deploymentOpsId)
		if err != nil {
			ui.Error(fmt.Sprintf("Could not retrieve deployment operations: %s\n to get the KeyVault, please manually delete it ", err))
			return
		}
		for _, deploymentOperation := range deploymentOperations {
			// Sometimes an empty operation is added to the list by Azure
			if deploymentOperation.Properties.TargetResource == nil {
				continue
			}
			resourceName := *deploymentOperation.Properties.TargetResource.ResourceName
			resourceType := *deploymentOperation.Properties.TargetResource.ResourceType

			if resourceType == KeyVaultResourceType {
				kvID := commonids.KeyVaultId{
					VaultName:         resourceName,
					ResourceGroupName: resourceGroupName,
					SubscriptionId:    subscriptionId,
				}
				err := s.deleteKV(ctx, kvID)
				if err != nil {
					s.reportResourceDeletionFailure(err, resourceGroupName)
				}
				return
			}
		}
		return
	}
	// Otherwise delete the Virtual Machine
	ui.Say("\nDeleting Virtual Machine deployment and its attached resources...")
	// Get image disk details before deleting the image; otherwise we won't be able to
	// delete the disk as the image request will return a 404
	computeName := state.Get(constants.ArmComputeName).(string)
	isManagedDisk := state.Get(constants.ArmIsManagedImage).(bool)
	isSIGImage := state.Get(constants.ArmIsSIGImage).(bool)
	armStorageAccountName := state.Get(constants.ArmStorageAccountName).(string)
	imageType, imageName, err := s.getDisk(ctx, subscriptionId, resourceGroupName, computeName)
	if err != nil {
		ui.Error(fmt.Sprintf("Could not retrieve OS Image details: %s", err))
	}

	deploymentOperations, err := s.listDeploymentOps(ctx, deploymentOpsId)
	if err != nil {
		ui.Error(fmt.Sprintf("Could not retrieve deployment operations: %s\n Virtual Machine %s, and its please manually delete it and its associated resources", err, computeName))
		return
	}
	resources := map[string]string{}
	var vmID *virtualmachines.VirtualMachineId
	var networkInterfaceID *commonids.NetworkInterfaceId
	for _, deploymentOperation := range deploymentOperations {
		// Sometimes an empty operation is added to the list by Azure
		if deploymentOperation.Properties.TargetResource == nil {
			continue
		}
		resourceName := *deploymentOperation.Properties.TargetResource.ResourceName
		resourceType := *deploymentOperation.Properties.TargetResource.ResourceType

		if resourceType == "Microsoft.Network/networkSecurityGroups" && config.NetworkSecurityGroupName != "" && resourceName == config.NetworkSecurityGroupName {
			continue
		}

		// Grab the Virtual Machine and Network ID resource names, and save them into Azure Resource IDs to be used later
		// We always want to delete the VM first, then the NIC, even if the ListDeployment endpoint doesn't return resources sorted in the order we want to delete them
		switch resourceType {
		case "Microsoft.Compute/virtualMachines":
			vmIDDeref := virtualmachines.NewVirtualMachineID(subscriptionId, resourceGroupName, resourceName)
			vmID = &vmIDDeref
		case "Microsoft.Network/networkInterfaces":
			networkInterfaceIDDeref := commonids.NewNetworkInterfaceID(subscriptionId, resourceGroupName, resourceName)
			networkInterfaceID = &networkInterfaceIDDeref
		default:
			resources[resourceType] = resourceName
		}
	}

	if vmID != nil {

		err := s.deleteVM(ctx, *vmID)
		if err != nil {
			return
		}
	}
	if networkInterfaceID != nil {
		err := s.deleteNic(ctx, *networkInterfaceID)
		if err != nil {
			return
		}
	}
	s.deleteDetachedResources(ctx, subscriptionId, resourceGroupName, resources)
	// The disk was not found on the VM, this is an error.
	if imageType == "" && imageName == "" {
		ui.Error(fmt.Sprintf("Failed to find temporary OS disk on VM.  Please delete manually.\n\n"+
			"VM Name: %s\n"+
			"Error: %s", computeName, err))
		return
	}
	if !state.Get(constants.ArmKeepOSDisk).(bool) {
		err = s.deleteDisk(ctx, imageName, resourceGroupName, (isManagedDisk || isSIGImage), subscriptionId, armStorageAccountName)
		if err != nil {
			s.reportResourceDeletionFailure(err, imageName)

		} else {
			ui.Say(fmt.Sprintf("Deleted -> %s : '%s'", imageType, imageName))
		}
	} else {
		ui.Say(fmt.Sprintf("Skipping deletion -> %s : '%s' since 'keep_os_disk' is set to true", imageType, imageName))
	}
	var dataDisks []string
	if disks := state.Get(constants.ArmAdditionalDiskVhds); disks != nil {
		dataDisks = disks.([]string)
	}
	for i, additionaldisk := range dataDisks {
		err := s.deleteImage(ctx, additionaldisk, resourceGroupName, (isManagedDisk || isSIGImage), subscriptionId, armStorageAccountName)
		if err == nil {
			s.say(fmt.Sprintf("Deleted Additional Disk -> %d: '%s'", i+1, additionaldisk))

		} else {
			s.reportResourceDeletionFailure(err, additionaldisk)
		}
	}

}

func (s *StepDeployTemplate) deployTemplate(ctx context.Context, subscriptionId string, resourceGroupName string, deploymentName string) error {
	deployment, err := s.factory(s.config)
	if err != nil {
		return err
	}
	pollingContext, cancel := context.WithTimeout(ctx, s.client.PollingDuration)
	defer cancel()
	id := deployments.NewResourceGroupProviderDeploymentID(subscriptionId, resourceGroupName, deploymentName)
	err = s.client.DeploymentsClient.CreateOrUpdateThenPoll(pollingContext, id, *deployment)
	if err != nil {
		s.say(s.client.LastError.Error())
		return err
	}
	return nil
}

func (s *StepDeployTemplate) deleteDeploymentObject(ctx context.Context, state multistep.StateBag) error {
	deploymentName := s.name
	resourceGroupName := state.Get(constants.ArmResourceGroupName).(string)
	subscriptionId := state.Get(constants.ArmSubscription).(string)
	ui := state.Get("ui").(packersdk.Ui)

	pollingContext, cancel := context.WithTimeout(ctx, s.client.PollingDuration)
	defer cancel()
	ui.Say(fmt.Sprintf("Removing the created Deployment object: '%s'", deploymentName))
	id := deployments.NewResourceGroupProviderDeploymentID(subscriptionId, resourceGroupName, deploymentName)
	err := s.client.DeploymentsClient.DeleteThenPoll(pollingContext, id)
	if err != nil {
		return err
	}
	return nil
}

func (s *StepDeployTemplate) getImageDetails(ctx context.Context, subscriptionId string, resourceGroupName string, computeName string) (string, string, error) {
	var imageName, imageType string
	pollingContext, cancel := context.WithTimeout(ctx, s.client.PollingDuration)
	defer cancel()
	vmID := virtualmachines.NewVirtualMachineID(subscriptionId, resourceGroupName, computeName)
	vm, err := s.client.VirtualMachinesClient.Get(pollingContext, vmID, virtualmachines.DefaultGetOperationOptions())
	if err != nil {
		s.say(s.client.LastError.Error())
		return "", "", err
	}
	if model := vm.Model; model == nil {
		return "", "", client.NullModelSDKErr
	}
	if vm.Model.Properties.StorageProfile.OsDisk.Vhd != nil {
		imageType = "image"
		imageName = *vm.Model.Properties.StorageProfile.OsDisk.Vhd.Uri
		return imageType, imageName, nil
	}

	if vm.Model.Properties.StorageProfile.OsDisk.ManagedDisk.Id == nil {
		return "", "", fmt.Errorf("unable to obtain a OS disk for %q, please check that the instance has been created", computeName)
	}

	imageType = "Microsoft.Compute/disks"
	imageName = *vm.Model.Properties.StorageProfile.OsDisk.ManagedDisk.Id

	return imageType, imageName, nil
}

// TODO Let's split this into two separate methods
// deleteVHD and deleteManagedDisk, and then just check in Cleanup which function to call
func (s *StepDeployTemplate) deleteImage(ctx context.Context, imageName string, resourceGroupName string, isManagedDisk bool, subscriptionId string, storageAccountName string) error {
	// Managed disk
	pollingContext, cancel := context.WithTimeout(ctx, s.client.PollingDuration)
	defer cancel()
	if isManagedDisk {
		xs := strings.Split(imageName, "/")
		diskName := xs[len(xs)-1]
		diskId := commonids.NewManagedDiskID(subscriptionId, resourceGroupName, diskName)

		if err := s.client.DisksClient.DeleteThenPoll(pollingContext, diskId); err != nil {
			return err
		}
		return nil
	}

	// VHD image
	u, err := url.Parse(imageName)
	if err != nil {
		return err
	}
	xs := strings.Split(u.Path, "/")
	var blobName = strings.Join(xs[2:], "/")
	if len(xs) < 3 {
		return errors.New("Unable to parse path of image " + imageName)
	}
	_, err = s.client.GiovanniBlobClient.Delete(pollingContext, "images", blobName, giovanniBlobStorageSDK.DeleteInput{})
	return err
}

func (s *StepDeployTemplate) retryDeletion(ctx context.Context, resourceType string, resourceName string, deleteResourceFunction func() error) error {
	log.Printf("[INFO] Attempting deletion -> %s : %s", resourceType, resourceName)
	retryConfig := retry.Config{
		Tries: 5,
		RetryDelay: (&retry.Backoff{
			InitialBackoff: 3 * time.Second,
			MaxBackoff:     15 * time.Second,
			Multiplier:     1.5,
		}).Linear,
	}
	err := retryConfig.Run(ctx, func(ctx context.Context) error {
		err := deleteResourceFunction()
		if err != nil {
			log.Printf("[INFO] Couldn't delete resource %s.%s, will retry", resourceType, resourceName)
		}
		return err
	})
	if err != nil {
		s.reportResourceDeletionFailure(err, resourceName)
	} else {
		s.say(fmt.Sprintf("Deleted -> %s : '%s'", resourceType, resourceName))
	}
	return err
}

func (s *StepDeployTemplate) deleteVirtualMachine(ctx context.Context, vmID virtualmachines.VirtualMachineId) error {
	pollingContext, cancel := context.WithTimeout(ctx, s.client.PollingDuration)
	defer cancel()
	err := s.retryDeletion(pollingContext, vmID.VirtualMachineName, VMResourceType, func() error {
		return s.client.VirtualMachinesClient.DeleteThenPoll(ctx, vmID, virtualmachines.DefaultDeleteOperationOptions())
	})
	return err
}

func (s *StepDeployTemplate) deleteKeyVault(ctx context.Context, id commonids.KeyVaultId) error {
	pollingContext, cancel := context.WithTimeout(ctx, s.client.PollingDuration)
	defer cancel()
	err := s.retryDeletion(pollingContext, id.VaultName, KeyVaultResourceType, func() error {
		_, err := s.client.VaultsClient.Delete(pollingContext, id)
		return err
	})
	return err
}

func (s *StepDeployTemplate) deleteNetworkInterface(ctx context.Context, id commonids.NetworkInterfaceId) error {
	pollingContext, cancel := context.WithTimeout(ctx, s.client.PollingDuration)
	defer cancel()
	err := s.retryDeletion(pollingContext, id.NetworkInterfaceName, NetworkInterfaceResourceType, func() error {
		return s.client.NetworkMetaClient.NetworkInterfaces.DeleteThenPoll(pollingContext, id)
	})
	return err
}

func (s *StepDeployTemplate) listDeploymentOperations(ctx context.Context, id deploymentoperations.ResourceGroupDeploymentId) ([]deploymentoperations.DeploymentOperation, error) {
	var maxResources int64 = 50
	options := deploymentoperations.DefaultListOperationOptions()
	options.Top = &maxResources
	pollingContext, cancel := context.WithTimeout(ctx, s.client.PollingDuration)
	defer cancel()

	deploymentOperations, err := s.client.DeploymentOperationsClient.ListComplete(pollingContext, id, options)
	if err != nil {
		return nil, err
	}
	return deploymentOperations.Items, nil
}

// This function is called to delete the resources remaining in the deployment after we delete the Virtual Machine and the deleteNic
// Trying to delete these resources before the VM and the NIC results in errors
func (s *StepDeployTemplate) deleteDetachedResourcesWithQueue(ctx context.Context, subscriptionId string, resourceGroupName string, resources map[string]string) {
	var wg sync.WaitGroup
	wg.Add(len(resources))

	for resourceType, resourceName := range resources {
		go func(resourceType, resourceName string) {
			defer wg.Done()
			// Failures here are logged, and will not stop further cleanup at this point
			_ = s.retryDeletion(ctx, resourceName, resourceType, func() error {
				return deleteResource(ctx, s.client,
					subscriptionId,
					resourceType,
					resourceName,
					resourceGroupName)
			})

		}(resourceType, resourceName)
	}

	wg.Wait()
}

func deleteResource(ctx context.Context, client *AzureClient, subscriptionId string, resourceType string, resourceName string, resourceGroupName string) error {
	pollingContext, cancel := context.WithTimeout(ctx, client.PollingDuration)
	defer cancel()

	var err error
	switch resourceType {
	case "Microsoft.Network/virtualNetworks":
		vnetID := commonids.NewVirtualNetworkID(subscriptionId, resourceGroupName, resourceName)
		err = client.NetworkMetaClient.VirtualNetworks.DeleteThenPoll(pollingContext, vnetID)
	case "Microsoft.Network/networkSecurityGroups":
		secGroupId := networksecuritygroups.NewNetworkSecurityGroupID(subscriptionId, resourceGroupName, resourceName)
		err = client.NetworkMetaClient.NetworkSecurityGroups.DeleteThenPoll(pollingContext, secGroupId)
	case "Microsoft.Network/publicIPAddresses":
		ipID := commonids.NewPublicIPAddressID(subscriptionId, resourceGroupName, resourceName)
		err = client.NetworkMetaClient.PublicIPAddresses.DeleteThenPoll(pollingContext, ipID)
	}
	return err
}

func (s *StepDeployTemplate) reportResourceDeletionFailure(err error, resourceName string) {
	s.say(fmt.Sprintf("Error deleting resource. Please delete manually.\n\n"+
		"Name: %s\n"+
		"Error: %s", resourceName, err.Error()))
	s.error(err)
}
