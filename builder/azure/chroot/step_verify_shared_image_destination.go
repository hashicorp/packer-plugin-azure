// Copyright IBM Corp. 2013, 2025
// SPDX-License-Identifier: MPL-2.0

package chroot

import (
	"context"
	"fmt"
	"strings"

	"github.com/hashicorp/packer-plugin-azure/builder/azure/common/log"

	"github.com/hashicorp/go-azure-sdk/resource-manager/compute/2023-07-03/galleryimages"
	"github.com/hashicorp/go-azure-sdk/resource-manager/compute/2023-07-03/galleryimageversions"
	"github.com/hashicorp/packer-plugin-azure/builder/azure/common/client"
	"github.com/hashicorp/packer-plugin-sdk/multistep"
	packersdk "github.com/hashicorp/packer-plugin-sdk/packer"
)

var _ multistep.Step = &StepVerifySharedImageDestination{}

// StepVerifySharedImageDestination verifies that the shared image location matches the Location field in the step.
// Also verifies that the OS Type is Linux.
type StepVerifySharedImageDestination struct {
	Image        SharedImageGalleryDestination
	Location     string
	listVersions func(context.Context, client.AzureClientSet, galleryimageversions.GalleryImageId) ([]galleryimageversions.GalleryImageVersion, error)
	getImage     func(context.Context, client.AzureClientSet, galleryimages.GalleryImageId) (*galleryimages.GalleryImage, error)
}

func NewStepVerifySharedImageDestination(step *StepVerifySharedImageDestination) *StepVerifySharedImageDestination {
	step.getImage = step.getGalleryImage
	step.listVersions = step.listGalleryVersions
	return step
}

// Run retrieves the image metadata from Azure and compares the location to Location. Verifies the OS Type.
func (s *StepVerifySharedImageDestination) Run(ctx context.Context, state multistep.StateBag) multistep.StepAction {
	azcli := state.Get("azureclient").(client.AzureClientSet)
	ui := state.Get("ui").(packersdk.Ui)

	errorMessage := func(message string, parameters ...interface{}) multistep.StepAction {
		err := fmt.Errorf(message, parameters...)
		log.Printf("StepVerifySharedImageDestination.Run: error: %+v", err)
		state.Put("error", err)
		ui.Error(err.Error())
		return multistep.ActionHalt
	}

	imageURI := fmt.Sprintf("/subscriptions/%s/resourcegroup/%s/providers/Microsoft.Compute/galleries/%s/images/%s",
		azcli.SubscriptionID(),
		s.Image.ResourceGroup,
		s.Image.GalleryName,
		s.Image.ImageName,
	)

	ui.Say(fmt.Sprintf("Validating that shared image %s exists", imageURI))
	galleryImageID := galleryimages.NewGalleryImageID(
		azcli.SubscriptionID(),
		s.Image.ResourceGroup,
		s.Image.GalleryName,
		s.Image.ImageName,
	)
	getImagePollingContext, getImageCancel := context.WithTimeout(ctx, azcli.PollingDuration())
	defer getImageCancel()
	image, err := s.getImage(getImagePollingContext, azcli, galleryImageID)

	if err != nil {
		return errorMessage("Error retrieving shared image %q: %+v ", imageURI, err)
	}

	if image.Id == nil || *image.Id == "" {
		return errorMessage("Error retrieving shared image %q: ID field in response is empty", imageURI)
	}
	if image.Properties == nil {
		return errorMessage("Could not retrieve shared image properties for image %q.", image.Id)
	}

	location := image.Location

	log.Printf("StepVerifySharedImageDestination:Run: Image %+v, Location: %+v, HvGen: %+v, osState: %+v",
		*(image.Id),
		location,
		image.Properties.HyperVGeneration,
		image.Properties.OsState)

	if !strings.EqualFold(location, s.Location) {
		return errorMessage("Destination shared image resource %q is in a different location (%q) than this VM (%q).",
			*image.Id,
			location,
			s.Location)
	}

	if image.Properties.OsType != galleryimages.OperatingSystemTypesLinux {
		return errorMessage("The shared image (%q) is not a Linux image (found %q). Currently only Linux images are supported.",
			*(image.Id),
			image.Properties.OsType)
	}

	ui.Say(fmt.Sprintf("Found image %s in location %s",
		*image.Id,
		image.Location,
	))

	// TODO Suggest moving gallery image ID to common IDs library
	// so we don't have to define two different versions of the same resource ID
	galleryImageIDForList := galleryimageversions.NewGalleryImageID(
		azcli.SubscriptionID(),
		s.Image.ResourceGroup,
		s.Image.GalleryName,
		s.Image.ImageName,
	)
	listVersionsCtx, listVersionsCancel := context.WithTimeout(ctx, azcli.PollingDuration())
	defer listVersionsCancel()
	versions, err := s.listVersions(listVersionsCtx, azcli,
		galleryImageIDForList)

	if err != nil {
		return errorMessage("Could not ListByGalleryImageComplete group:%v gallery:%v image:%v",
			s.Image.ResourceGroup, s.Image.GalleryName, s.Image.ImageName)
	}

	for _, version := range versions {
		if version.Name == nil {
			return errorMessage("Could not retrieve versions for image %q: unexpected nil name", image.Id)
		}
		if *version.Name == s.Image.ImageVersion {
			return errorMessage("Shared image version %q already exists for image %q.", s.Image.ImageVersion, *image.Id)
		}
	}

	return multistep.ActionContinue
}

func (s *StepVerifySharedImageDestination) getGalleryImage(ctx context.Context, azcli client.AzureClientSet, id galleryimages.GalleryImageId) (*galleryimages.GalleryImage, error) {
	pollingContext, cancel := context.WithTimeout(ctx, azcli.PollingDuration())
	defer cancel()
	res, err := azcli.GalleryImagesClient().Get(pollingContext, id)
	if err != nil {
		return nil, err
	}
	if res.Model == nil {
		return nil, client.NullModelSDKErr
	}
	return res.Model, nil
}

func (s *StepVerifySharedImageDestination) listGalleryVersions(ctx context.Context, azcli client.AzureClientSet, id galleryimageversions.GalleryImageId) ([]galleryimageversions.GalleryImageVersion, error) {
	pollingContext, cancel := context.WithTimeout(ctx, azcli.PollingDuration())
	defer cancel()
	res, err := azcli.GalleryImageVersionsClient().ListByGalleryImageComplete(pollingContext, id)
	if err != nil {
		return nil, err
	}
	if res.Items == nil {
		return nil, client.NullModelSDKErr
	}
	return res.Items, nil
}
func (*StepVerifySharedImageDestination) Cleanup(multistep.StateBag) {}
