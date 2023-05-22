// Copyright (c) HashiCorp, Inc.
// SPDX-License-Identifier: MPL-2.0

package arm

import (
	"context"
	"fmt"
	"log"
	"regexp"

	"github.com/Azure/azure-sdk-for-go/services/compute/mgmt/2021-11-01/compute"
	"github.com/hashicorp/packer-plugin-sdk/multistep"
	packersdk "github.com/hashicorp/packer-plugin-sdk/packer"
	"github.com/hashicorp/packer-plugin-sdk/packerbuilderdata"
)

type StepGetSourceImageName struct {
	client            *AzureClient
	config            *Config
	GeneratedData     *packerbuilderdata.GeneratedData
	getGalleryVersion func(context.Context) (compute.GalleryImageVersion, error)
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

	if s.config.SharedGallery.Subscription != "" {

		image, err := s.getGalleryVersion(ctx)
		if err != nil {
			log.Println("[TRACE] unable to derive managed image URL for shared gallery version image")
			s.GeneratedData.Put("SourceImageName", "ERR_SOURCE_IMAGE_NAME_NOT_FOUND")
			return multistep.ActionContinue
		}

		if image.GalleryImageVersionProperties != nil && image.GalleryImageVersionProperties.StorageProfile != nil &&
			image.GalleryImageVersionProperties.StorageProfile.Source != nil && image.GalleryImageVersionProperties.StorageProfile.Source.ID != nil {

			sourceID := *image.GalleryImageVersionProperties.StorageProfile.Source.ID
			isSIGSourcedFromManagedImage, _ := regexp.MatchString("/subscriptions/[^/]*/resourceGroups/[^/]*/providers/Microsoft.Compute/images/[^/]*$", sourceID)
			// If the Source SIG Image Version does not have its StorageProfile.Source set to a Managed Image, that means the image was directly sourced from a VM
			// Use the SIG ID itself if there isn't a managed image that the SIG was created from
			if !isSIGSourcedFromManagedImage {
				sourceID = *image.ID
			}

			s.say(fmt.Sprintf(" -> SourceImageName: '%s'", sourceID))
			s.GeneratedData.Put("SourceImageName", sourceID)
			return multistep.ActionContinue
		}

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

func (s *StepGetSourceImageName) GetGalleryImageVersion(ctx context.Context) (compute.GalleryImageVersion, error) {
	client := s.client.GalleryImageVersionsClient
	client.SubscriptionID = s.config.SharedGallery.Subscription

	return client.Get(ctx, s.config.SharedGallery.ResourceGroup,
		s.config.SharedGallery.GalleryName, s.config.SharedGallery.ImageName, s.config.SharedGallery.ImageVersion, "")
}

func (*StepGetSourceImageName) Cleanup(multistep.StateBag) {
}
