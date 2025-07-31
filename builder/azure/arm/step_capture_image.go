// Copyright (c) HashiCorp, Inc.
// SPDX-License-Identifier: MPL-2.0

package arm

import (
	"context"
	"fmt"
	"net/http"

	"github.com/hashicorp/go-azure-helpers/resourcemanager/commonids"
	"github.com/hashicorp/go-azure-sdk/resource-manager/compute/2022-03-01/images"
	"github.com/hashicorp/go-azure-sdk/resource-manager/compute/2022-03-01/virtualmachines"
	"github.com/hashicorp/go-azure-sdk/resource-manager/compute/2022-03-02/disks"
	sdkclient "github.com/hashicorp/go-azure-sdk/sdk/client"
	"github.com/hashicorp/go-azure-sdk/sdk/client/pollers"
	"github.com/hashicorp/go-azure-sdk/sdk/client/resourcemanager"
	"github.com/hashicorp/go-azure-sdk/sdk/odata"
	"github.com/hashicorp/packer-plugin-azure/builder/azure/common/client"
	"github.com/hashicorp/packer-plugin-azure/builder/azure/common/constants"
	"github.com/hashicorp/packer-plugin-sdk/multistep"
	packersdk "github.com/hashicorp/packer-plugin-sdk/packer"
	"github.com/tombuildsstuff/giovanni/storage/2020-08-04/blob/blobs"
)

type StepCaptureImage struct {
	client              *AzureClient
	config              *Config
	generalizeVM        func(ctx context.Context, vmId virtualmachines.VirtualMachineId) error
	getVMInternalID     func(ctx context.Context, vmId virtualmachines.VirtualMachineId) (string, error)
	captureManagedImage func(ctx context.Context, subscriptionId string, resourceGroupName string, imageName string, parameters *images.Image) error
	grantAccess         func(ctx context.Context, subscriptionId string, resourceGroupName string, osDiskName string) (string, error)
	revokeAccess        func(ctx context.Context, subscriptionId string, resourceGroupName string, osDiskName string) error
	copyToStorage       func(ctx context.Context, storageContainerName string, captureNamePrefix string, osDiskName string, accessUri string) error
	say                 func(message string)
	error               func(e error)
}

type GrantAccessOperationResponse struct {
	Poller       pollers.Poller
	HttpResponse *http.Response
	OData        *odata.OData
	Model        *AccessUri
}

type AccessUri struct {
	StartTime  *string              `json:"startTime,omitempty"`
	EndTime    *string              `json:"endTime,omitempty"`
	Status     *string              `json:"status,omitempty"`
	Name       *string              `json:"name,omitempty"`
	Properties *AccessUriProperties `json:"properties,omitempty"`
}

type AccessUriProperties struct {
	Output *AccessUriOutput `json:"output,omitempty"`
}

type AccessUriOutput struct {
	AccessSAS *string `json:"accessSAS,omitempty"`
}

func NewStepCaptureImage(client *AzureClient, ui packersdk.Ui, config *Config) *StepCaptureImage {
	var step = &StepCaptureImage{
		config: config,
		client: client,
		say: func(message string) {
			ui.Say(message)
		},
		error: func(e error) {
			ui.Error(e.Error())
		},
	}

	step.generalizeVM = step.generalize
	step.captureManagedImage = step.captureImageFromVM
	step.getVMInternalID = step.getVMID
	step.grantAccess = step.grantDiskAccess
	step.revokeAccess = step.revokeDiskAccess
	step.copyToStorage = step.copyVhdToStorage
	return step
}

func (s *StepCaptureImage) generalize(ctx context.Context, vmId virtualmachines.VirtualMachineId) error {
	pollingContext, cancel := context.WithTimeout(ctx, s.client.PollingDuration)
	defer cancel()
	_, err := s.client.Generalize(pollingContext, vmId)
	if err != nil {
		s.say(s.client.LastError.Error())
	}
	return err
}

func (s *StepCaptureImage) captureImageFromVM(ctx context.Context, subscriptionId string, resourceGroupName string, imageName string, image *images.Image) error {
	pollingContext, cancel := context.WithTimeout(ctx, s.client.PollingDuration)
	defer cancel()
	id := images.NewImageID(subscriptionId, resourceGroupName, imageName)
	err := s.client.ImagesClient.CreateOrUpdateThenPoll(pollingContext, id, *image)
	if err != nil {
		s.say(s.client.LastError.Error())
	}
	return err
}

func (s *StepCaptureImage) getVMID(ctx context.Context, vmId virtualmachines.VirtualMachineId) (string, error) {
	pollingContext, cancel := context.WithTimeout(ctx, s.client.PollingDuration)
	defer cancel()
	vmResponse, err := s.client.VirtualMachinesClient.Get(pollingContext, vmId, virtualmachines.DefaultGetOperationOptions())
	if err != nil {
		return "", err
	}
	if vmResponse.Model != nil {
		vmId := vmResponse.Model.Properties.VMId
		return *vmId, nil
	}
	return "", client.NullModelSDKErr
}

func (s *StepCaptureImage) Run(ctx context.Context, state multistep.StateBag) multistep.StepAction {
	var computeName = state.Get(constants.ArmComputeName).(string)
	var location = state.Get(constants.ArmLocation).(string)
	var resourceGroupName = state.Get(constants.ArmResourceGroupName).(string)
	var imageParameters = state.Get(constants.ArmImageParameters).(*images.Image)
	var subscriptionId = state.Get(constants.ArmSubscription).(string)
	var isVHDSaveToStorage = state.Get(constants.ArmIsVHDSaveToStorage).(bool)
	var isManagedImage = state.Get(constants.ArmIsManagedImage).(bool)
	var isSIGImage = state.Get(constants.ArmIsSIGImage).(bool)
	var skipGeneralization = state.Get(constants.ArmSharedImageGalleryDestinationSpecialized).(bool)

	vmId := virtualmachines.NewVirtualMachineID(subscriptionId, resourceGroupName, computeName)
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
			if err != nil {
				state.Put(constants.Error, err)
				s.error(err)

				return multistep.ActionHalt
			}
		}

		if isVHDSaveToStorage {
			// VHD Builds are created with a field called the VMId in its name
			// Get that ID before capturing the VM so that we know where the resultant VHD is stored
			vmInternalID, err := s.getVMInternalID(ctx, vmId)
			if err != nil {
				err = fmt.Errorf("failed to get build VM before capturing image with err : %s", err)
				state.Put(constants.Error, err)
				s.error(err)
				return multistep.ActionHalt
			}

			s.say(fmt.Sprintf(" -> VM Internal ID            : '%s'", vmInternalID))
			state.Put(constants.ArmBuildVMInternalId, vmInternalID)

			s.say("OS Disk ...")
			var osDiskName = s.config.tmpOSDiskName
			s.say(fmt.Sprintf(" -> osDiskName                : '%s'", osDiskName))
			err = s.captureVHD(ctx, subscriptionId, resourceGroupName, osDiskName)
			if err != nil {
				err = fmt.Errorf("failed to capture OS Disk with err : %s", err)
				state.Put(constants.Error, err)
				s.error(err)
				return multistep.ActionHalt
			}

			additionalDiskCount := len(s.config.AdditionalDiskSize)
			if additionalDiskCount > 0 {
				s.say("Data Disk ...")
				var dataDiskName = s.config.tmpDataDiskName
				s.say(fmt.Sprintf(" -> dataDiskName              : '%s'", dataDiskName))

				for i := 0; i < additionalDiskCount; i++ {
					subDataDiskName := fmt.Sprintf("%s-%d", dataDiskName, i+1)

					err = s.captureVHD(ctx, subscriptionId, resourceGroupName, subDataDiskName)
					if err != nil {
						err = fmt.Errorf("failed to capture data Disk with err : %s", err)
						state.Put(constants.Error, err)
						s.error(err)
						return multistep.ActionHalt
					}
				}
			}
		}

		if isSIGImage {
			return multistep.ActionContinue
		}
	}
	return multistep.ActionContinue
}

func (*StepCaptureImage) Cleanup(multistep.StateBag) {
}

func (s *StepCaptureImage) captureVHD(ctx context.Context, subscriptionId string, resourceGroupName string, diskName string) error {
	accessUri, err := s.grantAccess(ctx, subscriptionId, resourceGroupName, diskName)
	if err != nil {
		err = fmt.Errorf("failed to grant access with err : %s", err)
		return err
	}

	s.say(fmt.Sprintf(" -> accessUri                 : '%s'", accessUri))

	var storageContainerName = s.config.CaptureContainerName
	var captureNamePrefix = s.config.CaptureNamePrefix

	err = s.copyToStorage(ctx, storageContainerName, captureNamePrefix, diskName, accessUri)
	if err != nil {
		err = fmt.Errorf("failed to copy to storage with err : %s", err)
		return err
	}

	err = s.revokeAccess(ctx, subscriptionId, resourceGroupName, diskName)
	if err != nil {
		err = fmt.Errorf("failed to revoke access with err : %s", err)
		return err
	}

	return nil
}

func (s *StepCaptureImage) grantDiskAccess(ctx context.Context, subscriptionId string, resourceGroupName string, diskName string) (string, error) {
	pollingContext, cancel := context.WithTimeout(ctx, s.client.PollingDuration)
	defer cancel()

	diskID := commonids.NewManagedDiskID(subscriptionId, resourceGroupName, diskName)
	grantAccessData := disks.GrantAccessData{
		Access:            disks.AccessLevelRead,
		DurationInSeconds: 600,
	}

	opts := sdkclient.RequestOptions{
		ContentType: "application/json; charset=utf-8",
		ExpectedStatusCodes: []int{
			http.StatusAccepted,
			http.StatusOK,
		},
		HttpMethod: http.MethodPost,
		Path:       fmt.Sprintf("%s/beginGetAccess", diskID.ID()),
	}

	s.say("Capturing VHD ...")
	req, err := s.client.DisksClient.Client.NewRequest(pollingContext, opts)
	if err != nil {
		return "", err
	}

	if err = req.Marshal(grantAccessData); err != nil {
		return "", err
	}

	var resp *sdkclient.Response
	resp, err = req.Execute(ctx)

	var result GrantAccessOperationResponse

	if resp != nil {
		result.OData = resp.OData
		result.HttpResponse = resp.Response

		var model AccessUri
		result.Model = &model
	}
	if err != nil {
		return "", err
	}

	result.Poller, err = resourcemanager.PollerFromResponse(resp, s.client.DisksClient.Client)
	if err != nil {
		return "", err
	}

	if err := result.Poller.PollUntilDone(pollingContext); err != nil {
		return "", fmt.Errorf("polling after GrantAccess: %+v", err)
	}

	if err := result.Poller.FinalResult(result.Model); err != nil {
		return "", fmt.Errorf("performing FinalResult: %+v", err)
	}

	accessUri := result.Model.Properties.Output.AccessSAS

	return *accessUri, nil
}

func (s *StepCaptureImage) revokeDiskAccess(ctx context.Context, subscriptionId string, resourceGroupName string, diskName string) error {
	pollingContext, cancel := context.WithTimeout(ctx, s.client.PollingDuration)
	defer cancel()
	diskID := commonids.NewManagedDiskID(subscriptionId, resourceGroupName, diskName)

	s.say("Revoking access ...")
	err := s.client.DisksClient.RevokeAccessThenPoll(pollingContext, diskID)
	if err != nil {
		s.say(s.client.LastError.Error())
		return err
	}

	return nil
}

func (s *StepCaptureImage) copyVhdToStorage(ctx context.Context, storageContainerName string, captureNamePrefix string, diskName string, accessUri string) error {
	pollingContext, cancel := context.WithTimeout(ctx, s.client.PollingDuration)
	defer cancel()

	var vhdName = fmt.Sprintf("%s%s.vhd", captureNamePrefix, diskName)
	copyInput := blobs.CopyInput{
		CopySource: accessUri,
	}

	s.say("Copying VHD to Storage Account ...")
	s.say(fmt.Sprintf(" -> Storage Container Name    : '%s'", storageContainerName))
	s.say(fmt.Sprintf(" -> VHD Name                  : '%s'", vhdName))

	if err := s.client.GiovanniBlobClient.CopyAndWait(pollingContext, storageContainerName, vhdName, copyInput); err != nil {
		return fmt.Errorf("error copying: %s", err)
	}

	return nil
}
