// Copyright (c) HashiCorp, Inc.
// SPDX-License-Identifier: MPL-2.0

package arm

import (
	"bytes"
	"context"
	"testing"

	galleryimageversions "github.com/hashicorp/go-azure-sdk/resource-manager/compute/2023-07-03/galleryimageversions"
	"github.com/hashicorp/packer-plugin-azure/builder/azure/common/client"
	"github.com/hashicorp/packer-plugin-sdk/multistep"
	packersdk "github.com/hashicorp/packer-plugin-sdk/packer"
	"github.com/hashicorp/packer-plugin-sdk/packerbuilderdata"
)

func TestStepGetSourceImageName(t *testing.T) {
	ui := &packersdk.BasicUi{
		Reader: new(bytes.Buffer),
		Writer: new(bytes.Buffer),
	}
	state := new(multistep.BasicStateBag)
	genData := packerbuilderdata.GeneratedData{State: state}
	vmSourcedSigID := "/subscriptions/00000000-0000-0000-0000-000000000000/resourceGroups/pkr-Resource-Group-blah/providers/Microsoft.Compute/virtualMachines/pkrvmexample"
	managedImageSourcedSigID := "/subscriptions/00000000-0000-0000-0000-000000000000/resourceGroups/pkr-Resource-Group-blah/providers/Microsoft.Compute/images/exampleimage"
	sigArtifactID := "example-sig-id"
	vmSourcedSigImageVersion := &galleryimageversions.GalleryImageVersion{
		Id: &sigArtifactID,
		Properties: &galleryimageversions.GalleryImageVersionProperties{
			StorageProfile: galleryimageversions.GalleryImageVersionStorageProfile{
				Source: &galleryimageversions.GalleryArtifactVersionFullSource{
					Id:               &vmSourcedSigID,
					VirtualMachineId: &vmSourcedSigID,
				},
			},
		},
	}

	managedImageSourcedSigImageVersion := &galleryimageversions.GalleryImageVersion{
		Id: &sigArtifactID,
		Properties: &galleryimageversions.GalleryImageVersionProperties{
			StorageProfile: galleryimageversions.GalleryImageVersionStorageProfile{
				Source: &galleryimageversions.GalleryArtifactVersionFullSource{
					Id:               &managedImageSourcedSigID,
					VirtualMachineId: &managedImageSourcedSigID,
				},
			},
		},
	}

	tc := []struct {
		name               string
		config             *Config
		expected           string
		mockedGalleryImage *galleryimageversions.GalleryImageVersion
	}{
		{
			name:     "ImageUrl",
			config:   &Config{ImageUrl: "https://azure/vhd"},
			expected: "https://azure/vhd",
		},
		{
			name: "CustomManagedImageName",
			config: &Config{
				CustomManagedImageName: "/subscription/1/resource/name",
				// During build time the custom managed id source is resolved
				// and stored as customManagedImageID
				customManagedImageID: "/subscription/0/resource/managedimage/12",
			},
			expected: "/subscription/0/resource/managedimage/12",
		},
		{
			name: "MarketPlaceImage",
			config: &Config{
				ClientConfig:   client.Config{SubscriptionID: "1234"},
				Location:       "west",
				ImagePublisher: "Microsoft",
				ImageOffer:     "Server",
				ImageSku:       "0",
				ImageVersion:   "2019",
			},
			expected: "/subscriptions/1234/providers/Microsoft.Compute/locations/west/publishers/Microsoft/ArtifactTypes/vmimage/offers/Server/skus/0/versions/2019",
		},
		{
			name: "SharedImageGallery - VM Sourced (direct publish to SIG)",
			config: &Config{
				ClientConfig: client.Config{SubscriptionID: "1234"},
				SharedGallery: SharedImageGallery{
					Subscription: "1234",
				},
			},
			mockedGalleryImage: vmSourcedSigImageVersion,
			expected:           sigArtifactID,
		},
		{
			name: "SharedImageGallery - Managed Image Sourced",
			config: &Config{
				ClientConfig: client.Config{SubscriptionID: "1234"},
				SharedGallery: SharedImageGallery{
					Subscription: "1234",
				},
			},
			mockedGalleryImage: managedImageSourcedSigImageVersion,
			expected:           managedImageSourcedSigID,
		},
	}
	for _, tt := range tc {
		tt := tt
		t.Run(tt.name, func(t *testing.T) {
			var step StepGetSourceImageName
			step = StepGetSourceImageName{
				config:        tt.config,
				GeneratedData: &genData,
				say:           ui.Say,
				error:         func(e error) {},
			}
			if tt.mockedGalleryImage != nil {
				step = StepGetSourceImageName{
					config:        tt.config,
					GeneratedData: &genData,
					say:           ui.Say,
					error:         func(e error) {},
					getGalleryVersion: func(ctx context.Context) (*galleryimageversions.GalleryImageVersion, error) {
						return tt.mockedGalleryImage, nil
					},
				}
			}
			step.Run(context.TODO(), state)
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
