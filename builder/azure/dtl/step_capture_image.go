// Copyright (c) HashiCorp, Inc.
// SPDX-License-Identifier: MPL-2.0

package dtl

import (
	"context"
	"fmt"

	"github.com/hashicorp/go-azure-sdk/resource-manager/devtestlab/2018-09-15/customimages"
	"github.com/hashicorp/packer-plugin-azure/builder/azure/common/constants"
	"github.com/hashicorp/packer-plugin-sdk/multistep"
	packersdk "github.com/hashicorp/packer-plugin-sdk/packer"
)

type StepCaptureImage struct {
	client              *AzureClient
	captureManagedImage func(ctx context.Context) error
	config              *Config
	say                 func(message string)
	error               func(e error)
}

func NewStepCaptureImage(client *AzureClient, ui packersdk.Ui, config *Config) *StepCaptureImage {
	var step = &StepCaptureImage{
		client: client,
		config: config,
		say: func(message string) {
			ui.Say(message)
		},
		error: func(e error) {
			ui.Error(e.Error())
		},
	}

	// step.captureVhd = step.captureImage
	step.captureManagedImage = step.captureImageFromVM

	return step
}

func (s *StepCaptureImage) captureImageFromVM(ctx context.Context) error {
	imageID := fmt.Sprintf("/subscriptions/%s/resourceGroups/%s/providers/Microsoft.DevTestLab/labs/%s/virtualMachines/%s",
		s.config.ClientConfig.SubscriptionID,
		s.config.tmpResourceGroupName,
		s.config.LabName,
		s.config.tmpComputeName)

	customImageProperties := customimages.CustomImageProperties{}

	if s.config.OSType == constants.Target_Linux {
		deprovision := customimages.LinuxOsStateDeprovisionRequested
		if s.config.SkipSysprep {
			deprovision = customimages.LinuxOsStateDeprovisionApplied
		}
		customImageProperties = customimages.CustomImageProperties{
			VM: &customimages.CustomImagePropertiesFromVM{
				LinuxOsInfo: &customimages.LinuxOsInfo{
					LinuxOsState: &deprovision,
				},
				SourceVMId: &imageID,
			},
		}
	} else if s.config.OSType == constants.Target_Windows {
		deprovision := customimages.WindowsOsStateSysprepRequested
		if s.config.SkipSysprep {
			deprovision = customimages.WindowsOsStateSysprepApplied
		}
		customImageProperties = customimages.CustomImageProperties{
			VM: &customimages.CustomImagePropertiesFromVM{
				WindowsOsInfo: &customimages.WindowsOsInfo{
					WindowsOsState: &deprovision,
				},
				SourceVMId: &imageID,
			},
		}
	}

	customImage := &customimages.CustomImage{
		Name:       &s.config.ManagedImageName,
		Properties: customImageProperties,
	}

	customImageId := customimages.NewCustomImageID(s.config.ClientConfig.SubscriptionID, s.config.LabResourceGroupName, s.config.LabName, s.config.ManagedImageName)
	pollingContext, cancel := context.WithTimeout(ctx, s.client.CustomImageCaptureTimeout)
	defer cancel()
	err := s.client.DtlMetaClient.CustomImages.CreateOrUpdateThenPoll(pollingContext, customImageId, *customImage)
	if err != nil {
		s.say("Error from Capture Image")
		s.say(s.client.LastError.Error())
	}

	return err
}

func (s *StepCaptureImage) Run(ctx context.Context, state multistep.StateBag) multistep.StepAction {
	s.say("Capturing image ...")

	var computeName = state.Get(constants.ArmComputeName).(string)
	var location = state.Get(constants.ArmLocation).(string)
	var resourceGroupName = state.Get(constants.ArmResourceGroupName).(string)

	s.say(fmt.Sprintf(" -> Compute ResourceGroupName : '%s'", resourceGroupName))
	s.say(fmt.Sprintf(" -> Compute Name              : '%s'", computeName))
	s.say(fmt.Sprintf(" -> Compute Location          : '%s'", location))

	err := s.captureImageFromVM(ctx)

	if err != nil {
		s.error(err)
		state.Put(constants.Error, err)

		return multistep.ActionHalt
	}

	return multistep.ActionContinue
}

func (*StepCaptureImage) Cleanup(multistep.StateBag) {
}
