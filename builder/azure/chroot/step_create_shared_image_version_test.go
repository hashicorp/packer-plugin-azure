// Copyright IBM Corp. 2013, 2025
// SPDX-License-Identifier: MPL-2.0

package chroot

import (
	"context"
	"testing"

	"github.com/google/go-cmp/cmp"
	"github.com/hashicorp/go-azure-sdk/resource-manager/compute/2023-07-03/galleryimageversions"
	"github.com/hashicorp/packer-plugin-azure/builder/azure/common"
	"github.com/hashicorp/packer-plugin-azure/builder/azure/common/client"
	"github.com/hashicorp/packer-plugin-sdk/multistep"
	packersdk "github.com/hashicorp/packer-plugin-sdk/packer"
)

func TestStepCreateSharedImageVersion_Run(t *testing.T) {
	standardZRSStorageType := galleryimageversions.StorageAccountTypeStandardZRS
	hostCacheingRW := galleryimageversions.HostCachingReadWrite
	hostCacheingNone := galleryimageversions.HostCachingNone
	subscriptionID := "12345"
	type fields struct {
		Destination       SharedImageGalleryDestination
		OSDiskCacheType   string
		DataDiskCacheType string
		Location          string
	}
	tests := []struct {
		name                 string
		fields               fields
		snapshotset          Diskset
		want                 multistep.StepAction
		expectedImageVersion galleryimageversions.GalleryImageVersion
		expectedImageId      galleryimageversions.ImageVersionId
	}{
		{
			name: "happy path",
			fields: fields{
				Destination: SharedImageGalleryDestination{
					ResourceGroup: "ResourceGroup",
					GalleryName:   "GalleryName",
					ImageName:     "ImageName",
					ImageVersion:  "0.1.2",
					TargetRegions: []TargetRegion{
						{
							Name:               "region1",
							ReplicaCount:       5,
							StorageAccountType: "Standard_ZRS",
						},
					},
					ExcludeFromLatest: true,
				},
				OSDiskCacheType:   "ReadWrite",
				DataDiskCacheType: "None",
				Location:          "region2",
			},
			snapshotset: diskset(
				"/subscriptions/12345/resourceGroups/group1/providers/Microsoft.Compute/snapshots/osdisksnapshot",
				"/subscriptions/12345/resourceGroups/group1/providers/Microsoft.Compute/snapshots/datadisksnapshot0",
				"/subscriptions/12345/resourceGroups/group1/providers/Microsoft.Compute/snapshots/datadisksnapshot1",
				"/subscriptions/12345/resourceGroups/group1/providers/Microsoft.Compute/snapshots/datadisksnapshot2"),
			expectedImageId: galleryimageversions.NewImageVersionID(
				subscriptionID,
				"ResourceGroup",
				"GalleryName",
				"ImageName",
				"0.1.2",
			),
			expectedImageVersion: galleryimageversions.GalleryImageVersion{
				Location: "region2",
				Properties: &galleryimageversions.GalleryImageVersionProperties{
					PublishingProfile: &galleryimageversions.GalleryArtifactPublishingProfileBase{
						ExcludeFromLatest: common.BoolPtr(true),
						TargetRegions: &[]galleryimageversions.TargetRegion{
							{
								Name:                 "region1",
								RegionalReplicaCount: common.Int64Ptr(5),
								StorageAccountType:   &standardZRSStorageType,
							},
						},
					},
					StorageProfile: galleryimageversions.GalleryImageVersionStorageProfile{
						OsDiskImage: &galleryimageversions.GalleryDiskImage{
							Source: &galleryimageversions.GalleryDiskImageSource{
								Id: common.StringPtr("/subscriptions/12345/resourceGroups/group1/providers/Microsoft.Compute/snapshots/osdisksnapshot"),
							},
							HostCaching: &hostCacheingRW,
						},
						DataDiskImages: &[]galleryimageversions.GalleryDataDiskImage{
							{
								HostCaching: &hostCacheingNone,
								Lun:         0,
								Source: &galleryimageversions.GalleryDiskImageSource{
									Id: common.StringPtr("/subscriptions/12345/resourceGroups/group1/providers/Microsoft.Compute/snapshots/datadisksnapshot0"),
								},
							},
							{
								HostCaching: &hostCacheingNone,
								Lun:         1,
								Source: &galleryimageversions.GalleryDiskImageSource{
									Id: common.StringPtr("/subscriptions/12345/resourceGroups/group1/providers/Microsoft.Compute/snapshots/datadisksnapshot1"),
								},
							},
							{
								HostCaching: &hostCacheingNone,
								Lun:         2,
								Source: &galleryimageversions.GalleryDiskImageSource{
									Id: common.StringPtr("/subscriptions/12345/resourceGroups/group1/providers/Microsoft.Compute/snapshots/datadisksnapshot2"),
								},
							},
						},
					},
				},
			},
		},
	}
	for _, tt := range tests {
		state := new(multistep.BasicStateBag)
		state.Put("azureclient", &client.AzureClientSetMock{
			SubscriptionIDMock: subscriptionID,
		})
		state.Put("ui", packersdk.TestUi(t))
		state.Put(stateBagKey_Snapshotset, tt.snapshotset)

		t.Run(tt.name, func(t *testing.T) {
			var actualID galleryimageversions.ImageVersionId
			var actualImageVersion galleryimageversions.GalleryImageVersion
			s := &StepCreateSharedImageVersion{
				Destination:       tt.fields.Destination,
				OSDiskCacheType:   tt.fields.OSDiskCacheType,
				DataDiskCacheType: tt.fields.DataDiskCacheType,
				Location:          tt.fields.Location,
				create: func(ctx context.Context, azcli client.AzureClientSet, id galleryimageversions.ImageVersionId, imageVersion galleryimageversions.GalleryImageVersion) error {
					actualID = id
					actualImageVersion = imageVersion
					return nil
				},
			}

			action := s.Run(context.TODO(), state)
			if action != multistep.ActionContinue {
				t.Fatalf("Expected ActionContinue got %s", action)
			}
			if diff := cmp.Diff(actualImageVersion, tt.expectedImageVersion); diff != "" {
				t.Fatalf("unexpected image version %s", diff)
			}
			if actualID != tt.expectedImageId {
				t.Fatalf("Expected image ID %+v got %+v", tt.expectedImageId, actualID)
			}
		})
	}
}
