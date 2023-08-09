// Copyright (c) HashiCorp, Inc.
// SPDX-License-Identifier: MPL-2.0

package arm

import (
	"context"
	"errors"
	"fmt"
	"net/url"
	"strings"
	"sync"
	"time"

	"github.com/hashicorp/go-azure-helpers/resourcemanager/commonids"
	"github.com/hashicorp/go-azure-sdk/resource-manager/compute/2022-03-01/virtualmachines"
	"github.com/hashicorp/go-azure-sdk/resource-manager/compute/2022-03-02/disks"
	"github.com/hashicorp/go-azure-sdk/resource-manager/network/2022-09-01/networksecuritygroups"
	"github.com/hashicorp/go-azure-sdk/resource-manager/network/2022-09-01/virtualnetworks"
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
)

type StepDeployTemplate struct {
	client           *AzureClient
	deploy           func(ctx context.Context, subscriptionId string, resourceGroupName string, deploymentName string) error
	delete           func(ctx context.Context, subscriptionId, deploymentName, resourceGroupName string) error
	disk             func(ctx context.Context, subscriptionId string, resourceGroupName string, computeName string) (string, string, error)
	deleteDisk       func(ctx context.Context, imageName string, resourceGroupName string, isManagedDisk bool, subscriptionId string, storageAccountName string) error
	deleteDeployment func(ctx context.Context, state multistep.StateBag) error
	say              func(message string)
	error            func(e error)
	config           *Config
	factory          templateFactoryFunc
	name             string
	templateType     DeploymentTemplateType
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
	step.delete = step.deleteDeploymentResources
	step.disk = step.getImageDetails
	step.deleteDisk = step.deleteImage
	step.deleteDeployment = step.deleteDeploymentObject
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
	if s.templateType == KeyVaultTemplate {
		ui.Say("\nDeleting KeyVault created during build")
		err := s.delete(ctx, subscriptionId, deploymentName, resourceGroupName)
		if err != nil {
			s.reportIfError(err, resourceGroupName)
		}

	} else {
		ui.Say("\nDeleting Virtual Machine deployment and its attatched resources...")
		// Get image disk details before deleting the image; otherwise we won't be able to
		// delete the disk as the image request will return a 404
		computeName := state.Get(constants.ArmComputeName).(string)
		isManagedDisk := state.Get(constants.ArmIsManagedImage).(bool)
		isSIGImage := state.Get(constants.ArmIsSIGImage).(bool)
		armStorageAccountName := state.Get(constants.ArmStorageAccountName).(string)
		imageType, imageName, err := s.disk(ctx, subscriptionId, resourceGroupName, computeName)
		if err != nil {
			ui.Error(fmt.Sprintf("Could not retrieve OS Image details: %s", err))
		}
		err = s.delete(ctx, subscriptionId, deploymentName, resourceGroupName)
		if err != nil {
			s.reportIfError(err, resourceGroupName)
		}
		// The disk was not found on the VM, this is an error.
		if imageType == "" && imageName == "" {
			ui.Error(fmt.Sprintf("Failed to find temporary OS disk on VM.  Please delete manually.\n\n"+
				"VM Name: %s\n"+
				"Error: %s", computeName, err))
			return
		}
		if !state.Get(constants.ArmKeepOSDisk).(bool) {
			ui.Say(fmt.Sprintf(" Deleting -> %s : '%s'", imageType, imageName))
			err = s.deleteDisk(ctx, imageName, resourceGroupName, (isManagedDisk || isSIGImage), subscriptionId, armStorageAccountName)
			if err != nil {
				ui.Error(fmt.Sprintf("Error deleting resource.  Please delete manually.\n\n"+
					"Name: %s\n"+
					"Error: %s", imageName, err))
			}
		} else {
			ui.Say(fmt.Sprintf("Skipping deletion -> %s : '%s' since 'keep_os_disk' is set to true", imageType, imageName))
		}
		var dataDisks []string
		if disks := state.Get(constants.ArmAdditionalDiskVhds); disks != nil {
			dataDisks = disks.([]string)
		}
		for i, additionaldisk := range dataDisks {
			s.say(fmt.Sprintf(" Deleting Additional Disk -> %d: '%s'", i+1, additionaldisk))

			err := s.deleteImage(ctx, additionaldisk, resourceGroupName, (isManagedDisk || isSIGImage), subscriptionId, armStorageAccountName)
			if err != nil {
				s.say("Failed to delete the managed Additional Disk!")
			}
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
	//TODO is this still true?
	//We can't depend on constants.ArmOSDiskVhd being set
	var imageName, imageType string
	vmID := virtualmachines.NewVirtualMachineID(subscriptionId, resourceGroupName, computeName)
	vm, err := s.client.VirtualMachinesClient.Get(ctx, vmID, virtualmachines.DefaultGetOperationOptions())
	if err != nil {
		return imageName, imageType, err
	}
	if err != nil {
		s.say(s.client.LastError.Error())
		return "", "", err
	}
	if model := vm.Model; model == nil {
		return "", "", errors.New(client.SDK_NULL_MODEL_ERROR)
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

func deleteResource(ctx context.Context, client *AzureClient, subscriptionId string, resourceType string, resourceName string, resourceGroupName string) error {

	pollingContext, cancel := context.WithTimeout(ctx, client.PollingDuration)
	defer cancel()

	switch resourceType {
	case "Microsoft.Compute/virtualMachines":
		vmID := virtualmachines.NewVirtualMachineID(subscriptionId, resourceGroupName, resourceName)
		if err := client.VirtualMachinesClient.DeleteThenPoll(pollingContext, vmID, virtualmachines.DefaultDeleteOperationOptions()); err != nil {
			return err
		}
	case "Microsoft.KeyVault/vaults":
		id := commonids.NewKeyVaultID(subscriptionId, resourceGroupName, resourceName)
		_, err := client.VaultsClient.Delete(pollingContext, id)
		return err
	case "Microsoft.Network/networkInterfaces":
		interfaceID := commonids.NewNetworkInterfaceID(subscriptionId, resourceGroupName, resourceName)
		err := client.NetworkMetaClient.NetworkInterfaces.DeleteThenPoll(pollingContext, interfaceID)
		return err
	case "Microsoft.Network/virtualNetworks":
		vnetID := virtualnetworks.NewVirtualNetworkID(subscriptionId, resourceGroupName, resourceName)
		err := client.NetworkMetaClient.VirtualNetworks.DeleteThenPoll(pollingContext, vnetID)
		return err
	case "Microsoft.Network/networkSecurityGroups":
		secGroupId := networksecuritygroups.NewNetworkSecurityGroupID(subscriptionId, resourceGroupName, resourceName)
		err := client.NetworkMetaClient.NetworkSecurityGroups.DeleteThenPoll(pollingContext, secGroupId)
		return err
	case "Microsoft.Network/publicIPAddresses":
		ipID := commonids.NewPublicIPAddressID(subscriptionId, resourceGroupName, resourceName)
		err := client.NetworkMetaClient.PublicIPAddresses.DeleteThenPoll(pollingContext, ipID)
		return err
	}
	return nil
}

// TODO Let's split this into two seperate methods
// deleteVHD and deleteManagedDisk, and then just check in Cleanup which function to call
func (s *StepDeployTemplate) deleteImage(ctx context.Context, imageName string, resourceGroupName string, isManagedDisk bool, subscriptionId string, storageAccountName string) error {
	// Managed disk
	pollingContext, cancel := context.WithTimeout(ctx, s.client.PollingDuration)
	defer cancel()
	if isManagedDisk {
		xs := strings.Split(imageName, "/")
		diskName := xs[len(xs)-1]
		diskId := disks.NewDiskID(subscriptionId, resourceGroupName, diskName)

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
	_, err = s.client.GiovanniBlobClient.Delete(pollingContext, storageAccountName, "images", blobName, giovanniBlobStorageSDK.DeleteInput{})
	return err
}

func (s *StepDeployTemplate) deleteDeploymentResources(ctx context.Context, subscriptionId, deploymentName, resourceGroupName string) error {
	var maxResources int64 = 50
	options := deploymentoperations.DefaultListOperationOptions()
	options.Top = &maxResources
	id := deploymentoperations.NewResourceGroupDeploymentID(subscriptionId, resourceGroupName, deploymentName)
	deploymentOperations, err := s.client.DeploymentOperationsClient.ListComplete(ctx, id, options)
	if err != nil {
		s.reportIfError(err, resourceGroupName)
		return err
	}

	resources := map[string]string{}

	for _, deploymentOperation := range deploymentOperations.Items {
		// Sometimes an empty operation is added to the list by Azure
		if deploymentOperation.Properties.TargetResource == nil {
			continue
		}

		resourceName := *deploymentOperation.Properties.TargetResource.ResourceName
		resourceType := *deploymentOperation.Properties.TargetResource.ResourceType

		s.say(fmt.Sprintf("Adding to deletion queue -> %s : '%s'", resourceType, resourceName))
		resources[resourceType] = resourceName

	}

	var wg sync.WaitGroup
	wg.Add(len(resources))

	for resourceType, resourceName := range resources {
		go func(resourceType, resourceName string) {
			defer wg.Done()
			retryConfig := retry.Config{
				Tries:      10,
				RetryDelay: (&retry.Backoff{InitialBackoff: 5 * time.Second, MaxBackoff: 60 * time.Second, Multiplier: 1.5}).Linear,
			}

			err = retryConfig.Run(ctx, func(ctx context.Context) error {
				s.say(fmt.Sprintf("Attempting deletion -> %s : '%s'", resourceType, resourceName))
				err := deleteResource(ctx, s.client,
					subscriptionId,
					resourceType,
					resourceName,
					resourceGroupName)
				if err != nil {
					s.say(fmt.Sprintf("Couldn't delete %s resource. Will retry.\n"+
						"Name: %s",
						resourceType, resourceName))
				}
				return err
			})
			if err != nil {
				s.reportIfError(err, resourceName)
			}
		}(resourceType, resourceName)
	}

	s.say("Waiting for deletion of all resources...")
	wg.Wait()

	return nil
}

func (s *StepDeployTemplate) reportIfError(err error, resourceName string) {
	if err != nil {
		s.say(fmt.Sprintf("Error deleting resource. Please delete manually.\n\n"+
			"Name: %s\n"+
			"Error: %s", resourceName, err.Error()))
		s.error(err)
	}
}
