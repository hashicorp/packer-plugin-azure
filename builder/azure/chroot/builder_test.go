// Copyright (c) HashiCorp, Inc.
// SPDX-License-Identifier: MPL-2.0

package chroot

import (
	"strings"
	"testing"

	"github.com/hashicorp/go-azure-sdk/resource-manager/compute/2022-03-01/virtualmachines"

	"github.com/hashicorp/packer-plugin-azure/builder/azure/common/client"
	"github.com/hashicorp/packer-plugin-sdk/multistep"
	"github.com/hashicorp/packer-plugin-sdk/packerbuilderdata"
)

func TestBuilder_Prepare(t *testing.T) {
	type config map[string]interface{}

	tests := []struct {
		name     string
		config   config
		validate func(Config)
		wantErr  bool
	}{
		{
			name: "platform image to managed disk",
			config: config{
				"client_id":         "123",
				"client_secret":     "456",
				"subscription_id":   "789",
				"source":            "credativ:Debian:9:latest",
				"image_resource_id": "/subscriptions/789/resourceGroups/otherrgname/providers/Microsoft.Compute/images/MyDebianOSImage-{{timestamp}}",
				"shared_image_destination": config{
					"resource_group": "otherrgname",
					"gallery_name":   "myGallery",
					"image_name":     "imageName",
					"image_version":  "1.0.2",
				},
			},
			validate: func(c Config) {
				if c.OSDiskSizeGB != 0 {
					t.Errorf("Expected OSDiskSizeGB to be 0, was %+v", c.OSDiskSizeGB)
				}
				if c.MountPartition != "1" {
					t.Errorf("Expected MountPartition to be %s, but found %s", "1", c.MountPartition)
				}
				if c.OSDiskStorageAccountType != string(virtualmachines.StorageAccountTypesPremiumLRS) {
					t.Errorf("Expected OSDiskStorageAccountType to be %s, but found %s", string(virtualmachines.StorageAccountTypesPremiumLRS), c.OSDiskStorageAccountType)
				}
				if c.OSDiskCacheType != string(virtualmachines.CachingTypesReadOnly) {
					t.Errorf("Expected OSDiskCacheType to be %s, but found %s", string(virtualmachines.CachingTypesReadOnly), c.OSDiskCacheType)
				}
				if c.ImageHyperVGeneration != string(virtualmachines.HyperVGenerationTypeVOne) {
					t.Errorf("Expected ImageHyperVGeneration to be %s, but found %s", string(virtualmachines.HyperVGenerationTypeVOne), c.ImageHyperVGeneration)
				}
			},
		},
		{
			name: "disk to managed image, validate temp disk id expansion",
			config: config{
				"source":            "/subscriptions/789/resourceGroups/testrg/providers/Microsoft.Compute/disks/diskname",
				"image_resource_id": "/subscriptions/789/resourceGroups/otherrgname/providers/Microsoft.Compute/images/MyDebianOSImage-{{timestamp}}",
			},
			validate: func(c Config) {
				prefix := "/subscriptions/testSubscriptionID/resourceGroups/testResourceGroup/providers/Microsoft.Compute/disks/PackerTemp-osdisk-"
				if !strings.HasPrefix(c.TemporaryOSDiskID, prefix) {
					t.Errorf("Expected TemporaryOSDiskID to start with %q, but got %q", prefix, c.TemporaryOSDiskID)
				}
			},
		},
		{
			name: "disk to both managed image and shared image",
			config: config{
				"source":            "/subscriptions/789/resourceGroups/testrg/providers/Microsoft.Compute/disks/diskname",
				"image_resource_id": "/subscriptions/789/resourceGroups/otherrgname/providers/Microsoft.Compute/images/MyDebianOSImage-{{timestamp}}",
				"shared_image_destination": config{
					"resource_group": "rg",
					"gallery_name":   "galleryName",
					"image_name":     "imageName",
					"image_version":  "0.1.0",
				},
			},
		},
		{
			name: "disk to both managed image and shared image with missing property",
			config: config{
				"source":            "/subscriptions/789/resourceGroups/testrg/providers/Microsoft.Compute/disks/diskname",
				"image_resource_id": "/subscriptions/789/resourceGroups/otherrgname/providers/Microsoft.Compute/images/MyDebianOSImage-{{timestamp}}",
				"shared_image_destination": config{
					"resource_group": "rg",
					"gallery_name":   "galleryName",
					"image_version":  "0.1.0",
				},
			},
			wantErr: true,
		},
		{
			name: "from shared image",
			config: config{
				"shared_image_destination": config{
					"resource_group": "otherrgname",
					"gallery_name":   "myGallery",
					"image_name":     "imageName",
					"image_version":  "1.0.2",
				},
				"source": "/subscriptions/789/resourceGroups/testrg/providers/Microsoft.Compute/disks/diskname",
			},
			wantErr: false,
		},
		{
			name: "err: no output",
			config: config{
				"source": "/subscriptions/789/resourceGroups/testrg/providers/Microsoft.Compute/disks/diskname",
			},
			wantErr: true,
		},
	}
	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			withMetadataStub(func() {
				b := &Builder{}

				_, _, err := b.Prepare(tt.config)

				if (err != nil) != tt.wantErr {
					t.Errorf("Builder.Prepare() error = %v, wantErr %v", err, tt.wantErr)
					return
				}

				if tt.validate != nil {
					tt.validate(b.config)
				}
			})
		})
	}
}

func Test_buildsteps(t *testing.T) {
	info := &client.ComputeInfo{
		Location:          "northpole",
		Name:              "unittestVM",
		ResourceGroupName: "unittestResourceGroup",
		SubscriptionID:    "96854241-60c7-426d-9a27-3fdeec8957f4",
	}

	tests := []struct {
		name   string
		config Config
		verify func([]multistep.Step, *testing.T)
	}{
		{
			name:   "Source FromScrath creates empty disk",
			config: Config{FromScratch: true},
			verify: func(steps []multistep.Step, _ *testing.T) {
				for _, s := range steps {
					if s, ok := s.(*StepCreateNewDiskset); ok {
						if s.SourceOSDiskResourceID == "" &&
							s.SourcePlatformImage == nil {
							return
						}
						t.Errorf("found misconfigured StepCreateNewDisk: %+v", s)
					}
				}
				t.Error("did not find a StepCreateNewDisk")
			}},
		{
			name:   "Source Platform image disk creation",
			config: Config{Source: "publisher:offer:sku:version", sourceType: sourcePlatformImage},
			verify: func(steps []multistep.Step, _ *testing.T) {
				for _, s := range steps {
					if s, ok := s.(*StepCreateNewDiskset); ok {
						if s.SourceOSDiskResourceID == "" &&
							s.SourcePlatformImage != nil &&
							s.SourcePlatformImage.Publisher == "publisher" {
							return
						}
						t.Errorf("found misconfigured StepCreateNewDisk: %+v", s)
					}
				}
				t.Error("did not find a StepCreateNewDisk")
			}},
		{
			name:   "Source Platform image with version latest adds StepResolvePlatformImageVersion",
			config: Config{Source: "publisher:offer:sku:latest", sourceType: sourcePlatformImage},
			verify: func(steps []multistep.Step, _ *testing.T) {
				for _, s := range steps {
					if s, ok := s.(*StepResolvePlatformImageVersion); ok {
						if s.PlatformImage != nil &&
							s.Location == info.Location {
							return
						}
						t.Errorf("found misconfigured StepResolvePlatformImageVersion: %+v", s)
					}
				}
				t.Error("did not find a StepResolvePlatformImageVersion")
			}},
		{
			name:   "Source Disk adds correct disk creation",
			config: Config{Source: "diskresourceid", sourceType: sourceDisk},
			verify: func(steps []multistep.Step, _ *testing.T) {
				for _, s := range steps {
					if s, ok := s.(*StepCreateNewDiskset); ok {
						if s.SourceOSDiskResourceID == "diskresourceid" &&
							s.SourcePlatformImage == nil {
							return
						}
						t.Errorf("found misconfigured StepCreateNewDisk: %+v", s)
					}
				}
				t.Error("did not find a StepCreateNewDisk")
			}},
		{
			name:   "Source disk adds StepVerifySourceDisk",
			config: Config{Source: "diskresourceid", sourceType: sourceDisk},
			verify: func(steps []multistep.Step, _ *testing.T) {
				for _, s := range steps {
					if s, ok := s.(*StepVerifySourceDisk); ok {
						if s.SourceDiskResourceID == "diskresourceid" &&
							s.Location == info.Location {
							return
						}
						t.Errorf("found misconfigured StepVerifySourceDisk: %+v", s)
					}
				}
				t.Error("did not find a StepVerifySourceDisk")
			}},
	}
	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			withMetadataStub(func() { // ensure that values are taken from info, instead of retrieved again
				got := buildsteps(tt.config, info, &packerbuilderdata.GeneratedData{}, func(string) {})
				tt.verify(got, t)
			})
		})
	}
}
