// Copyright (c) HashiCorp, Inc.
// SPDX-License-Identifier: MPL-2.0

package arm

import (
	"context"
	"fmt"

	"github.com/hashicorp/go-azure-sdk/resource-manager/compute/2022-03-01/images"
	"github.com/hashicorp/go-azure-sdk/resource-manager/compute/2022-03-03/galleryimageversions"
	"github.com/hashicorp/packer-plugin-azure/builder/azure/common/constants"
	"github.com/hashicorp/packer-plugin-sdk/multistep"
	packersdk "github.com/hashicorp/packer-plugin-sdk/packer"
)

type StepPublishToSharedImageGallery struct {
	client  *AzureClient
	publish func(ctx context.Context, args PublishArgs) (string, error)
	say     func(message string)
	error   func(e error)
	toSIG   func() bool
}

type PublishArgs struct {
	SubscriptionID      string
	SourceID            string
	SharedImageGallery  SharedImageGalleryDestination
	EndOfLifeDate       string
	ExcludeFromLatest   bool
	ReplicaCount        int64
	Location            string
	DiskEncryptionSetId string
	ReplicationMode     galleryimageversions.ReplicationMode
	Tags                map[string]string
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
			return config.isPublishToSIG()
		},
	}

	step.publish = step.publishToSig
	return step
}

func getSigDestinationStorageAccountType(s string) (galleryimageversions.StorageAccountType, error) {
	if s == "" {
		return galleryimageversions.StorageAccountTypeStandardLRS, nil
	}
	for _, t := range galleryimageversions.PossibleValuesForStorageAccountType() {
		if s == t {
			return galleryimageversions.StorageAccountType(t), nil
		}
	}
	return "", fmt.Errorf("not an accepted value for shared_image_gallery_destination.storage_account_type")
}

func getSigDestination(state multistep.StateBag) SharedImageGalleryDestination {
	subscription := state.Get(constants.ArmManagedImageSubscription).(string)
	resourceGroup := state.Get(constants.ArmManagedImageSigPublishResourceGroup).(string)
	galleryName := state.Get(constants.ArmManagedImageSharedGalleryName).(string)
	imageName := state.Get(constants.ArmManagedImageSharedGalleryImageName).(string)
	imageVersion := state.Get(constants.ArmManagedImageSharedGalleryImageVersion).(string)
	replicationRegions := state.Get(constants.ArmManagedImageSharedGalleryReplicationRegions).([]string)
	storageAccountType := state.Get(constants.ArmManagedImageSharedGalleryImageVersionStorageAccountType).(string)

	return SharedImageGalleryDestination{
		SigDestinationSubscription:       subscription,
		SigDestinationResourceGroup:      resourceGroup,
		SigDestinationGalleryName:        galleryName,
		SigDestinationImageName:          imageName,
		SigDestinationImageVersion:       imageVersion,
		SigDestinationReplicationRegions: replicationRegions,
		SigDestinationStorageAccountType: storageAccountType,
	}
}

func (s *StepPublishToSharedImageGallery) publishToSig(ctx context.Context, args PublishArgs) (string, error) {
	replicationRegions := make([]galleryimageversions.TargetRegion, len(args.SharedImageGallery.SigDestinationReplicationRegions))
	for i, v := range args.SharedImageGallery.SigDestinationReplicationRegions {
		regionName := v
		replicationRegions[i] = galleryimageversions.TargetRegion{Name: regionName}
	}

	storageAccountType, err := getSigDestinationStorageAccountType(args.SharedImageGallery.SigDestinationStorageAccountType)
	if err != nil {
		s.error(err)
		return "", err
	}

	if args.DiskEncryptionSetId != "" {
		for index, targetRegion := range replicationRegions {
			targetRegion.Encryption = &galleryimageversions.EncryptionImages{
				OsDiskImage: &galleryimageversions.OSDiskImageEncryption{
					DiskEncryptionSetId: &args.DiskEncryptionSetId,
				},
			}
			replicationRegions[index] = targetRegion
		}
	}

	galleryImageVersion := galleryimageversions.GalleryImageVersion{
		Location: args.Location,
		Tags:     &args.Tags,
		Properties: &galleryimageversions.GalleryImageVersionProperties{
			StorageProfile: galleryimageversions.GalleryImageVersionStorageProfile{
				Source: &galleryimageversions.GalleryArtifactVersionFullSource{
					Id: &args.SourceID,
				},
			},
			PublishingProfile: &galleryimageversions.GalleryArtifactPublishingProfileBase{
				TargetRegions:      &replicationRegions,
				EndOfLifeDate:      &args.EndOfLifeDate,
				ExcludeFromLatest:  &args.ExcludeFromLatest,
				ReplicaCount:       &args.ReplicaCount,
				ReplicationMode:    &args.ReplicationMode,
				StorageAccountType: &storageAccountType,
			},
		},
	}

	pollingContext, cancel := context.WithTimeout(ctx, s.client.SharedGalleryTimeout)
	defer cancel()
	galleryImageVersionId := galleryimageversions.NewImageVersionID(args.SubscriptionID, args.SharedImageGallery.SigDestinationResourceGroup, args.SharedImageGallery.SigDestinationGalleryName, args.SharedImageGallery.SigDestinationImageName, args.SharedImageGallery.SigDestinationImageVersion)
	err = s.client.GalleryImageVersionsClient.CreateOrUpdateThenPoll(pollingContext, galleryImageVersionId, galleryImageVersion)
	if err != nil {
		s.say(s.client.LastError.Error())
		return "", err
	}

	createdSGImageVersion, err := s.client.GalleryImageVersionsClient.Get(ctx, galleryImageVersionId, galleryimageversions.DefaultGetOperationOptions())

	if err != nil {
		s.say(s.client.LastError.Error())
		return "", err
	}

	s.say(fmt.Sprintf(" -> Shared Gallery Image Version ID : '%s'", *(createdSGImageVersion.Model.Id)))
	return *(createdSGImageVersion.Model.Id), nil
}

func (s *StepPublishToSharedImageGallery) Run(ctx context.Context, stateBag multistep.StateBag) multistep.StepAction {
	if !s.toSIG() {
		return multistep.ActionContinue
	}

	s.say("Publishing to Shared Image Gallery ...")

	location := stateBag.Get(constants.ArmLocation).(string)
	tags := stateBag.Get(constants.ArmTags).(map[string]string)

	sharedImageGallery := getSigDestination(stateBag)
	var sourceID string

	var isManagedImage = stateBag.Get(constants.ArmIsManagedImage).(bool)
	if isManagedImage {
		targetManagedImageResourceGroupName := stateBag.Get(constants.ArmManagedImageResourceGroupName).(string)
		targetManagedImageName := stateBag.Get(constants.ArmManagedImageName).(string)

		managedImageSubscription := stateBag.Get(constants.ArmManagedImageSubscription).(string)
		sourceID = fmt.Sprintf("/subscriptions/%s/resourceGroups/%s/providers/Microsoft.Compute/images/%s", managedImageSubscription, targetManagedImageResourceGroupName, targetManagedImageName)
	} else {
		var imageParameters = stateBag.Get(constants.ArmImageParameters).(*images.Image)
		sourceID = *imageParameters.Properties.SourceVirtualMachine.Id
	}

	miSGImageVersionEndOfLifeDate, _ := stateBag.Get(constants.ArmManagedImageSharedGalleryImageVersionEndOfLifeDate).(string)
	miSGImageVersionExcludeFromLatest, _ := stateBag.Get(constants.ArmManagedImageSharedGalleryImageVersionExcludeFromLatest).(bool)
	miSigReplicaCount, _ := stateBag.Get(constants.ArmManagedImageSharedGalleryImageVersionReplicaCount).(int64)
	// Replica count must be between 1 and 100 inclusive
	if miSigReplicaCount <= 0 {
		miSigReplicaCount = constants.SharedImageGalleryImageVersionDefaultMinReplicaCount
	} else if miSigReplicaCount > constants.SharedImageGalleryImageVersionDefaultMaxReplicaCount {
		miSigReplicaCount = constants.SharedImageGalleryImageVersionDefaultMaxReplicaCount
	}

	var diskEncryptionSetId string
	if _, ok := stateBag.GetOk(constants.ArmBuildDiskEncryptionSetId); ok {
		diskEncryptionSetId = stateBag.Get(constants.ArmBuildDiskEncryptionSetId).(string)
	}

	s.say(fmt.Sprintf(" -> Source ID used for SIG publish        : '%s'", sourceID))
	s.say(fmt.Sprintf(" -> SIG publish resource group            : '%s'", sharedImageGallery.SigDestinationResourceGroup))
	s.say(fmt.Sprintf(" -> SIG gallery name                      : '%s'", sharedImageGallery.SigDestinationGalleryName))
	s.say(fmt.Sprintf(" -> SIG image name                        : '%s'", sharedImageGallery.SigDestinationImageName))
	s.say(fmt.Sprintf(" -> SIG image version                     : '%s'", sharedImageGallery.SigDestinationImageVersion))
	if diskEncryptionSetId != "" {
		s.say(fmt.Sprintf(" -> SIG Encryption Set : %s", diskEncryptionSetId))
	}
	s.say(fmt.Sprintf(" -> SIG replication regions               : '%v'", sharedImageGallery.SigDestinationReplicationRegions))
	s.say(fmt.Sprintf(" -> SIG storage account type              : '%s'", sharedImageGallery.SigDestinationStorageAccountType))
	s.say(fmt.Sprintf(" -> SIG image version endoflife date      : '%s'", miSGImageVersionEndOfLifeDate))
	s.say(fmt.Sprintf(" -> SIG image version exclude from latest : '%t'", miSGImageVersionExcludeFromLatest))
	s.say(fmt.Sprintf(" -> SIG replica count [1, 100]            : '%d'", miSigReplicaCount))
	replicationMode := galleryimageversions.ReplicationModeFull
	shallowReplicationMode := stateBag.Get(constants.ArmSharedImageGalleryDestinationShallowReplication).(bool)
	if shallowReplicationMode {
		s.say(" -> Creating SIG Image with Shallow Replication")
		replicationMode = galleryimageversions.ReplicationModeShallow
	}
	subscriptionID := stateBag.Get(constants.ArmSharedImageGalleryDestinationSubscription).(string)
	createdGalleryImageVersionID, err := s.publish(
		ctx,
		PublishArgs{
			SubscriptionID:      subscriptionID,
			SourceID:            sourceID,
			SharedImageGallery:  sharedImageGallery,
			EndOfLifeDate:       miSGImageVersionEndOfLifeDate,
			ExcludeFromLatest:   miSGImageVersionExcludeFromLatest,
			ReplicaCount:        miSigReplicaCount,
			Location:            location,
			DiskEncryptionSetId: diskEncryptionSetId,
			ReplicationMode:     replicationMode,
			Tags:                tags,
		},
	)

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
