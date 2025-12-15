// Copyright (c) HashiCorp, Inc.
// SPDX-License-Identifier: MPL-2.0

package arm

import (
	"fmt"
	"os"
	"strings"
	"testing"
	"time"

	"github.com/google/go-cmp/cmp"
	"github.com/hashicorp/go-azure-sdk/resource-manager/compute/2022-03-01/virtualmachines"
	"github.com/hashicorp/go-azure-sdk/resource-manager/network/2023-09-01/publicipaddresses"
	"github.com/hashicorp/packer-plugin-azure/builder/azure/common/constants"
	sdkconfig "github.com/hashicorp/packer-plugin-sdk/template/config"
)

// List of configuration parameters that are required by the ARM builder.
var requiredConfigValues = []string{
	"capture_container_name",
	"image_offer",
	"image_publisher",
	"image_sku",
	"location",
	"os_type",
	"storage_account",
	"resource_group_name",
}

func TestConfigShouldProvideReasonableDefaultValues(t *testing.T) {
	var c Config
	_, err := c.Prepare(getArmBuilderConfiguration(), getPackerConfiguration())

	if err != nil {
		t.Error("Expected configuration creation to succeed, but it failed!\n")
		t.Fatalf(" errors: %s\n", err)
	}

	if c.UserName == "" {
		t.Error("Expected 'UserName' to be populated, but it was empty!")
	}

	if c.VMSize == "" {
		t.Error("Expected 'VMSize' to be populated, but it was empty!")
	}

	if c.ClientConfig.ObjectID != "" {
		t.Errorf("Expected 'ObjectID' to be nil, but it was '%s'!", c.ClientConfig.ObjectID)
	}

	if c.managedImageStorageAccountType == "" {
		t.Errorf("Expected 'managedImageStorageAccountType' to be populated, but it was empty!")
	}

	if c.diskCachingType == "" {
		t.Errorf("Expected 'diskCachingType' to be populated, but it was empty!")
	}
}

func TestConfigUserNameOverride(t *testing.T) {
	builderValues := getArmBuilderConfiguration()
	builderValues["ssh_username"] = "override_username"
	builderValues["communicator"] = "ssh"

	var c Config
	_, err := c.Prepare(builderValues, getPackerConfiguration())

	if err != nil {
		t.Fatalf("newConfig failed: %s", err)
	}

	// SSH comm
	if c.Password != c.tmpAdminPassword {
		t.Errorf("Expected 'Password' to be set to generated password, but found %q!", c.Password)
	}
	if c.Comm.SSHPassword != c.tmpAdminPassword {
		t.Errorf("Expected 'c.Comm.SSHPassword' to set to generated password, but found %q!", c.Comm.SSHPassword)
	}
	if c.UserName != "override_username" {
		t.Errorf("Expected 'UserName' to be set to 'override_username', but found %q!", c.UserName)
	}
	if c.Comm.SSHUsername != "override_username" {
		t.Errorf("Expected 'c.Comm.SSHUsername' to be set to 'override_username', but found %q!", c.Comm.SSHUsername)
	}

	// Winrm comm
	c = Config{}
	builderValues = getArmBuilderConfiguration()
	builderValues["communicator"] = "winrm"
	builderValues["winrm_username"] = "override_winrm_username"
	_, err = c.Prepare(builderValues, getPackerConfiguration())
	if err != nil {
		t.Fatalf("newConfig failed: %s", err)
	}
	if c.Password != c.tmpAdminPassword {
		t.Errorf("Expected 'Password' to be set to generated password, but found %q!", c.Password)
	}
	if c.Comm.WinRMPassword != c.tmpAdminPassword {
		t.Errorf("Expected 'c.Comm.WinRMPassword' to be set to generated password, but found %q!", c.Comm.WinRMPassword)
	}
	if c.UserName != "override_winrm_username" {
		t.Errorf("Expected 'UserName' to be set to 'override_winrm_username', but found %q!", c.UserName)
	}
	if c.Comm.WinRMUser != "override_winrm_username" {
		t.Errorf("Expected 'c.Comm.WinRMUser' to be set to 'override_winrm_username', but found %q!", c.Comm.WinRMUser)
	}
}
func TestConfigShouldBeAbleToOverrideDefaultedValues(t *testing.T) {
	builderValues := getArmBuilderConfiguration()
	builderValues["ssh_password"] = "override_password"
	builderValues["ssh_username"] = "override_username"
	builderValues["vm_size"] = "override_vm_size"
	builderValues["communicator"] = "ssh"
	builderValues["managed_image_storage_account_type"] = "Premium_LRS"
	builderValues["disk_caching_type"] = "None"

	var c Config
	_, err := c.Prepare(builderValues, getPackerConfiguration())

	if err != nil {
		t.Fatalf("newConfig failed: %s", err)
	}

	if c.VMSize != "override_vm_size" {
		t.Errorf("Expected 'vm_size' to be set to 'override_vm_size', but found %q!", c.VMSize)
	}

	if c.managedImageStorageAccountType != virtualmachines.StorageAccountTypesPremiumLRS {
		t.Errorf("Expected 'managed_image_storage_account_type' to be set to 'Premium_LRS', but found %q!", c.managedImageStorageAccountType)
	}

	if c.diskCachingType != virtualmachines.CachingTypesNone {
		t.Errorf("Expected 'disk_caching_type' to be set to 'None', but found %q!", c.diskCachingType)
	}

	// SSH comm
	if c.Password != "override_password" {
		t.Errorf("Expected 'Password' to be set to 'override_password', but found %q!", c.Password)
	}
	if c.Comm.SSHPassword != "override_password" {
		t.Errorf("Expected 'c.Comm.SSHPassword' to be set to 'override_password', but found %q!", c.Comm.SSHPassword)
	}
	if c.UserName != "override_username" {
		t.Errorf("Expected 'UserName' to be set to 'override_username', but found %q!", c.UserName)
	}
	if c.Comm.SSHUsername != "override_username" {
		t.Errorf("Expected 'c.Comm.SSHUsername' to be set to 'override_username', but found %q!", c.Comm.SSHUsername)
	}

	// Winrm comm
	c = Config{}
	builderValues = getArmBuilderConfiguration()
	builderValues["communicator"] = "winrm"
	builderValues["winrm_password"] = "Override_winrm_password1"
	builderValues["winrm_username"] = "override_winrm_username"
	_, err = c.Prepare(builderValues, getPackerConfiguration())
	if err != nil {
		t.Fatalf("newConfig failed: %s", err)
	}
	if c.Password != "Override_winrm_password1" {
		t.Errorf("Expected 'Password' to be set to 'Override_winrm_password1', but found %q!", c.Password)
	}
	if c.Comm.WinRMPassword != "Override_winrm_password1" {
		t.Errorf("Expected 'c.Comm.WinRMPassword' to be set to 'Override_winrm_password1', but found %q!", c.Comm.WinRMPassword)
	}
	if c.UserName != "override_winrm_username" {
		t.Errorf("Expected 'UserName' to be set to 'override_winrm_username', but found %q!", c.UserName)
	}
	if c.Comm.WinRMUser != "override_winrm_username" {
		t.Errorf("Expected 'c.Comm.WinRMUser' to be set to 'override_winrm_username', but found %q!", c.Comm.WinRMUser)
	}
}

func TestConfigShouldDefaultVMSizeToStandardA1(t *testing.T) {
	var c Config
	_, err := c.Prepare(getArmBuilderConfiguration(), getPackerConfiguration())
	if err != nil {
		t.Fatal(err)
	}
	if c.VMSize != "Standard_A1" {
		t.Errorf("Expected 'VMSize' to default to 'Standard_A1', but got '%s'.", c.VMSize)
	}
}

func TestConfigShouldDefaultImageVersionToLatest(t *testing.T) {
	var c Config
	_, err := c.Prepare(getArmBuilderConfiguration(), getPackerConfiguration())
	if err != nil {
		t.Fatal(err)
	}
	if c.ImageVersion != "latest" {
		t.Errorf("Expected 'ImageVersion' to default to 'latest', but got '%s'.", c.ImageVersion)
	}
}

func TestConfigShouldNotDefaultImageVersionIfCustomImage(t *testing.T) {
	config := map[string]string{
		"capture_name_prefix":    "ignore",
		"capture_container_name": "ignore",
		"location":               "ignore",
		"image_url":              "ignore",
		"storage_account":        "ignore",
		"resource_group_name":    "ignore",
		"subscription_id":        "ignore",
		"os_type":                constants.Target_Linux,
		"communicator":           "none",
	}

	var c Config
	_, err := c.Prepare(config, getPackerConfiguration())
	if err != nil {
		t.Fatal(err)
	}
	if c.ImageVersion != "" {
		t.Errorf("Expected 'ImageVersion' to empty, but got '%s'.", c.ImageVersion)
	}
}

func TestConfigShouldNormalizeOSTypeCase(t *testing.T) {
	config := map[string]string{
		"capture_name_prefix":    "ignore",
		"capture_container_name": "ignore",
		"location":               "ignore",
		"image_url":              "ignore",
		"storage_account":        "ignore",
		"resource_group_name":    "ignore",
		"subscription_id":        "ignore",
		"communicator":           "none",
	}

	os_types := map[string][]string{
		constants.Target_Linux:   {"linux", "LiNuX"},
		constants.Target_Windows: {"windows", "WiNdOWs"},
	}

	for k, v := range os_types {
		for _, os_type := range v {
			config["os_type"] = os_type
			var c Config
			_, err := c.Prepare(config, getPackerConfiguration())
			if err != nil {
				t.Fatalf("Expected config to accept the value %q, but it did not", os_type)
			}

			if c.OSType != k {
				t.Fatalf("Expected config to normalize the value %q to %q, but it did not", os_type, constants.Target_Linux)
			}
		}
	}

	bad_os_types := []string{"", "does-not-exist"}
	for _, os_type := range bad_os_types {
		config["os_type"] = os_type
		var c Config
		_, err := c.Prepare(config, getPackerConfiguration())
		if err == nil {
			t.Fatalf("Expected config to not accept the value %q, but it did", os_type)
		}
	}
}

func TestConfigShouldRejectCustomImageAndMarketPlace(t *testing.T) {
	config := map[string]string{
		"capture_name_prefix":    "ignore",
		"capture_container_name": "ignore",
		"location":               "ignore",
		"image_url":              "ignore",
		"resource_group_name":    "ignore",
		"storage_account":        "ignore",
		"subscription_id":        "ignore",
		"os_type":                constants.Target_Linux,
		"communicator":           "none",
	}
	packerConfiguration := getPackerConfiguration()
	marketPlace := []string{"image_publisher", "image_offer", "image_sku"}

	for _, x := range marketPlace {
		config[x] = "ignore"
		var c Config
		_, err := c.Prepare(config, packerConfiguration)
		if err == nil {
			t.Errorf("Expected Config to reject image_url and %s, but it did not", x)
		}
	}
}

func TestConfigVirtualNetworkNameIsOptional(t *testing.T) {
	config := map[string]string{
		"capture_name_prefix":    "ignore",
		"capture_container_name": "ignore",
		"location":               "ignore",
		"image_url":              "ignore",
		"storage_account":        "ignore",
		"resource_group_name":    "ignore",
		"subscription_id":        "ignore",
		"os_type":                constants.Target_Linux,
		"communicator":           "none",
		"virtual_network_name":   "MyVirtualNetwork",
	}

	var c Config
	_, err := c.Prepare(config, getPackerConfiguration())
	if err != nil {
		t.Fatal(err)
	}
	if c.VirtualNetworkName != "MyVirtualNetwork" {
		t.Errorf("Expected Config to set virtual_network_name to MyVirtualNetwork, but got %q", c.VirtualNetworkName)
	}
	if c.VirtualNetworkResourceGroupName != "" {
		t.Errorf("Expected Config to leave virtual_network_resource_group_name to '', but got %q", c.VirtualNetworkResourceGroupName)
	}
	if c.VirtualNetworkSubnetName != "" {
		t.Errorf("Expected Config to leave virtual_network_subnet_name to '', but got %q", c.VirtualNetworkSubnetName)
	}
}

// The user can pass the value virtual_network_resource_group_name to avoid the lookup of
// a virtual network's resource group, or to help with disambiguation.  The value should
// only be set if virtual_network_name was set.
func TestConfigVirtualNetworkResourceGroupNameMustBeSetWithVirtualNetworkName(t *testing.T) {
	config := map[string]string{
		"capture_name_prefix":                 "ignore",
		"capture_container_name":              "ignore",
		"location":                            "ignore",
		"image_url":                           "ignore",
		"storage_account":                     "ignore",
		"resource_group_name":                 "ignore",
		"subscription_id":                     "ignore",
		"os_type":                             constants.Target_Linux,
		"communicator":                        "none",
		"virtual_network_resource_group_name": "MyVirtualNetworkRG",
	}

	var c Config
	_, err := c.Prepare(config, getPackerConfiguration())
	if err == nil {
		t.Error("Expected Config to reject virtual_network_resource_group_name, if virtual_network_name is not set.")
	}
}

// The user can pass the value virtual_network_subnet_name to avoid the lookup of
// a virtual network subnet's name, or to help with disambiguation.  The value should
// only be set if virtual_network_name was set.
func TestConfigVirtualNetworkSubnetNameMustBeSetWithVirtualNetworkName(t *testing.T) {
	config := map[string]string{
		"capture_name_prefix":         "ignore",
		"capture_container_name":      "ignore",
		"location":                    "ignore",
		"image_url":                   "ignore",
		"storage_account":             "ignore",
		"resource_group_name":         "ignore",
		"subscription_id":             "ignore",
		"os_type":                     constants.Target_Linux,
		"communicator":                "none",
		"virtual_network_subnet_name": "MyVirtualNetworkRG",
	}

	var c Config
	_, err := c.Prepare(config, getPackerConfiguration())
	if err == nil {
		t.Error("Expected Config to reject virtual_network_subnet_name, if virtual_network_name is not set.")
	}
}

func TestConfigAllowedInboundIpAddressesIsOptional(t *testing.T) {
	config := map[string]string{
		"capture_name_prefix":    "ignore",
		"capture_container_name": "ignore",
		"location":               "ignore",
		"image_url":              "ignore",
		"storage_account":        "ignore",
		"resource_group_name":    "ignore",
		"subscription_id":        "ignore",
		"os_type":                constants.Target_Linux,
		"communicator":           "none",
		"virtual_network_name":   "MyVirtualNetwork",
	}

	var c Config
	_, err := c.Prepare(config, getPackerConfiguration())
	if err != nil {
		t.Fatal(err)
	}
	if c.AllowedInboundIpAddresses != nil {
		t.Errorf("Expected Config to set allowed_inbound_ip_addresses to nil, but got %v", c.AllowedInboundIpAddresses)
	}
}

func TestConfigShouldAcceptCorrectInboundIpAddresses(t *testing.T) {
	ipValue0 := "127.0.0.1"
	ipValue1 := "127.0.0.2"
	cidrValue2 := "192.168.100.0/24"
	cidrValue3 := "10.10.1.16/32"
	config := map[string]interface{}{
		"capture_name_prefix":    "ignore",
		"capture_container_name": "ignore",
		"location":               "ignore",
		"image_url":              "ignore",
		"storage_account":        "ignore",
		"resource_group_name":    "ignore",
		"subscription_id":        "ignore",
		"os_type":                constants.Target_Linux,
		"communicator":           "none",
	}

	config["allowed_inbound_ip_addresses"] = ipValue0
	var c Config
	_, err := c.Prepare(config, getPackerConfiguration())
	if err != nil {
		t.Fatal(err)
	}
	if c.AllowedInboundIpAddresses == nil || len(c.AllowedInboundIpAddresses) != 1 ||
		c.AllowedInboundIpAddresses[0] != ipValue0 {
		t.Errorf("Expected 'allowed_inbound_ip_addresses' to have one element (%s), but got '%v'.", ipValue0, c.AllowedInboundIpAddresses)
	}

	config["allowed_inbound_ip_addresses"] = cidrValue2
	_, err = c.Prepare(config, getPackerConfiguration())
	if err != nil {
		t.Fatal(err)
	}
	if c.AllowedInboundIpAddresses == nil || len(c.AllowedInboundIpAddresses) != 1 ||
		c.AllowedInboundIpAddresses[0] != cidrValue2 {
		t.Errorf("Expected 'allowed_inbound_ip_addresses' to have one element (%s), but got '%v'.", cidrValue2, c.AllowedInboundIpAddresses)
	}

	config["allowed_inbound_ip_addresses"] = []string{ipValue0, cidrValue2, ipValue1, cidrValue3}
	_, err = c.Prepare(config, getPackerConfiguration())
	if err != nil {
		t.Fatal(err)
	}
	if c.AllowedInboundIpAddresses == nil || len(c.AllowedInboundIpAddresses) != 4 ||
		c.AllowedInboundIpAddresses[0] != ipValue0 || c.AllowedInboundIpAddresses[1] != cidrValue2 ||
		c.AllowedInboundIpAddresses[2] != ipValue1 || c.AllowedInboundIpAddresses[3] != cidrValue3 {
		t.Errorf("Expected 'allowed_inbound_ip_addresses' to have four elements (%s %s %s %s), but got '%v'.", ipValue0, cidrValue2, ipValue1,
			cidrValue3, c.AllowedInboundIpAddresses)
	}
}

func TestConfigShouldRejectIncorrectInboundIpAddresses(t *testing.T) {
	config := map[string]interface{}{
		"capture_name_prefix":    "ignore",
		"capture_container_name": "ignore",
		"location":               "ignore",
		"image_url":              "ignore",
		"storage_account":        "ignore",
		"resource_group_name":    "ignore",
		"subscription_id":        "ignore",
		"os_type":                constants.Target_Linux,
		"communicator":           "none",
	}

	config["allowed_inbound_ip_addresses"] = []string{"127.0.0.1", "127.0.0.two"}
	var c Config
	_, err := c.Prepare(config, getPackerConfiguration())
	if err == nil {
		t.Errorf("Expected configuration creation to fail, but it succeeded with the malformed allowed_inbound_ip_addresses set to %v", c.AllowedInboundIpAddresses)
	}

	config["allowed_inbound_ip_addresses"] = []string{"192.168.100.1000/24", "10.10.1.16/32"}
	_, err = c.Prepare(config, getPackerConfiguration())
	if err == nil {
		// 192.168.100.1000/24 is invalid
		t.Errorf("Expected configuration creation to fail, but it succeeded with the malformed allowed_inbound_ip_addresses set to %v", c.AllowedInboundIpAddresses)
	}
}

func TestConfigShouldRejectInboundIpAddressesWithVirtualNetwork(t *testing.T) {
	config := map[string]interface{}{
		"capture_name_prefix":          "ignore",
		"capture_container_name":       "ignore",
		"location":                     "ignore",
		"image_url":                    "ignore",
		"storage_account":              "ignore",
		"resource_group_name":          "ignore",
		"subscription_id":              "ignore",
		"os_type":                      constants.Target_Linux,
		"communicator":                 "none",
		"allowed_inbound_ip_addresses": "127.0.0.1",
	}

	var c Config
	_, err := c.Prepare(config, getPackerConfiguration())
	if err != nil {
		t.Fatal(err)
	}

	config["virtual_network_name"] = "some_vnet_name"
	_, err = c.Prepare(config, getPackerConfiguration())
	if err == nil {
		t.Errorf("Expected configuration creation to fail, but it succeeded with allowed_inbound_ip_addresses and virtual_network_name both specified")
	}
}

func TestConfigShouldDefaultToPublicCloud(t *testing.T) {
	var c Config
	_, err := c.Prepare(getArmBuilderConfiguration(), getPackerConfiguration())
	if err != nil {
		t.Fatal(err)
	}
	if c.ClientConfig.CloudEnvironmentName != "Public" {
		t.Errorf("Expected 'CloudEnvironmentName' to default to 'Public', but got '%s'.", c.ClientConfig.CloudEnvironmentName)
	}

	if c.ClientConfig.CloudEnvironment() == nil || c.ClientConfig.CloudEnvironment().Name != "Public" {
		t.Errorf("Expected 'cloudEnvironment' to be set to 'Public', but got '%s'.", c.ClientConfig.CloudEnvironment().Name)
	}
}

func TestConfigInstantiatesCorrectAzureEnvironment(t *testing.T) {
	config := map[string]string{
		"capture_name_prefix":    "ignore",
		"capture_container_name": "ignore",
		"image_offer":            "ignore",
		"image_publisher":        "ignore",
		"image_sku":              "ignore",
		"location":               "ignore",
		"storage_account":        "ignore",
		"resource_group_name":    "ignore",
		"subscription_id":        "ignore",
		"os_type":                constants.Target_Linux,
		"communicator":           "none",
	}

	// user input is fun :D
	var table = []struct {
		name            string
		environmentName string
	}{
		{"China", "China"},
		{"ChinaCloud", "China"},
		{"AzureChinaCloud", "China"},
		{"aZuReChInAcLoUd", "China"},

		{"USGovernment", "USGovernment"},
		{"USGovernmentCloud", "USGovernment"},
		{"AzureUSGovernmentCloud", "USGovernment"},
		{"aZuReUsGoVeRnMeNtClOuD", "USGovernment"},

		{"Public", "Public"},
		{"PublicCloud", "Public"},
		{"AzurePublicCloud", "Public"},
		{"aZuRePuBlIcClOuD", "Public"},
	}

	packerConfiguration := getPackerConfiguration()

	for _, x := range table {
		config["cloud_environment_name"] = x.name
		var c Config
		_, err := c.Prepare(config, packerConfiguration)
		if err != nil {
			t.Fatal(err)
		}

		if c.ClientConfig.CloudEnvironment() == nil || c.ClientConfig.CloudEnvironment().Name != x.environmentName {
			t.Errorf("Expected 'cloudEnvironment' to be set to '%s', but got '%s'.", x.environmentName, c.ClientConfig.CloudEnvironment().Name)
		}
	}
}

func TestUserShouldProvideRequiredValues(t *testing.T) {
	builderValues := getArmBuilderConfiguration()

	// Ensure we can successfully create a config.
	var c Config
	_, err := c.Prepare(builderValues, getPackerConfiguration())
	if err != nil {
		t.Error("Expected configuration creation to succeed, but it failed!\n")
		t.Fatalf(" -> %+v\n", builderValues)
	}

	// Take away a required element, and ensure construction fails.
	for _, v := range requiredConfigValues {
		originalValue := builderValues[v]
		delete(builderValues, v)

		var c Config
		_, err := c.Prepare(builderValues, getPackerConfiguration())
		if err == nil {
			t.Error("Expected configuration creation to fail, but it succeeded!\n")
			t.Fatalf(" -> %+v\n", builderValues)
		}

		builderValues[v] = originalValue
	}
}

func TestSystemShouldDefineRuntimeValues(t *testing.T) {
	var c Config
	_, err := c.Prepare(getArmBuilderConfiguration(), getPackerConfiguration())

	if err != nil {
		t.Fatal(err)
	}
	if c.Password == "" {
		t.Errorf("Expected Password to not be empty, but it was '%s'!", c.Password)
	}

	if c.tmpComputeName == "" {
		t.Errorf("Expected tmpComputeName to not be empty, but it was '%s'!", c.tmpComputeName)
	}

	if c.tmpDeploymentName == "" {
		t.Errorf("Expected tmpDeploymentName to not be empty, but it was '%s'!", c.tmpDeploymentName)
	}

	if c.tmpResourceGroupName == "" {
		t.Errorf("Expected tmpResourceGroupName to not be empty, but it was '%s'!", c.tmpResourceGroupName)
	}

	if c.tmpOSDiskName == "" {
		t.Errorf("Expected tmpOSDiskName to not be empty, but it was '%s'!", c.tmpOSDiskName)
	}

	if c.tmpDataDiskName == "" {
		t.Errorf("Expected tmpDataDiskName to not be empty, but it was '%s'!", c.tmpDataDiskName)
	}

	if c.tmpNsgName == "" {
		t.Errorf("Expected tmpNsgName to not be empty, but it was '%s'!", c.tmpNsgName)
	}
}

func TestConfigShouldTransformToVirtualMachineCaptureParameters(t *testing.T) {
	var c Config
	_, err := c.Prepare(getArmBuilderConfiguration(), getPackerConfiguration())
	if err != nil {
		t.Fatal(err)
	}
	parameters := c.toVirtualMachineCaptureParameters()

	if parameters.DestinationContainerName != c.CaptureContainerName {
		t.Errorf("Expected DestinationContainerName to be equal to config's CaptureContainerName, but they were '%s' and '%s' respectively.", parameters.DestinationContainerName, c.CaptureContainerName)
	}

	if parameters.VhdPrefix != c.CaptureNamePrefix {
		t.Errorf("Expected DestinationContainerName to be equal to config's CaptureContainerName, but they were '%s' and '%s' respectively.", parameters.VhdPrefix, c.CaptureNamePrefix)
	}

	if parameters.OverwriteVhds != false {
		t.Error("Expected OverwriteVhds to be false, but it was not.")
	}
}

func TestConfigShouldSupportPackersConfigElements(t *testing.T) {
	var c Config
	_, err := c.Prepare(
		getArmBuilderConfiguration(),
		getPackerConfiguration(),
		getPackerCommunicatorConfiguration())

	if err != nil {
		t.Fatal(err)
	}

	if c.Comm.SSHTimeout != 1*time.Hour {
		t.Errorf("Expected Comm.SSHTimeout to be a duration of an hour, but got '%s' instead.", c.Comm.SSHTimeout)
	}

	if c.Comm.WinRMTimeout != 2*time.Hour {
		t.Errorf("Expected Comm.WinRMTimeout to be a durationof two hours, but got '%s' instead.", c.Comm.WinRMTimeout)
	}
}

func TestWinRMConfigShouldSetRoundTripDecorator(t *testing.T) {
	config := getArmBuilderConfiguration()
	config["communicator"] = "winrm"
	config["winrm_username"] = "username"
	config["winrm_password"] = "Password123"

	var c Config
	_, err := c.Prepare(config, getPackerConfiguration())
	if err != nil {
		t.Fatal(err)
	}

	if c.Comm.WinRMTransportDecorator == nil {
		t.Error("Expected WinRMTransportDecorator to be set, but it was nil")
	}
}

func TestUserDeviceLoginIsEnabledForLinux(t *testing.T) {
	config := map[string]string{
		"capture_name_prefix":    "ignore",
		"capture_container_name": "ignore",
		"image_offer":            "ignore",
		"image_publisher":        "ignore",
		"image_sku":              "ignore",
		"location":               "ignore",
		"storage_account":        "ignore",
		"resource_group_name":    "ignore",
		"subscription_id":        "ignore",
		"os_type":                constants.Target_Linux,
		"communicator":           "none",
	}

	var c Config
	_, err := c.Prepare(config, getPackerConfiguration())
	if err != nil {
		t.Fatalf("failed to use device login for Linux: %s", err)
	}
}

func TestConfigShouldRejectMalformedCaptureContainerName(t *testing.T) {
	config := map[string]string{
		"capture_name_prefix": "ignore",
		"image_offer":         "ignore",
		"image_publisher":     "ignore",
		"image_sku":           "ignore",
		"location":            "ignore",
		"storage_account":     "ignore",
		"resource_group_name": "ignore",
		"subscription_id":     "ignore",
		"communicator":        "none",
		// Does not matter for this test case, just pick one.
		"os_type": constants.Target_Linux,
	}

	wellFormedCaptureContainerName := []string{
		"0leading",
		"aleading",
		"hype-hyphen",
		"abcdefghijklmnopqrstuvwxyz0123456789-abcdefghijklmnopqrstuvwxyz", // 63 characters
	}

	for _, x := range wellFormedCaptureContainerName {
		config["capture_container_name"] = x
		var c Config
		_, err := c.Prepare(config, getPackerConfiguration())

		if err != nil {
			t.Errorf("Expected test to pass, but it failed with the well-formed capture_container_name set to %q.", x)
		}
	}

	malformedCaptureContainerName := []string{
		"No-Capitals",
		"double--hyphens",
		"-leading-hyphen",
		"trailing-hyphen-",
		"punc-!@#$%^&*()_+-=-punc",
		"there-are-over-63-characters-in-this-string-and-that-is-a-bad-container-name",
	}

	for _, x := range malformedCaptureContainerName {
		config["capture_container_name"] = x
		var c Config
		_, err := c.Prepare(config, getPackerConfiguration())

		if err == nil {
			t.Errorf("Expected test to fail, but it succeeded with the malformed capture_container_name set to %q.", x)
		}
	}
}

func TestConfigShouldRejectMalformedManagedImageOSDiskSnapshotName(t *testing.T) {
	config := map[string]interface{}{
		"image_offer":                         "ignore",
		"image_publisher":                     "ignore",
		"image_sku":                           "ignore",
		"location":                            "ignore",
		"subscription_id":                     "ignore",
		"communicator":                        "none",
		"managed_image_resource_group_name":   "ignore",
		"managed_image_name":                  "ignore",
		"managed_image_os_disk_snapshot_name": "ignore",
		// Does not matter for this test case, just pick one.
		"os_type": constants.Target_Linux,
	}

	wellFormedManagedImageOSDiskSnapshotName := []string{
		"AbcdefghijklmnopqrstuvwX",
		"underscore_underscore",
		"0leading_number",
		"really_loooooooooooooooooooooooooooooooooooooooooooooooooong",
	}

	for _, x := range wellFormedManagedImageOSDiskSnapshotName {
		config["managed_image_os_disk_snapshot_name"] = x
		var c Config
		_, err := c.Prepare(config, getPackerConfiguration())

		if err != nil {
			t.Errorf("Expected test to pass, but it failed with the well-formed managed_image_os_disk_snapshot_name set to %q.", x)
		}
	}

	malformedManagedImageOSDiskSnapshotName := []string{
		"-leading-hyphen",
		"trailing-hyphen-",
		"trailing-period.",
		"punc-!@#$%^&*()_+-=-punc",
		"really_looooooooooooooooooooooooooooooooooooooooooooooooooooooong_exceeding_80_char_limit",
	}

	for _, x := range malformedManagedImageOSDiskSnapshotName {
		config["managed_image_os_disk_snapshot_name"] = x
		var c Config
		_, err := c.Prepare(config, getPackerConfiguration())

		if err == nil {
			t.Errorf("Expected test to fail, but it succeeded with the malformed managed_image_os_disk_snapshot_name set to %q.", x)
		}
	}
}

func TestConfigShouldRejectMalformedManagedImageDataDiskSnapshotPrefix(t *testing.T) {
	config := map[string]interface{}{
		"image_offer":                       "ignore",
		"image_publisher":                   "ignore",
		"image_sku":                         "ignore",
		"location":                          "ignore",
		"subscription_id":                   "ignore",
		"communicator":                      "none",
		"managed_image_resource_group_name": "ignore",
		"managed_image_name":                "ignore",
		"managed_image_data_disk_snapshot_prefix": "ignore",
		// Does not matter for this test case, just pick one.
		"os_type": constants.Target_Linux,
	}

	wellFormedManagedImageDataDiskSnapshotPrefix := []string{
		"min_ten_chars",
		"AbcdefghijklmnopqrstuvwX",
		"underscore_underscore",
		"0leading_number",
		"less_than_sixty_characters",
	}

	for _, x := range wellFormedManagedImageDataDiskSnapshotPrefix {
		config["managed_image_data_disk_snapshot_prefix"] = x
		var c Config
		_, err := c.Prepare(config, getPackerConfiguration())

		if err != nil {
			t.Errorf("Expected test to pass, but it failed with the well-formed managed_image_data_disk_snapshot_prefix set to %q.", x)
		}
	}

	malformedManagedImageDataDiskSnapshotPrefix := []string{
		"-leading-hyphen",
		"trailing-hyphen-",
		"trailing-period.",
		"punc-!@#$%^&*()_+-=-punc",
		"really_looooooooooooooooooooooooooooooooooooooooooooooooooooooong_exceeding_60_char_limit",
	}

	for _, x := range malformedManagedImageDataDiskSnapshotPrefix {
		config["managed_image_data_disk_snapshot_prefix"] = x
		var c Config
		_, err := c.Prepare(config, getPackerConfiguration())

		if err == nil {
			t.Errorf("Expected test to fail, but it succeeded with the malformed managed_image_data_disk_snapshot_prefix set to %q.", x)
		}
	}
}

func TestConfigShouldAcceptTags(t *testing.T) {
	config := map[string]interface{}{
		"capture_name_prefix":    "ignore",
		"capture_container_name": "ignore",
		"image_offer":            "ignore",
		"image_publisher":        "ignore",
		"image_sku":              "ignore",
		"location":               "ignore",
		"storage_account":        "ignore",
		"resource_group_name":    "ignore",
		"subscription_id":        "ignore",
		"communicator":           "none",
		// Does not matter for this test case, just pick one.
		"os_type": constants.Target_Linux,
		"azure_tags": map[string]string{
			"tag01": "value01",
			"tag02": "value02",
		},
	}

	c := Config{
		AzureTag: sdkconfig.NameValues{
			{Name: "tag03", Value: "value03"},
		},
	}
	_, err := c.Prepare(config, getPackerConfiguration())
	if err != nil {
		t.Fatal(err)
	}

	if diff := cmp.Diff(c.AzureTags, map[string]string{
		"tag01": "value01",
		"tag02": "value02",
		"tag03": "value03",
	}); diff != "" {
		t.Fatalf("unexpected azure tags: %s", diff)
	}
}

func TestConfigShouldAcceptTag(t *testing.T) {
	config := map[string]interface{}{
		"capture_name_prefix":    "ignore",
		"capture_container_name": "ignore",
		"image_offer":            "ignore",
		"image_publisher":        "ignore",
		"image_sku":              "ignore",
		"location":               "ignore",
		"storage_account":        "ignore",
		"resource_group_name":    "ignore",
		"subscription_id":        "ignore",
		"communicator":           "none",
		// Does not matter for this test case, just pick one.
		"os_type": constants.Target_Linux,
	}

	c := Config{
		AzureTag: sdkconfig.NameValues{
			{Name: "tag03", Value: "value03"},
		},
	}
	_, err := c.Prepare(config, getPackerConfiguration())
	if err != nil {
		t.Fatal(err)
	}

	if diff := cmp.Diff(c.AzureTags, map[string]string{
		"tag03": "value03",
	}); diff != "" {
		t.Fatalf("unexpected azure tags: %s", diff)
	}
}

func TestConfigShouldRejectTagsInExcessOf50AcceptTags(t *testing.T) {
	tooManyTags := map[string]string{}
	for i := 0; i < 51; i++ {
		tooManyTags[fmt.Sprintf("tag%.2d", i)] = "ignored"
	}

	config := map[string]interface{}{
		"capture_name_prefix":    "ignore",
		"capture_container_name": "ignore",
		"image_offer":            "ignore",
		"image_publisher":        "ignore",
		"image_sku":              "ignore",
		"location":               "ignore",
		"storage_account":        "ignore",
		"resource_group_name":    "ignore",
		"subscription_id":        "ignore",
		"communicator":           "none",
		// Does not matter for this test case, just pick one.
		"os_type":    constants.Target_Linux,
		"azure_tags": tooManyTags,
	}

	var c Config
	_, err := c.Prepare(config, getPackerConfiguration())

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

	config := map[string]interface{}{
		"capture_name_prefix":    "ignore",
		"capture_container_name": "ignore",
		"image_offer":            "ignore",
		"image_publisher":        "ignore",
		"image_sku":              "ignore",
		"location":               "ignore",
		"storage_account":        "ignore",
		"resource_group_name":    "ignore",
		"subscription_id":        "ignore",
		"communicator":           "none",
		// Does not matter for this test case, just pick one.
		"os_type":    constants.Target_Linux,
		"azure_tags": tags,
	}

	var c Config
	_, err := c.Prepare(config, getPackerConfiguration())
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

	config := map[string]interface{}{
		"capture_name_prefix":    "ignore",
		"capture_container_name": "ignore",
		"image_offer":            "ignore",
		"image_publisher":        "ignore",
		"image_sku":              "ignore",
		"location":               "ignore",
		"storage_account":        "ignore",
		"resource_group_name":    "ignore",
		"subscription_id":        "ignore",
		"communicator":           "none",
		// Does not matter for this test case, just pick one.
		"os_type":    constants.Target_Linux,
		"azure_tags": tags,
	}

	var c Config
	_, err := c.Prepare(config, getPackerConfiguration())
	if err == nil {
		t.Fatal("expected config to reject tag value based on length (> 256)")
	}
}

func TestConfigZoneResilientShouldDefaultToFalse(t *testing.T) {
	config := map[string]interface{}{
		"managed_image_name":                "ignore",
		"managed_image_resource_group_name": "ignore",
		"build_resource_group_name":         "ignore",
		"image_publisher":                   "ignore",
		"image_offer":                       "ignore",
		"image_sku":                         "ignore",
		"os_type":                           "linux",
	}

	var c Config
	_, err := c.Prepare(config, getPackerConfiguration())
	if err != nil {
		t.Fatal(err)
	}

	p := c.toImageParameters()
	if *p.Properties.StorageProfile.ZoneResilient {
		t.Fatal("expected zone resilient default to be false")
	}
}

func TestConfigZoneResilientSetFromConfig(t *testing.T) {
	config := map[string]interface{}{
		"managed_image_name":                "ignore",
		"managed_image_resource_group_name": "ignore",
		"build_resource_group_name":         "ignore",
		"image_publisher":                   "ignore",
		"image_offer":                       "ignore",
		"image_sku":                         "ignore",
		"os_type":                           "linux",
		"managed_image_zone_resilient":      true,
	}

	var c Config
	_, err := c.Prepare(config, getPackerConfiguration())
	if err != nil {
		t.Fatal(err)
	}

	p := c.toImageParameters()
	if *p.Properties.StorageProfile.ZoneResilient == false {
		t.Fatal("expected managed image zone resilient to be true from config")
	}
}

func TestConfigShouldRejectMissingCustomDataFile(t *testing.T) {
	config := map[string]interface{}{
		"capture_name_prefix":    "ignore",
		"capture_container_name": "ignore",
		"image_offer":            "ignore",
		"image_publisher":        "ignore",
		"image_sku":              "ignore",
		"location":               "ignore",
		"storage_account":        "ignore",
		"resource_group_name":    "ignore",
		"subscription_id":        "ignore",
		"communicator":           "none",
		// Does not matter for this test case, just pick one.
		"os_type":          constants.Target_Linux,
		"custom_data_file": "/this/file/does/not/exist",
	}

	var c Config
	_, err := c.Prepare(config, getPackerConfiguration())
	if err == nil {
		t.Fatal("expected config to reject missing custom data file")
	}
}

func TestConfigShouldRejectManagedImageOSDiskSnapshotNameWithoutManagedImageName(t *testing.T) {
	config := map[string]interface{}{
		"image_offer":                         "ignore",
		"image_publisher":                     "ignore",
		"image_sku":                           "ignore",
		"location":                            "ignore",
		"subscription_id":                     "ignore",
		"communicator":                        "none",
		"managed_image_resource_group_name":   "ignore",
		"managed_image_os_disk_snapshot_name": "ignore",
		// Does not matter for this test case, just pick one.
		"os_type": constants.Target_Linux,
	}

	var c Config
	_, err := c.Prepare(config, getPackerConfiguration())
	if err == nil {
		t.Fatal("expected config to reject Managed Image build with OS disk snapshot name but without managed image name")
	}
}

func TestConfigShouldRejectManagedImageOSDiskSnapshotNameWithoutManagedImageResourceGroupName(t *testing.T) {
	config := map[string]interface{}{
		"image_offer":                         "ignore",
		"image_publisher":                     "ignore",
		"image_sku":                           "ignore",
		"location":                            "ignore",
		"subscription_id":                     "ignore",
		"communicator":                        "none",
		"managed_image_name":                  "ignore",
		"managed_image_os_disk_snapshot_name": "ignore",
		// Does not matter for this test case, just pick one.
		"os_type": constants.Target_Linux,
	}

	var c Config
	_, err := c.Prepare(config, getPackerConfiguration())
	if err == nil {
		t.Fatal("expected config to reject Managed Image build with OS disk snapshot name but without managed image resource group name")
	}
}

func TestConfigShouldRejectImageDataDiskSnapshotPrefixWithoutManagedImageName(t *testing.T) {
	config := map[string]interface{}{
		"image_offer":                       "ignore",
		"image_publisher":                   "ignore",
		"image_sku":                         "ignore",
		"location":                          "ignore",
		"subscription_id":                   "ignore",
		"communicator":                      "none",
		"managed_image_resource_group_name": "ignore",
		"managed_image_data_disk_snapshot_prefix": "ignore",
		// Does not matter for this test case, just pick one.
		"os_type": constants.Target_Linux,
	}

	var c Config
	_, err := c.Prepare(config, getPackerConfiguration())
	if err == nil {
		t.Fatal("expected config to reject Managed Image build with data disk snapshot prefix but without managed image name")
	}
}

func TestConfigShouldRejectImageDataDiskSnapshotPrefixWithoutManagedImageResourceGroupName(t *testing.T) {
	config := map[string]interface{}{
		"image_offer":        "ignore",
		"image_publisher":    "ignore",
		"image_sku":          "ignore",
		"location":           "ignore",
		"subscription_id":    "ignore",
		"communicator":       "none",
		"managed_image_name": "ignore",
		"managed_image_data_disk_snapshot_prefix": "ignore",
		// Does not matter for this test case, just pick one.
		"os_type": constants.Target_Linux,
	}

	var c Config
	_, err := c.Prepare(config, getPackerConfiguration())
	if err == nil {
		t.Fatal("expected config to reject Managed Image build with data disk snapshot prefix but without managed image resource group name")
	}
}

func TestConfigShouldAcceptManagedImageOSDiskSnapshotNameAndManagedImageDataDiskSnapshotPrefix(t *testing.T) {
	config := map[string]interface{}{
		"image_offer":                             "ignore",
		"image_publisher":                         "ignore",
		"image_sku":                               "ignore",
		"location":                                "ignore",
		"subscription_id":                         "ignore",
		"communicator":                            "none",
		"managed_image_resource_group_name":       "ignore",
		"managed_image_name":                      "ignore",
		"managed_image_os_disk_snapshot_name":     "ignore_ignore",
		"managed_image_data_disk_snapshot_prefix": "ignore_ignore",
		// Does not matter for this test case, just pick one.
		"os_type": constants.Target_Linux,
	}

	var c Config
	_, err := c.Prepare(config, getPackerConfiguration())
	if err != nil {
		t.Fatal("expected config to accept platform managed image build")
	}
}

func TestConfigShouldAcceptAbsentManagedImageButPresentSharedImageGalleryDestination(t *testing.T) {
	config := map[string]interface{}{
		"image_offer":     "ignore",
		"image_publisher": "ignore",
		"image_sku":       "ignore",
		"location":        "ignore",
		"subscription_id": "ignore",
		"communicator":    "none",
		// Does not matter for this test case, just pick one.
		"os_type": constants.Target_Linux,

		"shared_image_gallery_destination": map[string]string{
			"resource_group":      "ignore",
			"gallery_name":        "ignore",
			"image_name":          "ignore",
			"image_version":       "1.0.1",
			"replication_regions": "ignore",
		},
	}

	var c Config
	_, err := c.Prepare(config, getPackerConfiguration())
	if err != nil {
		t.Fatalf("expected config to accept platform managed image build: %v", err)
	}
}

func TestConfigShouldRejectShallowReplicationWithInvalidReplicationCount(t *testing.T) {
	config := map[string]interface{}{
		"image_offer":                        "ignore",
		"image_publisher":                    "ignore",
		"image_sku":                          "ignore",
		"location":                           "ignore",
		"subscription_id":                    "ignore",
		"communicator":                       "none",
		"os_type":                            constants.Target_Linux,
		"shared_image_gallery_replica_count": "2",
		"shared_image_gallery_destination": map[string]string{
			"resource_group":          "ignore",
			"gallery_name":            "ignore",
			"image_name":              "ignore",
			"image_version":           "1.0.1",
			"replication_regions":     "ignore",
			"use_shallow_replication": "true",
		},
	}
	expectedErrorMessage := "When using shallow replication the replica count can only be 1, leaving this value unset will default to 1"
	var c Config
	_, err := c.Prepare(config, getPackerConfiguration())
	if err == nil {
		t.Fatalf("expected config to reject invalid replica count using shallow replication but it was accepted")
	} else if !strings.Contains(err.Error(), expectedErrorMessage) {
		t.Fatalf("unexpected rejection reason, expected %s to contain %s", err.Error(), expectedErrorMessage)
	}
}

func TestConfigShouldAcceptShallowReplicationWithReplicaCount(t *testing.T) {
	config := map[string]interface{}{
		"image_offer":                        "ignore",
		"image_publisher":                    "ignore",
		"image_sku":                          "ignore",
		"location":                           "ignore",
		"subscription_id":                    "ignore",
		"communicator":                       "none",
		"os_type":                            constants.Target_Linux,
		"shared_image_gallery_replica_count": "1",
		"shared_image_gallery_destination": map[string]string{
			"resource_group":          "ignore",
			"gallery_name":            "ignore",
			"image_name":              "ignore",
			"image_version":           "1.0.1",
			"replication_regions":     "ignore",
			"use_shallow_replication": "true",
		},
	}
	var c Config
	_, err := c.Prepare(config, getPackerConfiguration())
	if err != nil {
		t.Fatalf("expected config to accept shallow replication with set replica count (1) build: %v", err)
	}
}

func TestConfigShouldAcceptShallowReplicationWithWithUnsetReplicaCount(t *testing.T) {
	config := map[string]interface{}{
		"image_offer":     "ignore",
		"image_publisher": "ignore",
		"image_sku":       "ignore",
		"location":        "ignore",
		"subscription_id": "ignore",
		"communicator":    "none",
		"os_type":         constants.Target_Linux,
		"shared_image_gallery_destination": map[string]string{
			"resource_group":          "ignore",
			"gallery_name":            "ignore",
			"image_name":              "ignore",
			"image_version":           "1.0.1",
			"replication_regions":     "ignore",
			"use_shallow_replication": "true",
		},
	}
	var c Config
	_, err := c.Prepare(config, getPackerConfiguration())
	if err != nil {
		t.Fatalf("expected config to accept shallow replication with unset replica count build: %v", err)
	}
}

func TestConfigShouldRejectSigVersionReplicaWithTargetRegion(t *testing.T) {
	config := map[string]interface{}{
		"image_offer":                        "ignore",
		"image_publisher":                    "ignore",
		"image_sku":                          "ignore",
		"location":                           "ignore",
		"subscription_id":                    "ignore",
		"communicator":                       "none",
		"shared_image_gallery_replica_count": "2",
		"os_type":                            constants.Target_Linux,
		"shared_image_gallery_destination": map[string]interface{}{
			"resource_group": "ignore",
			"gallery_name":   "ignore",
			"image_name":     "ignore",
			"image_version":  "1.0.1",
			"target_region": map[string]interface{}{
				"name": "ignore",
			},
		},
	}
	var c Config
	expectedErrorMessage := "shared_image_gallery_replica_count` can not be defined alongside `target_region`; you can define `replicas` inside each target_region block to set the number replicas for each region"
	_, err := c.Prepare(config, getPackerConfiguration())
	if err == nil {
		t.Fatal("expected config to reject target_region block with shared_image_gallery_replica_count set")
	} else if !strings.Contains(err.Error(), expectedErrorMessage) {
		t.Fatalf("unexpected rejection reason, expected %s to contain %s", err.Error(), expectedErrorMessage)
	}
}

func TestConfigValidateShallowReplicationRegion(t *testing.T) {
	tt := []struct {
		name          string
		config        map[string]interface{}
		errorExpected bool
	}{
		{
			name: "with replication region",
			config: map[string]interface{}{
				"image_offer":     "ignore",
				"image_publisher": "ignore",
				"image_sku":       "ignore",
				"location":        "ignore",
				"subscription_id": "ignore",
				"communicator":    "none",
				"os_type":         constants.Target_Linux,
				"shared_image_gallery_destination": map[string]interface{}{
					"resource_group":          "ignore",
					"gallery_name":            "ignore",
					"image_name":              "ignore",
					"image_version":           "1.0.1",
					"replication_regions":     "ignore",
					"use_shallow_replication": "true",
				},
			},
		},
		{
			name: "with target region",
			config: map[string]interface{}{
				"image_offer":     "ignore",
				"image_publisher": "ignore",
				"image_sku":       "ignore",
				"location":        "ignore",
				"subscription_id": "ignore",
				"communicator":    "none",
				"os_type":         constants.Target_Linux,
				"shared_image_gallery_destination": map[string]interface{}{
					"resource_group": "ignore",
					"gallery_name":   "ignore",
					"image_name":     "ignore",
					"image_version":  "1.0.1",
					"target_region": map[string]interface{}{
						"name": "ignore",
					},
					"use_shallow_replication": "true",
				},
			},
		},
		{
			name:          "with multiple replication regions",
			errorExpected: true,
			config: map[string]interface{}{
				"image_offer":     "ignore",
				"image_publisher": "ignore",
				"image_sku":       "ignore",
				"location":        "ignore",
				"subscription_id": "ignore",
				"communicator":    "none",
				"os_type":         constants.Target_Linux,
				"shared_image_gallery_destination": map[string]interface{}{
					"resource_group":          "ignore",
					"gallery_name":            "ignore",
					"image_name":              "ignore",
					"image_version":           "1.0.1",
					"replication_regions":     []string{"one", "two"},
					"use_shallow_replication": "true",
				},
			},
		},
		{
			name:          "with multiple target region",
			errorExpected: true,
			config: map[string]interface{}{
				"image_offer":     "ignore",
				"image_publisher": "ignore",
				"image_sku":       "ignore",
				"location":        "ignore",
				"subscription_id": "ignore",
				"communicator":    "none",
				"os_type":         constants.Target_Linux,
				"shared_image_gallery_destination": map[string]interface{}{
					"resource_group": "ignore",
					"gallery_name":   "ignore",
					"image_name":     "ignore",
					"image_version":  "1.0.1",
					"target_region": map[string]interface{}{
						"name":  "ignore",
						"name2": "ignore",
					},
					"use_shallow_replication": "true",
				},
			},
		},
	}

	for _, tc := range tt {
		t.Run(tc.name, func(t *testing.T) {
			var c Config
			_, err := c.Prepare(tc.config, getPackerConfiguration())
			if !tc.errorExpected && err != nil {
				t.Errorf("unexpected error returned when validating shallow replication regions: %s", err)
			}
			if tc.errorExpected && err == nil {
				t.Errorf("expected an error but got none for %s", tc.name)
			}
		})
	}
}

func TestConfigShouldRejectManagedImageOSDiskSnapshotNameAndManagedImageDataDiskSnapshotPrefixWithCaptureContainerName(t *testing.T) {
	config := map[string]interface{}{
		"image_offer":                         "ignore",
		"image_publisher":                     "ignore",
		"image_sku":                           "ignore",
		"location":                            "ignore",
		"subscription_id":                     "ignore",
		"communicator":                        "none",
		"capture_container_name":              "ignore",
		"managed_image_os_disk_snapshot_name": "ignore_ignore",
		"managed_image_data_disk_snapshot_prefix": "ignore_ignore",
		// Does not matter for this test case, just pick one.
		"os_type": constants.Target_Linux,
	}

	var c Config
	_, err := c.Prepare(config, getPackerConfiguration())
	if err == nil {
		t.Fatal("expected config to reject Managed Image build with data disk snapshot prefix and OS disk snapshot name with capture container name")
	}
}

func TestConfigShouldRejectManagedImageOSDiskSnapshotNameAndManagedImageDataDiskSnapshotPrefixWithCaptureNamePrefix(t *testing.T) {
	config := map[string]interface{}{
		"image_offer":                         "ignore",
		"image_publisher":                     "ignore",
		"image_sku":                           "ignore",
		"location":                            "ignore",
		"subscription_id":                     "ignore",
		"communicator":                        "none",
		"capture_name_prefix":                 "ignore",
		"managed_image_os_disk_snapshot_name": "ignore_ignore",
		"managed_image_data_disk_snapshot_prefix": "ignore_ignore",
		// Does not matter for this test case, just pick one.
		"os_type": constants.Target_Linux,
	}

	var c Config
	_, err := c.Prepare(config, getPackerConfiguration())
	if err == nil {
		t.Fatal("expected config to reject Managed Image build with data disk snapshot prefix and OS disk snapshot name with capture name prefix")
	}
}

func TestConfigShouldRejectSecureBootWhenPublishingToAManagedImage(t *testing.T) {
	expectedErrorMessage := "A managed image (managed_image_name, managed_image_resource_group_name) can not set SecureBoot or VTpm, these features are only supported when directly publishing to a Shared Image Gallery"
	config := map[string]interface{}{
		"image_offer":                       "ignore",
		"image_publisher":                   "ignore",
		"image_sku":                         "ignore",
		"location":                          "ignore",
		"subscription_id":                   "ignore",
		"communicator":                      "none",
		"managed_image_resource_group_name": "ignore",
		"managed_image_name":                "ignore",
		"secure_boot_enabled":               "true",

		// Does not matter for this test case, just pick one.
		"os_type": constants.Target_Linux,
	}

	var c Config
	_, err := c.Prepare(config, getPackerConfiguration())
	if err == nil {
		t.Fatal("expected config to reject managed image with secure boot, secure boot is only allowed when direct publishing to SIG")
	} else if !strings.Contains(err.Error(), expectedErrorMessage) {
		t.Fatalf("unexpected rejection reason, expected %s to contain %s", err.Error(), expectedErrorMessage)
	}
}

func TestConfigShouldRejectVTPMWhenPublishingToAManagedImage(t *testing.T) {
	expectedErrorMessage := "A managed image (managed_image_name, managed_image_resource_group_name) can not set SecureBoot or VTpm, these features are only supported when directly publishing to a Shared Image Gallery"
	config := map[string]interface{}{
		"image_offer":                       "ignore",
		"image_publisher":                   "ignore",
		"image_sku":                         "ignore",
		"location":                          "ignore",
		"subscription_id":                   "ignore",
		"communicator":                      "none",
		"managed_image_resource_group_name": "ignore",
		"managed_image_name":                "ignore",
		"vtpm_enabled":                      "true",

		// Does not matter for this test case, just pick one.
		"os_type": constants.Target_Linux,
	}

	var c Config
	_, err := c.Prepare(config, getPackerConfiguration())
	if err == nil {
		t.Fatal("expected config to reject managed image with secure boot, secure boot is only allowed when direct publishing to SIG")
	} else if !strings.Contains(err.Error(), expectedErrorMessage) {
		t.Fatalf("unexpected rejection reason, expected %s to contain %s", err.Error(), expectedErrorMessage)
	}
}

func TestConfigShouldRejectSpecializedWhenPublishingManagedImage(t *testing.T) {
	expectedErrorMessage := "A managed image (managed_image_name, managed_image_resource_group_name) can not be Specialized (shared_image_gallery_destination.specialized can not be set), Specialized images are only supported when directly publishing to a Shared Image Gallery"
	config := map[string]interface{}{
		"image_offer":                       "ignore",
		"image_publisher":                   "ignore",
		"image_sku":                         "ignore",
		"location":                          "ignore",
		"subscription_id":                   "ignore",
		"communicator":                      "none",
		"managed_image_resource_group_name": "ignore",
		"managed_image_name":                "ignore",
		"shared_image_gallery_destination": map[string]interface{}{
			"resource_group": "ignore",
			"gallery_name":   "ignore",
			"image_name":     "ignore",
			"image_version":  "1.0.0",
			"specialized":    "true",
		},
		// Does not matter for this test case, just pick one.
		"os_type": constants.Target_Linux,
	}

	var c Config
	_, err := c.Prepare(config, getPackerConfiguration())
	if err == nil {
		t.Fatal("expected config to reject managed image with secure boot, secure boot is only allowed when direct publishing to SIG")
	} else if !strings.Contains(err.Error(), expectedErrorMessage) {
		t.Fatalf("unexpected rejection reason, expected %s to contain %s", err.Error(), expectedErrorMessage)
	}
}
func TestConfigShouldAcceptPlatformManagedImageBuild(t *testing.T) {
	config := map[string]interface{}{
		"image_offer":                       "ignore",
		"image_publisher":                   "ignore",
		"image_sku":                         "ignore",
		"location":                          "ignore",
		"subscription_id":                   "ignore",
		"communicator":                      "none",
		"managed_image_resource_group_name": "ignore",
		"managed_image_name":                "ignore",

		// Does not matter for this test case, just pick one.
		"os_type": constants.Target_Linux,
	}

	var c Config
	_, err := c.Prepare(config, getPackerConfiguration())
	if err != nil {
		t.Fatal("expected config to accept platform managed image build")
	}
}

// If the user specified a build for a VHD and a Managed Image it should be accepted.
func TestConfigShouldRejectVhdAndManagedImageOutput(t *testing.T) {
	config := map[string]interface{}{
		"image_offer":                       "ignore",
		"image_publisher":                   "ignore",
		"image_sku":                         "ignore",
		"location":                          "ignore",
		"subscription_id":                   "ignore",
		"communicator":                      "none",
		"capture_container_name":            "ignore",
		"capture_name_prefix":               "ignore",
		"managed_image_resource_group_name": "ignore",
		"managed_image_name":                "ignore",

		// Does not matter for this test case, just pick one.
		"os_type": constants.Target_Linux,
	}

	var c Config
	_, err := c.Prepare(config, getPackerConfiguration())
	if err != nil {
		t.Fatal("expected config to accept VHD and Managed Image build")
	}
}

// If the user specified a build of a VHD, but started with a managed image it should be rejected.
func TestConfigShouldRejectManagedImageSourceAndVhdOutput(t *testing.T) {
	config := map[string]interface{}{
		"image_url":                         "ignore",
		"location":                          "ignore",
		"subscription_id":                   "ignore",
		"communicator":                      "none",
		"managed_image_resource_group_name": "ignore",
		"managed_image_name":                "ignore",

		// Does not matter for this test case, just pick one.
		"os_type": constants.Target_Linux,
	}

	var c Config
	_, err := c.Prepare(config, getPackerConfiguration())
	if err == nil {
		t.Fatal("expected config to reject VHD and Managed Image build")
	}
}

func TestConfigShouldRejectCustomAndPlatformManagedImageBuild(t *testing.T) {
	config := map[string]interface{}{
		"custom_managed_image_resource_group_name": "ignore",
		"custom_managed_image_name":                "ignore",
		"image_offer":                              "ignore",
		"image_publisher":                          "ignore",
		"image_sku":                                "ignore",
		"location":                                 "ignore",
		"subscription_id":                          "ignore",
		"communicator":                             "none",
		"managed_image_resource_group_name":        "ignore",
		"managed_image_name":                       "ignore",

		// Does not matter for this test case, just pick one.
		"os_type": constants.Target_Linux,
	}

	var c Config
	_, err := c.Prepare(config, getPackerConfiguration())
	if err == nil {
		t.Fatal("expected config to reject custom and platform input for a managed image build")
	}
}

func TestConfigShouldRejectCustomAndImageUrlForManagedImageBuild(t *testing.T) {
	config := map[string]interface{}{
		"image_url": "ignore",
		"custom_managed_image_resource_group_name": "ignore",
		"custom_managed_image_name":                "ignore",
		"location":                                 "ignore",
		"subscription_id":                          "ignore",
		"communicator":                             "none",
		"managed_image_resource_group_name":        "ignore",
		"managed_image_name":                       "ignore",

		// Does not matter for this test case, just pick one.
		"os_type": constants.Target_Linux,
	}

	var c Config
	_, err := c.Prepare(config, getPackerConfiguration())
	if err == nil {
		t.Fatal("expected config to reject custom and platform input for a managed image build")
	}
}

func TestConfigShouldRejectMalformedManageImageStorageAccountTypes(t *testing.T) {
	config := map[string]interface{}{
		"custom_managed_image_resource_group_name": "ignore",
		"custom_managed_image_name":                "ignore",
		"location":                                 "ignore",
		"subscription_id":                          "ignore",
		"communicator":                             "none",
		"managed_image_resource_group_name":        "ignore",
		"managed_image_name":                       "ignore",
		"managed_image_storage_account_type":       "--invalid--",

		// Does not matter for this test case, just pick one.
		"os_type": constants.Target_Linux,
	}

	var c Config
	_, err := c.Prepare(config, getPackerConfiguration())
	if err == nil {
		t.Fatal("expected config to reject custom and platform input for a managed image build")
	}
}

func TestConfigShouldRejectMalformedDiskCachingType(t *testing.T) {
	config := map[string]interface{}{
		"custom_managed_image_resource_group_name": "ignore",
		"custom_managed_image_name":                "ignore",
		"location":                                 "ignore",
		"subscription_id":                          "ignore",
		"communicator":                             "none",
		"managed_image_resource_group_name":        "ignore",
		"managed_image_name":                       "ignore",
		"disk_caching_type":                        "--invalid--",

		// Does not matter for this test case, just pick one.
		"os_type": constants.Target_Linux,
	}

	var c Config
	_, err := c.Prepare(config, getPackerConfiguration())
	if err == nil {
		t.Fatal("expected config to reject custom and platform input for a managed image build")
	}
}

func TestConfigShouldAcceptManagedImageStorageAccountTypes(t *testing.T) {
	config := map[string]interface{}{
		"custom_managed_image_resource_group_name": "ignore",
		"custom_managed_image_name":                "ignore",
		"location":                                 "ignore",
		"subscription_id":                          "ignore",
		"communicator":                             "none",
		"managed_image_resource_group_name":        "ignore",
		"managed_image_name":                       "ignore",

		// Does not matter for this test case, just pick one.
		"os_type": constants.Target_Linux,
	}

	storage_account_types := []string{"Premium_LRS", "Standard_LRS"}

	for _, x := range storage_account_types {
		config["managed_image_storage_account_type"] = x
		var c Config
		_, err := c.Prepare(config, getPackerConfiguration())
		if err != nil {
			t.Fatalf("expected config to accept a managed_image_storage_account_type of %q", x)
		}
	}
}

func TestConfigShouldAcceptDiskCachingTypes(t *testing.T) {
	config := map[string]interface{}{
		"custom_managed_image_resource_group_name": "ignore",
		"custom_managed_image_name":                "ignore",
		"location":                                 "ignore",
		"subscription_id":                          "ignore",
		"communicator":                             "none",
		"managed_image_resource_group_name":        "ignore",
		"managed_image_name":                       "ignore",

		// Does not matter for this test case, just pick one.
		"os_type": constants.Target_Linux,
	}

	storage_account_types := []string{"None", "ReadOnly", "ReadWrite"}

	for _, x := range storage_account_types {
		config["disk_caching_type"] = x
		var c Config
		_, err := c.Prepare(config, getPackerConfiguration())
		if err != nil {
			t.Fatalf("expected config to accept a disk_caching_type of %q", x)
		}
	}
}

func TestConfigShouldRejectTempAndBuildResourceGroupName(t *testing.T) {
	config := map[string]interface{}{
		"capture_name_prefix":    "ignore",
		"capture_container_name": "ignore",
		"image_offer":            "ignore",
		"image_publisher":        "ignore",
		"image_sku":              "ignore",
		"location":               "ignore",
		"storage_account":        "ignore",
		"resource_group_name":    "ignore",
		"subscription_id":        "ignore",
		"communicator":           "none",

		// custom may define one or the other, but not both
		"temp_resource_group_name":  "rgn00",
		"build_resource_group_name": "rgn00",
	}

	var c Config
	_, err := c.Prepare(config, getPackerConfiguration())
	if err == nil {
		t.Fatal("expected config to reject the use of both temp_resource_group_name and build_resource_group_name")
	}
}

func TestConfigShouldRejectInvalidResourceGroupNames(t *testing.T) {
	config := map[string]interface{}{
		"capture_name_prefix":    "ignore",
		"capture_container_name": "ignore",
		"image_offer":            "ignore",
		"image_publisher":        "ignore",
		"image_sku":              "ignore",
		"location":               "ignore",
		"storage_account":        "ignore",
		"resource_group_name":    "ignore",
		"subscription_id":        "ignore",
		"communicator":           "none",
		"os_type":                "linux",
	}

	tests := []struct {
		name string
		ok   bool
	}{
		// The Good
		{"packer-Resource-Group-jt2j3fc", true},
		{"My", true},
		{"My-(with-parens)-Resource-Group", true},

		// The Bad
		{"My Resource Group", false},
		{"My-Resource-Group-", false},
		{"My.Resource.Group.", false},

		// The Ugly
		{"My!@#!@#%$%yM", false},
		{"   ", false},
		{"My10000000000000000000000000000000000000000000000000000000000000000000000000000000000000000000000000000000000", false},
	}

	settings := []string{"temp_resource_group_name", "build_resource_group_name"}

	for _, x := range settings {
		for _, y := range tests {
			config[x] = y.name

			var c Config
			_, err := c.Prepare(config, getPackerConfiguration())
			if !y.ok && err == nil {
				t.Errorf("expected config to reject %q for setting %q", y.name, x)
			} else if y.ok && err != nil {
				t.Errorf("expected config to accept %q for setting %q", y.name, x)
			}
		}

		delete(config, "location") // not valid for build_resource_group_name
		delete(config, x)
	}
}

func TestConfigShouldRejectManagedDiskNames(t *testing.T) {
	config := map[string]interface{}{
		"image_offer":                       "ignore",
		"image_publisher":                   "ignore",
		"image_sku":                         "ignore",
		"location":                          "ignore",
		"subscription_id":                   "ignore",
		"communicator":                      "none",
		"os_type":                           "linux",
		"managed_image_name":                "ignore",
		"managed_image_resource_group_name": "ignore",
	}

	testsResourceGroupNames := []struct {
		name string
		ok   bool
	}{
		// The Good
		{"packer-Resource-Group-jt2j3fc", true},
		{"My", true},
		{"My-(with-parens)-Resource-Group", true},

		// The Bad
		{"My Resource Group", false},
		{"My-Resource-Group-", false},
		{"My.Resource.Group.", false},

		// The Ugly
		{"My!@#!@#%$%yM", false},
		{"   ", false},
		{"My10000000000000000000000000000000000000000000000000000000000000000000000000000000000000000000000000000000000", false},
	}

	settingUnderTest := "managed_image_resource_group_name"
	for _, y := range testsResourceGroupNames {
		config[settingUnderTest] = y.name

		var c Config
		_, err := c.Prepare(config, getPackerConfiguration())
		if !y.ok && err == nil {
			t.Errorf("expected config to reject %q for setting %q", y.name, settingUnderTest)
		} else if y.ok && err != nil {
			t.Errorf("expected config to accept %q for setting %q", y.name, settingUnderTest)
		}
	}

	config["managed_image_resource_group_name"] = "ignored"

	testNames := []struct {
		name string
		ok   bool
	}{
		// The Good
		{"ManagedDiskName", true},
		{"Managed-Disk-Name", true},
		{"My33", true},

		// The Bad
		{"Managed Disk Name", false},
		{"Managed-Disk-Name-", false},
		{"Managed.Disk.Name.", false},

		// The Ugly
		{"My!@#!@#%$%yM", false},
		{"   ", false},
		{"My10000000000000000000000000000000000000000000000000000000000000000000000000000000000000000000000000000000000", false},
	}

	settingUnderTest = "managed_image_name"
	for _, y := range testNames {
		config[settingUnderTest] = y.name

		var c Config
		_, err := c.Prepare(config, getPackerConfiguration())
		if !y.ok && err == nil {
			t.Logf("expected config to reject %q for setting %q", y.name, settingUnderTest)
		} else if y.ok && err != nil {
			t.Logf("expected config to accept %q for setting %q", y.name, settingUnderTest)
		}
	}
}

func TestConfigAdditionalDiskDefaultIsNil(t *testing.T) {
	var c Config
	_, err := c.Prepare(getArmBuilderConfiguration(), getPackerConfiguration())
	if err != nil {
		t.Fatal(err)
	}
	if c.AdditionalDiskSize != nil {
		t.Errorf("Expected Config to not have a set of additional disks, but got a non nil value")
	}
}

func TestConfigAdditionalDiskOverrideDefault(t *testing.T) {
	config := map[string]string{
		"capture_name_prefix":    "ignore",
		"capture_container_name": "ignore",
		"location":               "ignore",
		"image_url":              "ignore",
		"storage_account":        "ignore",
		"resource_group_name":    "ignore",
		"subscription_id":        "ignore",
		"os_type":                constants.Target_Linux,
		"communicator":           "none",
	}

	diskconfig := map[string][]int32{
		"disk_additional_size": {32, 64},
	}

	var c Config
	_, err := c.Prepare(config, diskconfig, getPackerConfiguration())
	if err != nil {
		t.Fatal(err)
	}
	if c.AdditionalDiskSize == nil {
		t.Errorf("Expected Config to have a set of additional disks, but got nil")
	}
	if len(c.AdditionalDiskSize) != 2 {
		t.Errorf("Expected Config to have a 2 additional disks, but got %d additional disks", len(c.AdditionalDiskSize))
	}
	if c.AdditionalDiskSize[0] != 32 {
		t.Errorf("Expected Config to have the first additional disks of size 32Gb, but got %dGb", c.AdditionalDiskSize[0])
	}
	if c.AdditionalDiskSize[1] != 64 {
		t.Errorf("Expected Config to have the second additional disks of size 64Gb, but got %dGb", c.AdditionalDiskSize[1])
	}
}

// Test that configuration handles plan info
//
// The use of plan info requires that the following three properties are set.
//
//  1. plan_name
//  2. plan_product
//  3. plan_publisher
func TestPlanInfoConfiguration(t *testing.T) {
	config := map[string]interface{}{
		"capture_name_prefix":    "ignore",
		"capture_container_name": "ignore",
		"image_offer":            "ignore",
		"image_publisher":        "ignore",
		"image_sku":              "ignore",
		"location":               "ignore",
		"storage_account":        "ignore",
		"resource_group_name":    "ignore",
		"subscription_id":        "ignore",
		"os_type":                "linux",
		"communicator":           "none",
	}

	planInfo := map[string]string{
		"plan_name": "--plan-name--",
	}
	config["plan_info"] = planInfo

	var c Config
	_, err := c.Prepare(config, getPackerConfiguration())
	if err == nil {
		t.Fatal("expected config to reject the use of plan_name without plan_product and plan_publisher")
	}

	planInfo["plan_product"] = "--plan-product--"
	_, err = c.Prepare(config, getPackerConfiguration())
	if err == nil {
		t.Fatal("expected config to reject the use of plan_name and plan_product without plan_publisher")
	}

	planInfo["plan_publisher"] = "--plan-publisher--"
	_, err = c.Prepare(config, getPackerConfiguration())
	if err != nil {
		t.Fatalf("expected config to accept a complete plan configuration: %s", err)
	}

	if c.PlanInfo.PlanName != "--plan-name--" {
		t.Fatalf("Expected PlanName to be '--plan-name--', but got %q", c.PlanInfo.PlanName)
	}
	if c.PlanInfo.PlanProduct != "--plan-product--" {
		t.Fatalf("Expected PlanProduct to be '--plan-product--', but got %q", c.PlanInfo.PlanProduct)
	}
	if c.PlanInfo.PlanPublisher != "--plan-publisher--" {
		t.Fatalf("Expected PlanPublisher to be '--plan-publisher--, but got %q", c.PlanInfo.PlanPublisher)
	}
}

func TestPlanInfoPromotionCode(t *testing.T) {
	config := map[string]interface{}{
		"capture_name_prefix":    "ignore",
		"capture_container_name": "ignore",
		"image_offer":            "ignore",
		"image_publisher":        "ignore",
		"image_sku":              "ignore",
		"location":               "ignore",
		"storage_account":        "ignore",
		"resource_group_name":    "ignore",
		"subscription_id":        "ignore",
		"os_type":                "linux",
		"communicator":           "none",
		"plan_info": map[string]string{
			"plan_name":           "--plan-name--",
			"plan_product":        "--plan-product--",
			"plan_publisher":      "--plan-publisher--",
			"plan_promotion_code": "--plan-promotion-code--",
		},
	}

	var c Config
	_, err := c.Prepare(config, getPackerConfiguration())
	if err != nil {
		t.Fatalf("expected config to accept plan_info configuration, but got %s", err)
	}

	if c.PlanInfo.PlanName != "--plan-name--" {
		t.Fatalf("Expected PlanName to be '--plan-name--', but got %q", c.PlanInfo.PlanName)
	}
	if c.PlanInfo.PlanProduct != "--plan-product--" {
		t.Fatalf("Expected PlanProduct to be '--plan-product--', but got %q", c.PlanInfo.PlanProduct)
	}
	if c.PlanInfo.PlanPublisher != "--plan-publisher--" {
		t.Fatalf("Expected PlanPublisher to be '--plan-publisher--, but got %q", c.PlanInfo.PlanPublisher)
	}
	if c.PlanInfo.PlanPromotionCode != "--plan-promotion-code--" {
		t.Fatalf("Expected PlanPublisher to be '--plan-promotion-code----, but got %q", c.PlanInfo.PlanPromotionCode)
	}
}

// plan_info defines 3 or 4 tags based on plan data.
// The user can define up to 15 tags.  If the combination of these two
// exceeds the max tag amount, the builder should reject the configuration.
func TestPlanInfoTooManyTagsErrors(t *testing.T) {
	exactMaxNumberOfTags := map[string]string{}
	for i := 0; i < 50; i++ {
		exactMaxNumberOfTags[fmt.Sprintf("tag%.2d", i)] = "ignored"
	}

	config := map[string]interface{}{
		"capture_name_prefix":    "ignore",
		"capture_container_name": "ignore",
		"image_offer":            "ignore",
		"image_publisher":        "ignore",
		"image_sku":              "ignore",
		"location":               "ignore",
		"storage_account":        "ignore",
		"resource_group_name":    "ignore",
		"subscription_id":        "ignore",
		"os_type":                "linux",
		"communicator":           "none",
		"azure_tags":             exactMaxNumberOfTags,
		"plan_info": map[string]string{
			"plan_name":           "--plan-name--",
			"plan_product":        "--plan-product--",
			"plan_publisher":      "--plan-publisher--",
			"plan_promotion_code": "--plan-promotion-code--",
		},
	}

	var c Config
	_, err := c.Prepare(config, getPackerConfiguration())
	if err == nil {
		t.Fatal("expected config to reject configuration due to excess tags")
	}
}

// The Azure builder creates temporary resources, but the user has some control over
// these values. This test asserts those values are controllable by the user.
func TestConfigShouldAllowTempNameOverrides(t *testing.T) {
	config := map[string]interface{}{
		"image_offer":                       "ignore",
		"image_publisher":                   "ignore",
		"image_sku":                         "ignore",
		"location":                          "ignore",
		"subscription_id":                   "ignore",
		"communicator":                      "none",
		"os_type":                           "linux",
		"managed_image_name":                "ignore",
		"managed_image_resource_group_name": "ignore",
		"temp_resource_group_name":          "myTempResourceGroupName",
		"temp_compute_name":                 "myTempComputeName",
	}

	var c Config
	_, err := c.Prepare(config, getPackerConfiguration())
	if err != nil {
		t.Errorf("newConfig failed with %q", err)
	}

	if c.TempResourceGroupName != "myTempResourceGroupName" {
		t.Errorf("expected TempResourceGroupName to be %q, but got %q", "myTempResourceGroupName", c.TempResourceGroupName)
	}
	if c.tmpResourceGroupName != "myTempResourceGroupName" {
		t.Errorf("expected tmpResourceGroupName to be %q, but got %q", "myTempResourceGroupName", c.tmpResourceGroupName)
	}

	if c.TempComputeName != "myTempComputeName" {
		t.Errorf("expected TempComputeName to be %q, but got %q", "myTempComputeName", c.TempComputeName)
	}
	if c.tmpComputeName != "myTempComputeName" {
		t.Errorf("expected tmpComputeName to be %q, but got %q", "myTempComputeName", c.tmpResourceGroupName)
	}
}

func TestConfigShouldAllowAsyncResourceGroupOverride(t *testing.T) {
	config := map[string]interface{}{
		"image_offer":                       "ignore",
		"image_publisher":                   "ignore",
		"image_sku":                         "ignore",
		"location":                          "ignore",
		"subscription_id":                   "ignore",
		"communicator":                      "none",
		"os_type":                           "linux",
		"managed_image_name":                "ignore",
		"managed_image_resource_group_name": "ignore",
		"async_resourcegroup_delete":        "true",
	}

	var c Config
	_, err := c.Prepare(config, getPackerConfiguration())
	if err != nil {
		t.Errorf("newConfig failed with %q", err)
	}

	if c.AsyncResourceGroupDelete != true {
		t.Errorf("expected async_resourcegroup_delete to be %q, but got %t", "async_resourcegroup_delete", c.AsyncResourceGroupDelete)
	}
}
func TestConfigShouldAllowAsyncResourceGroupOverrideNoValue(t *testing.T) {
	config := map[string]interface{}{
		"image_offer":                       "ignore",
		"image_publisher":                   "ignore",
		"image_sku":                         "ignore",
		"location":                          "ignore",
		"subscription_id":                   "ignore",
		"communicator":                      "none",
		"os_type":                           "linux",
		"managed_image_name":                "ignore",
		"managed_image_resource_group_name": "ignore",
	}

	var c Config
	_, err := c.Prepare(config, getPackerConfiguration())
	if err != nil {
		t.Errorf("newConfig failed with %q", err)
	}

	if c.AsyncResourceGroupDelete != false {
		t.Errorf("expected async_resourcegroup_delete to be %q, but got %t", "async_resourcegroup_delete", c.AsyncResourceGroupDelete)
	}
}
func TestConfigShouldAllowAsyncResourceGroupOverrideBadValue(t *testing.T) {
	config := map[string]interface{}{
		"image_offer":                       "ignore",
		"image_publisher":                   "ignore",
		"image_sku":                         "ignore",
		"location":                          "ignore",
		"subscription_id":                   "ignore",
		"communicator":                      "none",
		"os_type":                           "linux",
		"managed_image_name":                "ignore",
		"managed_image_resource_group_name": "ignore",
		"async_resourcegroup_delete":        "asdasda",
	}

	var c Config
	_, err := c.Prepare(config, getPackerConfiguration())
	if err != nil {
		t.Log("newConfig failed which is expected ", err)
	}

}
func TestConfigShouldAllowSharedImageGalleryOptions(t *testing.T) {
	config := map[string]interface{}{
		"location":                          "ignore",
		"subscription_id":                   "ignore",
		"os_type":                           "linux",
		"managed_image_name":                "ignore",
		"managed_image_resource_group_name": "ignore",
		"shared_image_gallery": map[string]string{
			"subscription":   "ignore",
			"resource_group": "ignore",
			"gallery_name":   "ignore",
			"image_name":     "ignore",
			"image_version":  "ignore",
		},
	}

	var c Config
	_, err := c.Prepare(config, getPackerConfiguration())
	if err != nil {
		t.Errorf("expected config to accept Shared Image Gallery options - but failed with %q", err)
	}
}

func TestConfigShouldRejectSharedImageGalleryDestinationNoVersionSet(t *testing.T) {
	config := map[string]interface{}{
		"location":        "ignore",
		"subscription_id": "ignore",
		"os_type":         "linux",
		"image_sku":       "ignore",
		"image_offer":     "ignore",
		"image_publisher": "ignore",
		"shared_image_gallery_destination": map[string]string{
			"resource_group":      "ignore",
			"gallery_name":        "ignore",
			"image_name":          "ignore",
			"replication_regions": "ignore",
		},
	}

	var c Config
	_, err := c.Prepare(config, getPackerConfiguration())
	if err == nil {
		t.Fatal("expected config to reject invalid shared image gallery destination version", err)
	}
	errorMessage := "An image_version must be specified for shared_image_gallery_destination and must follow the Major(int).Minor(int).Patch(int) format"
	if !strings.Contains(err.Error(), errorMessage) {
		t.Errorf("expected config to reject with error containing %s but got %s", errorMessage, err)
	}
}

func TestConfigShouldRejectSharedImageGalleryDestinationInvalidVersion(t *testing.T) {
	config := map[string]interface{}{
		"location":        "ignore",
		"subscription_id": "ignore",
		"os_type":         "linux",
		"image_sku":       "ignore",
		"image_offer":     "ignore",
		"image_publisher": "ignore",
		"shared_image_gallery_destination": map[string]string{
			"resource_group":      "ignore",
			"gallery_name":        "ignore",
			"image_name":          "ignore",
			"image_version":       "a.0.1",
			"replication_regions": "ignore",
		},
	}

	var c Config
	_, err := c.Prepare(config, getPackerConfiguration())
	if err == nil {
		t.Fatal("expected config to reject invalid shared image gallery destination version", err)
	}
	errorMessage := "An image_version must be specified for shared_image_gallery_destination and must follow the Major(int).Minor(int).Patch(int) format"
	if !strings.Contains(err.Error(), errorMessage) {
		t.Errorf("expected config to reject with error containing %s but got %s", errorMessage, err)
	}
}

func TestSharedImageGalleryWithSkipImageCreateOptions(t *testing.T) {
	config := map[string]interface{}{
		"location":                          "ignore",
		"subscription_id":                   "ignore",
		"os_type":                           "linux",
		"managed_image_name":                "ignore",
		"managed_image_resource_group_name": "ignore",
		"skip_create_image":                 true,
		"shared_image_gallery": map[string]string{
			"subscription":   "ignore",
			"resource_group": "ignore",
			"gallery_name":   "ignore",
			"image_name":     "ignore",
			"image_version":  "ignore",
		},
	}

	var c Config
	_, err := c.Prepare(config, getPackerConfiguration())
	if err != nil {
		t.Errorf("expected config to accept Shared Image Gallery with skip create options - but failed with %q", err)
	}

}

func TestConfigShouldAllowCommunityGalleryOptions(t *testing.T) {
	config := map[string]interface{}{
		"location":                          "ignore",
		"subscription_id":                   "ignore",
		"os_type":                           "linux",
		"managed_image_name":                "ignore",
		"managed_image_resource_group_name": "ignore",
		"async_resourcegroup_delete":        "true",
		"shared_image_gallery": map[string]string{
			"community_gallery_image_id": "/CommunityGalleries/cg/Images/img",
		},
	}

	var c Config
	_, err := c.Prepare(config, getPackerConfiguration())
	if err != nil {
		t.Errorf("community gallery might not be accepted - failed with %q", err)
	}

}

func TestConfigShouldAllowDirectSharedGalleryOptions(t *testing.T) {
	config := map[string]interface{}{
		"location":                          "ignore",
		"subscription_id":                   "ignore",
		"os_type":                           "linux",
		"managed_image_name":                "ignore",
		"managed_image_resource_group_name": "ignore",
		"async_resourcegroup_delete":        "true",
		"shared_image_gallery": map[string]string{
			"direct_shared_gallery_image_id": "/SharedGalleries/cg/Images/img",
		},
	}

	var c Config
	_, err := c.Prepare(config, getPackerConfiguration())
	if err != nil {
		t.Errorf("direct shared gallery might not be accepted - failed with %q", err)
	}

}

func TestConfigShouldNotAllowBothDirectSharedGalleryAndCommunityGalleryOptions(t *testing.T) {
	config := map[string]interface{}{
		"location":                          "ignore",
		"subscription_id":                   "ignore",
		"os_type":                           "linux",
		"managed_image_name":                "ignore",
		"managed_image_resource_group_name": "ignore",
		"async_resourcegroup_delete":        "true",
		"shared_image_gallery": map[string]string{
			"direct_shared_gallery_image_id": "/SharedGalleries/cg/Images/img",
			"community_gallery_image_id":     "/CommunityGalleries/cg/Images/img",
		},
	}

	var c Config
	_, err := c.Prepare(config, getPackerConfiguration())
	if err == nil {
		t.Errorf("Provided both direct shared and community gallery as inputs and did not get error.")
	}

}

func TestConfigShouldNotAllowBothCommunityAndSharedImageGalleryOptions(t *testing.T) {
	config := map[string]interface{}{
		"location":                          "ignore",
		"subscription_id":                   "ignore",
		"os_type":                           "linux",
		"managed_image_name":                "ignore",
		"managed_image_resource_group_name": "ignore",
		"shared_image_gallery": map[string]string{
			"subscription":               "ignore",
			"resource_group":             "ignore",
			"gallery_name":               "ignore",
			"image_name":                 "ignore",
			"image_version":              "ignore",
			"community_gallery_image_id": "/CommunityGalleries/cg/Images/img",
		},
	}

	var c Config
	_, err := c.Prepare(config, getPackerConfiguration())
	if err == nil {
		t.Errorf("Provided both normal private gallery and community gallery as inputs and did not get error.")
	}

}

func TestConfigShouldRejectSharedImageGalleryInvalidStorageAccountType(t *testing.T) {
	config := map[string]interface{}{
		"location":        "ignore",
		"subscription_id": "ignore",
		"os_type":         "linux",
		"shared_image_gallery": map[string]string{
			"subscription":         "ignore",
			"resource_group":       "ignore",
			"gallery_name":         "ignore",
			"image_name":           "ignore",
			"image_version":        "ignore",
			"storage_account_type": "--invalid--",
		},
	}

	var c Config
	_, err := c.Prepare(config, getPackerConfiguration())
	if err != nil {
		t.Log("config Shared Image Gallery with unsupported storage account type failed which is expected", err)
	}

}

func TestConfigShouldRejectSharedImageGalleryWithVhdTarget(t *testing.T) {
	config := map[string]interface{}{
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
		"resource_group_name":    "ignore",
		"storage_account":        "ignore",
		"capture_container_name": "ignore",
		"capture_name_prefix":    "ignore",
	}

	var c Config
	_, err := c.Prepare(config, getPackerConfiguration())
	if err != nil {
		t.Log("expected an error if Shared Image Gallery source is used with VHD target", err)
	}
}

func Test_GivenZoneNotSupportingResiliency_ConfigValidate_ShouldWarn(t *testing.T) {
	builderValues := getArmBuilderConfiguration()
	builderValues["managed_image_zone_resilient"] = "true"
	builderValues["location"] = "ukwest"

	var c Config
	_, err := c.Prepare(builderValues, getPackerConfiguration())
	if err != nil {
		t.Errorf("newConfig failed with %q", err)
	}

	var m = ""
	c.validateLocationZoneResiliency(func(s string) { m = s })

	if m != "WARNING: Zone resiliency may not be supported in ukwest, checkout the docs at https://docs.microsoft.com/en-us/azure/availability-zones/" {
		t.Errorf("warning message not as expected: %s", m)
	}
}

func Test_GivenZoneSupportingResiliency_ConfigValidate_ShouldNotWarn(t *testing.T) {
	builderValues := getArmBuilderConfiguration()
	builderValues["managed_image_zone_resilient"] = "true"
	builderValues["location"] = "westeurope"

	var c Config
	_, err := c.Prepare(builderValues, getPackerConfiguration())
	if err != nil {
		t.Errorf("newConfig failed with %q", err)
	}

	var m = ""
	c.validateLocationZoneResiliency(func(s string) { m = s })

	if m != "" {
		t.Errorf("warning message not as expected: %s", m)
	}
}

func TestConfig_PrepareProvidedWinRMPassword(t *testing.T) {
	config := getArmBuilderConfiguration()
	config["communicator"] = "winrm"

	var c Config
	tc := []struct {
		name       string
		password   string
		shouldFail bool
	}{
		{
			name:       "password should be longer than 8 characters",
			password:   "packer",
			shouldFail: true,
		},
		{
			name:       "password should be shorter than 123 characters",
			password:   "1Aaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaa",
			shouldFail: true,
		},
		{
			name:       "password should have valid size but only lower and upper case letters",
			password:   "AAAbbbCCC",
			shouldFail: true,
		},
		{
			name:       "password should have valid size but only digits and upper case letters",
			password:   "AAA12345",
			shouldFail: true,
		},
		{
			name:       "password should have valid size, digits, upper and lower case letters",
			password:   "AAA12345bbb",
			shouldFail: false,
		},
		{
			name:       "password should have valid size, digits, special characters and lower case letters",
			password:   "//12345bbb",
			shouldFail: false,
		},
	}

	for _, tt := range tc {
		t.Run(tt.name, func(t *testing.T) {
			config["winrm_password"] = tt.password
			_, err := c.Prepare(config)
			fail := err != nil
			if tt.shouldFail != fail {
				t.Fatalf("bad: %s. Expected fail is: %t but it was %t", tt.name, tt.shouldFail, fail)
			}
		})
	}
}

func getArmBuilderConfiguration() map[string]interface{} {
	m := make(map[string]interface{})
	for _, v := range requiredConfigValues {
		m[v] = "ignored00"
	}

	m["communicator"] = "none"
	m["os_type"] = constants.Target_Linux
	return m
}

func getArmBuilderConfigurationWithWindows() map[string]string {
	m := make(map[string]string)
	for _, v := range requiredConfigValues {
		m[v] = "ignored00"
	}

	m["object_id"] = "ignored00"
	m["tenant_id"] = "ignored00"
	m["subscription_id"] = "ignored00"
	m["use_azure_cli_auth"] = "true"
	m["winrm_username"] = "ignored00"
	m["communicator"] = "winrm"
	m["os_type"] = constants.Target_Windows
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

func getPackerSSHPasswordCommunicatorConfiguration() map[string]string {
	config := map[string]string{
		"ssh_password": "superS3cret",
	}

	return config
}

func TestConfigShouldRejectMalformedUserAssignedManagedIdentities(t *testing.T) {
	config := map[string]interface{}{
		"capture_name_prefix":    "ignore",
		"capture_container_name": "ignore",
		"image_offer":            "ignore",
		"image_publisher":        "ignore",
		"image_sku":              "ignore",
		"location":               "ignore",
		"storage_account":        "ignore",
		"resource_group_name":    "ignore",
		"subscription_id":        "ignore",
		"communicator":           "none",
		// Does not matter for this test case, just pick one.
		"os_type": constants.Target_Linux,
	}

	config["user_assigned_managed_identities"] = []string{"/subscriptions/00000000-0000-0000-0000-000000000000/resourceGroups/rg1/providers/Microsoft.ManagedIdentity/userAssignedIdentities/id"}
	var c Config
	if _, err := c.Prepare(config, getPackerConfiguration()); err != nil {
		t.Error("Expected test to pass, but it failed with the well-formed user_assigned_managed_identities.")
	}

	malformedUserAssignedManagedIdentityResourceIds := []string{
		"not_a_resource_id",
		"/subscriptions/00000000-0000-0000-0000-000000000000/resourceGroups/rg1/providers/Microsoft.ManagedIdentity/userAssignedIdentities/",
		"/subscriptions/00000000-0000-0000-0000-000000000000/resourceGroups/rg1/providers/Microsoft.Compute/images/im",
		"/subscriptions/00000000-0000-0000-0000-000000000000/resourceGroups/rg1/providers/Microsoft.ManagedIdentity/userAssignedIdentitie/id",
	}

	for _, x := range malformedUserAssignedManagedIdentityResourceIds {
		config["user_assigned_managed_identities"] = x
		var c Config
		if _, err := c.Prepare(config, getPackerConfiguration()); err == nil {
			t.Errorf("Expected test to fail, but it succeeded with the malformed user_assigned_managed_identities set to %q.", x)
		}
	}
}

func TestConfigShouldRejectUserDataAndUserDataFile(t *testing.T) {
	tmpfile, err := os.CreateTemp("", "userdata")
	if err != nil {
		t.Fatalf("failed creating tempfile: %s", err)
	}
	config := map[string]interface{}{
		"capture_container_name": "ignore",
		"capture_name_prefix":    "ignore",
		"image_publisher":        "ignore",
		"image_offer":            "ignore",
		"image_sku":              "ignore",
		"location":               "ignore",
		"storage_account":        "ignore",
		"resource_group_name":    "ignore",
		"subscription_id":        "ignore",
		"os_type":                constants.Target_Linux,
		"communicator":           "none",
		// custom may define one or the other, but not both
		"user_data":      "user_data",
		"user_data_file": tmpfile.Name(),
	}

	var c Config
	_, err = c.Prepare(config, getPackerConfiguration())

	defer os.Remove(tmpfile.Name())
	if err == nil {
		t.Fatal("expected config to reject the use of both user_data and user_data_file")
	}
}

func TestConfigShouldRejectCustomDataAndCustomDataFile(t *testing.T) {
	tmpfile, err := os.CreateTemp("", "customdata")
	if err != nil {
		t.Fatalf("failed creating tempfile: %s", err)
	}
	config := map[string]interface{}{
		"capture_container_name": "ignore",
		"capture_name_prefix":    "ignore",
		"image_publisher":        "ignore",
		"image_offer":            "ignore",
		"image_sku":              "ignore",
		"location":               "ignore",
		"storage_account":        "ignore",
		"resource_group_name":    "ignore",
		"subscription_id":        "ignore",
		"os_type":                constants.Target_Linux,
		"communicator":           "none",
		// custom may define one or the other, but not both
		"custom_data":      "custom_data",
		"custom_data_file": tmpfile.Name(),
	}

	var c Config
	_, err = c.Prepare(config, getPackerConfiguration())

	defer os.Remove(tmpfile.Name())
	if err == nil {
		t.Fatal("expected config to reject the use of both custom_data and custom_data_file")
	}
}

func TestConfigShouldRejectInvalidCustomResourceBuildPrefix(t *testing.T) {
	config := map[string]interface{}{
		"location":               "ignore",
		"subscription_id":        "ignore",
		"image_offer":            "ignore",
		"image_publisher":        "ignore",
		"image_sku":              "ignore",
		"os_type":                "linux",
		"resource_group_name":    "ignore",
		"storage_account":        "ignore",
		"capture_container_name": "ignore",
		"capture_name_prefix":    "ignore",
	}

	badResourcePrefixes := []string{"pkr_123456", "pkr-1234567", "-pkr123", "pkr.123"}
	for _, resourcePrefix := range badResourcePrefixes {
		config["custom_resource_build_prefix"] = resourcePrefix
		var c Config
		_, err := c.Prepare(config, getPackerConfiguration())
		if err == nil {
			t.Fatalf("expected config to reject %s as custom_resource_build_prefix", resourcePrefix)
		}
	}
}

func TestConfigShouldAcceptValidCustomResourceBuildPrefix(t *testing.T) {
	config := map[string]interface{}{
		"location":               "ignore",
		"subscription_id":        "ignore",
		"image_offer":            "ignore",
		"image_publisher":        "ignore",
		"image_sku":              "ignore",
		"os_type":                "linux",
		"resource_group_name":    "ignore",
		"storage_account":        "ignore",
		"capture_container_name": "ignore",
		"capture_name_prefix":    "ignore",
	}

	goodResourcePrefixes := []string{"pkr-123456", "pkr-12345-", "pkr123"}
	for _, resourcePrefix := range goodResourcePrefixes {
		config["custom_resource_build_prefix"] = resourcePrefix
		var c Config
		_, err := c.Prepare(config, getPackerConfiguration())
		if err != nil {
			t.Fatalf("expected config to accept %s as custom_resource_build_prefix but got error: %s", resourcePrefix, err)
		}
	}
}

func TestEnvVarSetsCustomResourceBuildPrefix_Invalid(t *testing.T) {
	// Invalid env var should cause validation to fail when field not explicitly set
	t.Setenv("PACKER_AZURE_CUSTOM_RESOURCE_BUILD_PREFIX", "pkr_123456")

	config := map[string]interface{}{
		"location":               "ignore",
		"subscription_id":        "ignore",
		"image_offer":            "ignore",
		"image_publisher":        "ignore",
		"image_sku":              "ignore",
		"os_type":                "linux",
		"resource_group_name":    "ignore",
		"storage_account":        "ignore",
		"capture_container_name": "ignore",
		"capture_name_prefix":    "ignore",
	}

	var c Config
	if _, err := c.Prepare(config, getPackerConfiguration()); err == nil {
		t.Fatal("expected config to reject invalid env var value for custom_resource_build_prefix")
	}
}

func TestConfigCustomResourceBuildPrefixTakesPrecedenceOverEnv(t *testing.T) {
	// When both are present, the config value should win
	t.Setenv("PACKER_AZURE_CUSTOM_RESOURCE_BUILD_PREFIX", "pkr-env99")

	config := map[string]interface{}{
		"location":                     "ignore",
		"subscription_id":              "ignore",
		"image_offer":                  "ignore",
		"image_publisher":              "ignore",
		"image_sku":                    "ignore",
		"os_type":                      "linux",
		"resource_group_name":          "ignore",
		"storage_account":              "ignore",
		"capture_container_name":       "ignore",
		"capture_name_prefix":          "ignore",
		"custom_resource_build_prefix": "pkr-12345-",
	}

	var c Config
	if _, err := c.Prepare(config, getPackerConfiguration()); err != nil {
		t.Fatalf("expected config to succeed when both env and config set, got error: %s", err)
	}
	if c.CustomResourcePrefix != "pkr-12345-" {
		t.Fatalf("expected CustomResourcePrefix to be set from config (precedence), got %q", c.CustomResourcePrefix)
	}
}

func TestConfigShouldNormalizeLicenseTypeCase(t *testing.T) {
	config := map[string]string{
		"capture_name_prefix":    "ignore",
		"capture_container_name": "ignore",
		"location":               "ignore",
		"image_url":              "ignore",
		"storage_account":        "ignore",
		"resource_group_name":    "ignore",
		"subscription_id":        "ignore",
		"communicator":           "none",
	}

	test_inputs := map[string]map[string][]string{
		constants.Target_Linux: {
			constants.License_RHEL: {"rhel_byos", "rHEL_byos"},
			constants.License_SUSE: {"sles_byos", "sLes_BYoS"},
		},
		constants.Target_Windows: {
			constants.License_Windows_Client: {"windows_client", "WINdOWS_CLIenT"},
			constants.License_Windows_Server: {"windows_server", "WINdOWS_SErVER"},
		},
	}

	for os_type, license_types := range test_inputs {
		for expected, v := range license_types {
			for _, license_type := range v {
				config["license_type"] = license_type
				config["os_type"] = os_type
				var c Config
				_, err := c.Prepare(config, getPackerConfiguration())
				if err != nil {
					t.Fatalf("Expected config to accept the value %q, but it did not", license_type)
				}

				if c.LicenseType != expected {
					t.Fatalf("Expected config to normalize the value %q to %q, but it did not", license_type, expected)
				}
			}
		}
	}
}

func TestConfigShouldValidateLicenseType(t *testing.T) {
	config := map[string]string{
		"capture_name_prefix":    "ignore",
		"capture_container_name": "ignore",
		"location":               "ignore",
		"image_url":              "ignore",
		"storage_account":        "ignore",
		"resource_group_name":    "ignore",
		"subscription_id":        "ignore",
		"communicator":           "none",
	}

	good_inputs := map[string]map[string][]string{
		constants.Target_Linux: {
			constants.License_RHEL: {"rhel_byos", "rHEL_byos"},
			constants.License_SUSE: {"sles_byos", "sLes_BYoS"},
		},
		constants.Target_Windows: {
			constants.License_Windows_Client: {"windows_client", "WINdOWS_CLIenT"},
			constants.License_Windows_Server: {"windows_server", "WINdOWS_SErVER"},
		},
	}

	for os_type, license_types := range good_inputs {
		for _, v := range license_types {
			for _, license_type := range v {
				config["license_type"] = license_type
				config["os_type"] = os_type
				var c Config
				_, err := c.Prepare(config, getPackerConfiguration())
				if err != nil {
					t.Fatalf("Expected config to accept the value %q, but it did not", license_type)
				}
			}
		}
	}

	bad_inputs := map[string]map[string][]string{
		constants.Target_Linux: {
			constants.License_RHEL: {"windows_client", "windows"},
			constants.License_SUSE: {"WINdOWS_CLIenT", "server"},
		},
		constants.Target_Windows: {
			constants.License_Windows_Client: {"sles_byos", "rHEL"},
			constants.License_Windows_Server: {"rhel_byos", "sLes"},
		},
	}

	for os_type, license_types := range bad_inputs {
		for _, v := range license_types {
			for _, license_type := range v {
				config["license_type"] = license_type
				config["os_type"] = os_type
				var c Config
				_, err := c.Prepare(config, getPackerConfiguration())
				if err == nil {
					t.Fatalf("Expected config to not accept the value %q for os_type %q, but it did", license_type, os_type)
				}
			}
		}
	}
}

func TestConfigSpot(t *testing.T) {
	config := map[string]interface{}{
		"capture_container_name": "ignore",
		"capture_name_prefix":    "ignore",
		"image_publisher":        "ignore",
		"image_offer":            "ignore",
		"image_sku":              "ignore",
		"location":               "ignore",
		"storage_account":        "ignore",
		"resource_group_name":    "ignore",
		"subscription_id":        "ignore",
		"os_type":                constants.Target_Linux,
		"communicator":           "none",
		"spot": map[string]interface{}{
			"eviction_policy": "Deallocate",
		},
	}
	var c Config
	_, err := c.Prepare(config, getPackerConfiguration())
	if err != nil {
		t.Fatal("expected config to accept spot settings", err)
	}
}

func TestConfigSpotInvalidEvictionPolicy(t *testing.T) {
	config := map[string]interface{}{
		"capture_container_name": "ignore",
		"capture_name_prefix":    "ignore",
		"image_publisher":        "ignore",
		"image_offer":            "ignore",
		"image_sku":              "ignore",
		"location":               "ignore",
		"storage_account":        "ignore",
		"resource_group_name":    "ignore",
		"subscription_id":        "ignore",
		"os_type":                constants.Target_Linux,
		"communicator":           "none",
		"spot": map[string]interface{}{
			"eviction_policy": "test",
		},
	}
	var c Config
	_, err := c.Prepare(config, getPackerConfiguration())
	if err == nil {
		t.Fatal("expected config to not accept spot settings", err)
	}
}

func TestConfigSpotEmptyEvictionPolicy(t *testing.T) {
	config := map[string]interface{}{
		"capture_container_name": "ignore",
		"capture_name_prefix":    "ignore",
		"image_publisher":        "ignore",
		"image_offer":            "ignore",
		"image_sku":              "ignore",
		"location":               "ignore",
		"storage_account":        "ignore",
		"resource_group_name":    "ignore",
		"subscription_id":        "ignore",
		"os_type":                constants.Target_Linux,
		"communicator":           "none",
		"spot":                   map[string]interface{}{},
	}
	var c Config
	_, err := c.Prepare(config, getPackerConfiguration())
	if err != nil {
		t.Fatal("expected config to accept spot settings", err)
	}
}

func TestConfigSpotEmptyEvictionPolicyMaxPriceSet(t *testing.T) {
	config := map[string]interface{}{
		"capture_container_name": "ignore",
		"capture_name_prefix":    "ignore",
		"image_publisher":        "ignore",
		"image_offer":            "ignore",
		"image_sku":              "ignore",
		"location":               "ignore",
		"storage_account":        "ignore",
		"resource_group_name":    "ignore",
		"subscription_id":        "ignore",
		"os_type":                constants.Target_Linux,
		"communicator":           "none",
		"spot": map[string]interface{}{
			"max_price": 100,
		},
	}
	var c Config
	_, err := c.Prepare(config, getPackerConfiguration())
	if err == nil {
		t.Fatal("expected config to not accept spot settings", err)
	}
}

func TestConfigShouldRejectSharedImageGalleryDestinationReplicationRegions(t *testing.T) {
	config := map[string]interface{}{
		"location":        "ignore",
		"subscription_id": "ignore",
		"os_type":         "linux",
		"image_sku":       "ignore",
		"image_offer":     "ignore",
		"image_publisher": "ignore",
		"shared_image_gallery_destination": map[string]interface{}{
			"resource_group":      "ignore",
			"gallery_name":        "ignore",
			"image_name":          "ignore",
			"image_version":       "1.0.1",
			"replication_regions": "ignore",
			"target_region": map[string]string{
				"name": "useast",
			},
		},
	}

	var c Config
	_, err := c.Prepare(config, getPackerConfiguration())
	if err == nil {
		t.Fatal("expected config to reject invalid shared image gallery destination the defines both replication_regions and target_region block", err)
	}
	errorMessage := "`replicated_regions` can not be defined alongside `target_region`; you can define a target_region for each destination region you wish to replicate to."
	if !strings.Contains(err.Error(), errorMessage) {
		t.Errorf("expected config to reject with error containing %s but got %s", errorMessage, err)
	}
}

func TestConfigShouldRejectCVMSourceToBuildManagedImage(t *testing.T) {
	config := map[string]interface{}{
		"image_offer":                       "ignore",
		"image_publisher":                   "ignore",
		"image_sku":                         "ignore",
		"location":                          "ignore",
		"subscription_id":                   "ignore",
		"communicator":                      "none",
		"managed_image_name":                "ignore",
		"managed_image_resource_group_name": "ignore",
		"os_type":                           constants.Target_Linux, // Does not matter for this test case, just pick one.
		"security_type":                     "ConfidentialVM",
		"security_encryption_type":          "VMGuestStateOnly",
	}

	var c Config
	_, err := c.Prepare(config, getPackerConfiguration())
	errorMessage := `Setting a security type of 'ConfidentialVM' is not allowed when building a VHD or creating a Managed Image, only when publishing directly to Shared Image Gallery`
	if err == nil {
		t.Fatalf("expected config to reject with the following error: %q",
			errorMessage,
		)
	}
	if !strings.Contains(err.Error(), errorMessage) {
		t.Errorf("expected config to reject with error containing %s but got %s", errorMessage, err)
	}
}

func TestConfigShouldRejectNonMatchingSecurityType(t *testing.T) {
	invalidSecurityType := "ignore"
	config := map[string]interface{}{
		"image_offer":                       "ignore",
		"image_publisher":                   "ignore",
		"image_sku":                         "ignore",
		"location":                          "ignore",
		"subscription_id":                   "ignore",
		"communicator":                      "none",
		"managed_image_name":                "ignore",
		"managed_image_resource_group_name": invalidSecurityType,
		"os_type":                           constants.Target_Linux, // Does not matter for this test case, just pick one.
		"security_type":                     "ignore",
	}

	var c Config
	_, err := c.Prepare(config, getPackerConfiguration())
	errorMessage := fmt.Sprintf(`The security_type "%s" must match either "TrustedLaunch" or "ConfidentialVM".`, invalidSecurityType)
	if err == nil {
		t.Fatalf("expected config to reject with the following error: %q",
			errorMessage,
		)
	}
	if !strings.Contains(err.Error(), errorMessage) {
		t.Errorf("expected config to reject with error containing %s but got %s", errorMessage, err)
	}
}

func TestConfigShouldRejectNonMatchingSecurityEncryptionType(t *testing.T) {
	invalidSecurityType := "ignore"
	config := map[string]interface{}{
		"image_offer":              "ignore",
		"image_publisher":          "ignore",
		"image_sku":                "ignore",
		"location":                 "ignore",
		"subscription_id":          "ignore",
		"communicator":             "none",
		"os_type":                  constants.Target_Linux, // Does not matter for this test case, just pick one.
		"security_type":            "ConfidentialVM",
		"security_encryption_type": invalidSecurityType,
		"shared_image_gallery_destination": map[string]interface{}{
			"resource_group": "ignore",
			"gallery_name":   "ignore",
			"image_name":     "ignore",
			"image_version":  "1.0.1",
			"target_region": map[string]string{
				"name": "useast",
			},
			"confidential_vm_image_encryption_type": "EncryptedVMGuestStateOnlyWithPmk",
		},
	}

	var c Config
	_, err := c.Prepare(config, getPackerConfiguration())
	errorMessage := fmt.Sprintf(`The security_encryption_type "%s" must match either "VMGuestStateOnly" or "DiskWithVMGuestState"`, invalidSecurityType)
	if err == nil {
		t.Fatalf("expected config to reject with the following error: %q",
			errorMessage,
		)
	}
	if !strings.Contains(err.Error(), errorMessage) {
		t.Errorf("expected config to reject with error containing %s but got %s", errorMessage, err)
	}
}

func TestConfigShouldRejectNonMatchingSIGDestinationCVMEncryptionType(t *testing.T) {
	config := map[string]interface{}{
		"image_offer":              "ignore",
		"image_publisher":          "ignore",
		"image_sku":                "ignore",
		"location":                 "ignore",
		"subscription_id":          "ignore",
		"communicator":             "none",
		"os_type":                  constants.Target_Linux, // Does not matter for this test case, just pick one.
		"security_type":            "ConfidentialVM",
		"security_encryption_type": "DiskWithVMGuestState",
		"shared_image_gallery_destination": map[string]interface{}{
			"resource_group": "ignore",
			"gallery_name":   "ignore",
			"image_name":     "ignore",
			"image_version":  "1.0.1",
			"target_region": map[string]string{
				"name": "useast",
			},
			"confidential_vm_image_encryption_type": "ignore",
		},
	}

	var c Config
	_, err := c.Prepare(config, getPackerConfiguration())
	errorMessage := `The shared_image_gallery_destination setting confidential_vm_image_encryption_type must match either "EncryptedWithCmk", "EncryptedVMGuestStateOnlyWithPmk" or "EncryptedWithPmk"`
	if err == nil {
		t.Fatalf("expected config to reject with the following error: %q",
			errorMessage,
		)
	}
	if !strings.Contains(err.Error(), errorMessage) {
		t.Errorf("expected config to reject with error containing %s but got %s", errorMessage, err)
	}
}

func TestConfigShouldRejectSecurityEncryptionTypeIfSecurityTypeIsNotCVM(t *testing.T) {
	config := map[string]interface{}{
		"image_offer":              "ignore",
		"image_publisher":          "ignore",
		"image_sku":                "ignore",
		"location":                 "ignore",
		"subscription_id":          "ignore",
		"communicator":             "none",
		"os_type":                  constants.Target_Linux, // Does not matter for this test case, just pick one.
		"security_type":            "TrustedLaunch",
		"security_encryption_type": "DiskWithVMGuestState",
		"shared_image_gallery_destination": map[string]interface{}{
			"resource_group": "ignore",
			"gallery_name":   "ignore",
			"image_name":     "ignore",
			"image_version":  "1.0.1",
			"target_region": map[string]string{
				"name": "useast",
			},
			"confidential_vm_image_encryption_type": "EncryptedWithPmk",
		},
	}

	var c Config
	_, err := c.Prepare(config, getPackerConfiguration())
	errorMessage := `security_encryption_type must be unset if the security_type is not set to "ConfidentialVM"`
	if err == nil {
		t.Fatalf("expected config to reject with the following error: %q",
			errorMessage,
		)
	}
	if !strings.Contains(err.Error(), errorMessage) {
		t.Errorf("expected config to reject with error containing %s but got %s", errorMessage, err)
	}
}

func TestConfigShouldRejectIfSecurityEncryptionTypeDoesNotMatchSIGDestinationCVMEncryptionType01(t *testing.T) {
	config := map[string]interface{}{
		"image_offer":              "ignore",
		"image_publisher":          "ignore",
		"image_sku":                "ignore",
		"location":                 "ignore",
		"subscription_id":          "ignore",
		"communicator":             "none",
		"os_type":                  constants.Target_Linux, // Does not matter for this test case, just pick one.
		"security_type":            "ConfidentialVM",
		"security_encryption_type": "DiskWithVMGuestState",
		"shared_image_gallery_destination": map[string]interface{}{
			"resource_group": "ignore",
			"gallery_name":   "ignore",
			"image_name":     "ignore",
			"image_version":  "1.0.1",
			"target_region": map[string]string{
				"name": "useast",
			},
			"confidential_vm_image_encryption_type": "EncryptedVMGuestStateOnlyWithPmk",
		},
	}

	var c Config
	_, err := c.Prepare(config, getPackerConfiguration())
	errorMessage := "The security_encryption_type setting \"DiskWithVMGuestState\" does not match the shared_image_gallery_destination confidential_vm_image_encryption_type setting \"EncryptedVMGuestStateOnlyWithPmk\". security_encryption type \"DiskWithVMGuestState\" needs to match \"EncryptedWithPMK\" or \"EncryptedWithCMK\"."
	if err == nil {
		t.Fatalf("expected config to reject with the following error: %q",
			errorMessage,
		)
	}
	if !strings.Contains(err.Error(), errorMessage) {
		t.Errorf("expected config to reject with error containing %s but got %s", errorMessage, err)
	}
}

func TestConfigShouldRejectIfSecurityEncryptionTypeDoesNotMatchSIGDestinationCVMEncryptionType02(t *testing.T) {
	config := map[string]interface{}{
		"image_offer":              "ignore",
		"image_publisher":          "ignore",
		"image_sku":                "ignore",
		"location":                 "ignore",
		"subscription_id":          "ignore",
		"communicator":             "none",
		"os_type":                  constants.Target_Linux, // Does not matter for this test case, just pick one.
		"security_type":            "ConfidentialVM",
		"security_encryption_type": "VMGuestStateOnly",
		"shared_image_gallery_destination": map[string]interface{}{
			"resource_group": "ignore",
			"gallery_name":   "ignore",
			"image_name":     "ignore",
			"image_version":  "1.0.1",
			"target_region": map[string]string{
				"name": "useast",
			},
			"confidential_vm_image_encryption_type": "EncryptedWithPmk",
		},
	}

	var c Config
	_, err := c.Prepare(config, getPackerConfiguration())
	errorMessage := `The security_encryption_type setting "VMGuestStateOnly" does not match the shared_image_gallery_destination confidential_vm_image_encryption_type setting "EncryptedWithPmk". security_encryption type "VMGuestStateOnly" needs to match "EncryptedVMGuestStateOnlyWithPmk".`
	if err == nil {
		t.Fatalf("expected config to reject with the following error: %q",
			errorMessage,
		)
	}
	if !strings.Contains(err.Error(), errorMessage) {
		t.Errorf("expected config to reject with error containing %s but got %s", errorMessage, err)
	}

}

func TestConfigShouldRejectIfDiskEncryptionIDIsSetWithNonCMKEncryptionType(t *testing.T) {
	config := map[string]interface{}{
		"image_offer":              "ignore",
		"image_publisher":          "ignore",
		"image_sku":                "ignore",
		"location":                 "ignore",
		"subscription_id":          "ignore",
		"communicator":             "none",
		"os_type":                  constants.Target_Linux, // Does not matter for this test case, just pick one.
		"security_type":            "ConfidentialVM",
		"security_encryption_type": "DiskWithVMGuestState",
		"shared_image_gallery_destination": map[string]interface{}{
			"resource_group": "ignore",
			"gallery_name":   "ignore",
			"image_name":     "ignore",
			"image_version":  "1.0.1",
			"target_region": map[string]string{
				"name":                   "useast",
				"disk_encryption_set_id": "ignore",
			},
			"confidential_vm_image_encryption_type": "EncryptedWithPmk",
		},
	}

	var c Config
	_, err := c.Prepare(config, getPackerConfiguration())
	errorMessage := `confidential_vm_image_encryption_type must be set to "EncryptedWithCmk" when passing a disk_encryption_set_id in the target_region block`
	if err == nil {
		t.Fatalf("expected config to reject with the following error: %q",
			errorMessage,
		)
	}
	if !strings.Contains(err.Error(), errorMessage) {
		t.Errorf("expected config to reject with error containing %s but got %s", errorMessage, err)
	}
}

func TestConfigShouldRejectIfNoDiskEncryptionIDIsSetWithSIGCMKEncryptionType(t *testing.T) {
	config := map[string]interface{}{
		"image_offer":              "ignore",
		"image_publisher":          "ignore",
		"image_sku":                "ignore",
		"location":                 "ignore",
		"subscription_id":          "ignore",
		"communicator":             "none",
		"os_type":                  constants.Target_Linux, // Does not matter for this test case, just pick one.
		"security_type":            "ConfidentialVM",
		"security_encryption_type": "DiskWithVMGuestState",
		"shared_image_gallery_destination": map[string]interface{}{
			"resource_group": "ignore",
			"gallery_name":   "ignore",
			"image_name":     "ignore",
			"image_version":  "1.0.1",
			"target_region": map[string]string{
				"name":                   "useast",
				"disk_encryption_set_id": "ignore",
			},
			"confidential_vm_image_encryption_type": "EncryptedWithCmk",
		},
	}

	var c Config
	_, err := c.Prepare(config, getPackerConfiguration())
	errorMessage := `when using a confidential vm as source to an cvm image version and a confidential_vm_image_encryption_type of "EncryptedWithCmk", the source cvm must have a disk_encryption_set_id set`
	if err == nil {
		t.Fatalf("expected config to reject with the following error: %q",
			errorMessage,
		)
	}
	if !strings.Contains(err.Error(), errorMessage) {
		t.Errorf("expected config to reject with error containing %s but got %s", errorMessage, err)
	}
}

func TestConfigShouldRejectIfNoDiskEncryptionIDIsSetInSIGTargetRegions(t *testing.T) {
	config := map[string]interface{}{
		"image_offer":              "ignore",
		"image_publisher":          "ignore",
		"image_sku":                "ignore",
		"location":                 "ignore",
		"subscription_id":          "ignore",
		"communicator":             "none",
		"os_type":                  constants.Target_Linux, // Does not matter for this test case, just pick one.
		"security_type":            "ConfidentialVM",
		"security_encryption_type": "DiskWithVMGuestState",
		"disk_encryption_set_id":   "ignore",
		"shared_image_gallery_destination": map[string]interface{}{
			"resource_group": "ignore",
			"gallery_name":   "ignore",
			"image_name":     "ignore",
			"image_version":  "1.0.1",
			"target_region": []map[string]string{
				{"name": "eastus"},
				{"name": "westus"},
			},
			"confidential_vm_image_encryption_type": "EncryptedWithCmk",
		},
	}
	errorMessageRegion1 := `when using a confidential vm as source to an cvm image version and a confidential_vm_image_encryption_type of "EncryptedWithCmk", the target region "eastus" must have a disk_encryption_set_id set`
	errorMessageRegion2 := `when using a confidential vm as source to an cvm image version and a confidential_vm_image_encryption_type of "EncryptedWithCmk", the target region "westus" must have a disk_encryption_set_id set`
	var c Config
	_, err := c.Prepare(config, getPackerConfiguration())
	if err == nil {
		t.Fatalf("expected config to reject with the following errors: %q\n%q",
			errorMessageRegion1,
			errorMessageRegion2,
		)
	}
	if !strings.Contains(err.Error(), errorMessageRegion1) {
		t.Errorf("expected config to reject with error containing %s but got %s", errorMessageRegion1, err)
	}
	if !strings.Contains(err.Error(), errorMessageRegion2) {
		t.Errorf("expected config to reject with error containing %s but got %s", errorMessageRegion2, err)
	}
}

func TestConfigShouldRejectInvalidIPSku(t *testing.T) {
	config := map[string]interface{}{
		"image_offer":              "ignore",
		"image_publisher":          "ignore",
		"image_sku":                "ignore",
		"location":                 "ignore",
		"subscription_id":          "ignore",
		"communicator":             "none",
		"public_ip_sku":            "invalid",
		"os_type":                  constants.Target_Linux,
		"security_type":            "ConfidentialVM",
		"security_encryption_type": "DiskWithVMGuestState",
		"disk_encryption_set_id":   "ignore",
		"shared_image_gallery_destination": map[string]interface{}{
			"resource_group": "ignore",
			"gallery_name":   "ignore",
			"image_name":     "ignore",
			"image_version":  "1.0.1",
		},
	}
	errorMessageInvalidPublicIPSku := `The provided value of "invalid" for public_ip_sku does not match the allowed values of "Basic" or "Standard"`
	var c Config
	_, err := c.Prepare(config, getPackerConfiguration())
	if err == nil {
		t.Fatalf("expected config to reject with the following error: %q",
			errorMessageInvalidPublicIPSku,
		)
	}
	if !strings.Contains(err.Error(), errorMessageInvalidPublicIPSku) {
		t.Errorf("expected config to reject with error containing %s but got %s", errorMessageInvalidPublicIPSku, err)
	}
}

func TestConfigShouldRejectIPSkuWithUserProvidedNetwork(t *testing.T) {
	config := map[string]interface{}{
		"image_offer":              "ignore",
		"image_publisher":          "ignore",
		"image_sku":                "ignore",
		"location":                 "ignore",
		"subscription_id":          "ignore",
		"communicator":             "none",
		"public_ip_sku":            "Standard",
		"virtual_network_name":     "somenet",
		"os_type":                  constants.Target_Linux,
		"security_type":            "ConfidentialVM",
		"security_encryption_type": "DiskWithVMGuestState",
		"disk_encryption_set_id":   "ignore",
		"shared_image_gallery_destination": map[string]interface{}{
			"resource_group": "ignore",
			"gallery_name":   "ignore",
			"image_name":     "ignore",
			"image_version":  "1.0.1",
		},
	}
	errorMessagePreExistingNetwork := `If virtual_network_name is specified, public_ip_sku cannot be specified, since a new network will not be created`
	var c Config
	_, err := c.Prepare(config, getPackerConfiguration())
	if err == nil {
		t.Fatalf("expected config to reject with the following error: %q",
			errorMessagePreExistingNetwork,
		)
	}
	if !strings.Contains(err.Error(), errorMessagePreExistingNetwork) {
		t.Errorf("expected config to reject with error containing %s but got %s", errorMessagePreExistingNetwork, err)
	}
}
func TestConfigShouldAcceptValidIPSkus(t *testing.T) {
	basicConfig := map[string]interface{}{
		"image_offer":              "ignore",
		"image_publisher":          "ignore",
		"image_sku":                "ignore",
		"location":                 "ignore",
		"subscription_id":          "ignore",
		"communicator":             "none",
		"public_ip_sku":            "Basic",
		"os_type":                  constants.Target_Linux,
		"security_type":            "ConfidentialVM",
		"security_encryption_type": "DiskWithVMGuestState",
		"disk_encryption_set_id":   "ignore",
		"shared_image_gallery_destination": map[string]interface{}{
			"resource_group": "ignore",
			"gallery_name":   "ignore",
			"image_name":     "ignore",
			"image_version":  "1.0.1",
		},
	}
	standardConfig := map[string]interface{}{
		"image_offer":              "ignore",
		"image_publisher":          "ignore",
		"image_sku":                "ignore",
		"location":                 "ignore",
		"subscription_id":          "ignore",
		"communicator":             "none",
		"public_ip_sku":            "Standard",
		"os_type":                  constants.Target_Linux,
		"security_type":            "ConfidentialVM",
		"security_encryption_type": "DiskWithVMGuestState",
		"disk_encryption_set_id":   "ignore",
		"shared_image_gallery_destination": map[string]interface{}{
			"resource_group": "ignore",
			"gallery_name":   "ignore",
			"image_name":     "ignore",
			"image_version":  "1.0.1",
		},
	}
	var basic Config
	var standard Config
	_, err := basic.Prepare(basicConfig, getPackerConfiguration())
	if err != nil {
		t.Fatalf("expected config to not reject basic IP sku but rejected with error %s", err)
	}
	_, err = standard.Prepare(standardConfig, getPackerConfiguration())
	if err != nil {
		t.Fatalf("expected config to not reject standard IP sku but rejected with error %s", err)
	}

	// Check case insensitivity
	basicConfig["public_ip_sku"] = "BaSiC"
	standardConfig["public_ip_sku"] = "StAnDaRd"

	_, err = basic.Prepare(basicConfig, getPackerConfiguration())
	if err != nil {
		t.Fatalf("expected config to not reject basic IP sku but rejected with error %s", err)
	}
	_, err = standard.Prepare(standardConfig, getPackerConfiguration())
	if err != nil {
		t.Fatalf("expected config to not reject standard IP sku but rejected with error %s", err)
	}

	if basic.PublicIpSKU != string(publicipaddresses.PublicIPAddressSkuNameBasic) {
		t.Fatalf("Expected basic ip sku to be normalized to %s, but was %s", publicipaddresses.PublicIPAddressSkuNameBasic, basicConfig["public_ip_sku"])
	}

	if standard.PublicIpSKU != string(publicipaddresses.PublicIPAddressSkuNameStandard) {
		t.Fatalf("Expected standard ip sku to be normalized to %s, but was %s", publicipaddresses.PublicIPAddressSkuNameBasic, basicConfig["public_ip_sku"])
	}
}

func TestConfigShouldRejectSIGIDWhenSIGNameSet(t *testing.T) {
	config := map[string]interface{}{
		"subscription_id":        "ignore",
		"os_type":                constants.Target_Linux,
		"communicator":           "none",
		"location":               "ignore",
		"disk_encryption_set_id": "ignore",
		"shared_image_gallery": map[string]interface{}{
			"id":         "/subscriptions/00000000-0000-0000-0000-000000000000/resourceGroups/myResourceGroup/providers/Microsoft.Compute/galleries/myGallery/images/myImageDefinition/versions/1.0.0",
			"image_name": "blorp",
		},
		"shared_image_gallery_destination": map[string]interface{}{
			"resource_group": "ignore",
			"gallery_name":   "ignore",
			"image_name":     "ignore",
			"image_version":  "1.0.1",
		},
	}
	var cfg Config
	_, err := cfg.Prepare(config, getPackerConfiguration())
	if err == nil {
		t.Fatal("Expected config to reject but didn't")
	}
	if !strings.Contains(err.Error(), "When setting shared_image_gallery.id, shared_image_gallery.image_name must not be specified") {
		t.Fatalf("Unexpected err %s", err)
	}
}

func TestConfigShouldRejectSIGIDWhenSIGVersionSet(t *testing.T) {
	config := map[string]interface{}{
		"subscription_id":        "ignore",
		"os_type":                constants.Target_Linux,
		"communicator":           "none",
		"location":               "ignore",
		"disk_encryption_set_id": "ignore",
		"shared_image_gallery": map[string]interface{}{
			"id":            "/subscriptions/00000000-0000-0000-0000-000000000000/resourceGroups/myResourceGroup/providers/Microsoft.Compute/galleries/myGallery/images/myImageDefinition/versions/1.0.0",
			"image_version": "1.0.1",
		},
		"shared_image_gallery_destination": map[string]interface{}{
			"resource_group": "ignore",
			"gallery_name":   "ignore",
			"image_name":     "ignore",
			"image_version":  "1.0.1",
		},
	}
	var cfg Config
	_, err := cfg.Prepare(config, getPackerConfiguration())
	if err == nil {
		t.Fatal("Expected config to reject but didn't")
	}
	if !strings.Contains(err.Error(), "When setting shared_image_gallery.id, shared_image_gallery.image_version must not be specified") {
		t.Fatalf("Unexpected err %s", err)
	}
}

func TestConfigShouldRejectSIGIDWhenSIGSubscriptionSet(t *testing.T) {
	config := map[string]interface{}{
		"subscription_id":        "ignore",
		"os_type":                constants.Target_Linux,
		"communicator":           "none",
		"location":               "ignore",
		"disk_encryption_set_id": "ignore",
		"shared_image_gallery": map[string]interface{}{
			"id":           "/subscriptions/00000000-0000-0000-0000-000000000000/resourceGroups/myResourceGroup/providers/Microsoft.Compute/galleries/myGallery/images/myImageDefinition/versions/1.0.0",
			"subscription": "whatever",
		},
		"shared_image_gallery_destination": map[string]interface{}{
			"resource_group": "ignore",
			"gallery_name":   "ignore",
			"image_name":     "ignore",
			"image_version":  "1.0.1",
		},
	}
	var cfg Config
	_, err := cfg.Prepare(config, getPackerConfiguration())
	if err == nil {
		t.Fatal("Expected config to reject but didn't")
	}
	if !strings.Contains(err.Error(), "When setting shared_image_gallery.id, shared_image_gallery.subscription must not be specified") {
		t.Fatalf("Unexpected err %s", err)
	}
}

func TestConfigShouldRejectSIGIDWhenSIGrgSet(t *testing.T) {
	config := map[string]interface{}{
		"subscription_id":        "ignore",
		"os_type":                constants.Target_Linux,
		"communicator":           "none",
		"location":               "ignore",
		"disk_encryption_set_id": "ignore",
		"shared_image_gallery": map[string]interface{}{
			"id":             "/subscriptions/00000000-0000-0000-0000-000000000000/resourceGroups/myResourceGroup/providers/Microsoft.Compute/galleries/myGallery/images/myImageDefinition/versions/1.0.0",
			"resource_group": "ignore",
		},
		"shared_image_gallery_destination": map[string]interface{}{
			"resource_group": "ignore",
			"gallery_name":   "ignore",
			"image_name":     "ignore",
			"image_version":  "1.0.1",
		},
	}
	var cfg Config
	_, err := cfg.Prepare(config, getPackerConfiguration())
	if err == nil {
		t.Fatal("Expected config to reject but didn't")
	}
	if !strings.Contains(err.Error(), "When setting shared_image_gallery.id, shared_image_gallery.resource_group must not be specified") {
		t.Fatalf("Unexpected err %s", err)
	}
}

func TestConfigShouldAcceptSigID(t *testing.T) {
	config := map[string]interface{}{
		"subscription_id":        "ignore",
		"os_type":                constants.Target_Linux,
		"communicator":           "none",
		"location":               "ignore",
		"disk_encryption_set_id": "ignore",
		"shared_image_gallery": map[string]interface{}{
			"id": "/subscriptions/00000000-0000-0000-0000-000000000000/resourceGroups/myResourceGroup/providers/Microsoft.Compute/galleries/myGallery/images/myImageDefinition/versions/1.0.0",
		},
		"shared_image_gallery_destination": map[string]interface{}{
			"resource_group": "ignore",
			"gallery_name":   "ignore",
			"image_name":     "ignore",
			"image_version":  "1.0.1",
		},
	}
	var cfg Config
	_, err := cfg.Prepare(config, getPackerConfiguration())
	if err != nil {
		t.Fatalf("Unexpected err %s", err)
	}
}

func TestConfigShouldParseValidSigID(t *testing.T) {
	cfg := &Config{}
	cfg.SharedGallery = SharedImageGallery{
		ID: "/subscriptions/00000000-0000-0000-0000-000000000000/resourceGroups/myResourceGroup/providers/Microsoft.Compute/galleries/myGallery/images/myImageDefinition/versions/1.0.0",
	}
	sigObject := cfg.getSharedImageGalleryObjectFromId()
	if sigObject == nil {
		t.Fatalf("getSharedImageGalleryObjectFromId did not parse valid SIG ID")
	}

	expectedSigObject := &SharedImageGallery{
		Subscription:  "00000000-0000-0000-0000-000000000000",
		ResourceGroup: "myResourceGroup",
		GalleryName:   "myGallery",
		ImageName:     "myImageDefinition",
		ImageVersion:  "1.0.0",
	}

	if diff := cmp.Diff(expectedSigObject, sigObject); diff != "" {
		t.Fatalf("unexpected diff %s", diff)
	}
}

func TestConfigShouldntParseInvalidSigIDs(t *testing.T) {
	cfg := &Config{}
	cfg.SharedGallery = SharedImageGallery{
		ID: "/bufo/some-bad-id",
	}
	sigObject := cfg.getSharedImageGalleryObjectFromId()
	if sigObject != nil {
		t.Fatalf("getSharedImageGalleryObjectFromId unexpectedly parsed invalid ID")
	}

	cfg.SharedGallery = SharedImageGallery{}
	sigObject = cfg.getSharedImageGalleryObjectFromId()
	if sigObject != nil {
		t.Fatalf("getSharedImageGalleryObjectFromId unexpectedly parsed invalid ID")
	}

	cfg = &Config{}
	sigObject = cfg.getSharedImageGalleryObjectFromId()
	if sigObject != nil {
		t.Fatalf("getSharedImageGalleryObjectFromId unexpectedly parsed invalid ID")
	}
}
