package arm

import (
	"context"
	"fmt"
	"log"

	"github.com/hashicorp/packer-plugin-sdk/multistep"
	packersdk "github.com/hashicorp/packer-plugin-sdk/packer"
	"github.com/hashicorp/packer-plugin-sdk/packerbuilderdata"
)

type StepSetSourceImageName struct {
	client *AzureClient
	config *Config
	say    func(message string)
	error  func(e error)
}

func NewStepSetSourceImageName(client *AzureClient, config *Config, ui packersdk.Ui) *StepSetSourceImageName {
	var step = &StepSetSourceImageName{
		client: client,
		config: config,
		say:    func(message string) { ui.Say(message) },
		error:  func(e error) { ui.Error(e.Error()) },
	}

	return step
}

func (s *StepSetSourceImageName) Run(ctx context.Context, state multistep.StateBag) multistep.StepAction {
	s.say("Storing reference for source image to use in deployment ...")

	generatedData := packerbuilderdata.GeneratedData{State: state}
	if s.config.ImageUrl != "" { //err = builder.SetImageUrl(s.config.ImageUrl, osType, s.config.diskCachingType)
		s.say(fmt.Sprintf(" -> SourceImageName used for deployment           : '%s'", s.config.ImageUrl))
		generatedData.Put("SourceImageName", s.config.ImageUrl)
		return multistep.ActionContinue
	}

	if s.config.CustomManagedImageName != "" {
		//	err = builder.SetManagedDiskUrl(s.config.customManagedImageID, s.config.managedImageStorageAccountType, s.config.diskCachingType)
		s.say(fmt.Sprintf(" -> SourceImageName used for deployment           : '%s'", s.config.customManagedImageID))
		generatedData.Put("SourceImageName", s.config.customManagedImageID)
		return multistep.ActionContinue
	}

	if s.config.SharedGallery.Subscription != "" {
		imageID := fmt.Sprintf("/subscriptions/%s/resourceGroups/%s/providers/Microsoft.Compute/galleries/%s/images/%s",
			s.config.SharedGallery.Subscription,
			s.config.SharedGallery.ResourceGroup,
			s.config.SharedGallery.GalleryName,
			s.config.SharedGallery.ImageName)

		if s.config.SharedGallery.ImageVersion != "" {
			imageID += fmt.Sprintf("/versions/%s",
				s.config.SharedGallery.ImageVersion)
		}

		// Lets keep track of the old subscription settings and reset once we have our
		// managed image source.
		sigID := s.client.GalleryImageVersionsClient.SubscriptionID
		defer func() {
			s.client.GalleryImageVersionsClient.SubscriptionID = sigID
		}()

		s.client.GalleryImageVersionsClient.SubscriptionID = s.config.SharedGallery.Subscription
		i, err := s.client.GalleryImageVersionsClient.Get(ctx, s.config.SharedGallery.ResourceGroup, s.config.SharedGallery.GalleryName, s.config.SharedGallery.ImageName, s.config.SharedGallery.ImageVersion, "")
		if err != nil {
			log.Println("[TRACE] unable to derive managed image URL for shared gallery version image")
		}

		if i.GalleryImageVersionProperties.StorageProfile.Source.ID != nil {
			imageID = *i.GalleryImageVersionProperties.StorageProfile.Source.ID
		}

		s.say(fmt.Sprintf(" -> SourceImageName used for deployment           : '%s'", imageID))
		generatedData.Put("SourceImageName", imageID)
		return multistep.ActionContinue
	}

	imageID := fmt.Sprintf("/subscriptions/%s/providers/Microsoft.Compute/locations/%s/publishers/%s/ArtifactTypes/vmimage/offers/%s/skus/%s/versions/%s",
		s.config.ClientConfig.SubscriptionID,
		s.config.Location,
		s.config.ImagePublisher,
		s.config.ImageOffer,
		s.config.ImageSku,
		s.config.ImageVersion)

	s.say(fmt.Sprintf(" -> SourceImageName used for deployment           : '%s'", imageID))
	generatedData.Put("SourceImageName", imageID)
	return multistep.ActionContinue
}

func (*StepSetSourceImageName) Cleanup(multistep.StateBag) {
}
