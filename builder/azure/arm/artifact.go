// Copyright (c) HashiCorp, Inc.
// SPDX-License-Identifier: MPL-2.0

package arm

import (
	"bytes"
	"fmt"
	"strings"

	"github.com/hashicorp/packer-plugin-azure/builder/azure/common/constants"
	registryimage "github.com/hashicorp/packer-plugin-sdk/packer/registry/image"
)

const (
	BuilderId = "Azure.ResourceManagement.VMImage"
)

type AdditionalDiskArtifact struct {
	AdditionalDiskUri string
}

type ManagedImageArtifact struct {
	// Managed Image
	ManagedImageResourceGroupName      string
	ManagedImageName                   string
	ManagedImageLocation               string
	ManagedImageId                     string
	ManagedImageOSDiskSnapshotName     string
	ManagedImageOSDiskUri              string // this is used when ArmKeepOSDisk is true
	ManagedImageDataDiskSnapshotPrefix string
}

type VHDArtifact struct {
	// VHD
	StorageAccountLocation string
	OSDiskUri              string
	AdditionalDisks        *[]AdditionalDiskArtifact
}

type SharedImageGalleryArtifact struct {
	// Shared Image Gallery
	// ARM resource id for Shared Image Gallery
	ManagedImageSharedImageGalleryId string
	SharedImageGalleryLocation       string
}

type Artifact struct {
	// OS type: Linux, Windows
	OSType string

	VHD                VHDArtifact
	ManagedImage       ManagedImageArtifact
	SharedImageGallery SharedImageGalleryArtifact

	// StateData should store data such as GeneratedData
	// to be shared with post-processors
	StateData map[string]interface{}
}

func NewArtifact(osType string, vhd VHDArtifact, managedImage ManagedImageArtifact, sig SharedImageGalleryArtifact, stateData map[string]interface{}) *Artifact {
	return &Artifact{
		OSType:             osType,
		VHD:                vhd,
		ManagedImage:       managedImage,
		SharedImageGallery: sig,
		StateData:          stateData,
	}
}

func (a *Artifact) isManagedImage() bool {
	return a.ManagedImage.ManagedImageResourceGroupName != ""
}

func (a *Artifact) isVHDCopyToStorage() bool {
	return a.VHD.OSDiskUri != ""
}

func (a *Artifact) isPublishedToSIG() bool {
	return a.SharedImageGallery.ManagedImageSharedImageGalleryId != ""
}

func (*Artifact) BuilderId() string {
	return BuilderId
}

func (*Artifact) Files() []string {
	return []string{}
}

func (a *Artifact) Id() string {
	if a.ManagedImage.ManagedImageOSDiskUri != "" {
		return a.ManagedImage.ManagedImageOSDiskUri
	}
	if a.ManagedImage.ManagedImageId != "" {
		return a.ManagedImage.ManagedImageId
	}
	if a.SharedImageGallery.ManagedImageSharedImageGalleryId != "" {
		return a.SharedImageGallery.ManagedImageSharedImageGalleryId
	}
	if a.VHD.OSDiskUri != "" {
		return a.VHD.OSDiskUri
	}
	return "UNKNOWN ID"
}

func (a *Artifact) State(name string) interface{} {
	if name == registryimage.ArtifactStateURI {
		return a.hcpPackerRegistryMetadata()
	}

	if _, ok := a.StateData[name]; ok {
		return a.StateData[name]
	}

	return nil
}

func (a *Artifact) String() string {
	var buf bytes.Buffer

	buf.WriteString(fmt.Sprintf("%s:\n\n", a.BuilderId()))
	buf.WriteString(fmt.Sprintf("OSType: %s\n", a.OSType))
	// fix to string
	if a.isManagedImage() {
		buf.WriteString(fmt.Sprintf("ManagedImageResourceGroupName: %s\n", a.ManagedImage.ManagedImageResourceGroupName))
		buf.WriteString(fmt.Sprintf("ManagedImageName: %s\n", a.ManagedImage.ManagedImageName))
		buf.WriteString(fmt.Sprintf("ManagedImageId: %s\n", a.ManagedImage.ManagedImageId))
		buf.WriteString(fmt.Sprintf("ManagedImageLocation: %s\n", a.ManagedImage.ManagedImageLocation))
		if a.ManagedImage.ManagedImageOSDiskSnapshotName != "" {
			buf.WriteString(fmt.Sprintf("ManagedImageOSDiskSnapshotName: %s\n", a.ManagedImage.ManagedImageOSDiskSnapshotName))
		}
		if a.ManagedImage.ManagedImageDataDiskSnapshotPrefix != "" {
			buf.WriteString(fmt.Sprintf("ManagedImageDataDiskSnapshotPrefix: %s\n", a.ManagedImage.ManagedImageDataDiskSnapshotPrefix))
		}
		if a.ManagedImage.ManagedImageOSDiskUri != "" {
			buf.WriteString(fmt.Sprintf("OSDiskUri: %s\n", a.ManagedImage.ManagedImageOSDiskUri))
		}
	}

	if a.isVHDCopyToStorage() {
		buf.WriteString(fmt.Sprintf("StorageAccountLocation: %s\n", a.VHD.StorageAccountLocation))
		buf.WriteString(fmt.Sprintf("VHDOSDiskUri: %s\n", a.VHD.OSDiskUri))
		if a.VHD.AdditionalDisks != nil {
			for i, additionalDisk := range *a.VHD.AdditionalDisks {
				buf.WriteString(fmt.Sprintf("AdditionalDiskUri (datadisk-%d): %s\n", i+1, additionalDisk.AdditionalDiskUri))
			}
		}
	}

	if a.isPublishedToSIG() {
		buf.WriteString(fmt.Sprintf("ManagedImageSharedImageGalleryId: %s\n", a.SharedImageGallery.ManagedImageSharedImageGalleryId))
		if x, ok := a.State(constants.ArmManagedImageSigPublishResourceGroup).(string); ok {
			buf.WriteString(fmt.Sprintf("SharedImageGalleryResourceGroup: %s\n", x))
		}
		if x, ok := a.State(constants.ArmManagedImageSharedGalleryName).(string); ok {
			buf.WriteString(fmt.Sprintf("SharedImageGalleryName: %s\n", x))
		}
		if x, ok := a.State(constants.ArmManagedImageSharedGalleryImageName).(string); ok {
			buf.WriteString(fmt.Sprintf("SharedImageGalleryImageName: %s\n", x))
		}
		if x, ok := a.State(constants.ArmManagedImageSharedGalleryImageVersion).(string); ok {
			buf.WriteString(fmt.Sprintf("SharedImageGalleryImageVersion: %s\n", x))
		}
		if rr, ok := a.State(constants.ArmManagedImageSharedGalleryReplicationRegions).([]string); ok {
			buf.WriteString(fmt.Sprintf("SharedImageGalleryReplicatedRegions: %s\n", strings.Join(rr, ", ")))
		}
	}

	return buf.String()
}

func (*Artifact) Destroy() error {
	return nil
}

func (a *Artifact) hcpPackerRegistryMetadata() *registryimage.Image {
	var generatedData map[string]interface{}

	if a.StateData != nil {
		generatedData = a.StateData["generated_data"].(map[string]interface{})
	}

	var sourceID string
	if sourceImage, ok := generatedData["SourceImageName"].(string); ok {
		sourceID = sourceImage
	}

	labels := make(map[string]interface{})

	var id, location string

	// If image is a VHD
	if a.isVHDCopyToStorage() {
		id = a.VHD.OSDiskUri
		location = a.VHD.StorageAccountLocation

		labels["storage_account_location"] = a.VHD.StorageAccountLocation
	}

	// If image is published to SharedImageGallery
	if a.isPublishedToSIG() {
		id = a.SharedImageGallery.ManagedImageSharedImageGalleryId
		location = a.SharedImageGallery.SharedImageGalleryLocation

		labels["sig_resource_group"] = a.State(constants.ArmManagedImageSigPublishResourceGroup).(string)
		labels["sig_name"] = a.State(constants.ArmManagedImageSharedGalleryName).(string)
		labels["sig_image_name"] = a.State(constants.ArmManagedImageSharedGalleryImageName).(string)
		labels["sig_image_version"] = a.State(constants.ArmManagedImageSharedGalleryImageVersion).(string)
		if rr, ok := a.State(constants.ArmManagedImageSharedGalleryReplicationRegions).([]string); ok {
			labels["sig_replicated_regions"] = strings.Join(rr, ", ")
		}
	}

	// If image is captured as a managed image
	if a.isManagedImage() {
		id = a.ManagedImage.ManagedImageId
		location = a.ManagedImage.ManagedImageLocation

		labels["os_type"] = a.OSType
		labels["managed_image_resourcegroup_name"] = a.ManagedImage.ManagedImageResourceGroupName
		labels["managed_image_name"] = a.ManagedImage.ManagedImageName

		if a.ManagedImage.ManagedImageOSDiskUri != "" {
			labels["os_disk_uri"] = a.ManagedImage.ManagedImageOSDiskUri
		}
	}

	img, _ := registryimage.FromArtifact(a,
		registryimage.WithID(id),
		registryimage.WithRegion(location),
		registryimage.WithProvider("azure"),
		registryimage.WithSourceID(sourceID),
		registryimage.SetLabels(labels),
	)
	return img
}
