// Copyright (c) HashiCorp, Inc.
// SPDX-License-Identifier: MPL-2.0

package arm

import (
	"context"
	"fmt"
	"regexp"

	"github.com/hashicorp/packer-plugin-azure/builder/azure/common/log"

	"github.com/hashicorp/go-azure-sdk/resource-manager/compute/2023-07-03/galleryimageversions"
	"github.com/hashicorp/packer-plugin-sdk/multistep"
	packersdk "github.com/hashicorp/packer-plugin-sdk/packer"
	"github.com/hashicorp/packer-plugin-sdk/packerbuilderdata"
)

type StepGetSourceImageName struct {
	client            *AzureClient
	config            *Config
	GeneratedData     *packerbuilderdata.GeneratedData
	getGalleryVersion func(context.Context, SharedImageGallery) (*galleryimageversions.GalleryImageVersion, error)
	say               func(message string)
	error             func(e error)
}

func NewStepGetSourceImageName(client *AzureClient, ui packersdk.Ui, config *Config, GeneratedData *packerbuilderdata.GeneratedData) *StepGetSourceImageName {
	var step = &StepGetSourceImageName{
		client:        client,
		say:           func(message string) { ui.Say(message) },
		error:         func(e error) { ui.Error(e.Error()) },
		config:        config,
		GeneratedData: GeneratedData,
	}
	step.getGalleryVersion = step.GetGalleryImageVersion
	return step
}

func (s *StepGetSourceImageName) Run(ctx context.Context, state multistep.StateBag) multistep.StepAction {
	s.say("Getting source image id for the deployment ...")

	if s.config.ImageUrl != "" {
		s.say(fmt.Sprintf(" -> SourceImageName: '%s'", s.config.ImageUrl))
		s.GeneratedData.Put("SourceImageName", s.config.ImageUrl)
		return multistep.ActionContinue
	}

	if s.config.CustomManagedImageName != "" {
		s.say(fmt.Sprintf(" -> SourceImageName: '%s'", s.config.customManagedImageID))
		s.GeneratedData.Put("SourceImageName", s.config.customManagedImageID)
		return multistep.ActionContinue
	}

	if s.config.SharedGallery.Subscription != "" || s.config.SharedGallery.ID != "" {
		acg := s.config.SharedGallery
		if acg.Subscription == "" {
			acgPtr := s.config.getSharedImageGalleryObjectFromId()
			if acgPtr == nil {
				s.say("Failed to parse Shared Image Gallery object from ID, this is always a Packer Azure plugin bug")
				return multistep.ActionHalt
			}
			acg = *acgPtr
		}
		image, err := s.getGalleryVersion(ctx, acg)
		if err != nil {
			s.say(fmt.Sprintf("Failed to fetch source gallery image from Azure API, HCP Packer will not be able to track the ancestry of this build: %s", err.Error()))
			s.GeneratedData.Put("SourceImageName", "ERR_SOURCE_IMAGE_NAME_NOT_FOUND")
			return multistep.ActionContinue
		}

		if image.Properties != nil &&
			image.Properties.StorageProfile.Source != nil && image.Properties.StorageProfile.Source.Id != nil {

			// Shared Image Galleries can be created in two different ways
			// Either directly from a VM (in the builder this means not setting managed_image_name), for these types of images we set the artifact ID as the Gallery Image ID
			// Or through an intermediate managed image. in which case we use that managed image as the artifact ID.

			// First check if the parent Gallery Image Version source ID is a managed image, if so we use that as our source image name
			parentSourceID := *image.Properties.StorageProfile.Source.Id
			isSIGSourcedFromManagedImage, _ := regexp.MatchString("/subscriptions/[^/]*/resourceGroups/[^/]*/providers/Microsoft.Compute/images/[^/]*$", parentSourceID)

			if isSIGSourcedFromManagedImage {
				s.say(fmt.Sprintf(" -> SourceImageName: '%s'", parentSourceID))
				s.GeneratedData.Put("SourceImageName", parentSourceID)
				return multistep.ActionContinue
			}
		}
		// If the Gallery Image Version was not sourced from a Managed Image, that means it was captured directly from a VM, so we just use the gallery ID itself as the source image
		if image.Id != nil {
			s.say(fmt.Sprintf(" -> SourceImageName: '%s'", *image.Id))
			s.GeneratedData.Put("SourceImageName", *image.Id)
			return multistep.ActionContinue
		}

		s.say("Received unexpected response from Azure API, HCP Packer will not be able to track the ancestry of this build.")
		log.Println("[TRACE] unable to identify the source image for provided gallery image version")
		s.GeneratedData.Put("SourceImageName", "ERR_SOURCE_IMAGE_NAME_NOT_FOUND")
		return multistep.ActionContinue
	}

	imageID := fmt.Sprintf("/subscriptions/%s/providers/Microsoft.Compute/locations/%s/publishers/%s/ArtifactTypes/vmimage/offers/%s/skus/%s/versions/%s",
		s.config.ClientConfig.SubscriptionID,
		s.config.Location,
		s.config.ImagePublisher,
		s.config.ImageOffer,
		s.config.ImageSku,
		s.config.ImageVersion)

	s.say(fmt.Sprintf(" -> SourceImageName: '%s'", imageID))
	s.GeneratedData.Put("SourceImageName", imageID)
	return multistep.ActionContinue
}

func (s *StepGetSourceImageName) GetGalleryImageVersion(ctx context.Context, acg SharedImageGallery) (*galleryimageversions.GalleryImageVersion, error) {
	client := s.client.GalleryImageVersionsClient
	pollingContext, cancel := context.WithTimeout(ctx, s.client.PollingDuration)
	defer cancel()
	galleryVersionId := galleryimageversions.NewImageVersionID(acg.Subscription, acg.ResourceGroup, acg.GalleryName, acg.ImageName, acg.ImageVersion)
	result, err := client.Get(pollingContext, galleryVersionId, galleryimageversions.DefaultGetOperationOptions())
	if err != nil {
		return nil, err
	}
	return result.Model, nil
}

func (*StepGetSourceImageName) Cleanup(multistep.StateBag) {
}
