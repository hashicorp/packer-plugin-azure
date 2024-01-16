// Copyright (c) HashiCorp, Inc.
// SPDX-License-Identifier: MPL-2.0

package chroot

import (
	"context"
	"fmt"
	"reflect"
	"testing"

	"github.com/hashicorp/go-azure-sdk/resource-manager/compute/2022-03-03/galleryimages"
	"github.com/hashicorp/go-azure-sdk/resource-manager/compute/2022-03-03/galleryimageversions"
	"github.com/hashicorp/packer-plugin-azure/builder/azure/common"
	"github.com/hashicorp/packer-plugin-azure/builder/azure/common/client"
	"github.com/hashicorp/packer-plugin-sdk/multistep"
	packersdk "github.com/hashicorp/packer-plugin-sdk/packer"
)

func TestStepVerifySharedImageDestination_Run(t *testing.T) {

	type fields struct {
		Image    SharedImageGalleryDestination
		Location string
	}
	tests := []struct {
		name    string
		fields  fields
		want    multistep.StepAction
		wantErr string
	}{
		{
			name: "happy path",
			want: multistep.ActionContinue,
			fields: fields{
				Image: SharedImageGalleryDestination{
					ResourceGroup: "rg",
					GalleryName:   "gallery",
					ImageName:     "image",
					ImageVersion:  "1.2.3",
				},
				Location: "region1",
			},
		},
		{
			name:    "not found",
			want:    multistep.ActionHalt,
			wantErr: "Error retrieving shared image \"/subscriptions/subscriptionID/resourcegroup/other-rg/providers/Microsoft.Compute/galleries/gallery/images/image\": Not Found ",
			fields: fields{
				Image: SharedImageGalleryDestination{
					ResourceGroup: "other-rg",
					GalleryName:   "gallery",
					ImageName:     "image",
					ImageVersion:  "1.2.3",
				},
				Location: "region1",
			},
		},
		{
			name:    "wrong region",
			want:    multistep.ActionHalt,
			wantErr: "Destination shared image resource \"image-resourceid-goes-here\" is in a different location (\"region1\") than this VM (\"other-region\").",
			fields: fields{
				Image: SharedImageGalleryDestination{
					ResourceGroup: "rg",
					GalleryName:   "gallery",
					ImageName:     "image",
					ImageVersion:  "1.2.3",
				},
				Location: "other-region",
			},
		},
		{
			name:    "version exists",
			want:    multistep.ActionHalt,
			wantErr: "Shared image version \"2.3.4\" already exists for image \"image-resourceid-goes-here\".",
			fields: fields{
				Image: SharedImageGalleryDestination{
					ResourceGroup: "rg",
					GalleryName:   "gallery",
					ImageName:     "image",
					ImageVersion:  "2.3.4",
				},
				Location: "region1",
			},
		},
		{
			name:    "not Linux",
			want:    multistep.ActionHalt,
			wantErr: "The shared image (\"windows-image-resourceid-goes-here\") is not a Linux image (found \"Windows\"). Currently only Linux images are supported.",
			fields: fields{
				Image: SharedImageGalleryDestination{
					ResourceGroup: "rg",
					GalleryName:   "gallery",
					ImageName:     "windowsimage",
					ImageVersion:  "1.2.3",
				},
				Location: "region1",
			},
		},
	}
	for _, tt := range tests {
		state := new(multistep.BasicStateBag)
		state.Put("azureclient", &client.AzureClientSetMock{
			SubscriptionIDMock: "subscriptionID",
		})
		state.Put("ui", packersdk.TestUi(t))

		t.Run(tt.name, func(t *testing.T) {
			s := &StepVerifySharedImageDestination{
				Image:    tt.fields.Image,
				Location: tt.fields.Location,
				getImage: func(ctx context.Context, acs client.AzureClientSet, id galleryimages.GalleryImageId) (*galleryimages.GalleryImage, error) {
					switch {
					case id.ImageName == "image" && id.GalleryName == "gallery" && id.ResourceGroupName == "rg":
						return &galleryimages.GalleryImage{
							Id:       common.StringPtr("image-resourceid-goes-here"),
							Location: "region1",
							Properties: &galleryimages.GalleryImageProperties{
								OsType: galleryimages.OperatingSystemTypesLinux,
							},
						}, nil
					case id.ImageName == "windowsimage" && id.GalleryName == "gallery" && id.ResourceGroupName == "rg":
						return &galleryimages.GalleryImage{
							Id:       common.StringPtr("windows-image-resourceid-goes-here"),
							Location: "region1",
							Properties: &galleryimages.GalleryImageProperties{
								OsType: galleryimages.OperatingSystemTypesWindows,
							},
						}, nil
					}
					return nil, fmt.Errorf("Not Found")
				},
				listVersions: func(ctx context.Context, acs client.AzureClientSet, id galleryimageversions.GalleryImageId) ([]galleryimageversions.GalleryImageVersion, error) {
					return []galleryimageversions.GalleryImageVersion{
						{
							Name: common.StringPtr("2.3.4"),
						},
					}, nil
				},
			}

			if got := s.Run(context.TODO(), state); !reflect.DeepEqual(got, tt.want) {
				t.Errorf("StepVerifySharedImageDestination.Run() = %v, want %v", got, tt.want)
			}

			if err, ok := state.GetOk("error"); ok {
				if err.(error).Error() != tt.wantErr {
					t.Errorf("Unexpected error, got: %q, want: %q", err, tt.wantErr)
				}
			} else if tt.wantErr != "" {
				t.Errorf("Expected error, but didn't get any")
			}
		})
	}
}
