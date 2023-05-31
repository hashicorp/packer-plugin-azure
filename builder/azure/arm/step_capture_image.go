// Copyright (c) HashiCorp, Inc.
// SPDX-License-Identifier: MPL-2.0

package arm

import (
	"context"
	"fmt"

	hashiImagesSDK "github.com/hashicorp/go-azure-sdk/resource-manager/compute/2022-03-01/images"
	hashiVMSDK "github.com/hashicorp/go-azure-sdk/resource-manager/compute/2022-03-01/virtualmachines"
	"github.com/hashicorp/packer-plugin-azure/builder/azure/common/constants"
	"github.com/hashicorp/packer-plugin-sdk/multistep"
	packersdk "github.com/hashicorp/packer-plugin-sdk/packer"
)

type StepCaptureImage struct {
	client              *AzureClient
	generalizeVM        func(vmId hashiVMSDK.VirtualMachineId) error
	captureVhd          func(ctx context.Context, vmId hashiVMSDK.VirtualMachineId, parameters *hashiVMSDK.VirtualMachineCaptureParameters) error
	captureManagedImage func(ctx context.Context, subscriptionId string, resourceGroupName string, imageName string, parameters *hashiImagesSDK.Image) error
	get                 func(client *AzureClient) *CaptureTemplate
	say                 func(message string)
	error               func(e error)
}

func NewStepCaptureImage(client *AzureClient, ui packersdk.Ui) *StepCaptureImage {
	var step = &StepCaptureImage{
		client: client,
		get: func(client *AzureClient) *CaptureTemplate {
			return client.Template
		},
		say: func(message string) {
			ui.Say(message)
		},
		error: func(e error) {
			ui.Error(e.Error())
		},
	}

	step.generalizeVM = step.generalize
	step.captureVhd = step.captureImage
	step.captureManagedImage = step.captureImageFromVM

	return step
}

func (s *StepCaptureImage) generalize(vmId hashiVMSDK.VirtualMachineId) error {
	_, err := s.client.Generalize(context.TODO(), vmId)
	if err != nil {
		s.say(s.client.LastError.Error())
	}
	return err
}

func (s *StepCaptureImage) captureImageFromVM(ctx context.Context, subscriptionId string, resourceGroupName string, imageName string, image *hashiImagesSDK.Image) error {
	id := hashiImagesSDK.NewImageID(subscriptionId, resourceGroupName, imageName)
	err := s.client.ImagesClient.CreateOrUpdateThenPoll(ctx, id, *image)
	if err != nil {
		s.say(s.client.LastError.Error())
	}
	return err
}

func (s *StepCaptureImage) captureImage(ctx context.Context, vmId hashiVMSDK.VirtualMachineId, parameters *hashiVMSDK.VirtualMachineCaptureParameters) error {
	if err := s.client.VirtualMachinesClient.CaptureThenPoll(ctx, vmId, *parameters); err != nil {
		s.say(s.client.LastError.Error())
		return err
	}
	return nil
}

func (s *StepCaptureImage) Run(ctx context.Context, state multistep.StateBag) multistep.StepAction {

	var computeName = state.Get(constants.ArmComputeName).(string)
	var location = state.Get(constants.ArmLocation).(string)
	var resourceGroupName = state.Get(constants.ArmResourceGroupName).(string)
	var vmCaptureParameters = state.Get(constants.ArmNewVirtualMachineCaptureParameters).(*hashiVMSDK.VirtualMachineCaptureParameters)
	var imageParameters = state.Get(constants.ArmImageParameters).(*hashiImagesSDK.Image)
	var subscriptionId = state.Get(constants.ArmSubscription).(string)
	var isManagedImage = state.Get(constants.ArmIsManagedImage).(bool)
	var isSIGImage = state.Get(constants.ArmIsSIGImage).(bool)
	var skipGeneralization = state.Get(constants.ArmSharedImageGalleryDestinationSpecialized).(bool)

	vmId := hashiVMSDK.NewVirtualMachineID(subscriptionId, resourceGroupName, computeName)
	s.say(fmt.Sprintf(" -> Compute ResourceGroupName : '%s'", resourceGroupName))
	s.say(fmt.Sprintf(" -> Compute Name              : '%s'", computeName))
	s.say(fmt.Sprintf(" -> Compute Location          : '%s'", location))

	if skipGeneralization {
		s.say("Skipping generalization of Compute Gallery Image")
	} else {
		s.say("Generalizing machine ...")
		err := s.generalizeVM(vmId)

		if err == nil {
			if isManagedImage {
				s.say("Capturing image ...")
				var targetManagedImageResourceGroupName = state.Get(constants.ArmManagedImageResourceGroupName).(string)
				var targetManagedImageName = state.Get(constants.ArmManagedImageName).(string)
				var targetManagedImageLocation = state.Get(constants.ArmLocation).(string)
				s.say(fmt.Sprintf(" -> Image ResourceGroupName   : '%s'", targetManagedImageResourceGroupName))
				s.say(fmt.Sprintf(" -> Image Name                : '%s'", targetManagedImageName))
				s.say(fmt.Sprintf(" -> Image Location            : '%s'", targetManagedImageLocation))
				err = s.captureManagedImage(ctx, subscriptionId, targetManagedImageResourceGroupName, targetManagedImageName, imageParameters)
			} else if isSIGImage {
				// It's possible to create SIG image
				return multistep.ActionContinue
			} else {
				s.say("Capturing VHD ...")
				err = s.captureVhd(ctx, vmId, vmCaptureParameters)
			}
		}
		if err != nil {
			state.Put(constants.Error, err)
			s.error(err)

			return multistep.ActionHalt
		}

		// HACK(chrboum): I do not like this.  The capture method should be returning this value
		// instead having to pass in another lambda.
		//
		// Having to resort to capturing the template via an inspector is hack, and once I can
		// resolve that I can cleanup this code too.  See the comments in azure_client.go for more
		// details.
		// [paulmey]: autorest.Future now has access to the last http.Response, but I'm not sure if
		// the body is still accessible.
		template := s.get(s.client)
		state.Put(constants.ArmCaptureTemplate, template)
	}
	return multistep.ActionContinue
}

func (*StepCaptureImage) Cleanup(multistep.StateBag) {
}
