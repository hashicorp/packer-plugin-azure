package arm

import (
	"fmt"
	"strings"
	"testing"

	"github.com/hashicorp/packer-plugin-azure/builder/azure/common/constants"
	registryimage "github.com/hashicorp/packer-plugin-sdk/packer/registry/image"
	"github.com/mitchellh/mapstructure"
)

func getFakeSasUrl(name string) string {
	return fmt.Sprintf("SAS-%s", name)
}

func generatedData() map[string]interface{} {
	return make(map[string]interface{})
}

func TestArtifactIdVHD(t *testing.T) {
	template := CaptureTemplate{
		Resources: []CaptureResources{
			{
				Properties: CaptureProperties{
					StorageProfile: CaptureStorageProfile{
						OSDisk: CaptureDisk{
							Image: CaptureUri{
								Uri: "https://storage.blob.core.windows.net/system/Microsoft.Compute/Images/images/packer-osDisk.4085bb15-3644-4641-b9cd-f575918640b4.vhd",
							},
						},
					},
				},
				Location: "southcentralus",
			},
		},
	}

	artifact, err := NewArtifact(&template, getFakeSasUrl, "Linux", generatedData())
	if err != nil {
		t.Fatalf("err=%s", err)
	}

	expected := "https://storage.blob.core.windows.net/system/Microsoft.Compute/Images/images/packer-osDisk.4085bb15-3644-4641-b9cd-f575918640b4.vhd"

	result := artifact.Id()
	if result != expected {
		t.Fatalf("bad: %s", result)
	}
}

func TestArtifactIDManagedImage(t *testing.T) {
	template := CaptureTemplate{
		Resources: []CaptureResources{
			{
				Properties: CaptureProperties{
					StorageProfile: CaptureStorageProfile{
						OSDisk: CaptureDisk{
							Image: CaptureUri{
								Uri: "https://storage.blob.core.windows.net/system/Microsoft.Compute/Images/images/packer-osDisk.4085bb15-3644-4641-b9cd-f575918640b4.vhd",
							},
						},
					},
				},
				Location: "southcentralus",
			},
		},
	}

	artifact, err := NewManagedImageArtifact("Linux", "fakeResourceGroup", "fakeName", "fakeLocation", "fakeID", "fakeOsDiskSnapshotName", "fakeDataDiskSnapshotPrefix", generatedData(), false, &template, getFakeSasUrl)
	if err != nil {
		t.Fatalf("err=%s", err)
	}

	expected := `Azure.ResourceManagement.VMImage:

OSType: Linux
ManagedImageResourceGroupName: fakeResourceGroup
ManagedImageName: fakeName
ManagedImageId: fakeID
ManagedImageLocation: fakeLocation
ManagedImageOSDiskSnapshotName: fakeOsDiskSnapshotName
ManagedImageDataDiskSnapshotPrefix: fakeDataDiskSnapshotPrefix
`

	result := artifact.String()
	if result != expected {
		t.Fatalf("bad: %s", result)
	}
}

func TestArtifactIDManagedImageWithoutOSDiskSnapshotName(t *testing.T) {
	template := CaptureTemplate{
		Resources: []CaptureResources{
			{
				Properties: CaptureProperties{
					StorageProfile: CaptureStorageProfile{
						OSDisk: CaptureDisk{
							Image: CaptureUri{
								Uri: "https://storage.blob.core.windows.net/system/Microsoft.Compute/Images/images/packer-osDisk.4085bb15-3644-4641-b9cd-f575918640b4.vhd",
							},
						},
					},
				},
				Location: "southcentralus",
			},
		},
	}

	artifact, err := NewManagedImageArtifact("Linux", "fakeResourceGroup", "fakeName", "fakeLocation", "fakeID", "", "fakeDataDiskSnapshotPrefix", generatedData(), false, &template, getFakeSasUrl)
	if err != nil {
		t.Fatalf("err=%s", err)
	}

	expected := `Azure.ResourceManagement.VMImage:

OSType: Linux
ManagedImageResourceGroupName: fakeResourceGroup
ManagedImageName: fakeName
ManagedImageId: fakeID
ManagedImageLocation: fakeLocation
ManagedImageDataDiskSnapshotPrefix: fakeDataDiskSnapshotPrefix
`

	result := artifact.String()
	if result != expected {
		t.Fatalf("bad: %s", result)
	}
}

func TestArtifactIDManagedImageWithoutDataDiskSnapshotPrefix(t *testing.T) {
	template := CaptureTemplate{
		Resources: []CaptureResources{
			{
				Properties: CaptureProperties{
					StorageProfile: CaptureStorageProfile{
						OSDisk: CaptureDisk{
							Image: CaptureUri{
								Uri: "https://storage.blob.core.windows.net/system/Microsoft.Compute/Images/images/packer-osDisk.4085bb15-3644-4641-b9cd-f575918640b4.vhd",
							},
						},
					},
				},
				Location: "southcentralus",
			},
		},
	}

	artifact, err := NewManagedImageArtifact("Linux", "fakeResourceGroup", "fakeName", "fakeLocation", "fakeID", "fakeOsDiskSnapshotName", "", generatedData(), false, &template, getFakeSasUrl)
	if err != nil {
		t.Fatalf("err=%s", err)
	}

	expected := `Azure.ResourceManagement.VMImage:

OSType: Linux
ManagedImageResourceGroupName: fakeResourceGroup
ManagedImageName: fakeName
ManagedImageId: fakeID
ManagedImageLocation: fakeLocation
ManagedImageOSDiskSnapshotName: fakeOsDiskSnapshotName
`

	result := artifact.String()
	if result != expected {
		t.Fatalf("bad: %s", result)
	}
}

func TestArtifactIDManagedImageWithKeepingTheOSDisk(t *testing.T) {
	template := CaptureTemplate{
		Resources: []CaptureResources{
			{
				Properties: CaptureProperties{
					StorageProfile: CaptureStorageProfile{
						OSDisk: CaptureDisk{
							Image: CaptureUri{
								Uri: "https://storage.blob.core.windows.net/system/Microsoft.Compute/Images/images/packer-osDisk.4085bb15-3644-4641-b9cd-f575918640b4.vhd",
							},
						},
					},
				},
				Location: "southcentralus",
			},
		},
	}

	artifact, err := NewManagedImageArtifact("Linux", "fakeResourceGroup", "fakeName", "fakeLocation", "fakeID", "fakeOsDiskSnapshotName", "", generatedData(), true, &template, getFakeSasUrl)
	if err != nil {
		t.Fatalf("err=%s", err)
	}

	expected := `Azure.ResourceManagement.VMImage:

OSType: Linux
ManagedImageResourceGroupName: fakeResourceGroup
ManagedImageName: fakeName
ManagedImageId: fakeID
ManagedImageLocation: fakeLocation
ManagedImageOSDiskSnapshotName: fakeOsDiskSnapshotName
OSDiskUri: https://storage.blob.core.windows.net/system/Microsoft.Compute/Images/images/packer-osDisk.4085bb15-3644-4641-b9cd-f575918640b4.vhd
OSDiskUriReadOnlySas: SAS-Images/images/packer-osDisk.4085bb15-3644-4641-b9cd-f575918640b4.vhd
`

	result := artifact.String()
	if result != expected {
		t.Fatalf("bad: %s", result)
	}
}

func TestArtifactIDManagedImageWithSharedImageGalleryId(t *testing.T) {
	artifact, err := NewManagedImageArtifactWithSIGAsDestination("Linux", "fakeResourceGroup", "fakeName", "fakeLocation", "fakeID", "fakeOsDiskSnapshotName", "fakeDataDiskSnapshotPrefix", "fakeSharedImageGallery", generatedData())
	if err != nil {
		t.Fatalf("err=%s", err)
	}

	expected := `Azure.ResourceManagement.VMImage:

OSType: Linux
ManagedImageResourceGroupName: fakeResourceGroup
ManagedImageName: fakeName
ManagedImageId: fakeID
ManagedImageLocation: fakeLocation
ManagedImageOSDiskSnapshotName: fakeOsDiskSnapshotName
ManagedImageDataDiskSnapshotPrefix: fakeDataDiskSnapshotPrefix
ManagedImageSharedImageGalleryId: fakeSharedImageGallery
`

	result := artifact.String()
	if result != expected {
		t.Fatalf("bad: %s", result)
	}
}

func TestArtifactIDManagedImageWithSharedImageGallery_PARMetadata(t *testing.T) {

	fakeGalleryResourceGroup := "fakeResourceGroup"
	fakeGalleryName := "fakeName"
	fakeGalleryImageName := "fakeGalleryImageName"
	fakeGalleryImageVersion := "fakeGalleryImageVersion"
	fakeGalleryReplicationRegions := []string{"fake-region-1", "fake-region-2"}

	stateData := map[string]interface{}{
		// Previous Artifact code base used these state key from generated_data; providing duplicate info with empty strings.
		"generated_data": map[string]interface{}{
			"SharedImageGalleryName":               "",
			"SharedImageGalleryImageName":          "",
			"SharedImageGalleryImageVersion":       "",
			"SharedImageGalleryResourceGroup":      "",
			"SharedImageGalleryReplicationRegions": []string{},
		},
	}

	stateData[constants.ArmManagedImageSigPublishResourceGroup] = fakeGalleryResourceGroup
	stateData[constants.ArmManagedImageSharedGalleryName] = fakeGalleryName
	stateData[constants.ArmManagedImageSharedGalleryImageName] = fakeGalleryImageName
	stateData[constants.ArmManagedImageSharedGalleryImageVersion] = fakeGalleryImageVersion
	stateData[constants.ArmManagedImageSharedGalleryReplicationRegions] = fakeGalleryReplicationRegions

	artifact, err := NewManagedImageArtifactWithSIGAsDestination("Linux", "fakeResourceGroup", "fakeName", "fakeLocation", "fakeID", "fakeOsDiskSnapshotName", "fakeDataDiskSnapshotPrefix", "fakeSharedImageGallery", stateData)
	if err != nil {
		t.Fatalf("err=%s", err)
	}

	expected := `Azure.ResourceManagement.VMImage:

OSType: Linux
ManagedImageResourceGroupName: fakeResourceGroup
ManagedImageName: fakeName
ManagedImageId: fakeID
ManagedImageLocation: fakeLocation
ManagedImageOSDiskSnapshotName: fakeOsDiskSnapshotName
ManagedImageDataDiskSnapshotPrefix: fakeDataDiskSnapshotPrefix
ManagedImageSharedImageGalleryId: fakeSharedImageGallery
`

	result := artifact.String()
	if result != expected {
		t.Fatalf("bad: %s", result)
	}

	hcpImage := artifact.State(registryimage.ArtifactStateURI)
	if hcpImage == nil {
		t.Fatalf("Bad: HCP Packer registry image data was nil")
	}

	var image registryimage.Image
	err = mapstructure.Decode(hcpImage, &image)
	if err != nil {
		t.Errorf("Bad: unexpected error when trying to decode state into registryimage.Image %v", err)
	}

	expectedSIGLabels := []string{
		"sig_resource_group",
		"sig_name",
		"sig_image_name",
		"sig_image_version",
		"sig_replicated_regions",
	}
	for _, key := range expectedSIGLabels {
		key := key
		v, ok := image.Labels[key]
		if !ok {
			t.Errorf("expected labels to have %q but no entry was found", key)
		}
		if v == "" {
			t.Errorf("expected labels[%q] to have a non-empty string value, but got %#v", key, v)
		}

	}
}

func TestArtifactString(t *testing.T) {
	template := CaptureTemplate{
		Resources: []CaptureResources{
			{
				Properties: CaptureProperties{
					StorageProfile: CaptureStorageProfile{
						OSDisk: CaptureDisk{
							Image: CaptureUri{
								Uri: "https://storage.blob.core.windows.net/system/Microsoft.Compute/Images/images/packer-osDisk.4085bb15-3644-4641-b9cd-f575918640b4.vhd",
							},
						},
					},
				},
				Location: "southcentralus",
			},
		},
	}

	artifact, err := NewArtifact(&template, getFakeSasUrl, "Linux", generatedData())
	if err != nil {
		t.Fatalf("err=%s", err)
	}

	testSubject := artifact.String()
	if !strings.Contains(testSubject, "OSDiskUri: https://storage.blob.core.windows.net/system/Microsoft.Compute/Images/images/packer-osDisk.4085bb15-3644-4641-b9cd-f575918640b4.vhd") {
		t.Errorf("Expected String() output to contain OSDiskUri")
	}
	if !strings.Contains(testSubject, "OSDiskUriReadOnlySas: SAS-Images/images/packer-osDisk.4085bb15-3644-4641-b9cd-f575918640b4.vhd") {
		t.Errorf("Expected String() output to contain OSDiskUriReadOnlySas")
	}
	if !strings.Contains(testSubject, "TemplateUri: https://storage.blob.core.windows.net/system/Microsoft.Compute/Images/images/packer-vmTemplate.4085bb15-3644-4641-b9cd-f575918640b4.json") {
		t.Errorf("Expected String() output to contain TemplateUri")
	}
	if !strings.Contains(testSubject, "TemplateUriReadOnlySas: SAS-Images/images/packer-vmTemplate.4085bb15-3644-4641-b9cd-f575918640b4.json") {
		t.Errorf("Expected String() output to contain TemplateUriReadOnlySas")
	}
	if !strings.Contains(testSubject, "StorageAccountLocation: southcentralus") {
		t.Errorf("Expected String() output to contain StorageAccountLocation")
	}
	if !strings.Contains(testSubject, "OSType: Linux") {
		t.Errorf("Expected String() output to contain OSType")
	}
}

func TestAdditionalDiskArtifactString(t *testing.T) {
	template := CaptureTemplate{
		Resources: []CaptureResources{
			{
				Properties: CaptureProperties{
					StorageProfile: CaptureStorageProfile{
						OSDisk: CaptureDisk{
							Image: CaptureUri{
								Uri: "https://storage.blob.core.windows.net/system/Microsoft.Compute/Images/images/packer-osDisk.4085bb15-3644-4641-b9cd-f575918640b4.vhd",
							},
						},
						DataDisks: []CaptureDisk{
							{
								Image: CaptureUri{
									Uri: "https://storage.blob.core.windows.net/system/Microsoft.Compute/Images/images/packer-datadisk-1.4085bb15-3644-4641-b9cd-f575918640b4.vhd",
								},
							},
						},
					},
				},
				Location: "southcentralus",
			},
		},
	}

	artifact, err := NewArtifact(&template, getFakeSasUrl, "Linux", generatedData())
	if err != nil {
		t.Fatalf("err=%s", err)
	}

	testSubject := artifact.String()
	if !strings.Contains(testSubject, "OSDiskUri: https://storage.blob.core.windows.net/system/Microsoft.Compute/Images/images/packer-osDisk.4085bb15-3644-4641-b9cd-f575918640b4.vhd") {
		t.Errorf("Expected String() output to contain OSDiskUri")
	}
	if !strings.Contains(testSubject, "OSDiskUriReadOnlySas: SAS-Images/images/packer-osDisk.4085bb15-3644-4641-b9cd-f575918640b4.vhd") {
		t.Errorf("Expected String() output to contain OSDiskUriReadOnlySas")
	}
	if !strings.Contains(testSubject, "TemplateUri: https://storage.blob.core.windows.net/system/Microsoft.Compute/Images/images/packer-vmTemplate.4085bb15-3644-4641-b9cd-f575918640b4.json") {
		t.Errorf("Expected String() output to contain TemplateUri")
	}
	if !strings.Contains(testSubject, "TemplateUriReadOnlySas: SAS-Images/images/packer-vmTemplate.4085bb15-3644-4641-b9cd-f575918640b4.json") {
		t.Errorf("Expected String() output to contain TemplateUriReadOnlySas")
	}
	if !strings.Contains(testSubject, "StorageAccountLocation: southcentralus") {
		t.Errorf("Expected String() output to contain StorageAccountLocation")
	}
	if !strings.Contains(testSubject, "OSType: Linux") {
		t.Errorf("Expected String() output to contain OSType")
	}
	if !strings.Contains(testSubject, "AdditionalDiskUri (datadisk-1): https://storage.blob.core.windows.net/system/Microsoft.Compute/Images/images/packer-datadisk-1.4085bb15-3644-4641-b9cd-f575918640b4.vhd") {
		t.Errorf("Expected String() output to contain AdditionalDiskUri")
	}
	if !strings.Contains(testSubject, "AdditionalDiskUriReadOnlySas (datadisk-1): SAS-Images/images/packer-datadisk-1.4085bb15-3644-4641-b9cd-f575918640b4.vhd") {
		t.Errorf("Expected String() output to contain AdditionalDiskUriReadOnlySas")
	}
}

func TestArtifactProperties(t *testing.T) {
	template := CaptureTemplate{
		Resources: []CaptureResources{
			{
				Properties: CaptureProperties{
					StorageProfile: CaptureStorageProfile{
						OSDisk: CaptureDisk{
							Image: CaptureUri{
								Uri: "https://storage.blob.core.windows.net/system/Microsoft.Compute/Images/images/packer-osDisk.4085bb15-3644-4641-b9cd-f575918640b4.vhd",
							},
						},
					},
				},
				Location: "southcentralus",
			},
		},
	}

	testSubject, err := NewArtifact(&template, getFakeSasUrl, "Linux", generatedData())
	if err != nil {
		t.Fatalf("err=%s", err)
	}

	if testSubject.OSDiskUri != "https://storage.blob.core.windows.net/system/Microsoft.Compute/Images/images/packer-osDisk.4085bb15-3644-4641-b9cd-f575918640b4.vhd" {
		t.Errorf("Expected template to be 'https://storage.blob.core.windows.net/system/Microsoft.Compute/Images/images/packer-osDisk.4085bb15-3644-4641-b9cd-f575918640b4.vhd', but got %s", testSubject.OSDiskUri)
	}
	if testSubject.OSDiskUriReadOnlySas != "SAS-Images/images/packer-osDisk.4085bb15-3644-4641-b9cd-f575918640b4.vhd" {
		t.Errorf("Expected template to be 'SAS-Images/images/packer-osDisk.4085bb15-3644-4641-b9cd-f575918640b4.vhd', but got %s", testSubject.OSDiskUriReadOnlySas)
	}
	if testSubject.TemplateUri != "https://storage.blob.core.windows.net/system/Microsoft.Compute/Images/images/packer-vmTemplate.4085bb15-3644-4641-b9cd-f575918640b4.json" {
		t.Errorf("Expected template to be 'https://storage.blob.core.windows.net/system/Microsoft.Compute/Images/images/packer-vmTemplate.4085bb15-3644-4641-b9cd-f575918640b4.json', but got %s", testSubject.TemplateUri)
	}
	if testSubject.TemplateUriReadOnlySas != "SAS-Images/images/packer-vmTemplate.4085bb15-3644-4641-b9cd-f575918640b4.json" {
		t.Errorf("Expected template to be 'SAS-Images/images/packer-vmTemplate.4085bb15-3644-4641-b9cd-f575918640b4.json', but got %s", testSubject.TemplateUriReadOnlySas)
	}
	if testSubject.StorageAccountLocation != "southcentralus" {
		t.Errorf("Expected StorageAccountLocation to be 'southcentral', but got %s", testSubject.StorageAccountLocation)
	}
	if testSubject.OSType != "Linux" {
		t.Errorf("Expected OSType to be 'Linux', but got %s", testSubject.OSType)
	}
}

func TestAdditionalDiskArtifactProperties(t *testing.T) {
	template := CaptureTemplate{
		Resources: []CaptureResources{
			{
				Properties: CaptureProperties{
					StorageProfile: CaptureStorageProfile{
						OSDisk: CaptureDisk{
							Image: CaptureUri{
								Uri: "https://storage.blob.core.windows.net/system/Microsoft.Compute/Images/images/packer-osDisk.4085bb15-3644-4641-b9cd-f575918640b4.vhd",
							},
						},
						DataDisks: []CaptureDisk{
							{
								Image: CaptureUri{
									Uri: "https://storage.blob.core.windows.net/system/Microsoft.Compute/Images/images/packer-datadisk-1.4085bb15-3644-4641-b9cd-f575918640b4.vhd",
								},
							},
						},
					},
				},
				Location: "southcentralus",
			},
		},
	}

	testSubject, err := NewArtifact(&template, getFakeSasUrl, "Linux", generatedData())
	if err != nil {
		t.Fatalf("err=%s", err)
	}

	if testSubject.OSDiskUri != "https://storage.blob.core.windows.net/system/Microsoft.Compute/Images/images/packer-osDisk.4085bb15-3644-4641-b9cd-f575918640b4.vhd" {
		t.Errorf("Expected template to be 'https://storage.blob.core.windows.net/system/Microsoft.Compute/Images/images/packer-osDisk.4085bb15-3644-4641-b9cd-f575918640b4.vhd', but got %s", testSubject.OSDiskUri)
	}
	if testSubject.OSDiskUriReadOnlySas != "SAS-Images/images/packer-osDisk.4085bb15-3644-4641-b9cd-f575918640b4.vhd" {
		t.Errorf("Expected template to be 'SAS-Images/images/packer-osDisk.4085bb15-3644-4641-b9cd-f575918640b4.vhd', but got %s", testSubject.OSDiskUriReadOnlySas)
	}
	if testSubject.TemplateUri != "https://storage.blob.core.windows.net/system/Microsoft.Compute/Images/images/packer-vmTemplate.4085bb15-3644-4641-b9cd-f575918640b4.json" {
		t.Errorf("Expected template to be 'https://storage.blob.core.windows.net/system/Microsoft.Compute/Images/images/packer-vmTemplate.4085bb15-3644-4641-b9cd-f575918640b4.json', but got %s", testSubject.TemplateUri)
	}
	if testSubject.TemplateUriReadOnlySas != "SAS-Images/images/packer-vmTemplate.4085bb15-3644-4641-b9cd-f575918640b4.json" {
		t.Errorf("Expected template to be 'SAS-Images/images/packer-vmTemplate.4085bb15-3644-4641-b9cd-f575918640b4.json', but got %s", testSubject.TemplateUriReadOnlySas)
	}
	if testSubject.StorageAccountLocation != "southcentralus" {
		t.Errorf("Expected StorageAccountLocation to be 'southcentral', but got %s", testSubject.StorageAccountLocation)
	}
	if testSubject.OSType != "Linux" {
		t.Errorf("Expected OSType to be 'Linux', but got %s", testSubject.OSType)
	}
	if testSubject.AdditionalDisks == nil {
		t.Errorf("Expected AdditionalDisks to be not nil")
	}
	if len(*testSubject.AdditionalDisks) != 1 {
		t.Errorf("Expected AdditionalDisks to have one additional disk, but got %d", len(*testSubject.AdditionalDisks))
	}
	if (*testSubject.AdditionalDisks)[0].AdditionalDiskUri != "https://storage.blob.core.windows.net/system/Microsoft.Compute/Images/images/packer-datadisk-1.4085bb15-3644-4641-b9cd-f575918640b4.vhd" {
		t.Errorf("Expected additional disk uri to be 'https://storage.blob.core.windows.net/system/Microsoft.Compute/Images/images/packer-datadisk-1.4085bb15-3644-4641-b9cd-f575918640b4.vhd', but got %s", (*testSubject.AdditionalDisks)[0].AdditionalDiskUri)
	}
	if (*testSubject.AdditionalDisks)[0].AdditionalDiskUriReadOnlySas != "SAS-Images/images/packer-datadisk-1.4085bb15-3644-4641-b9cd-f575918640b4.vhd" {
		t.Errorf("Expected additional disk sas to be 'SAS-Images/images/packer-datadisk-1.4085bb15-3644-4641-b9cd-f575918640b4.vhd', but got %s", (*testSubject.AdditionalDisks)[0].AdditionalDiskUriReadOnlySas)
	}
}

func TestArtifactOverHyphenatedCaptureUri(t *testing.T) {
	template := CaptureTemplate{
		Resources: []CaptureResources{
			{
				Properties: CaptureProperties{
					StorageProfile: CaptureStorageProfile{
						OSDisk: CaptureDisk{
							Image: CaptureUri{
								Uri: "https://storage.blob.core.windows.net/system/Microsoft.Compute/Images/images/pac-ker-osDisk.4085bb15-3644-4641-b9cd-f575918640b4.vhd",
							},
						},
					},
				},
				Location: "southcentralus",
			},
		},
	}

	testSubject, err := NewArtifact(&template, getFakeSasUrl, "Linux", generatedData())
	if err != nil {
		t.Fatalf("err=%s", err)
	}

	if testSubject.TemplateUri != "https://storage.blob.core.windows.net/system/Microsoft.Compute/Images/images/pac-ker-vmTemplate.4085bb15-3644-4641-b9cd-f575918640b4.json" {
		t.Errorf("Expected template to be 'https://storage.blob.core.windows.net/system/Microsoft.Compute/Images/images/pac-ker-vmTemplate.4085bb15-3644-4641-b9cd-f575918640b4.json', but got %s", testSubject.TemplateUri)
	}
}

func TestArtifactRejectMalformedTemplates(t *testing.T) {
	template := CaptureTemplate{}

	_, err := NewArtifact(&template, getFakeSasUrl, "Linux", generatedData())
	if err == nil {
		t.Fatalf("Expected artifact creation to fail, but it succeeded.")
	}
}

func TestArtifactRejectMalformedStorageUri(t *testing.T) {
	template := CaptureTemplate{
		Resources: []CaptureResources{
			{
				Properties: CaptureProperties{
					StorageProfile: CaptureStorageProfile{
						OSDisk: CaptureDisk{
							Image: CaptureUri{
								Uri: "bark",
							},
						},
					},
				},
			},
		},
	}

	_, err := NewArtifact(&template, getFakeSasUrl, "Linux", generatedData())
	if err == nil {
		t.Fatalf("Expected artifact creation to fail, but it succeeded.")
	}
}

func TestArtifactState_StateData(t *testing.T) {
	expectedData := "this is the data"
	artifact := &Artifact{
		StateData: map[string]interface{}{"state_data": expectedData},
	}

	// Valid state
	result := artifact.State("state_data")
	if result != expectedData {
		t.Fatalf("Bad: State data was %s instead of %s", result, expectedData)
	}

	// Invalid state
	result = artifact.State("invalid_key")
	if result != nil {
		t.Fatalf("Bad: State should be nil for invalid state data name")
	}

	// Nil StateData should not fail and should return nil
	artifact = &Artifact{}
	result = artifact.State("key")
	if result != nil {
		t.Fatalf("Bad: State should be nil for nil StateData")
	}
}
