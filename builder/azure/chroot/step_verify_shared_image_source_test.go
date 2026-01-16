// Copyright IBM Corp. 2013, 2025
// SPDX-License-Identifier: MPL-2.0

package chroot

import (
	"context"
	"fmt"
	"reflect"
	"strings"
	"testing"

	"github.com/hashicorp/go-azure-sdk/resource-manager/compute/2023-07-03/galleryimages"
	"github.com/hashicorp/go-azure-sdk/resource-manager/compute/2023-07-03/galleryimageversions"
	"github.com/hashicorp/packer-plugin-azure/builder/azure/common"
	"github.com/hashicorp/packer-plugin-azure/builder/azure/common/client"
	"github.com/hashicorp/packer-plugin-sdk/multistep"
	packersdk "github.com/hashicorp/packer-plugin-sdk/packer"
)

func TestStepVerifySharedImageSource_Run(t *testing.T) {
	type fields struct {
		SharedImageID  string
		SubscriptionID string
		Location       string
	}
	tests := []struct {
		name                 string
		fields               fields
		want                 multistep.StepAction
		wantErr              string
		shouldCallGetVersion bool
		shouldCallGetImage   bool
	}{
		{
			name: "happy path",
			fields: fields{
				SharedImageID: "/subscriptions/subscriptionID/resourceGroups/rg/providers/Microsoft.Compute/galleries/myGallery/images/myImage/versions/1.2.3",
				Location:      "VM location",
			},
			shouldCallGetVersion: true,
			shouldCallGetImage:   true,
		},
		{
			name: "resource is not a shared image",
			fields: fields{
				SharedImageID: "/subscriptions/subscriptionID/resourceGroups/rg/providers/Microsoft.Compute/disks/myDisk",
				Location:      "VM location",
			},
			want:    multistep.ActionHalt,
			wantErr: "does not identify a shared image version",
		},
		{
			name: "error in resource id",
			fields: fields{
				SharedImageID: "not-a-resource-id",
			},
			want:    multistep.ActionHalt,
			wantErr: "Could not parse resource id",
		},
		{
			name: "wrong location",
			fields: fields{
				SharedImageID: "/subscriptions/subscriptionID/resourceGroups/rg/providers/Microsoft.Compute/galleries/myGallery/images/myImage/versions/1.2.3",
				Location:      "other location",
			},
			want:                 multistep.ActionHalt,
			wantErr:              "does not include VM location",
			shouldCallGetVersion: true,
		},
		{
			name: "image not found",
			fields: fields{
				SharedImageID: "/subscriptions/subscriptionID/resourceGroups/rg/providers/Microsoft.Compute/galleries/myGallery/images/myImage/versions/2.3.4",
				Location:      "vm location",
			},
			want:                 multistep.ActionHalt,
			wantErr:              "Error retrieving shared image version",
			shouldCallGetVersion: true,
		},
		{
			name: "windows image",
			fields: fields{
				SharedImageID: "/subscriptions/subscriptionID/resourceGroups/rg/providers/Microsoft.Compute/galleries/myGallery/images/windowsImage/versions/1.2.3",
				Location:      "VM location",
			},
			want:                 multistep.ActionHalt,
			wantErr:              "not a Linux image",
			shouldCallGetVersion: true,
			shouldCallGetImage:   true,
		},
	}
	for _, tt := range tests {

		state := new(multistep.BasicStateBag)
		state.Put("azureclient", &client.AzureClientSetMock{
			SubscriptionIDMock: "subscriptionID",
		})
		state.Put("ui", packersdk.TestUi(t))

		t.Run(tt.name, func(t *testing.T) {
			s := &StepVerifySharedImageSource{
				SharedImageID:  tt.fields.SharedImageID,
				SubscriptionID: tt.fields.SubscriptionID,
				Location:       tt.fields.Location,
				getImage: func(ctx context.Context, acs client.AzureClientSet, id galleryimages.GalleryImageId) (*galleryimages.GalleryImage, error) {
					if !tt.shouldCallGetImage {
						t.Fatalf("Expected test to not call getImage but it did")
					}
					switch {
					case strings.HasSuffix(id.ImageName, "windowsImage"):
						return &galleryimages.GalleryImage{
							Id: common.StringPtr("image-id"),
							Properties: &galleryimages.GalleryImageProperties{
								OsType: galleryimages.OperatingSystemTypesWindows,
							},
						}, nil
					case strings.HasSuffix(id.ImageName, "myImage"):
						return &galleryimages.GalleryImage{
							Id: common.StringPtr("image-id"),
							Properties: &galleryimages.GalleryImageProperties{
								OsType: galleryimages.OperatingSystemTypesLinux,
							},
						}, nil
					default:
						return nil, fmt.Errorf("Unexpected image")
					}
				},
				getVersion: func(ctx context.Context, azcli client.AzureClientSet, id galleryimageversions.ImageVersionId) (*galleryimageversions.GalleryImageVersion, error) {
					if !tt.shouldCallGetVersion {
						t.Fatalf("Expected test to not call getVersion but it did")
					}
					switch id.VersionName {
					case "1.2.3":
						return &galleryimageversions.GalleryImageVersion{
							Id: common.StringPtr("image-version-id"),
							Properties: &galleryimageversions.GalleryImageVersionProperties{
								PublishingProfile: &galleryimageversions.GalleryArtifactPublishingProfileBase{
									TargetRegions: &[]galleryimageversions.TargetRegion{
										{
											Name: "vm Location",
										},
									},
								},
							},
						}, nil
					default:
						return nil, fmt.Errorf("Not found")
					}
				},
			}
			if got := s.Run(context.TODO(), state); !reflect.DeepEqual(got, tt.want) {
				t.Errorf("StepVerifySharedImageSource.Run() = %v, want %v", got, tt.want)
			}
			d, _ := state.GetOk("error")
			err, _ := d.(error)
			if tt.wantErr != "" {
				if !strings.Contains(err.Error(), tt.wantErr) {
					t.Errorf("Wanted error %q, got %q", tt.wantErr, err)
				}
			} else if err != nil && err.Error() != "" {
				t.Errorf("Unexpected error: %v", err)
			}
		})
	}
}
