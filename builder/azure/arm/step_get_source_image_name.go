package arm

import (
	"context"
	"fmt"
	"log"

	"github.com/hashicorp/packer-plugin-sdk/multistep"
	packersdk "github.com/hashicorp/packer-plugin-sdk/packer"
	"github.com/hashicorp/packer-plugin-sdk/packerbuilderdata"
)

type StepGetSourceImageName struct {
	client        *AzureClient
	config        *Config
	generatedData *packerbuilderdata.GeneratedData
	say           func(message string)
	error         func(e error)
}

func NewStepGetSourceImageName(client *AzureClient, config *Config, ui packersdk.Ui) *StepGetSourceImageName {
	var step = &StepGetSourceImageName{
		client: client,
		config: config,
		say:    func(message string) { ui.Say(message) },
		error:  func(e error) { ui.Error(e.Error()) },
	}

	return step
}

func (s *StepGetSourceImageName) Run(ctx context.Context, state multistep.StateBag) multistep.StepAction {
	s.say("Getting source image id for the deployment ...")

	if s.config.ImageUrl != "" {
		s.say(fmt.Sprintf(" -> SourceImageName: '%s'", s.config.ImageUrl))
		s.generatedData.Put("SourceImageName", s.config.ImageUrl)
		return multistep.ActionContinue
	}

	if s.config.CustomManagedImageName != "" {
		s.say(fmt.Sprintf(" -> SourceImageName: '%s'", s.config.customManagedImageID))
		s.generatedData.Put("SourceImageName", s.config.customManagedImageID)
		return multistep.ActionContinue
	}

	if s.config.SharedGallery.Subscription != "" {
		client := s.client.GalleryImageVersionsClient
		client.SubscriptionID = s.config.SharedGallery.Subscription
		image, err := client.Get(ctx, s.config.SharedGallery.ResourceGroup, s.config.SharedGallery.GalleryName, s.config.SharedGallery.ImageName, s.config.SharedGallery.ImageVersion, "")
		if err != nil {
			log.Println("[TRACE] unable to derive managed image URL for shared gallery version image")
		}

		var imageID string
		if image.GalleryImageVersionProperties.StorageProfile.Source.ID != nil {
			log.Println("[TRACE] unable to derive managed image URL for shared gallery version image")
			imageID = *image.GalleryImageVersionProperties.StorageProfile.Source.ID
		}

		s.say(fmt.Sprintf(" -> SourceImageName: '%s'", imageID))
		s.generatedData.Put("SourceImageName", imageID)
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
	s.generatedData.Put("SourceImageName", imageID)
	return multistep.ActionContinue
}

func (*StepGetSourceImageName) Cleanup(multistep.StateBag) {
}
