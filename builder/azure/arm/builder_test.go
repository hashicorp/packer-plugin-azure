// Copyright (c) HashiCorp, Inc.
// SPDX-License-Identifier: MPL-2.0

package arm

import (
	"testing"

	"github.com/stretchr/testify/assert"

	"github.com/hashicorp/packer-plugin-azure/builder/azure/common/constants"
	"github.com/hashicorp/packer-plugin-sdk/communicator"
)

func TestStateBagShouldBePopulatedExpectedValues(t *testing.T) {
	var testSubject Builder
	_, _, err := testSubject.Prepare(getArmBuilderConfiguration(), getPackerConfiguration())
	if err != nil {
		t.Fatalf("failed to prepare: %s", err)
	}

	var expectedStateBagKeys = []string{
		constants.AuthorizedKey,

		constants.ArmTags,
		constants.ArmComputeName,
		constants.ArmDeploymentName,
		constants.ArmNicName,
		constants.ArmResourceGroupName,
		constants.ArmStorageAccountName,
		constants.ArmPublicIPAddressName,
		constants.ArmAsyncResourceGroupDelete,
	}

	for _, v := range expectedStateBagKeys {
		if _, ok := testSubject.stateBag.GetOk(v); ok == false {
			t.Errorf("Expected the builder's state bag to contain '%s', but it did not.", v)
		}
	}
}

func TestStateBagShouldPoluateExpectedTags(t *testing.T) {
	var testSubject Builder

	expectedTags := map[string]string{
		"env":     "test",
		"builder": "packer",
	}
	armConfig := getArmBuilderConfiguration()
	armConfig["azure_tags"] = expectedTags

	_, _, err := testSubject.Prepare(armConfig, getPackerConfiguration())
	if err != nil {
		t.Fatalf("failed to prepare: %s", err)
	}

	tags, ok := testSubject.stateBag.Get(constants.ArmTags).(map[string]string)
	if !ok {
		t.Errorf("Expected the builder's state bag to contain tags of type %T, but didn't.", testSubject.config.AzureTags)
	}

	if len(tags) != len(expectedTags) {
		t.Errorf("expect tags from state to be the same length as tags from config")
	}

	for k, v := range tags {
		if expectedTags[k] != v {
			t.Errorf("expect tag value of %s to be %s, but got %s", k, expectedTags[k], v)
		}
	}

}

func TestManagedImageArtifactWithSIGAsDestinationNoImage(t *testing.T) {
	var testSubject Builder

	_, _, err := testSubject.Prepare(getArmBuilderConfiguration(), getPackerConfiguration())
	assert.NoErrorf(t, err, "failed to prepare: %s", err)

	_, err = testSubject.managedImageArtifactWithSIGAsDestination("fakeID", generatedData())
	assert.ErrorIs(t, err, ErrNoImage)
}

func TestBuildSharedImageGalleryArtifact_withState(t *testing.T) {

	var testSubject Builder
	_, _, err := testSubject.Prepare(getArmBuilderConfiguration(), getPackerConfiguration())
	if err != nil {
		t.Fatalf("failed to prepare: %s", err)
	}

	// During the publishing state to a shared image gallery this information is added to the builder StateBag.
	// Adding it to the test to mimic a successful SIG publishing step.
	testSubject.stateBag.Put(constants.ArmManagedImageSigPublishResourceGroup, "fakeGalleryResourceGroup")
	testSubject.stateBag.Put(constants.ArmManagedImageSharedGalleryName, "fakeGalleryName")
	testSubject.stateBag.Put(constants.ArmManagedImageSharedGalleryImageName, "fakeGalleryImageName")
	testSubject.stateBag.Put(constants.ArmManagedImageSharedGalleryImageVersion, "fakeGalleryImageVersion")
	testSubject.stateBag.Put(constants.ArmManagedImageSharedGalleryReplicationRegions, []string{"fake-region-1", "fake-region-2"})
	testSubject.stateBag.Put(constants.ArmManagedImageSharedGalleryId, "fakeSharedImageGallery")

	testSubject.config.ManagedImageResourceGroupName = "fakeResourceGroup"
	testSubject.config.ManagedImageName = "fakeName"
	testSubject.config.Location = "fakeLocation"
	testSubject.config.ManagedImageOSDiskSnapshotName = "fakeOsDiskSnapshotName"
	testSubject.config.ManagedImageDataDiskSnapshotPrefix = "fakeDataDiskSnapshotPrefix"

	artifact, err := testSubject.managedImageArtifactWithSIGAsDestination("fakeID", generatedData())
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
SharedImageGalleryResourceGroup: fakeGalleryResourceGroup
SharedImageGalleryName: fakeGalleryName
SharedImageGalleryImageName: fakeGalleryImageName
SharedImageGalleryImageVersion: fakeGalleryImageVersion
SharedImageGalleryReplicatedRegions: fake-region-1, fake-region-2
`

	result := artifact.String()
	if result != expected {
		t.Fatalf("bad: %s", result)
	}

	if v, ok := artifact.State(constants.ArmManagedImageSigPublishResourceGroup).(string); !ok {
		t.Errorf("expected artifact.State(%s) to return a value for the expected type but it returned %#v", constants.ArmManagedImageSigPublishResourceGroup, v)
	}
	if v, ok := artifact.State(constants.ArmManagedImageSharedGalleryName).(string); !ok {
		t.Errorf("expected artifact.State(%s) to return a value for the expected type but it returned %#v", constants.ArmManagedImageSharedGalleryName, v)
	}
	if v, ok := artifact.State(constants.ArmManagedImageSharedGalleryImageName).(string); !ok {
		t.Errorf("expected artifact.State(%s) to return a value for the expected type but it returned %#v", constants.ArmManagedImageSharedGalleryImageName, v)
	}
	if v, ok := artifact.State(constants.ArmManagedImageSharedGalleryImageVersion).(string); !ok {
		t.Errorf("expected artifact.State(%s) to return a value for the expected type but it returned %#v", constants.ArmManagedImageSharedGalleryImageVersion, v)
	}
	if v, ok := artifact.State(constants.ArmManagedImageSharedGalleryReplicationRegions).([]string); !ok {
		t.Errorf("expected artifact.State(%s) to return a value for the expected type but it returned %#v", constants.ArmManagedImageSharedGalleryReplicationRegions, v)
	}
}

func TestBuilderConfig_SSHHost(t *testing.T) {
	var testSubject Builder
	builderValues := getArmBuilderConfiguration()
	builderValues["communicator"] = "ssh"
	builderValues["ssh_username"] = "override_username"
	builderValues["ssh_host"] = "172.10.10.3"
	_, _, err := testSubject.Prepare(builderValues)
	if err != nil {
		t.Fatalf("failed to prepare: %s", err)

	}

	//inject Fake IP into state for SSHHost
	testSubject.stateBag.Put(constants.SSHHost, "127.0.0.1")

	if _, ok := testSubject.stateBag.GetOk(constants.SSHHost); ok == false {
		t.Fatal("Expected the state bag to contain '127.0.0.1' for SSHHost but it did not.")

	}

	sshHostFn := communicator.CommHost(testSubject.config.Comm.SSHHost, constants.SSHHost)
	host, err := sshHostFn(testSubject.stateBag)
	if err != nil {
		t.Errorf("Unexpected error occurred obtaining ssh_host: %s", err)

	}
	if host != "172.10.10.3" {
		t.Errorf("Expected custom ssh_host to take precedences over state value but it got %q", host)

	}

}
