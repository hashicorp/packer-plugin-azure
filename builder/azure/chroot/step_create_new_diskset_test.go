// Copyright IBM Corp. 2013, 2025
// SPDX-License-Identifier: MPL-2.0

package chroot

import (
	"context"
	"reflect"
	"testing"

	"github.com/google/go-cmp/cmp"
	"github.com/hashicorp/go-azure-helpers/resourcemanager/commonids"
	"github.com/hashicorp/go-azure-sdk/resource-manager/compute/2023-04-02/disks"
	"github.com/hashicorp/go-azure-sdk/resource-manager/compute/2023-07-03/galleryimageversions"
	"github.com/hashicorp/packer-plugin-azure/builder/azure/common"
	"github.com/hashicorp/packer-plugin-azure/builder/azure/common/client"
	"github.com/hashicorp/packer-plugin-sdk/multistep"
	packersdk "github.com/hashicorp/packer-plugin-sdk/packer"
)

func TestStepCreateNewDisk_Run(t *testing.T) {
	osType := disks.OperatingSystemTypesLinux
	hyperVGeneration := disks.HyperVGenerationVOne
	premiumLRS := disks.DiskStorageAccountTypesPremiumLRS
	standardLRS := disks.DiskStorageAccountTypesStandardLRS

	tests := []struct {
		name                  string
		fields                StepCreateNewDiskset
		expectedPutDiskBodies []string
		want                  multistep.StepAction
		verifyDiskset         *Diskset
		disks                 []disks.Disk
	}{
		{
			name: "from disk",
			fields: StepCreateNewDiskset{
				OSDiskID:                 "/subscriptions/SubscriptionID/resourcegroups/ResourceGroupName/providers/Microsoft.Compute/disks/TemporaryOSDiskName",
				OSDiskSizeGB:             42,
				OSDiskStorageAccountType: string(disks.DiskStorageAccountTypesStandardLRS),
				HyperVGeneration:         string(disks.HyperVGenerationVOne),
				Location:                 "westus",
				SourceOSDiskResourceID:   "SourceDisk",
			},
			disks: []disks.Disk{
				{
					Location: "westus",
					Sku: &disks.DiskSku{
						Name: &standardLRS,
					},
					Properties: &disks.DiskProperties{
						HyperVGeneration: &hyperVGeneration,
						OsType:           &osType,
						CreationData: disks.CreationData{
							CreateOption:     disks.DiskCreateOptionCopy,
							SourceResourceId: common.StringPtr("SourceDisk"),
						},
						DiskSizeGB: common.Int64Ptr(42),
					},
				},
			},
			expectedPutDiskBodies: []string{`
				{
					"location": "westus",
					"properties": {
						"creationData": {
							"createOption": "Copy",
							"sourceResourceId": "SourceDisk"
						},
						"diskSizeGB": 42,
						"hyperVGeneration": "V1",
						"osType": "Linux"
					},
					"sku": {
						"name": "Premium_LRS"
					}
				}`},
			want:          multistep.ActionContinue,
			verifyDiskset: &Diskset{-1: resource("/subscriptions/SubscriptionID/resourceGroups/ResourceGroupName/providers/Microsoft.Compute/disks/TemporaryOSDiskName")},
		},
		{
			name: "from platform image",
			fields: StepCreateNewDiskset{
				OSDiskID:                 "/subscriptions/SubscriptionID/resourcegroups/ResourceGroupName/providers/Microsoft.Compute/disks/TemporaryOSDiskName",
				OSDiskStorageAccountType: string(disks.DiskStorageAccountTypesStandardLRS),
				HyperVGeneration:         string(disks.HyperVGenerationVOne),
				Location:                 "westus",
				SourcePlatformImage: &client.PlatformImage{
					Publisher: "Microsoft",
					Offer:     "Windows",
					Sku:       "2016-DataCenter",
					Version:   "2016.1.4",
				},
			},
			disks: []disks.Disk{
				{
					Location: "westus",
					Sku: &disks.DiskSku{
						Name: &standardLRS,
					},
					Properties: &disks.DiskProperties{
						HyperVGeneration: &hyperVGeneration,
						OsType:           &osType,
						CreationData: disks.CreationData{
							CreateOption: disks.DiskCreateOptionFromImage,
							ImageReference: &disks.ImageDiskReference{
								Id: common.StringPtr("/subscriptions/SubscriptionID/providers/Microsoft.Compute/locations/westus/publishers/Microsoft/artifacttypes/vmimage/offers/Windows/skus/2016-DataCenter/versions/2016.1.4"),
							},
						},
					},
				},
			},
			expectedPutDiskBodies: []string{`
				{
					"location": "westus",
					"properties": {
						"creationData": {
							"createOption":"FromImage",
							"imageReference": {
								"id":"/subscriptions/SubscriptionID/providers/Microsoft.Compute/locations/westus/publishers/Microsoft/artifacttypes/vmimage/offers/Windows/skus/2016-DataCenter/versions/2016.1.4"
							}
						},
						"hyperVGeneration": "V1",
						"osType": "Linux"
					},
					"sku": {
						"name": "Standard_LRS"
					}
				}`},
			want:          multistep.ActionContinue,
			verifyDiskset: &Diskset{-1: resource("/subscriptions/SubscriptionID/resourceGroups/ResourceGroupName/providers/Microsoft.Compute/disks/TemporaryOSDiskName")},
		},
		{
			name: "from shared image",
			fields: StepCreateNewDiskset{
				OSDiskID:                   "/subscriptions/SubscriptionID/resourcegroups/ResourceGroupName/providers/Microsoft.Compute/disks/TemporaryOSDiskName",
				OSDiskStorageAccountType:   string(disks.DiskStorageAccountTypesStandardLRS),
				DataDiskStorageAccountType: string(disks.DiskStorageAccountTypesPremiumLRS),
				DataDiskIDPrefix:           "/subscriptions/SubscriptionID/resourcegroups/ResourceGroupName/providers/Microsoft.Compute/disks/TemporaryDataDisk-",
				HyperVGeneration:           string(disks.HyperVGenerationVOne),
				Location:                   "westus",
				SourceImageResourceID:      "/subscriptions/ImageSubscriptionID/resourcegroups/imagegroup/providers/Microsoft.Compute/galleries/MyGallery/images/MyImage/versions/1.2.3",
			},

			disks: []disks.Disk{
				{
					Location: "westus",
					Properties: &disks.DiskProperties{
						CreationData: disks.CreationData{
							CreateOption: disks.DiskCreateOptionFromImage,
							GalleryImageReference: &disks.ImageDiskReference{
								Id: common.StringPtr("/subscriptions/ImageSubscriptionID/resourcegroups/imagegroup/providers/Microsoft.Compute/galleries/MyGallery/images/MyImage/versions/1.2.3"),
							},
						},
						HyperVGeneration: &hyperVGeneration,
						OsType:           &osType,
					},
					Sku: &disks.DiskSku{
						Name: &standardLRS,
					},
				},
				{
					Location: "westus",
					Properties: &disks.DiskProperties{
						CreationData: disks.CreationData{
							CreateOption: disks.DiskCreateOptionFromImage,
							GalleryImageReference: &disks.ImageDiskReference{
								Id:  common.StringPtr("/subscriptions/ImageSubscriptionID/resourcegroups/imagegroup/providers/Microsoft.Compute/galleries/MyGallery/images/MyImage/versions/1.2.3"),
								Lun: common.Int64Ptr(5),
							},
						},
					},
					Sku: &disks.DiskSku{
						Name: &premiumLRS,
					},
				},
				{
					Location: "westus",
					Properties: &disks.DiskProperties{
						CreationData: disks.CreationData{
							CreateOption: disks.DiskCreateOptionFromImage,
							GalleryImageReference: &disks.ImageDiskReference{
								Id:  common.StringPtr("/subscriptions/ImageSubscriptionID/resourcegroups/imagegroup/providers/Microsoft.Compute/galleries/MyGallery/images/MyImage/versions/1.2.3"),
								Lun: common.Int64Ptr(9),
							},
						},
					},
					Sku: &disks.DiskSku{
						Name: &premiumLRS,
					},
				},
				{
					Location: "westus",
					Properties: &disks.DiskProperties{
						CreationData: disks.CreationData{
							CreateOption: disks.DiskCreateOptionFromImage,
							GalleryImageReference: &disks.ImageDiskReference{
								Id:  common.StringPtr("/subscriptions/ImageSubscriptionID/resourcegroups/imagegroup/providers/Microsoft.Compute/galleries/MyGallery/images/MyImage/versions/1.2.3"),
								Lun: common.Int64Ptr(3),
							},
						},
					},
					Sku: &disks.DiskSku{
						Name: &premiumLRS,
					},
				},
			},

			want: multistep.ActionContinue,
			verifyDiskset: &Diskset{
				-1: resource("/subscriptions/SubscriptionID/resourceGroups/ResourceGroupName/providers/Microsoft.Compute/disks/TemporaryOSDiskName"),
				3:  resource("/subscriptions/SubscriptionID/resourceGroups/ResourceGroupName/providers/Microsoft.Compute/disks/TemporaryDataDisk-3"),
				5:  resource("/subscriptions/SubscriptionID/resourceGroups/ResourceGroupName/providers/Microsoft.Compute/disks/TemporaryDataDisk-5"),
				9:  resource("/subscriptions/SubscriptionID/resourceGroups/ResourceGroupName/providers/Microsoft.Compute/disks/TemporaryDataDisk-9"),
			},
		},
		{
			name: "from disk with availability zone",
			fields: StepCreateNewDiskset{
				OSDiskID:                 "/subscriptions/SubscriptionID/resourcegroups/ResourceGroupName/providers/Microsoft.Compute/disks/TemporaryOSDiskName",
				OSDiskSizeGB:             42,
				OSDiskStorageAccountType: string(disks.DiskStorageAccountTypesStandardLRS),
				HyperVGeneration:         string(disks.HyperVGenerationVOne),
				Location:                 "westus",
				Zone:                     "3",
				SourceOSDiskResourceID:   "SourceDisk",
			},
			disks: []disks.Disk{
				{
					Location: "westus",
					Zones:    &[]string{"3"},
					Sku: &disks.DiskSku{
						Name: &standardLRS,
					},
					Properties: &disks.DiskProperties{
						HyperVGeneration: &hyperVGeneration,
						OsType:           &osType,
						CreationData: disks.CreationData{
							CreateOption:     disks.DiskCreateOptionCopy,
							SourceResourceId: common.StringPtr("SourceDisk"),
						},
						DiskSizeGB: common.Int64Ptr(42),
					},
				},
			},
			want:          multistep.ActionContinue,
			verifyDiskset: &Diskset{-1: resource("/subscriptions/SubscriptionID/resourceGroups/ResourceGroupName/providers/Microsoft.Compute/disks/TemporaryOSDiskName")},
		},
	}
	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			bodyCount := 0
			s := StepCreateNewDiskset{
				OSDiskID:                   tt.fields.OSDiskID,
				OSDiskSizeGB:               tt.fields.OSDiskSizeGB,
				OSDiskStorageAccountType:   tt.fields.OSDiskStorageAccountType,
				DataDiskStorageAccountType: tt.fields.DataDiskStorageAccountType,
				DataDiskIDPrefix:           tt.fields.DataDiskIDPrefix,
				HyperVGeneration:           tt.fields.HyperVGeneration,
				Location:                   tt.fields.Location,
				Zone:                       tt.fields.Zone,
				SourceOSDiskResourceID:     tt.fields.SourceOSDiskResourceID,
				SourceImageResourceID:      tt.fields.SourceImageResourceID,
				SourcePlatformImage:        tt.fields.SourcePlatformImage,
				getVersion: func(ctx context.Context, acs client.AzureClientSet, id galleryimageversions.ImageVersionId) (*galleryimageversions.GalleryImageVersion, error) {
					if id.SubscriptionId != "ImageSubscriptionID" {
						t.Fatalf("expected gallery image version lookup in subscription 'ImageSubscriptionID', got '%s'", id.SubscriptionId)
					}
					return &galleryimageversions.GalleryImageVersion{
						Properties: &galleryimageversions.GalleryImageVersionProperties{
							StorageProfile: galleryimageversions.GalleryImageVersionStorageProfile{
								DataDiskImages: &[]galleryimageversions.GalleryDataDiskImage{
									{
										Lun: 5,
									},
									{
										Lun: 9,
									},
									{
										Lun: 3,
									},
								},
							},
						},
					}, nil
				},
				create: func(ctx context.Context, acs client.AzureClientSet, id commonids.ManagedDiskId, disk disks.Disk) error {
					if diff := cmp.Diff(disk, tt.disks[bodyCount]); diff != "" {
						t.Fatalf("unexpected disk for call %d diff %s", bodyCount+1, diff)
					}
					bodyCount++
					return nil
				},
			}

			state := new(multistep.BasicStateBag)
			state.Put("azureclient", &client.AzureClientSetMock{
				SubscriptionIDMock: "SubscriptionID",
			})
			state.Put("ui", packersdk.TestUi(t))

			if got := s.Run(context.TODO(), state); !reflect.DeepEqual(got, tt.want) {
				t.Errorf("StepCreateNewDisk.Run() = %v, want %v", got, tt.want)
			}

			ds := state.Get(stateBagKey_Diskset)
			if tt.verifyDiskset != nil && !reflect.DeepEqual(*tt.verifyDiskset, ds) {
				t.Errorf("Error verifying diskset after Run(), got %v, want %v", ds, *tt.verifyDiskset)
			}
		})
	}
}

func resource(id string) client.Resource {
	v, err := client.ParseResourceID(id)
	if err != nil {
		panic(err)
	}
	return v
}
