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

	"github.com/hashicorp/packer-plugin-azure/builder/azure/common/constants"
	"github.com/hashicorp/packer-plugin-sdk/multistep"
	packersdk "github.com/hashicorp/packer-plugin-sdk/packer"
	"github.com/hashicorp/packer-plugin-sdk/retry"
)

type DeploymentTemplateType int

const (
	VirtualMachineTemplate DeploymentTemplateType = iota
	KeyVaultTemplate
)

type StepDeployTemplate struct {
	client           *AzureClient
	deploy           func(ctx context.Context, resourceGroupName string, deploymentName string) error
	delete           func(ctx context.Context, deploymentName, resourceGroupName string) error
	disk             func(ctx context.Context, resourceGroupName string, computeName string) (string, string, error)
	deleteDisk       func(ctx context.Context, imageName string, resourceGroupName string, isManagedDisk bool) error
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
	s.say(fmt.Sprintf(" -> ResourceGroupName : '%s'", resourceGroupName))
	s.say(fmt.Sprintf(" -> DeploymentName    : '%s'", s.name))

	return processStepResult(
		s.deploy(ctx, resourceGroupName, s.name),
		s.error, state)
}

func (s *StepDeployTemplate) Cleanup(state multistep.StateBag) {
	defer func() {
		err := s.deleteDeployment(context.Background(), state)
		if err != nil {
			s.say(err.Error())
		}
	}()

	ui := state.Get("ui").(packersdk.Ui)
	deploymentName := s.name
	resourceGroupName := state.Get(constants.ArmResourceGroupName).(string)
	if s.templateType == KeyVaultTemplate {
		ui.Say("\nDeleting KeyVault created during build")
		err := s.delete(context.TODO(), deploymentName, resourceGroupName)
		if err != nil {
			s.reportIfError(err, resourceGroupName)
		}

	} else {
		ui.Say("\nDeleting Virtual Machine deployment and its attatched resources...")
		// Get image disk details before deleting the image; otherwise we won't be able to
		// delete the disk as the image request will return a 404
		computeName := state.Get(constants.ArmComputeName).(string)
		isManagedDisk := state.Get(constants.ArmIsManagedImage).(bool)
		imageType, imageName, err := s.disk(context.TODO(), resourceGroupName, computeName)
		if err != nil {
			ui.Error(fmt.Sprintf("Could not retrieve OS Image details: %s", err))
		}
		err = s.delete(context.TODO(), deploymentName, resourceGroupName)
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
			err = s.deleteDisk(context.TODO(), imageName, resourceGroupName, isManagedDisk)
			if err != nil {
				ui.Error(fmt.Sprintf("Error deleting resource.  Please delete manually.\n\n"+
					"Name: %s\n"+
					"Error: %s", imageName, err))
			}
		}

		var dataDisks []string
		if disks := state.Get(constants.ArmAdditionalDiskVhds); disks != nil {
			dataDisks = disks.([]string)
		}
		for i, additionaldisk := range dataDisks {
			s.say(fmt.Sprintf(" Deleting Additional Disk -> %d: '%s'", i+1, additionaldisk))

			err := s.deleteImage(context.TODO(), additionaldisk, resourceGroupName, isManagedDisk)
			if err != nil {
				s.say("Failed to delete the managed Additional Disk!")
			}
		}

	}
}

func (s *StepDeployTemplate) deployTemplate(ctx context.Context, resourceGroupName string, deploymentName string) error {
	deployment, err := s.factory(s.config)
	if err != nil {
		return err
	}

	f, err := s.client.DeploymentsClient.CreateOrUpdate(ctx, resourceGroupName, deploymentName, *deployment)
	if err != nil {
		s.say(s.client.LastError.Error())
		return err
	}

	err = f.WaitForCompletionRef(ctx, s.client.DeploymentsClient.Client)
	if err != nil {
		s.say(s.client.LastError.Error())
	}

	return err
}

func (s *StepDeployTemplate) deleteDeploymentObject(ctx context.Context, state multistep.StateBag) error {
	deploymentName := s.name
	resourceGroupName := state.Get(constants.ArmResourceGroupName).(string)
	ui := state.Get("ui").(packersdk.Ui)

	ui.Say(fmt.Sprintf("Removing the created Deployment object: '%s'", deploymentName))
	f, err := s.client.DeploymentsClient.Delete(ctx, resourceGroupName, deploymentName)
	if err != nil {
		return err
	}

	return f.WaitForCompletionRef(ctx, s.client.DeploymentsClient.Client)
}

func (s *StepDeployTemplate) getImageDetails(ctx context.Context, resourceGroupName string, computeName string) (string, string, error) {
	//We can't depend on constants.ArmOSDiskVhd being set
	var imageName, imageType string
	vm, err := s.client.VirtualMachinesClient.Get(ctx, resourceGroupName, computeName, "")
	if err != nil {
		return imageName, imageType, err
	}

	if vm.StorageProfile.OsDisk.Vhd != nil {
		imageType = "image"
		imageName = *vm.StorageProfile.OsDisk.Vhd.URI
		return imageType, imageName, nil
	}

	if vm.StorageProfile.OsDisk.ManagedDisk.ID == nil {
		return "", "", fmt.Errorf("unable to obtain a OS disk for %q, please check that the instance has been created", computeName)
	}

	imageType = "Microsoft.Compute/disks"
	imageName = *vm.StorageProfile.OsDisk.ManagedDisk.ID

	return imageType, imageName, nil
}

func deleteResource(ctx context.Context, client *AzureClient, resourceType string, resourceName string, resourceGroupName string) error {
	switch resourceType {
	case "Microsoft.Compute/virtualMachines":
		forcedelete := false
		f, err := client.VirtualMachinesClient.Delete(ctx, resourceGroupName, resourceName, &forcedelete)
		if err == nil {
			err = f.WaitForCompletionRef(ctx, client.VirtualMachinesClient.Client)
		}
		return err
	case "Microsoft.KeyVault/vaults":
		_, err := client.VaultClientDelete.Delete(ctx, resourceGroupName, resourceName)
		return err
	case "Microsoft.Network/networkInterfaces":
		f, err := client.InterfacesClient.Delete(ctx, resourceGroupName, resourceName)
		if err == nil {
			err = f.WaitForCompletionRef(ctx, client.InterfacesClient.Client)
		}
		return err
	case "Microsoft.Network/virtualNetworks":
		f, err := client.VirtualNetworksClient.Delete(ctx, resourceGroupName, resourceName)
		if err == nil {
			err = f.WaitForCompletionRef(ctx, client.VirtualNetworksClient.Client)
		}
		return err
	case "Microsoft.Network/networkSecurityGroups":
		f, err := client.SecurityGroupsClient.Delete(ctx, resourceGroupName, resourceName)
		if err == nil {
			err = f.WaitForCompletionRef(ctx, client.SecurityGroupsClient.Client)
		}
		return err
	case "Microsoft.Network/publicIPAddresses":
		f, err := client.PublicIPAddressesClient.Delete(ctx, resourceGroupName, resourceName)
		if err == nil {
			err = f.WaitForCompletionRef(ctx, client.PublicIPAddressesClient.Client)
		}
		return err
	}
	return nil
}

func (s *StepDeployTemplate) deleteImage(ctx context.Context, imageName string, resourceGroupName string, isManagedDisk bool) error {
	// Managed disk
	if isManagedDisk {
		xs := strings.Split(imageName, "/")
		diskName := xs[len(xs)-1]
		f, err := s.client.DisksClient.Delete(ctx, resourceGroupName, diskName)
		if err == nil {
			err = f.WaitForCompletionRef(ctx, s.client.DisksClient.Client)
		}
		return err
	}

	// VHD image
	u, err := url.Parse(imageName)
	if err != nil {
		return err
	}
	xs := strings.Split(u.Path, "/")
	if len(xs) < 3 {
		return errors.New("Unable to parse path of image " + imageName)
	}
	var storageAccountName = xs[1]
	var blobName = strings.Join(xs[2:], "/")

	blob := s.client.BlobStorageClient.GetContainerReference(storageAccountName).GetBlobReference(blobName)
	_, err = blob.BreakLease(nil)
	if err != nil && !strings.Contains(err.Error(), "LeaseNotPresentWithLeaseOperation") {
		s.say(s.client.LastError.Error())
		return err
	}

	return blob.Delete(nil)
}

func (s *StepDeployTemplate) deleteDeploymentResources(ctx context.Context, deploymentName, resourceGroupName string) error {
	var maxResources int32 = 50
	deploymentOperations, err := s.client.DeploymentOperationsClient.ListComplete(ctx, resourceGroupName, deploymentName, &maxResources)
	if err != nil {
		s.reportIfError(err, resourceGroupName)
		return err
	}

	resources := map[string]string{}

	for deploymentOperations.NotDone() {
		deploymentOperation := deploymentOperations.Value()
		// Sometimes an empty operation is added to the list by Azure
		if deploymentOperation.Properties.TargetResource == nil {
			_ = deploymentOperations.Next()
			continue
		}

		resourceName := *deploymentOperation.Properties.TargetResource.ResourceName
		resourceType := *deploymentOperation.Properties.TargetResource.ResourceType

		s.say(fmt.Sprintf("Adding to deletion queue -> %s : '%s'", resourceType, resourceName))
		resources[resourceType] = resourceName

		if err = deploymentOperations.Next(); err != nil {
			return err
		}
	}

	var wg sync.WaitGroup
	wg.Add(len(resources))

	for resourceType, resourceName := range resources {
		go func(resourceType, resourceName string) {
			defer wg.Done()
			retryConfig := retry.Config{
				Tries:      10,
				RetryDelay: (&retry.Backoff{InitialBackoff: 10 * time.Second, MaxBackoff: 600 * time.Second, Multiplier: 2}).Linear,
			}

			err = retryConfig.Run(ctx, func(ctx context.Context) error {
				s.say(fmt.Sprintf("Attempting deletion -> %s : '%s'", resourceType, resourceName))
				err := deleteResource(ctx, s.client,
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
