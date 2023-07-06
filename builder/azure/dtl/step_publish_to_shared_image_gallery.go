// Copyright (c) HashiCorp, Inc.
// SPDX-License-Identifier: MPL-2.0

package dtl

import (
	"context"
	"fmt"

	hashiGalleryImageVersionsSDK "github.com/hashicorp/go-azure-sdk/resource-manager/compute/2022-03-03/galleryimageversions"
	"github.com/hashicorp/packer-plugin-azure/builder/azure/common/constants"
	"github.com/hashicorp/packer-plugin-sdk/multistep"
	packersdk "github.com/hashicorp/packer-plugin-sdk/packer"
)

type StepPublishToSharedImageGallery struct {
	client  *AzureClient
	publish func(ctx context.Context, subscriptionID, managedImageID, sigDestinationResourceGroup, sigDestinationGalleryName, sigDestinationImageName, sigDestinationImageVersion string, sigReplicationRegions []string, location string, tags map[string]string) (string, error)
	say     func(message string)
	error   func(e error)
	toSIG   func() bool
}

func NewStepPublishToSharedImageGallery(client *AzureClient, ui packersdk.Ui, config *Config) *StepPublishToSharedImageGallery {
	var step = &StepPublishToSharedImageGallery{
		client: client,
		say: func(message string) {
			ui.Say(message)
		},
		error: func(e error) {
			ui.Error(e.Error())
		},
		toSIG: func() bool {
			return config.isManagedImage() && config.SharedGalleryDestination.SigDestinationGalleryName != ""
		},
	}

	step.publish = step.publishToSig
	return step
}

func (s *StepPublishToSharedImageGallery) publishToSig(ctx context.Context, subscriptionID, managedImageID, sigDestinationResourceGroup, sigDestinationGalleryName, sigDestinationImageName, sigDestinationImageVersion string, sigReplicationRegions []string, location string, tags map[string]string) (string, error) {

	replicationRegions := make([]hashiGalleryImageVersionsSDK.TargetRegion, len(sigReplicationRegions))
	for i, v := range sigReplicationRegions {
		regionName := v
		replicationRegions[i] = hashiGalleryImageVersionsSDK.TargetRegion{Name: regionName}
	}

	galleryImageVersion := hashiGalleryImageVersionsSDK.GalleryImageVersion{
		Location: location,
		Tags:     &tags,
		Properties: &hashiGalleryImageVersionsSDK.GalleryImageVersionProperties{
			StorageProfile: hashiGalleryImageVersionsSDK.GalleryImageVersionStorageProfile{
				Source: &hashiGalleryImageVersionsSDK.GalleryArtifactVersionFullSource{
					Id: &managedImageID,
				},
			},
			PublishingProfile: &hashiGalleryImageVersionsSDK.GalleryArtifactPublishingProfileBase{
				TargetRegions: &replicationRegions,
			},
		},
	}

	galleryImageVersionId := hashiGalleryImageVersionsSDK.NewImageVersionID(subscriptionID, sigDestinationResourceGroup, sigDestinationGalleryName, sigDestinationImageName, sigDestinationImageVersion)
	err := s.client.GalleryImageVersionsClient.CreateOrUpdateThenPoll(ctx, galleryImageVersionId, galleryImageVersion)

	if err != nil {
		s.say(s.client.LastError.Error())
		return "", err
	}
	createdSIGImageVersion, err := s.client.GalleryImageVersionsClient.Get(ctx, galleryImageVersionId, hashiGalleryImageVersionsSDK.DefaultGetOperationOptions())

	if err != nil {
		s.say(s.client.LastError.Error())
		return "", err
	}

	s.say(fmt.Sprintf(" -> Shared Gallery Image Version ID : '%s'", *(createdSIGImageVersion.Model.Id)))
	return *(createdSIGImageVersion.Model.Id), nil
}

func (s *StepPublishToSharedImageGallery) Run(ctx context.Context, stateBag multistep.StateBag) multistep.StepAction {
	if !s.toSIG() {
		return multistep.ActionContinue
	}

	s.say("Publishing to Shared Image Gallery ...")

	var miSigPubRg = stateBag.Get(constants.ArmManagedImageSigPublishResourceGroup).(string)
	var miSIGalleryName = stateBag.Get(constants.ArmManagedImageSharedGalleryName).(string)
	var miSGImageName = stateBag.Get(constants.ArmManagedImageSharedGalleryImageName).(string)
	var miSGImageVersion = stateBag.Get(constants.ArmManagedImageSharedGalleryImageVersion).(string)
	var location = stateBag.Get(constants.ArmLocation).(string)
	var tags = stateBag.Get(constants.ArmNewSDKTags).(map[string]string)
	var miSigReplicationRegions = stateBag.Get(constants.ArmManagedImageSharedGalleryReplicationRegions).([]string)
	var targetManagedImageResourceGroupName = stateBag.Get(constants.ArmManagedImageResourceGroupName).(string)
	var targetManagedImageName = stateBag.Get(constants.ArmManagedImageName).(string)
	var managedImageSubscription = stateBag.Get(constants.ArmManagedImageSubscription).(string)
	var managedImageID = fmt.Sprintf("/subscriptions/%s/resourceGroups/%s/providers/Microsoft.Compute/images/%s", managedImageSubscription, targetManagedImageResourceGroupName, targetManagedImageName)

	s.say(fmt.Sprintf(" -> MDI ID used for SIG publish     : '%s'", managedImageID))
	s.say(fmt.Sprintf(" -> SIG publish resource group     : '%s'", miSigPubRg))
	s.say(fmt.Sprintf(" -> SIG gallery name     : '%s'", miSIGalleryName))
	s.say(fmt.Sprintf(" -> SIG image name     : '%s'", miSGImageName))
	s.say(fmt.Sprintf(" -> SIG image version     : '%s'", miSGImageVersion))
	s.say(fmt.Sprintf(" -> SIG replication regions    : '%v'", miSigReplicationRegions))
	createdGalleryImageVersionID, err := s.publish(ctx, managedImageSubscription, managedImageID, miSigPubRg, miSIGalleryName, miSGImageName, miSGImageVersion, miSigReplicationRegions, location, tags)

	if err != nil {
		stateBag.Put(constants.Error, err)
		s.error(err)

		return multistep.ActionHalt
	}

	stateBag.Put(constants.ArmManagedImageSharedGalleryId, createdGalleryImageVersionID)
	return multistep.ActionContinue
}

func (*StepPublishToSharedImageGallery) Cleanup(multistep.StateBag) {
}
