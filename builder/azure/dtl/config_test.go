// Copyright (c) HashiCorp, Inc.
// SPDX-License-Identifier: MPL-2.0

package dtl

import (
	"fmt"
	"testing"
	"time"

	"github.com/hashicorp/packer-plugin-azure/builder/azure/common/constants"
)

// List of configuration parameters that are required by the DTL builder.
var requiredConfigValues = []string{
	"capture_name_prefix",
	"capture_container_name",
	"client_id",
	"client_secret",
	"image_offer",
	"image_publisher",
	"image_sku",
	"location",
	"os_type",
	"subscription_id",
	"lab_resource_group_name",
	"lab_virtual_network_name",
}

func TestConfigShouldProvideReasonableDefaultValues(t *testing.T) {
	config := Config{}
	_, err := config.Prepare(getDtlBuilderConfiguration(), getPackerConfiguration())
	if err != nil {
		t.Errorf("Expected configuration creation to succeed, but it failed (%s)!\n", err)
	}

	if config.UserName == "" {
		t.Error("Expected 'UserName' to be populated, but it was empty!")
	}

	if config.VMSize == "" {
		t.Error("Expected 'VMSize' to be populated, but it was empty!")
	}

	if config.ClientConfig.ObjectID != "" {
		t.Errorf("Expected 'ObjectID' to be nil, but it was '%s'!", config.ClientConfig.ObjectID)
	}

	if config.managedImageStorageAccountType == "" {
		t.Errorf("Expected 'managedImageStorageAccountType' to be populated, but it was empty!")
	}

	if config.diskCachingType == "" {
		t.Errorf("Expected 'diskCachingType' to be populated, but it was empty!")
	}
}

func TestConfigShouldDefaultVMSizeToStandardA1(t *testing.T) {
	config := Config{}
	_, err := config.Prepare(getDtlBuilderConfiguration(), getPackerConfiguration())
	if err != nil {
		t.Errorf("Expected configuration creation to succeed, but it failed (%s)!\n", err)
	}

	if config.VMSize != "Standard_A1" {
		t.Errorf("Expected 'VMSize' to default to 'Standard_A1', but got '%s'.", config.VMSize)
	}
}

func TestConfigShouldDefaultImageVersionToLatest(t *testing.T) {
	config := Config{}
	_, err := config.Prepare(getDtlBuilderConfiguration(), getPackerConfiguration())
	if err != nil {
		t.Errorf("Expected configuration creation to succeed, but it failed (%s)!\n", err)
	}

	if config.ImageVersion != "latest" {
		t.Errorf("Expected 'ImageVersion' to default to 'latest', but got '%s'.", config.ImageVersion)
	}
}

// The user can pass the value virtual_network_resource_group_name to avoid the lookup of
// a virtual network's resource group, or to help with disambiguation.  The value should
// only be set if virtual_network_name was set.
func TestConfigVirtualNetworkResourceGroupNameMustBeSetWithVirtualNetworkName(t *testing.T) {
	config_map := map[string]string{
		"capture_name_prefix":                 "ignore",
		"capture_container_name":              "ignore",
		"location":                            "ignore",
		"image_url":                           "ignore",
		"subscription_id":                     "ignore",
		"os_type":                             constants.Target_Linux,
		"communicator":                        "none",
		"virtual_network_resource_group_name": "MyVirtualNetworkRG",
	}

	config := Config{}
	_, err := config.Prepare(config_map, getPackerConfiguration())
	if err == nil {
		t.Error("Expected Config to reject virtual_network_resource_group_name, if virtual_network_name is not set.")
	}
}

// The user can pass the value virtual_network_subnet_name to avoid the lookup of
// a virtual network subnet's name, or to help with disambiguation.  The value should
// only be set if virtual_network_name was set.
func TestConfigVirtualNetworkSubnetNameMustBeSetWithVirtualNetworkName(t *testing.T) {
	config_map := map[string]string{
		"capture_name_prefix":         "ignore",
		"capture_container_name":      "ignore",
		"location":                    "ignore",
		"image_url":                   "ignore",
		"subscription_id":             "ignore",
		"os_type":                     constants.Target_Linux,
		"communicator":                "none",
		"virtual_network_subnet_name": "MyVirtualNetworkRG",
	}

	config := Config{}
	_, err := config.Prepare(config_map, getPackerConfiguration())
	if err == nil {
		t.Error("Expected Config to reject virtual_network_subnet_name, if virtual_network_name is not set.")
	}
}

func TestSystemShouldDefineRuntimeValues(t *testing.T) {
	config := Config{}
	_, err := config.Prepare(getDtlBuilderConfiguration(), getPackerConfiguration())
	if err != nil {
		t.Errorf("Expected configuration creation to succeed, but it failed (%s)!\n", err)
	}

	if config.Password == "" {
		t.Errorf("Expected Password to not be empty, but it was '%s'!", config.Password)
	}

	if config.tmpComputeName == "" {
		t.Errorf("Expected tmpComputeName to not be empty, but it was '%s'!", config.tmpComputeName)
	}

	if config.tmpDeploymentName == "" {
		t.Errorf("Expected tmpDeploymentName to not be empty, but it was '%s'!", config.tmpDeploymentName)
	}

	if config.tmpResourceGroupName == "" {
		t.Errorf("Expected tmpResourceGroupName to not be empty, but it was '%s'!", config.tmpResourceGroupName)
	}

	if config.tmpOSDiskName == "" {
		t.Errorf("Expected tmpOSDiskName to not be empty, but it was '%s'!", config.tmpOSDiskName)
	}
}

func TestConfigShouldTransformToVirtualMachineCaptureParameters(t *testing.T) {
	config := Config{}
	_, err := config.Prepare(getDtlBuilderConfiguration(), getPackerConfiguration())
	if err != nil {
		t.Errorf("Expected configuration creation to succeed, but it failed (%s)!\n", err)
	}

	parameters := config.toVirtualMachineCaptureParameters()

	if parameters.DestinationContainerName != config.CaptureContainerName {
		t.Errorf("Expected DestinationContainerName to be equal to config's CaptureContainerName, but they were '%s' and '%s' respectively.", parameters.DestinationContainerName, config.CaptureContainerName)
	}

	if parameters.VhdPrefix != config.CaptureNamePrefix {
		t.Errorf("Expected DestinationContainerName to be equal to config's CaptureContainerName, but they were '%s' and '%s' respectively.", parameters.VhdPrefix, config.CaptureNamePrefix)
	}

	if parameters.OverwriteVhds != false {
		t.Error("Expected OverwriteVhds to be false, but it was not.")
	}
}

func TestConfigShouldSupportPackersConfigElements(t *testing.T) {
	config := Config{}
	_, err := config.Prepare(getDtlBuilderConfiguration(), getPackerConfiguration(), getPackerCommunicatorConfiguration())
	if err != nil {
		t.Errorf("Expected configuration creation to succeed, but it failed (%s)!\n", err)
	}

	if config.Comm.SSHTimeout != 1*time.Hour {
		t.Errorf("Expected Comm.SSHTimeout to be a duration of an hour, but got '%s' instead.", config.Comm.SSHTimeout)
	}

	if config.Comm.WinRMTimeout != 2*time.Hour {
		t.Errorf("Expected Comm.WinRMTimeout to be a durationof two hours, but got '%s' instead.", config.Comm.WinRMTimeout)
	}
}

func TestWinRMConfigShouldSetRoundTripDecorator(t *testing.T) {
	config_dtl := getDtlBuilderConfiguration()
	config_dtl["communicator"] = "winrm"
	config_dtl["winrm_username"] = "username"
	config_dtl["winrm_password"] = "password"

	config := Config{}
	_, err := config.Prepare(config_dtl, getPackerConfiguration())
	if err != nil {
		t.Errorf("Expected configuration creation to succeed, but it failed (%s)!\n", err)
	}

	if config.Comm.WinRMTransportDecorator == nil {
		t.Error("Expected WinRMTransportDecorator to be set, but it was nil")
	}
}

func TestUserDeviceLoginIsEnabledForLinux(t *testing.T) {
	config_map := map[string]string{
		"capture_name_prefix":      "ignore",
		"capture_container_name":   "ignore",
		"image_offer":              "ignore",
		"image_publisher":          "ignore",
		"image_sku":                "ignore",
		"location":                 "ignore",
		"subscription_id":          "ignore",
		"os_type":                  constants.Target_Linux,
		"communicator":             "none",
		"lab_resource_group_name":  "ignore",
		"lab_virtual_network_name": "ignore",
	}

	config := Config{}
	_, err := config.Prepare(config_map, getPackerConfiguration())
	if err != nil {
		t.Fatalf("failed to use device login for Linux: %s", err)
	}
}

func TestConfigShouldAcceptTags(t *testing.T) {
	config_map := map[string]interface{}{
		"capture_name_prefix":      "ignore",
		"capture_container_name":   "ignore",
		"image_offer":              "ignore",
		"image_publisher":          "ignore",
		"image_sku":                "ignore",
		"location":                 "ignore",
		"subscription_id":          "ignore",
		"communicator":             "none",
		"lab_resource_group_name":  "ignore",
		"lab_virtual_network_name": "ignore",
		// Does not matter for this test case, just pick one.
		"os_type": constants.Target_Linux,
		"azure_tags": map[string]string{
			"tag01": "value01",
			"tag02": "value02",
		},
	}

	config := Config{}
	_, err := config.Prepare(config_map, getPackerConfiguration())
	if err != nil {
		t.Errorf("Expected configuration creation to succeed, but it failed (%s)!\n", err)
	}

	if len(config.AzureTags) != 2 {
		t.Fatalf("expected to find 2 tags, but got %d", len(config.AzureTags))
	}

	if _, ok := config.AzureTags["tag01"]; !ok {
		t.Error("expected to find key=\"tag01\", but did not")
	}
	if _, ok := config.AzureTags["tag02"]; !ok {
		t.Error("expected to find key=\"tag02\", but did not")
	}

	value := config.AzureTags["tag01"]
	if value != "value01" {
		t.Errorf("expected AzureTags[\"tag01\"] to have value \"value01\", but got %q", value)
	}

	value = config.AzureTags["tag02"]
	if value != "value02" {
		t.Errorf("expected AzureTags[\"tag02\"] to have value \"value02\", but got %q", value)
	}
}

func TestConfigShouldRejectTagsInExcessOf15AcceptTags(t *testing.T) {
	tooManyTags := map[string]string{}
	for i := 0; i < 16; i++ {
		tooManyTags[fmt.Sprintf("tag%.2d", i)] = "ignored"
	}

	config_map := map[string]interface{}{
		"capture_name_prefix":      "ignore",
		"capture_container_name":   "ignore",
		"image_offer":              "ignore",
		"image_publisher":          "ignore",
		"image_sku":                "ignore",
		"location":                 "ignore",
		"subscription_id":          "ignore",
		"communicator":             "none",
		"lab_resource_group_name":  "ignore",
		"lab_virtual_network_name": "ignore",
		// Does not matter for this test case, just pick one.
		"os_type":    constants.Target_Linux,
		"azure_tags": tooManyTags,
	}

	config := Config{}
	_, err := config.Prepare(config_map, getPackerConfiguration())
	if err == nil {
		t.Fatal("expected config to reject based on an excessive amount of tags (> 15)")
	}
}

func TestConfigShouldRejectExcessiveTagNameLength(t *testing.T) {
	nameTooLong := make([]byte, 513)
	for i := range nameTooLong {
		nameTooLong[i] = 'a'
	}

	tags := map[string]string{}
	tags[string(nameTooLong)] = "ignored"

	config_map := map[string]interface{}{
		"capture_name_prefix":      "ignore",
		"capture_container_name":   "ignore",
		"image_offer":              "ignore",
		"image_publisher":          "ignore",
		"image_sku":                "ignore",
		"location":                 "ignore",
		"subscription_id":          "ignore",
		"communicator":             "none",
		"lab_resource_group_name":  "ignore",
		"lab_virtual_network_name": "ignore",
		// Does not matter for this test case, just pick one.
		"os_type":    constants.Target_Linux,
		"azure_tags": tags,
	}

	config := Config{}
	_, err := config.Prepare(config_map, getPackerConfiguration())
	if err == nil {
		t.Fatal("expected config to reject tag name based on length (> 512)")
	}
}

func TestConfigShouldRejectExcessiveTagValueLength(t *testing.T) {
	valueTooLong := make([]byte, 257)
	for i := range valueTooLong {
		valueTooLong[i] = 'a'
	}

	tags := map[string]string{}
	tags["tag01"] = string(valueTooLong)

	config_map := map[string]interface{}{
		"capture_name_prefix":      "ignore",
		"capture_container_name":   "ignore",
		"image_offer":              "ignore",
		"image_publisher":          "ignore",
		"image_sku":                "ignore",
		"location":                 "ignore",
		"subscription_id":          "ignore",
		"communicator":             "none",
		"lab_resource_group_name":  "ignore",
		"lab_virtual_network_name": "ignore",
		// Does not matter for this test case, just pick one.
		"os_type":    constants.Target_Linux,
		"azure_tags": tags,
	}

	config := Config{}
	_, err := config.Prepare(config_map, getPackerConfiguration())
	if err == nil {
		t.Fatal("expected config to reject tag value based on length (> 256)")
	}
}

func TestConfigShouldAcceptPlatformManagedImageBuild(t *testing.T) {
	config_map := map[string]interface{}{
		"image_offer":                       "ignore",
		"image_publisher":                   "ignore",
		"image_sku":                         "ignore",
		"location":                          "ignore",
		"subscription_id":                   "ignore",
		"communicator":                      "none",
		"managed_image_resource_group_name": "ignore",
		"managed_image_name":                "ignore",
		"lab_resource_group_name":           "ignore",
		"lab_virtual_network_name":          "ignore",

		// Does not matter for this test case, just pick one.
		"os_type": constants.Target_Linux,
	}

	config := Config{}
	_, err := config.Prepare(config_map, getPackerConfiguration())
	if err != nil {
		t.Fatal("expected config to accept platform managed image build")
	}
}

func TestConfigShouldAcceptManagedImageStorageAccountTypes(t *testing.T) {
	config_map := map[string]interface{}{
		"custom_managed_image_resource_group_name": "ignore",
		"custom_managed_image_name":                "ignore",
		"location":                                 "ignore",
		"subscription_id":                          "ignore",
		"communicator":                             "none",
		"managed_image_resource_group_name":        "ignore",
		"managed_image_name":                       "ignore",
		"lab_resource_group_name":                  "ignore",
		"lab_virtual_network_name":                 "ignore",

		// Does not matter for this test case, just pick one.
		"os_type": constants.Target_Linux,
	}

	storage_account_types := []string{"Premium_LRS", "Standard_LRS"}

	for _, x := range storage_account_types {
		config_map["managed_image_storage_account_type"] = x
		config := Config{}
		_, err := config.Prepare(config_map, getPackerConfiguration())
		if err != nil {
			t.Fatalf("expected config to accept a managed_image_storage_account_type of %q", x)
		}
	}
}

func TestConfigShouldAcceptDiskCachingTypes(t *testing.T) {
	config_map := map[string]interface{}{
		"custom_managed_image_resource_group_name": "ignore",
		"custom_managed_image_name":                "ignore",
		"location":                                 "ignore",
		"subscription_id":                          "ignore",
		"communicator":                             "none",
		"managed_image_resource_group_name":        "ignore",
		"managed_image_name":                       "ignore",
		"lab_resource_group_name":                  "ignore",
		"lab_virtual_network_name":                 "ignore",

		// Does not matter for this test case, just pick one.
		"os_type": constants.Target_Linux,
	}

	storage_account_types := []string{"None", "ReadOnly", "ReadWrite"}

	for _, x := range storage_account_types {
		config_map["disk_caching_type"] = x
		config := Config{}
		_, err := config.Prepare(config_map, getPackerConfiguration())
		if err != nil {
			t.Fatalf("expected config to accept a disk_caching_type of %q", x)
		}
	}
}

func TestConfigAdditionalDiskDefaultIsNil(t *testing.T) {
	config := Config{}
	_, err := config.Prepare(getDtlBuilderConfiguration(), getPackerConfiguration())
	if err != nil {
		t.Errorf("Expected configuration creation to succeed, but it failed (%s)!\n", err)
	}

	if config.AdditionalDiskSize != nil {
		t.Errorf("Expected Config to not have a set of additional disks, but got a non nil value")
	}
}

func TestConfigAdditionalDiskOverrideDefault(t *testing.T) {
	config_map := map[string]string{
		"capture_name_prefix":      "ignore",
		"capture_container_name":   "ignore",
		"location":                 "ignore",
		"image_url":                "ignore",
		"subscription_id":          "ignore",
		"os_type":                  constants.Target_Linux,
		"communicator":             "none",
		"lab_resource_group_name":  "ignore",
		"lab_virtual_network_name": "ignore",
	}

	diskconfig := map[string][]int32{
		"disk_additional_size": {32, 64},
	}

	config := Config{}
	_, err := config.Prepare(config_map, diskconfig, getPackerConfiguration())
	if err != nil {
		t.Errorf("Expected configuration creation to succeed, but it failed (%s)!\n", err)
	}

	if config.AdditionalDiskSize == nil {
		t.Errorf("Expected Config to have a set of additional disks, but got nil")
	}
	if len(config.AdditionalDiskSize) != 2 {
		t.Errorf("Expected Config to have a 2 additional disks, but got %d additional disks", len(config.AdditionalDiskSize))
	}
	if config.AdditionalDiskSize[0] != 32 {
		t.Errorf("Expected Config to have the first additional disks of size 32Gb, but got %dGb", config.AdditionalDiskSize[0])
	}
	if config.AdditionalDiskSize[1] != 64 {
		t.Errorf("Expected Config to have the second additional disks of size 64Gb, but got %dGb", config.AdditionalDiskSize[1])
	}
}

func TestConfigShouldAllowSharedImageGalleryOptions(t *testing.T) {
	config_map := map[string]interface{}{
		"location":                 "ignore",
		"subscription_id":          "ignore",
		"os_type":                  "linux",
		"lab_resource_group_name":  "ignore",
		"lab_virtual_network_name": "ignore",
		"shared_image_gallery": map[string]string{
			"subscription":   "ignore",
			"resource_group": "ignore",
			"gallery_name":   "ignore",
			"image_name":     "ignore",
			"image_version":  "ignore",
		},
	}

	config := Config{}
	_, err := config.Prepare(config_map, getPackerConfiguration())
	if err == nil {
		t.Log("expected config to accept Shared Image Gallery options", err)
	}

}

func TestConfigShouldRejectSharedImageGalleryWithVhdTarget(t *testing.T) {
	config_map := map[string]interface{}{
		"location":        "ignore",
		"subscription_id": "ignore",
		"os_type":         "linux",
		"shared_image_gallery": map[string]string{
			"subscription":   "ignore",
			"resource_group": "ignore",
			"gallery_name":   "ignore",
			"image_name":     "ignore",
			"image_version":  "ignore",
		},
		"capture_container_name":   "ignore",
		"capture_name_prefix":      "ignore",
		"lab_resource_group_name":  "ignore",
		"lab_virtual_network_name": "ignore",
	}

	config := Config{}
	_, err := config.Prepare(config_map, getPackerConfiguration())
	if err != nil {
		t.Log("expected an error if Shared Image Gallery source is used with VHD target", err)
	}
}

func getDtlBuilderConfiguration() map[string]string {
	m := make(map[string]string)
	for _, v := range requiredConfigValues {
		m[v] = "ignored00"
	}

	m["communicator"] = "none"
	m["os_type"] = constants.Target_Linux
	return m
}

func getPackerConfiguration() interface{} {
	config := map[string]interface{}{
		"packer_build_name":    "azure-arm-vm",
		"packer_builder_type":  "azure-arm-vm",
		"packer_debug":         "false",
		"packer_force":         "false",
		"packer_template_path": "/home/jenkins/azure-arm-vm/template.json",
	}

	return config
}

func getPackerCommunicatorConfiguration() map[string]string {
	config := map[string]string{
		"ssh_timeout":   "1h",
		"winrm_timeout": "2h",
	}

	return config
}
