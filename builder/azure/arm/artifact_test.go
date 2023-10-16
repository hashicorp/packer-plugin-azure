// Copyright (c) HashiCorp, Inc.
// SPDX-License-Identifier: MPL-2.0

package arm

import (
	"strings"
	"testing"

	"github.com/hashicorp/packer-plugin-azure/builder/azure/common/constants"
	registryimage "github.com/hashicorp/packer-plugin-sdk/packer/registry/image"
	"github.com/mitchellh/mapstructure"
)

func generatedData() map[string]interface{} {
	return make(map[string]interface{})
}

func TestArtifactIdVHD(t *testing.T) {
	artifact, err := NewArtifact("4085bb15-3644-4641-b9cd-f575918640b4", "packer", "images", "https://storage.blob.core.windows.net/", "southcentralus", "Linux", 0, generatedData())
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
	artifact, err := NewManagedImageArtifact("Linux", "fakeResourceGroup", "fakeName", "fakeLocation", "fakeID", "fakeOsDiskSnapshotName", "fakeDataDiskSnapshotPrefix", generatedData(), "")
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
	artifact, err := NewManagedImageArtifact("Linux", "fakeResourceGroup", "fakeName", "fakeLocation", "fakeID", "", "fakeDataDiskSnapshotPrefix", generatedData(), "")
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
	artifact, err := NewManagedImageArtifact("Linux", "fakeResourceGroup", "fakeName", "fakeLocation", "fakeID", "fakeOsDiskSnapshotName", "", generatedData(), "")
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
	artifact, err := NewManagedImageArtifact("Linux", "fakeResourceGroup", "fakeName", "fakeLocation", "fakeID", "fakeOsDiskSnapshotName", "", generatedData(), "/subscriptions/subscription/resourceGroups/test/providers/Microsoft.Compute/images/myimage")
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
OSDiskUri: /subscriptions/subscription/resourceGroups/test/providers/Microsoft.Compute/images/myimage
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

func TestArtifactIDManagedImageWithSharedImageGalleryWithoutManagedImage_PARMetadata(t *testing.T) {

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

	artifact, err := NewSharedImageArtifact("Linux", "fakeSharedImageGallery", "fakeLocation", stateData)
	if err != nil {
		t.Fatalf("err=%s", err)
	}

	expected := `Azure.ResourceManagement.VMImage:

OSType: Linux
ManagedImageSharedImageGalleryId: fakeSharedImageGallery
SharedImageGalleryResourceGroup: fakeResourceGroup
SharedImageGalleryName: fakeName
SharedImageGalleryImageName: fakeGalleryImageName
SharedImageGalleryImageVersion: fakeGalleryImageVersion
SharedImageGalleryReplicatedRegions: fake-region-1, fake-region-2
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
	if artifact.SharedImageGalleryLocation != "fakeLocation" {
		t.Errorf("expected fakeLocation got %s", artifact.SharedImageGalleryLocation)
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
SharedImageGalleryResourceGroup: fakeResourceGroup
SharedImageGalleryName: fakeName
SharedImageGalleryImageName: fakeGalleryImageName
SharedImageGalleryImageVersion: fakeGalleryImageVersion
SharedImageGalleryReplicatedRegions: fake-region-1, fake-region-2
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
	artifact, err := NewArtifact("4085bb15-3644-4641-b9cd-f575918640b4", "packer", "images", "https://storage.blob.core.windows.net/", "southcentralus", "Linux", 0, generatedData())
	if err != nil {
		t.Fatalf("err=%s", err)
	}

	testSubject := artifact.String()
	if !strings.Contains(testSubject, "OSDiskUri: https://storage.blob.core.windows.net/system/Microsoft.Compute/Images/images/packer-osDisk.4085bb15-3644-4641-b9cd-f575918640b4.vhd") {
		t.Errorf("Expected String() output to contain OSDiskUri")
	}
	if !strings.Contains(testSubject, "TemplateUri: https://storage.blob.core.windows.net/system/Microsoft.Compute/Images/images/packer-vmTemplate.4085bb15-3644-4641-b9cd-f575918640b4.json") {
		t.Errorf("Expected String() output to contain TemplateUri")
	}
	if !strings.Contains(testSubject, "StorageAccountLocation: southcentralus") {
		t.Errorf("Expected String() output to contain StorageAccountLocation")
	}
	if !strings.Contains(testSubject, "OSType: Linux") {
		t.Errorf("Expected String() output to contain OSType")
	}
}

func TestAdditionalDiskArtifactString(t *testing.T) {
	artifact, err := NewArtifact("4085bb15-3644-4641-b9cd-f575918640b4", "packer", "images", "https://storage.blob.core.windows.net/", "southcentralus", "Linux", 1, generatedData())
	if err != nil {
		t.Fatalf("err=%s", err)
	}

	testSubject := artifact.String()
	if !strings.Contains(testSubject, "OSDiskUri: https://storage.blob.core.windows.net/system/Microsoft.Compute/Images/images/packer-osDisk.4085bb15-3644-4641-b9cd-f575918640b4.vhd") {
		t.Errorf("Expected String() output to contain OSDiskUri")
	}
	if !strings.Contains(testSubject, "TemplateUri: https://storage.blob.core.windows.net/system/Microsoft.Compute/Images/images/packer-vmTemplate.4085bb15-3644-4641-b9cd-f575918640b4.json") {
		t.Errorf("Expected String() output to contain TemplateUri")
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
}

func TestArtifactProperties(t *testing.T) {
	testSubject, err := NewArtifact("4085bb15-3644-4641-b9cd-f575918640b4", "packer", "images", "https://storage.blob.core.windows.net/", "southcentralus", "Linux", 0, generatedData())
	if err != nil {
		t.Fatalf("err=%s", err)
	}

	if testSubject.OSDiskUri != "https://storage.blob.core.windows.net/system/Microsoft.Compute/Images/images/packer-osDisk.4085bb15-3644-4641-b9cd-f575918640b4.vhd" {
		t.Errorf("Expected template to be 'https://storage.blob.core.windows.net/system/Microsoft.Compute/Images/images/packer-osDisk.4085bb15-3644-4641-b9cd-f575918640b4.vhd', but got %s", testSubject.OSDiskUri)
	}
	if testSubject.TemplateUri != "https://storage.blob.core.windows.net/system/Microsoft.Compute/Images/images/packer-vmTemplate.4085bb15-3644-4641-b9cd-f575918640b4.json" {
		t.Errorf("Expected template to be 'https://storage.blob.core.windows.net/system/Microsoft.Compute/Images/images/packer-vmTemplate.4085bb15-3644-4641-b9cd-f575918640b4.json', but got %s", testSubject.TemplateUri)
	}
	if testSubject.StorageAccountLocation != "southcentralus" {
		t.Errorf("Expected StorageAccountLocation to be 'southcentral', but got %s", testSubject.StorageAccountLocation)
	}
	if testSubject.OSType != "Linux" {
		t.Errorf("Expected OSType to be 'Linux', but got %s", testSubject.OSType)
	}
}

func TestAdditionalDiskArtifactProperties(t *testing.T) {
	testSubject, err := NewArtifact("4085bb15-3644-4641-b9cd-f575918640b4", "packer", "images", "https://storage.blob.core.windows.net/", "southcentralus", "Linux", 1, generatedData())
	if err != nil {
		t.Fatalf("err=%s", err)
	}

	if testSubject.OSDiskUri != "https://storage.blob.core.windows.net/system/Microsoft.Compute/Images/images/packer-osDisk.4085bb15-3644-4641-b9cd-f575918640b4.vhd" {
		t.Errorf("Expected template to be 'https://storage.blob.core.windows.net/system/Microsoft.Compute/Images/images/packer-osDisk.4085bb15-3644-4641-b9cd-f575918640b4.vhd', but got %s", testSubject.OSDiskUri)
	}
	if testSubject.TemplateUri != "https://storage.blob.core.windows.net/system/Microsoft.Compute/Images/images/packer-vmTemplate.4085bb15-3644-4641-b9cd-f575918640b4.json" {
		t.Errorf("Expected template to be 'https://storage.blob.core.windows.net/system/Microsoft.Compute/Images/images/packer-vmTemplate.4085bb15-3644-4641-b9cd-f575918640b4.json', but got %s", testSubject.TemplateUri)
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
