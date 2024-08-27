// Copyright (c) HashiCorp, Inc.
// SPDX-License-Identifier: MPL-2.0

package arm

import (
	"context"
	"errors"
	"fmt"

	"github.com/hashicorp/go-azure-sdk/resource-manager/compute/2022-03-01/images"
	"github.com/hashicorp/go-azure-sdk/resource-manager/compute/2023-07-03/galleryimageversions"
	"github.com/hashicorp/packer-plugin-azure/builder/azure/common"
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

	// Maps to `target_region` blocks inside of `shared_image_gallery_destination` block
	TargetRegions []TargetRegion

	// Maps to `replication_region` field inside of `shared_image_gallery_destination` block
	// This field is ignored if TargetRegions is passed in
	ReplicatedRegons []string

	BuildDiskEncryptionSetId string
}

type PublishArgs struct {
	SubscriptionID     string
	SharedImageGallery SharedImageGalleryDestination
	EndOfLifeDate      string
	ExcludeFromLatest  bool
	ReplicaCount       *int64
	Location           string
	ReplicationMode    galleryimageversions.ReplicationMode
	Tags               map[string]string
	GallerySource      galleryimageversions.GalleryArtifactVersionFullSource
	TargetRegions      []galleryimageversions.TargetRegion
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
	step.TargetRegions = config.SharedGalleryDestination.SigDestinationTargetRegions
	step.ReplicatedRegons = config.SharedGalleryDestination.SigDestinationReplicationRegions
	step.BuildDiskEncryptionSetId = config.DiskEncryptionSetId
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
	storageAccountType := state.Get(constants.ArmManagedImageSharedGalleryImageVersionStorageAccountType).(string)

	confidentialVMEncryptionType, ok := state.Get(constants.ArmSharedImageGalleryDestinationConfidentialVMImageEncryptionType).(string)
	if !ok {
		confidentialVMEncryptionType = ""
	}

	return SharedImageGalleryDestination{
		SigDestinationSubscription:                      subscription,
		SigDestinationResourceGroup:                     resourceGroup,
		SigDestinationGalleryName:                       galleryName,
		SigDestinationImageName:                         imageName,
		SigDestinationImageVersion:                      imageVersion,
		SigDestinationStorageAccountType:                storageAccountType,
		SigDestinationConfidentialVMImageEncryptionType: confidentialVMEncryptionType,
	}
}

func (s *StepPublishToSharedImageGallery) publishToSig(ctx context.Context, args PublishArgs) (string, error) {
	storageAccountType, err := getSigDestinationStorageAccountType(args.SharedImageGallery.SigDestinationStorageAccountType)
	if err != nil {
		s.error(err)
		return "", err
	}

	galleryImageVersion := galleryimageversions.GalleryImageVersion{
		Location: args.Location,
		Tags:     &args.Tags,
		Properties: &galleryimageversions.GalleryImageVersionProperties{
			StorageProfile: galleryimageversions.GalleryImageVersionStorageProfile{
				Source: &args.GallerySource,
			},
			PublishingProfile: &galleryimageversions.GalleryArtifactPublishingProfileBase{
				TargetRegions:      &args.TargetRegions,
				EndOfLifeDate:      &args.EndOfLifeDate,
				ExcludeFromLatest:  &args.ExcludeFromLatest,
				ReplicaCount:       args.ReplicaCount,
				ReplicationMode:    &args.ReplicationMode,
				StorageAccountType: &storageAccountType,
			},
		},
	}

	publishSigContext, publishSigCancel := context.WithTimeout(ctx, s.client.SharedGalleryTimeout)
	defer publishSigCancel()
	galleryImageVersionId := galleryimageversions.NewImageVersionID(args.SubscriptionID, args.SharedImageGallery.SigDestinationResourceGroup, args.SharedImageGallery.SigDestinationGalleryName, args.SharedImageGallery.SigDestinationImageName, args.SharedImageGallery.SigDestinationImageVersion)
	err = s.client.GalleryImageVersionsClient.CreateOrUpdateThenPoll(publishSigContext, galleryImageVersionId, galleryImageVersion)
	if err != nil {
		s.say(s.client.LastError.Error())
		return "", err
	}

	pollingContext, pollingCancel := context.WithTimeout(ctx, s.client.PollingDuration)
	defer pollingCancel()
	createdSGImageVersion, err := s.client.GalleryImageVersionsClient.Get(pollingContext, galleryImageVersionId, galleryimageversions.DefaultGetOperationOptions())

	if err != nil {
		s.say(s.client.LastError.Error())
		return "", err
	}

	s.say(fmt.Sprintf(" -> Successfully Created Shared Gallery Image Version ID : '%s'", *(createdSGImageVersion.Model.Id)))
	return *(createdSGImageVersion.Model.Id), nil
}

func (s *StepPublishToSharedImageGallery) Run(ctx context.Context, stateBag multistep.StateBag) multistep.StepAction {
	if !s.toSIG() {
		return multistep.ActionContinue
	}

	s.say("Preparing to publish to Shared Image Gallery ...")

	location := stateBag.Get(constants.ArmLocation).(string)
	tags := stateBag.Get(constants.ArmTags).(map[string]string)

	sharedImageGallery := getSigDestination(stateBag)
	sharedImageGallery.SigDestinationTargetRegions = s.TargetRegions
	var sourceID string

	gallerySource := galleryimageversions.GalleryArtifactVersionFullSource{}
	var isManagedImage = stateBag.Get(constants.ArmIsManagedImage).(bool)
	if isManagedImage {
		targetManagedImageResourceGroupName := stateBag.Get(constants.ArmManagedImageResourceGroupName).(string)
		targetManagedImageName := stateBag.Get(constants.ArmManagedImageName).(string)

		managedImageSubscription := stateBag.Get(constants.ArmManagedImageSubscription).(string)
		sourceID = fmt.Sprintf("/subscriptions/%s/resourceGroups/%s/providers/Microsoft.Compute/images/%s", managedImageSubscription, targetManagedImageResourceGroupName, targetManagedImageName)
		gallerySource.Id = &sourceID
	} else {
		var imageParameters = stateBag.Get(constants.ArmImageParameters).(*images.Image)
		sourceID = *imageParameters.Properties.SourceVirtualMachine.Id
		gallerySource.VirtualMachineId = &sourceID
	}

	miSGImageVersionEndOfLifeDate, _ := stateBag.Get(constants.ArmManagedImageSharedGalleryImageVersionEndOfLifeDate).(string)
	miSGImageVersionExcludeFromLatest, _ := stateBag.Get(constants.ArmManagedImageSharedGalleryImageVersionExcludeFromLatest).(bool)
	var defaultReplicaCount *int64
	miSigReplicaCount, _ := stateBag.Get(constants.ArmManagedImageSharedGalleryImageVersionReplicaCount).(int64)
	if miSigReplicaCount > 0 {
		if miSigReplicaCount > constants.SharedImageGalleryImageVersionDefaultMaxReplicaCount {
			miSigReplicaCount = constants.SharedImageGalleryImageVersionDefaultMaxReplicaCount
		}
		defaultReplicaCount = common.Int64Ptr(miSigReplicaCount)
	}

	s.say(fmt.Sprintf(" -> Source ID used for SIG publish        : '%s'", sourceID))
	s.say(fmt.Sprintf(" -> SIG publish resource group            : '%s'", sharedImageGallery.SigDestinationResourceGroup))
	s.say(fmt.Sprintf(" -> SIG gallery name                      : '%s'", sharedImageGallery.SigDestinationGalleryName))
	s.say(fmt.Sprintf(" -> SIG image name                        : '%s'", sharedImageGallery.SigDestinationImageName))
	s.say(fmt.Sprintf(" -> SIG image version                     : '%s'", sharedImageGallery.SigDestinationImageVersion))
	if defaultReplicaCount != nil {
		s.say(fmt.Sprintf(" -> SIG default Replica Count             : '%d'", *defaultReplicaCount))
	}
	if sharedImageGallery.SigDestinationConfidentialVMImageEncryptionType != "" {
		s.say(fmt.Sprintf(" -> SIG Confidential VM Encryption Type   : '%s'", sharedImageGallery.SigDestinationConfidentialVMImageEncryptionType))
	}
	if sharedImageGallery.SigDestinationStorageAccountType != "" {
		s.say(fmt.Sprintf(" -> SIG storage account type              : '%s'", sharedImageGallery.SigDestinationStorageAccountType))
	}
	if miSGImageVersionEndOfLifeDate != "" {
		s.say(fmt.Sprintf(" -> SIG image version endoflife date      : '%s'", miSGImageVersionEndOfLifeDate))
	}
	imageVersionRegions, ok := stateBag.Get(constants.ArmSharedImageGalleryDestinationTargetRegions).([]galleryimageversions.TargetRegion)
	if !ok {
		// TODO
		err := errors.New("failed to parse TargetRegions, this is always a Packer Azure plugin bug")
		stateBag.Put(constants.Error, err)
		s.error(err)
		return multistep.ActionHalt
	}
	s.say(fmt.Sprintf(" -> SIG image version exclude from latest : '%t'", miSGImageVersionExcludeFromLatest))
	s.say(" -> Target Regions")
	for _, targetRegion := range imageVersionRegions {
		s.say(fmt.Sprintf(" Normalized region name                : '%s'", targetRegion.Name))
		if targetRegion.RegionalReplicaCount != nil && *targetRegion.RegionalReplicaCount != 0 {
			s.say(fmt.Sprintf(" -> Replica count                         : '%d'", *targetRegion.RegionalReplicaCount))
		}

		// TODO
		//if len(targetRegion.DiskEncryptionSetId) > 0 {
		//	s.say(fmt.Sprintf(" -> Disk Encryption Set ID                : '%s'", targetRegion.DiskEncryptionSetId))
		//}
	}

	replicationMode := galleryimageversions.ReplicationModeFull
	shallowReplicationMode := stateBag.Get(constants.ArmSharedImageGalleryDestinationShallowReplication).(bool)
	if shallowReplicationMode {
		s.say(" -> Creating SIG Image with Shallow Replication")
		replicationMode = galleryimageversions.ReplicationModeShallow
	}
	subscriptionID := stateBag.Get(constants.ArmSharedImageGalleryDestinationSubscription).(string)
	s.say("Publishing to Shared Image Gallery ...")
	createdGalleryImageVersionID, err := s.publish(
		ctx,
		PublishArgs{
			SubscriptionID:     subscriptionID,
			SharedImageGallery: sharedImageGallery,
			EndOfLifeDate:      miSGImageVersionEndOfLifeDate,
			ExcludeFromLatest:  miSGImageVersionExcludeFromLatest,
			Location:           location,
			ReplicationMode:    replicationMode,
			Tags:               tags,
			ReplicaCount:       defaultReplicaCount,
			GallerySource:      gallerySource,
			TargetRegions:      imageVersionRegions,
		},
	)

	if err != nil {
		stateBag.Put(constants.Error, err)
		s.error(err)

		return multistep.ActionHalt
	}

	stateBag.Put(constants.ArmManagedImageSharedGalleryReplicationRegions, sharedImageGallery.SigDestinationReplicationRegions)
	stateBag.Put(constants.ArmManagedImageSharedGalleryId, createdGalleryImageVersionID)
	return multistep.ActionContinue
}

func (*StepPublishToSharedImageGallery) Cleanup(multistep.StateBag) {
}
