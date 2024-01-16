// Copyright (c) HashiCorp, Inc.
// SPDX-License-Identifier: MPL-2.0

package chroot

import (
	"bytes"
	"context"
	"fmt"
	"testing"

	"github.com/hashicorp/go-azure-sdk/resource-manager/compute/2022-03-03/galleryimageversions"
	"github.com/hashicorp/packer-plugin-azure/builder/azure/common"
	"github.com/hashicorp/packer-plugin-azure/builder/azure/common/client"
	"github.com/hashicorp/packer-plugin-sdk/multistep"
	packersdk "github.com/hashicorp/packer-plugin-sdk/packer"
	"github.com/hashicorp/packer-plugin-sdk/packerbuilderdata"
)

func TestChrootStepGetSourceImageName(t *testing.T) {
	ui := &packersdk.BasicUi{
		Reader: new(bytes.Buffer),
		Writer: new(bytes.Buffer),
	}
	state := new(multistep.BasicStateBag)

	state.Put("azureclient", &client.AzureClientSetMock{SubscriptionIDMock: "1234"})
	state.Put("ui", ui)

	tc := []struct {
		name     string
		step     *StepGetSourceImageName
		expected string
	}{
		{
			name: "SourceOSDisk",
			step: &StepGetSourceImageName{
				SourceOSDiskResourceID: "https://azure/vhd",
				GeneratedData:          &packerbuilderdata.GeneratedData{State: state},
			},
			expected: "https://azure/vhd",
		},
		{
			name: "MarketPlaceImage",
			step: &StepGetSourceImageName{
				SourcePlatformImage: &client.PlatformImage{
					Publisher: "Microsoft",
					Offer:     "Server",
					Sku:       "0",
					Version:   "2019",
				},
				Location:      "west",
				GeneratedData: &packerbuilderdata.GeneratedData{State: state},
			},
			expected: "/subscriptions/1234/providers/Microsoft.Compute/locations/west/publishers/Microsoft/ArtifactTypes/vmimage/offers/Server/skus/0/versions/2019",
		},
	}
	for _, tt := range tc {
		tt := tt
		t.Run(tt.name, func(t *testing.T) {
			tt.step.Run(context.TODO(), state)
			got := state.Get("generated_data").(map[string]interface{})
			v, ok := got["SourceImageName"]
			if !ok {
				t.Errorf("expected SourceImageName to be set in generatedData")
			}

			if v != tt.expected {
				t.Errorf("expected SourceImageName to be set to %q but got %q", tt.expected, v)
			}
		})
	}

}

func TestChrootStepGetSourceImageName_SharedImage(t *testing.T) {
	ui := &packersdk.BasicUi{
		Reader: new(bytes.Buffer),
		Writer: new(bytes.Buffer),
	}
	state := new(multistep.BasicStateBag)
	state.Put("ui", ui)

	genData := packerbuilderdata.GeneratedData{State: state}

	tc := []struct {
		name                  string
		step                  *StepGetSourceImageName
		expected              string
		mockedGalleryReturn   *galleryimageversions.GalleryImageVersion
		expectedImageId       galleryimageversions.ImageVersionId
		SourceImageResourceID string
		GeneratedData         packerbuilderdata.GeneratedData
	}{
		{
			name:                  "SharedImageWithMangedImageSource",
			SourceImageResourceID: "/subscriptions/1234/resourceGroups/bar/providers/Microsoft.Compute/galleries/test/images/foo/versions/1.0.6",
			GeneratedData:         genData,
			mockedGalleryReturn: &galleryimageversions.GalleryImageVersion{
				Properties: &galleryimageversions.GalleryImageVersionProperties{
					StorageProfile: galleryimageversions.GalleryImageVersionStorageProfile{
						Source: &galleryimageversions.GalleryArtifactVersionFullSource{
							Id: common.StringPtr("/subscription/resource/managed/image/name/as/source"),
						},
					},
				},
			},
			expectedImageId: galleryimageversions.ImageVersionId{
				SubscriptionId:    "1234",
				ResourceGroupName: "bar",
				GalleryName:       "test",
				ImageName:         "foo",
				VersionName:       "1.0.6",
			},
			expected: "/subscription/resource/managed/image/name/as/source",
		},
		{
			name:                  "SimulatedBadImageResponse",
			SourceImageResourceID: "/subscriptions/1234/resourceGroups/bar/providers/Microsoft.Compute/galleries/test/images/foo/versions/0.0.0",
			GeneratedData:         genData,
			expected:              "ERR_SOURCE_IMAGE_NAME_NOT_FOUND",
		},
	}
	for _, tt := range tc {
		tt := tt
		t.Run(tt.name, func(t *testing.T) {

			var actualID galleryimageversions.ImageVersionId
			step := StepGetSourceImageName{
				SourceImageResourceID: tt.SourceImageResourceID,
				GeneratedData:         &tt.GeneratedData,
				get: func(ctx context.Context, azcli client.AzureClientSet, id galleryimageversions.ImageVersionId) (*galleryimageversions.GalleryImageVersion, error) {
					actualID = id
					if tt.mockedGalleryReturn == nil {
						return nil, fmt.Errorf("Generic error")
					}
					return tt.mockedGalleryReturn, nil
				},
			}
			state.Put("azureclient", &client.AzureClientSetMock{
				SubscriptionIDMock: "1234",
			})
			step.Run(context.TODO(), state)
			got := state.Get("generated_data").(map[string]interface{})
			v, ok := got["SourceImageName"]
			if !ok {
				t.Errorf("expected SourceImageName to be set in generatedData")
			}

			if v != tt.expected {
				t.Errorf("expected SourceImageName to be set to %q but got %q", tt.expected, v)
			}

			if tt.mockedGalleryReturn != nil {
				if actualID != tt.expectedImageId {
					t.Errorf("expected %s but got %s Gallery Image Version ID passed into client", tt.expectedImageId, actualID)
				}
			}
		})
	}
}
