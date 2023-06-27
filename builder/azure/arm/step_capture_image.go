// Copyright (c) HashiCorp, Inc.
// SPDX-License-Identifier: MPL-2.0

package arm

import (
	"context"
	"errors"
	"fmt"

	hashiImagesSDK "github.com/hashicorp/go-azure-sdk/resource-manager/compute/2022-03-01/images"
	hashiVMSDK "github.com/hashicorp/go-azure-sdk/resource-manager/compute/2022-03-01/virtualmachines"
	"github.com/hashicorp/packer-plugin-azure/builder/azure/common/constants"
	"github.com/hashicorp/packer-plugin-sdk/multistep"
	packersdk "github.com/hashicorp/packer-plugin-sdk/packer"
)

type StepCaptureImage struct {
	client              *AzureClient
	generalizeVM        func(ctx context.Context, vmId hashiVMSDK.VirtualMachineId) error
	getVMInternalID     func(ctx context.Context, vmId hashiVMSDK.VirtualMachineId) (string, error)
	captureVhd          func(ctx context.Context, vmId hashiVMSDK.VirtualMachineId, parameters *hashiVMSDK.VirtualMachineCaptureParameters) error
	captureManagedImage func(ctx context.Context, subscriptionId string, resourceGroupName string, imageName string, parameters *hashiImagesSDK.Image) error
	say                 func(message string)
	error               func(e error)
}

func NewStepCaptureImage(client *AzureClient, ui packersdk.Ui) *StepCaptureImage {
	var step = &StepCaptureImage{
		client: client,
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
	step.getVMInternalID = step.getVMID
	return step
}

func (s *StepCaptureImage) generalize(ctx context.Context, vmId hashiVMSDK.VirtualMachineId) error {
	_, err := s.client.Generalize(ctx, vmId)
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

func (s *StepCaptureImage) getVMID(ctx context.Context, vmId hashiVMSDK.VirtualMachineId) (string, error) {
	vmResponse, err := s.client.VirtualMachinesClient.Get(ctx, vmId, hashiVMSDK.DefaultGetOperationOptions())
	if err != nil {
		return "", err
	}
	if vmResponse.Model != nil {
		vmId := vmResponse.Model.Properties.VMId
		return *vmId, nil
	}
	return "", nil
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
		err := s.generalizeVM(ctx, vmId)
		if err != nil {
			state.Put(constants.Error, err)
			s.error(err)

			return multistep.ActionHalt
		}
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
			// It's possible to create SIG image without a managed image
			return multistep.ActionContinue
		} else {
			// VHD Builds are created with a field called the VMId in its name
			// Get that ID before capturing the VM so that we know where the resultant VHD is stored
			vmInternalID, err := s.getVMInternalID(ctx, vmId)
			if err != nil {
				err = fmt.Errorf("Failed to get build VM before capturing image with err : %s", err)
				state.Put(constants.Error, err)
				s.error(err)
				return multistep.ActionHalt
			} else {
				if vmInternalID == "" {
					err = errors.New("Failed to get build VM before capturing image, Azure did not return the field VirtualMachine.Properties.VMId")
					state.Put(constants.Error, err)
					s.error(err)
					return multistep.ActionHalt
				} else {
					s.say(fmt.Sprintf(" -> VM Internal ID            : '%s'", vmInternalID))
					state.Put(constants.ArmBuildVMInternalId, vmInternalID)
					s.say("Capturing VHD ...")
					err = s.captureVhd(ctx, vmId, vmCaptureParameters)
					if err != nil {
						state.Put(constants.Error, err)
						s.error(err)
						return multistep.ActionHalt
					}
				}
			}
		}
	}
	return multistep.ActionContinue
}

func (s *StepCaptureImage) haltAndError(state multistep.StateBag, err error) multistep.StepAction {
	state.Put(constants.Error, err)
	s.error(err)

	return multistep.ActionHalt
}

func (*StepCaptureImage) Cleanup(multistep.StateBag) {
}
