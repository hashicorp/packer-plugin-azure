package arm

import (
	"bytes"
	"fmt"
	"log"
	"net/url"
	"path"
	"strings"

	"github.com/hashicorp/packer-plugin-azure/builder/azure/common/constants"
	registryimage "github.com/hashicorp/packer-plugin-sdk/packer/registry/image"
)

const (
	BuilderId = "Azure.ResourceManagement.VMImage"
)

type AdditionalDiskArtifact struct {
	AdditionalDiskUri            string
	AdditionalDiskUriReadOnlySas string
}

type Artifact struct {
	// OS type: Linux, Windows
	OSType string

	// VHD
	StorageAccountLocation string
	OSDiskUri              string
	TemplateUri            string
	OSDiskUriReadOnlySas   string
	TemplateUriReadOnlySas string

	// Managed Image
	ManagedImageResourceGroupName      string
	ManagedImageName                   string
	ManagedImageLocation               string
	ManagedImageId                     string
	ManagedImageOSDiskSnapshotName     string
	ManagedImageDataDiskSnapshotPrefix string
	// ARM resource id for Shared Image Gallery
	ManagedImageSharedImageGalleryId string

	// Additional Disks
	AdditionalDisks *[]AdditionalDiskArtifact

	// StateData should store data such as GeneratedData
	// to be shared with post-processors
	StateData map[string]interface{}
}

func NewManagedImageArtifact(osType, resourceGroup, name, location, id, osDiskSnapshotName, dataDiskSnapshotPrefix string, generatedData map[string]interface{}, keepOSDisk bool, template *CaptureTemplate, getSasUrl func(name string) string) (*Artifact, error) {
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

	if keepOSDisk {
		if template == nil {
			log.Printf("artifact error: nil capture template")
			return &res, nil
		}

		if len(template.Resources) != 1 {
			log.Printf("artifact error: malformed capture template, expected one resource")
			return &res, nil
		}

		vhdUri, err := url.Parse(template.Resources[0].Properties.StorageProfile.OSDisk.Image.Uri)
		if err != nil {
			log.Printf("artifact error: Error parsing osdisk url: %s", err)
			return &res, nil
		}

		res.OSDiskUri = vhdUri.String()
		res.OSDiskUriReadOnlySas = getSasUrl(getStorageUrlPath(vhdUri))
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

func NewSharedImageArtifact(osType, destinationSharedImageGalleryId string, generatedData map[string]interface{}) (*Artifact, error) {
	return &Artifact{
		OSType:                           osType,
		ManagedImageSharedImageGalleryId: destinationSharedImageGalleryId,
		StateData:                        generatedData,
	}, nil
}

func NewArtifact(template *CaptureTemplate, getSasUrl func(name string) string, osType string, generatedData map[string]interface{}) (*Artifact, error) {
	if template == nil {
		return nil, fmt.Errorf("nil capture template")
	}

	if len(template.Resources) != 1 {
		return nil, fmt.Errorf("malformed capture template, expected one resource")
	}

	vhdUri, err := url.Parse(template.Resources[0].Properties.StorageProfile.OSDisk.Image.Uri)
	if err != nil {
		return nil, err
	}

	templateUri, err := storageUriToTemplateUri(vhdUri)
	if err != nil {
		return nil, err
	}

	var additional_disks *[]AdditionalDiskArtifact
	if template.Resources[0].Properties.StorageProfile.DataDisks != nil {
		data_disks := make([]AdditionalDiskArtifact, len(template.Resources[0].Properties.StorageProfile.DataDisks))
		for i, additionaldisk := range template.Resources[0].Properties.StorageProfile.DataDisks {
			additionalVhdUri, err := url.Parse(additionaldisk.Image.Uri)
			if err != nil {
				return nil, err
			}
			data_disks[i].AdditionalDiskUri = additionalVhdUri.String()
			data_disks[i].AdditionalDiskUriReadOnlySas = getSasUrl(getStorageUrlPath(additionalVhdUri))
		}
		additional_disks = &data_disks
	}

	return &Artifact{
		OSType:                 osType,
		OSDiskUri:              vhdUri.String(),
		OSDiskUriReadOnlySas:   getSasUrl(getStorageUrlPath(vhdUri)),
		TemplateUri:            templateUri.String(),
		TemplateUriReadOnlySas: getSasUrl(getStorageUrlPath(templateUri)),

		AdditionalDisks: additional_disks,

		StorageAccountLocation: template.Resources[0].Location,

		StateData: generatedData,
	}, nil
}

func getStorageUrlPath(u *url.URL) string {
	parts := strings.Split(u.Path, "/")
	return strings.Join(parts[3:], "/")
}

func storageUriToTemplateUri(su *url.URL) (*url.URL, error) {
	// packer-osDisk.4085bb15-3644-4641-b9cd-f575918640b4.vhd -> 4085bb15-3644-4641-b9cd-f575918640b4
	filename := path.Base(su.Path)
	parts := strings.Split(filename, ".")

	if len(parts) < 3 {
		return nil, fmt.Errorf("malformed URL")
	}

	// packer-osDisk.4085bb15-3644-4641-b9cd-f575918640b4.vhd -> packer
	prefixParts := strings.Split(parts[0], "-")
	prefix := strings.Join(prefixParts[:len(prefixParts)-1], "-")

	templateFilename := fmt.Sprintf("%s-vmTemplate.%s.json", prefix, parts[1])

	// https://storage.blob.core.windows.net/system/Microsoft.Compute/Images/images/packer-osDisk.4085bb15-3644-4641-b9cd-f575918640b4.vhd"
	//   ->
	// https://storage.blob.core.windows.net/system/Microsoft.Compute/Images/images/packer-vmTemplate.4085bb15-3644-4641-b9cd-f575918640b4.json"
	return url.Parse(strings.Replace(su.String(), filename, templateFilename, 1))
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
		if a.OSDiskUriReadOnlySas != "" {
			buf.WriteString(fmt.Sprintf("OSDiskUriReadOnlySas: %s\n", a.OSDiskUriReadOnlySas))
		}
	} else if !a.isPublishedToSIG() {
		buf.WriteString(fmt.Sprintf("StorageAccountLocation: %s\n", a.StorageAccountLocation))
		buf.WriteString(fmt.Sprintf("OSDiskUri: %s\n", a.OSDiskUri))
		buf.WriteString(fmt.Sprintf("OSDiskUriReadOnlySas: %s\n", a.OSDiskUriReadOnlySas))
		buf.WriteString(fmt.Sprintf("TemplateUri: %s\n", a.TemplateUri))
		buf.WriteString(fmt.Sprintf("TemplateUriReadOnlySas: %s\n", a.TemplateUriReadOnlySas))
		if a.AdditionalDisks != nil {
			for i, additionaldisk := range *a.AdditionalDisks {
				buf.WriteString(fmt.Sprintf("AdditionalDiskUri (datadisk-%d): %s\n", i+1, additionaldisk.AdditionalDiskUri))
				buf.WriteString(fmt.Sprintf("AdditionalDiskUriReadOnlySas (datadisk-%d): %s\n", i+1, additionaldisk.AdditionalDiskUriReadOnlySas))
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
