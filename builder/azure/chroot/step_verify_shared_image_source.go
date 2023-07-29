// Copyright (c) HashiCorp, Inc.
// SPDX-License-Identifier: MPL-2.0

package chroot

import (
	"context"
	"fmt"
	"log"
	"strings"

	"github.com/hashicorp/go-azure-sdk/resource-manager/compute/2022-03-03/galleryimages"
	"github.com/hashicorp/go-azure-sdk/resource-manager/compute/2022-03-03/galleryimageversions"
	"github.com/hashicorp/packer-plugin-azure/builder/azure/common/client"
	"github.com/hashicorp/packer-plugin-sdk/multistep"
	packersdk "github.com/hashicorp/packer-plugin-sdk/packer"
)

var _ multistep.Step = &StepVerifySharedImageSource{}

// StepVerifySharedImageSource verifies that the shared image location matches the Location field in the step.
// Also verifies that the OS Type is Linux.
type StepVerifySharedImageSource struct {
	SharedImageID  string
	SubscriptionID string
	Location       string

	getVersion func(context.Context, client.AzureClientSet, galleryimageversions.ImageVersionId) (*galleryimageversions.GalleryImageVersion, error)
	getImage   func(context.Context, client.AzureClientSet, galleryimages.GalleryImageId) (*galleryimages.GalleryImage, error)
}

func NewStepVerifySharedImageSource(step *StepVerifySharedImageSource) *StepVerifySharedImageSource {
	step.getImage = step.getGalleryImage
	step.getVersion = step.getGalleryVersion
	return step
}

// Run retrieves the image metadata from Azure and compares the location to Location. Verifies the OS Type.
func (s *StepVerifySharedImageSource) Run(ctx context.Context, state multistep.StateBag) multistep.StepAction {
	azcli := state.Get("azureclient").(client.AzureClientSet)
	ui := state.Get("ui").(packersdk.Ui)

	errorMessage := func(message string, parameters ...interface{}) multistep.StepAction {
		err := fmt.Errorf(message, parameters...)
		log.Printf("StepVerifySharedImageSource.Run: error: %+v", err)
		state.Put("error", err)
		ui.Error(err.Error())
		return multistep.ActionHalt
	}

	resource, err := client.ParseResourceID(s.SharedImageID)
	if err != nil {
		return errorMessage("Could not parse resource id %q: %w", s.SharedImageID, err)
	}

	if !strings.EqualFold(resource.Provider, "Microsoft.Compute") ||
		!strings.EqualFold(resource.ResourceType.String(), "galleries/images/versions") {
		return errorMessage("Resource id %q does not identify a shared image version, expected Microsoft.Compute/galleries/images/versions", s.SharedImageID)
	}

	ui.Say(fmt.Sprintf("Validating that shared image version %q exists",
		s.SharedImageID))

	galleryImageVersionID := galleryimageversions.NewImageVersionID(
		azcli.SubscriptionID(),
		resource.ResourceGroup,
		resource.ResourceName[0],
		resource.ResourceName[1],
		resource.ResourceName[2],
	)
	version, err := s.getVersion(ctx, azcli, galleryImageVersionID)

	if err != nil {
		return errorMessage("Error retrieving shared image version %q: %+v ", s.SharedImageID, err)
	}

	if version.Id == nil || *version.Id == "" {
		return errorMessage("Error retrieving shared image version %q: ID field in response is empty", s.SharedImageID)
	}

	if version.Properties == nil ||
		version.Properties.PublishingProfile == nil ||
		version.Properties.PublishingProfile.TargetRegions == nil {
		return errorMessage("Could not retrieve shared image version properties for image %q.", s.SharedImageID)
	}

	targetLocations := make([]string, 0, len(*version.Properties.PublishingProfile.TargetRegions))
	vmLocation := client.NormalizeLocation(s.Location)
	locationFound := false
	for _, tr := range *version.Properties.PublishingProfile.TargetRegions {
		location := client.NormalizeLocation(tr.Name)
		targetLocations = append(targetLocations, location)
		if strings.EqualFold(vmLocation, location) {
			locationFound = true
			break
		}
	}
	if !locationFound {
		return errorMessage("Target locations %q for %q does not include VM location %q",
			targetLocations, s.SharedImageID, vmLocation)
	}

	imageResource, _ := resource.Parent()
	galleryImageID := galleryimages.NewGalleryImageID(
		azcli.SubscriptionID(),
		resource.ResourceGroup,
		resource.ResourceName[0],
		resource.ResourceName[1],
	)
	image, err := s.getImage(ctx, azcli, galleryImageID)
	if err != nil {
		return errorMessage("Error retrieving shared image %q: %+v ", imageResource.String(), err)
	}

	if image.Id == nil || *image.Id == "" {
		return errorMessage("Error retrieving shared image %q: ID field in response is empty", imageResource.String())
	}

	if image.Properties == nil {
		return errorMessage("Could not retrieve shared image properties for image %q.", imageResource.String())
	}

	log.Printf("StepVerifySharedImageSource:Run: Image %+v, HvGen: %+v, osState: %+v",
		&image.Id,
		image.Properties.HyperVGeneration,
		image.Properties.OsState)

	if image.Properties.OsType != galleryimages.OperatingSystemTypesLinux {
		return errorMessage("The shared image (%q) is not a Linux image (found %q). Currently only Linux images are supported.",
			&image.Id,
			image.Properties.OsType)
	}

	ui.Say(fmt.Sprintf("Found image source image version %q, available in location %s",
		s.SharedImageID,
		s.Location))

	return multistep.ActionContinue
}

func (s *StepVerifySharedImageSource) getGalleryVersion(ctx context.Context, azcli client.AzureClientSet, id galleryimageversions.ImageVersionId) (*galleryimageversions.GalleryImageVersion, error) {
	res, err := azcli.GalleryImageVersionsClient().Get(ctx, id, galleryimageversions.DefaultGetOperationOptions())
	if err != nil {
		return nil, err
	}
	if res.Model == nil {
		return nil, fmt.Errorf("SDK returned empty model")
	}
	return res.Model, nil
}

func (s *StepVerifySharedImageSource) getGalleryImage(ctx context.Context, azcli client.AzureClientSet, id galleryimages.GalleryImageId) (*galleryimages.GalleryImage, error) {
	res, err := azcli.GalleryImagesClient().Get(ctx, id)
	if err != nil {
		return nil, err
	}
	if res.Model == nil {
		return nil, fmt.Errorf("SDK returned empty model")
	}
	return res.Model, nil
}
func (*StepVerifySharedImageSource) Cleanup(multistep.StateBag) {}
