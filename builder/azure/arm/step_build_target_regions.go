// Copyright (c) HashiCorp, Inc.
// SPDX-License-Identifier: MPL-2.0

package arm

import (
	"context"
	"fmt"
	"strings"

	"github.com/hashicorp/go-azure-sdk/resource-manager/compute/2023-07-03/galleryimageversions"
	"github.com/hashicorp/packer-plugin-azure/builder/azure/common"
	"github.com/hashicorp/packer-plugin-azure/builder/azure/common/constants"
	"github.com/hashicorp/packer-plugin-sdk/multistep"
	packersdk "github.com/hashicorp/packer-plugin-sdk/packer"
)

type StepBuildTargetRegions struct {
	client *AzureClient
	say    func(message string)
	error  func(e error)
	toSIG  func() bool

	// Maps to `target_region` blocks inside of `shared_image_gallery_destination` block
	TargetRegions []TargetRegion

	// Maps to `replication_region` field inside of `shared_image_gallery_destination` block
	// This field is ignored if TargetRegions is passed in
	ReplicatedRegons []string

	BuildDiskEncryptionSetId string
}

func NewStepBuildTargetRegions(client *AzureClient, ui packersdk.Ui, config *Config) *StepBuildTargetRegions {
	var step = &StepBuildTargetRegions{
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
	return step
}

func buildAzureImageTargetRegionsWithEncryption(diskEncryptionSetId string, confidentialVMImageEncryptionType string) *galleryimageversions.EncryptionImages {
	var e *galleryimageversions.EncryptionImages = nil
	var toReturnDiskEncryptionSetId *string = nil
	var toReturnSecureVMDiskEncryptionSetId *string = nil
	var toReturnConfidentialVMEncryptionType *galleryimageversions.ConfidentialVMEncryptionType = nil

	if diskEncryptionSetId != "" && confidentialVMImageEncryptionType == "" {
		// If the diskEncryptionSetId is set, but the confidentialVMImageEncryptionType is not set, then the image is encrypted with the diskEncryptionSetId
		toReturnDiskEncryptionSetId = common.StringPtr(diskEncryptionSetId)
	} else if diskEncryptionSetId != "" && confidentialVMImageEncryptionType == string(galleryimageversions.ConfidentialVMEncryptionTypeEncryptedWithCmk) {
		// If the diskEncryptionSetId is set and the confidentialVMImageEncryptionType is set to EncryptedWithCmk, then the cvm image will be encrypted with the diskEncryptionSetId in this target region
		toReturnSecureVMDiskEncryptionSetId = common.StringPtr(diskEncryptionSetId)
		cvmCmk := galleryimageversions.ConfidentialVMEncryptionTypeEncryptedWithCmk
		toReturnConfidentialVMEncryptionType = &cvmCmk
	} else if diskEncryptionSetId == "" && (confidentialVMImageEncryptionType == string(galleryimageversions.ConfidentialVMEncryptionTypeEncryptedVMGuestStateOnlyWithPmk) || confidentialVMImageEncryptionType == string(galleryimageversions.ConfidentialVMEncryptionTypeEncryptedWithPmk)) {
		// If the diskEncryptionSetId is not set and the confidentialVMImageEncryptionType is set to EncryptedVMGuestStateOnlyWithPmk or EncryptedWithPmk, then the cvm image will be encrypted with a PaaS key in this target region
		switch confidentialVMImageEncryptionType {
		case string(galleryimageversions.ConfidentialVMEncryptionTypeEncryptedVMGuestStateOnlyWithPmk):
			withPmkOnly := galleryimageversions.ConfidentialVMEncryptionTypeEncryptedVMGuestStateOnlyWithPmk
			toReturnConfidentialVMEncryptionType = &withPmkOnly
		case string(galleryimageversions.ConfidentialVMEncryptionTypeEncryptedWithPmk):
			withPmk := galleryimageversions.ConfidentialVMEncryptionTypeEncryptedWithPmk
			toReturnConfidentialVMEncryptionType = &withPmk
		}
	}

	if toReturnDiskEncryptionSetId != nil {
		e = &galleryimageversions.EncryptionImages{
			OsDiskImage: &galleryimageversions.OSDiskImageEncryption{
				DiskEncryptionSetId: toReturnDiskEncryptionSetId,
			},
		}
	}

	if toReturnConfidentialVMEncryptionType != nil {
		e = &galleryimageversions.EncryptionImages{
			OsDiskImage: &galleryimageversions.OSDiskImageEncryption{
				SecurityProfile: &galleryimageversions.OSDiskImageSecurityProfile{
					ConfidentialVMEncryptionType: toReturnConfidentialVMEncryptionType,
					SecureVMDiskEncryptionSetId:  toReturnSecureVMDiskEncryptionSetId,
				},
			},
		}
	}

	return e
}

func (s *StepBuildTargetRegions) publishToSig(ctx context.Context, args PublishArgs) (string, error) {
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

func (s *StepBuildTargetRegions) Run(ctx context.Context, stateBag multistep.StateBag) multistep.StepAction {
	if !s.toSIG() {
		return multistep.ActionContinue
	}

	location := stateBag.Get(constants.ArmLocation).(string)
	buildLocation := normalizeAzureRegion(location)
	var foundMandatoryReplicationRegion bool
	if len(s.TargetRegions) > 0 {
		normalizedRegions := make([]TargetRegion, 0, len(s.TargetRegions))
		for _, tr := range s.TargetRegions {
			tr.Name = normalizeAzureRegion(tr.Name)
			if strings.EqualFold(tr.Name, buildLocation) {
				foundMandatoryReplicationRegion = true
				if tr.DiskEncryptionSetId == "" && s.BuildDiskEncryptionSetId != "" {
					// TODO this is a behavior change but I think it makes sense
					s.say(fmt.Sprintf("Using build disk encryption set ID %s for publishing SIG image to build location %s", s.BuildDiskEncryptionSetId, buildLocation))
					tr.DiskEncryptionSetId = s.BuildDiskEncryptionSetId
				}
			}
			normalizedRegions = append(normalizedRegions, tr)
		}
		s.TargetRegions = normalizedRegions

	} else if len(s.ReplicatedRegons) > 0 {
		// Convert deprecated replication_regions to []TargetRegion
		normalizedRegions := make([]TargetRegion, 0, len(s.ReplicatedRegons))
		for _, region := range s.ReplicatedRegons {
			region := normalizeAzureRegion(region)
			if strings.EqualFold(region, buildLocation) {
				normalizedRegions = append(
					normalizedRegions, TargetRegion{
						Name: region,
						// Add the build disk encryption set ID to the build location
						DiskEncryptionSetId: s.BuildDiskEncryptionSetId,
					},
				)
				foundMandatoryReplicationRegion = true
				continue
			}
			normalizedRegions = append(normalizedRegions, TargetRegion{Name: region})
		}

		s.TargetRegions = normalizedRegions
	}

	if !foundMandatoryReplicationRegion {
		s.TargetRegions = append(s.TargetRegions, TargetRegion{
			Name:                buildLocation,
			DiskEncryptionSetId: s.BuildDiskEncryptionSetId,
		},
		)
	}
	confidentialVMEncryptionType, ok := stateBag.Get(constants.ArmSharedImageGalleryDestinationConfidentialVMImageEncryptionType).(string)
	if !ok {
		confidentialVMEncryptionType = ""
	}
	sdkTargetRegions := make([]galleryimageversions.TargetRegion, 0, len(s.TargetRegions))
	for _, r := range s.TargetRegions {
		name := r.Name
		tr := galleryimageversions.TargetRegion{Name: name}

		encryption := buildAzureImageTargetRegionsWithEncryption(r.DiskEncryptionSetId, confidentialVMEncryptionType)
		tr.Encryption = encryption
		replicas := r.ReplicaCount
		if replicas > 0 {
			if replicas > constants.SharedImageGalleryImageVersionDefaultMaxReplicaCount {
				replicas = constants.SharedImageGalleryImageVersionDefaultMaxReplicaCount
			}
			tr.RegionalReplicaCount = &replicas
		}

		sdkTargetRegions = append(sdkTargetRegions, tr)
	}
	stateBag.Put(
		constants.ArmSharedImageGalleryDestinationTargetRegions,
		sdkTargetRegions,
	)

	return multistep.ActionContinue
}

func (*StepBuildTargetRegions) Cleanup(multistep.StateBag) {
}
