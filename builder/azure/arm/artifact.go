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

type Artifact struct {
	// OS type: Linux, Windows
	OSType string

	// VHD
	StorageAccountLocation string
	OSDiskUri              string
	TemplateUri            string

	// Managed Image
	ManagedImageResourceGroupName      string
	ManagedImageName                   string
	ManagedImageLocation               string
	ManagedImageId                     string
	ManagedImageOSDiskSnapshotName     string
	ManagedImageDataDiskSnapshotPrefix string

	// Shared Image Gallery
	// ARM resource id for Shared Image Gallery
	ManagedImageSharedImageGalleryId string
	SharedImageGalleryLocation       string

	// Additional Disks
	AdditionalDisks *[]AdditionalDiskArtifact

	// StateData should store data such as GeneratedData
	// to be shared with post-processors
	StateData map[string]interface{}
}

func NewManagedImageArtifact(osType, resourceGroup, name, location, id, osDiskSnapshotName, dataDiskSnapshotPrefix string, generatedData map[string]interface{}, osDiskUri string) (*Artifact, error) {
	res := Artifact{
		ManagedImageResourceGroupName:      resourceGroup,
		ManagedImageName:                   name,
		ManagedImageLocation:               location,
		ManagedImageId:                     id,
		OSType:                             osType,
		ManagedImageOSDiskSnapshotName:     osDiskSnapshotName,
		ManagedImageDataDiskSnapshotPrefix: dataDiskSnapshotPrefix,
		StateData:                          generatedData,
	}

	if osDiskUri != "" {
		res.OSDiskUri = osDiskUri
	}

	return &res, nil
}

func NewManagedImageArtifactWithSIGAsDestination(osType, resourceGroup, name, location, id, osDiskSnapshotName, dataDiskSnapshotPrefix, destinationSharedImageGalleryId string, generatedData map[string]interface{}) (*Artifact, error) {
	return &Artifact{
		ManagedImageResourceGroupName:      resourceGroup,
		ManagedImageName:                   name,
		ManagedImageLocation:               location,
		ManagedImageId:                     id,
		OSType:                             osType,
		ManagedImageOSDiskSnapshotName:     osDiskSnapshotName,
		ManagedImageDataDiskSnapshotPrefix: dataDiskSnapshotPrefix,
		ManagedImageSharedImageGalleryId:   destinationSharedImageGalleryId,
		StateData:                          generatedData,
	}, nil
}

func NewSharedImageArtifact(osType, destinationSharedImageGalleryId string, location string, generatedData map[string]interface{}) (*Artifact, error) {
	return &Artifact{
		OSType:                           osType,
		ManagedImageSharedImageGalleryId: destinationSharedImageGalleryId,
		StateData:                        generatedData,
		SharedImageGalleryLocation:       location,
	}, nil
}

func NewArtifact(vmInternalID string, storageAccountUrl string, storageAccountLocation string, osType string, additionalDiskCount int, generatedData map[string]interface{}) (*Artifact, error) {
	vhdUri := fmt.Sprintf("%ssystem/Microsoft.Compute/Images/images/packer-osDisk.%s.vhd", storageAccountUrl, vmInternalID)

	templateUri := fmt.Sprintf("%ssystem/Microsoft.Compute/Images/images/packer-vmTemplate.%s.json", storageAccountUrl, vmInternalID)

	var additional_disks *[]AdditionalDiskArtifact
	if additionalDiskCount > 0 {
		data_disks := make([]AdditionalDiskArtifact, additionalDiskCount)
		for i := 0; i < additionalDiskCount; i++ {
			data_disks[i].AdditionalDiskUri = fmt.Sprintf("%ssystem/Microsoft.Compute/Images/images/packer-datadisk-%d.%s.vhd", storageAccountUrl, i+1, vmInternalID)
		}
		additional_disks = &data_disks
	}

	return &Artifact{
		OSType:      osType,
		OSDiskUri:   vhdUri,
		TemplateUri: templateUri,

		AdditionalDisks: additional_disks,

		StorageAccountLocation: storageAccountLocation,

		StateData: generatedData,
	}, nil
}

func (a *Artifact) isManagedImage() bool {
	return a.ManagedImageResourceGroupName != ""
}

func (a *Artifact) isPublishedToSIG() bool {
	return a.ManagedImageSharedImageGalleryId != ""
}

func (*Artifact) BuilderId() string {
	return BuilderId
}

func (*Artifact) Files() []string {
	return []string{}
}

func (a *Artifact) Id() string {
	if a.OSDiskUri != "" {
		return a.OSDiskUri
	}
	if a.ManagedImageId != "" {
		return a.ManagedImageId
	}
	if a.ManagedImageSharedImageGalleryId != "" {
		return a.ManagedImageSharedImageGalleryId
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
	if a.isManagedImage() {
		buf.WriteString(fmt.Sprintf("ManagedImageResourceGroupName: %s\n", a.ManagedImageResourceGroupName))
		buf.WriteString(fmt.Sprintf("ManagedImageName: %s\n", a.ManagedImageName))
		buf.WriteString(fmt.Sprintf("ManagedImageId: %s\n", a.ManagedImageId))
		buf.WriteString(fmt.Sprintf("ManagedImageLocation: %s\n", a.ManagedImageLocation))
		if a.ManagedImageOSDiskSnapshotName != "" {
			buf.WriteString(fmt.Sprintf("ManagedImageOSDiskSnapshotName: %s\n", a.ManagedImageOSDiskSnapshotName))
		}
		if a.ManagedImageDataDiskSnapshotPrefix != "" {
			buf.WriteString(fmt.Sprintf("ManagedImageDataDiskSnapshotPrefix: %s\n", a.ManagedImageDataDiskSnapshotPrefix))
		}
		if a.OSDiskUri != "" {
			buf.WriteString(fmt.Sprintf("OSDiskUri: %s\n", a.OSDiskUri))
		}
	} else if !a.isPublishedToSIG() {
		buf.WriteString(fmt.Sprintf("StorageAccountLocation: %s\n", a.StorageAccountLocation))
		buf.WriteString(fmt.Sprintf("OSDiskUri: %s\n", a.OSDiskUri))
		buf.WriteString(fmt.Sprintf("TemplateUri: %s\n", a.TemplateUri))
		if a.AdditionalDisks != nil {
			for i, additionaldisk := range *a.AdditionalDisks {
				buf.WriteString(fmt.Sprintf("AdditionalDiskUri (datadisk-%d): %s\n", i+1, additionaldisk.AdditionalDiskUri))
			}
		}
	}
	if a.isPublishedToSIG() {
		buf.WriteString(fmt.Sprintf("ManagedImageSharedImageGalleryId: %s\n", a.ManagedImageSharedImageGalleryId))
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

	if a.isPublishedToSIG() {
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
		id := a.ManagedImageId
		location := a.ManagedImageLocation

		labels["os_type"] = a.OSType
		labels["managed_image_resourcegroup_name"] = a.ManagedImageResourceGroupName
		labels["managed_image_name"] = a.ManagedImageName

		if a.OSDiskUri != "" {
			labels["os_disk_uri"] = a.OSDiskUri
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

	if a.isPublishedToSIG() {
		img, _ := registryimage.FromArtifact(a,
			registryimage.WithID(a.ManagedImageSharedImageGalleryId),
			registryimage.WithRegion(a.SharedImageGalleryLocation),
			registryimage.WithProvider("azure"),
			registryimage.WithSourceID(sourceID),
			registryimage.SetLabels(labels),
		)

		return img
	}

	// If image is a VHD
	labels["storage_account_location"] = a.StorageAccountLocation
	labels["template_uri"] = a.TemplateUri

	id := a.OSDiskUri
	location := a.StorageAccountLocation
	img, _ := registryimage.FromArtifact(a,
		registryimage.WithID(id),
		registryimage.WithRegion(location),
		registryimage.WithProvider("azure"),
		registryimage.WithSourceID(sourceID),
		registryimage.SetLabels(labels),
	)
	return img
}
