package chroot

import (
	"context"
	"fmt"
	"log"

	"github.com/hashicorp/packer-plugin-azure/builder/azure/common/client"
	"github.com/hashicorp/packer-plugin-sdk/multistep"
	packersdk "github.com/hashicorp/packer-plugin-sdk/packer"
	"github.com/hashicorp/packer-plugin-sdk/packerbuilderdata"
)

type StepGetSourceImageName struct {
	// Copy another disk
	SourceOSDiskResourceID string

	// Extract from platform image
	SourcePlatformImage *client.PlatformImage
	// Extract from shared image
	SourceImageResourceID string

	Location      string
	GeneratedData *packerbuilderdata.GeneratedData
}

func (s *StepGetSourceImageName) Run(ctx context.Context, state multistep.StateBag) multistep.StepAction {
	azcli := state.Get("azureclient").(client.AzureClientSet)
	ui := state.Get("ui").(packersdk.Ui)
	ui.Say("Getting source image id for the deployment ...")

	if s.SourceOSDiskResourceID != "" {
		ui.Say(fmt.Sprintf(" -> SourceImageName: '%s'", s.SourceOSDiskResourceID))
		s.GeneratedData.Put("SourceImageName", s.SourceOSDiskResourceID)
		return multistep.ActionContinue
	}

	if s.SourceImageResourceID != "" {
		imageID, err := client.ParseResourceID(s.SourceImageResourceID)
		if err != nil {
			log.Printf("[TRACE] could not parse source image id %q: %v", s.SourceImageResourceID, err)
			s.GeneratedData.Put("SourceImageName", "ERR_SOURCE_IMAGE_NAME_NOT_FOUND")
			return multistep.ActionContinue
		}

		client := azcli.GalleryImageVersionsClient()
		image, err := client.Get(ctx, imageID.ResourceGroup, imageID.ResourceName[0], imageID.ResourceName[1], imageID.ResourceName[2], "")
		if err != nil {
			log.Printf("[TRACE] error retrieving managed image name for shared source image %q: %v", s.SourceImageResourceID, err)
			s.GeneratedData.Put("SourceImageName", "ERR_SOURCE_IMAGE_NAME_NOT_FOUND")
			return multistep.ActionContinue
		}

		if image.GalleryImageVersionProperties != nil && image.GalleryImageVersionProperties.StorageProfile != nil &&
			image.GalleryImageVersionProperties.StorageProfile.Source != nil && image.GalleryImageVersionProperties.StorageProfile.Source.ID != nil {
			id := *image.GalleryImageVersionProperties.StorageProfile.Source.ID
			ui.Say(fmt.Sprintf(" -> SourceImageName: '%s'", id))
			s.GeneratedData.Put("SourceImageName", id)
			return multistep.ActionContinue
		}

		log.Println("[TRACE] unable to identify the source image for provided gallery image version")
		s.GeneratedData.Put("SourceImageName", "ERR_SOURCE_IMAGE_NAME_NOT_FOUND")
		return multistep.ActionContinue
	}

	imageID := fmt.Sprintf(
		"/subscriptions/%s/providers/Microsoft.Compute/locations/%s/publishers/%s/ArtifactTypes/vmimage/offers/%s/skus/%s/versions/%s", azcli.SubscriptionID(), s.Location,
		s.SourcePlatformImage.Publisher, s.SourcePlatformImage.Offer, s.SourcePlatformImage.Sku, s.SourcePlatformImage.Version)

	ui.Say(fmt.Sprintf(" -> SourceImageName: '%s'", imageID))
	s.GeneratedData.Put("SourceImageName", imageID)
	return multistep.ActionContinue
}

func (*StepGetSourceImageName) Cleanup(multistep.StateBag) {
}
