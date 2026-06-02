// Copyright IBM Corp. 2013, 2026
// SPDX-License-Identifier: MPL-2.0

package arm

import (
	"errors"
	"net"
	"strings"
	"testing"

	"github.com/hashicorp/packer-plugin-azure/builder/azure/common/constants"
)

func TestOutboundDenyAddressExpansion_ReusesSharedMixedAddressHelper(t *testing.T) {
	defaultAddressLookup = fakeLookup(
		map[string][]net.IPAddr{
			"backend.example.com": {
				{IP: net.ParseIP("198.51.100.11")},
				{IP: net.ParseIP("198.51.100.10")},
			},
		},
		nil,
	)
	defer func() { defaultAddressLookup = net.DefaultResolver.LookupIPAddr }()

	config := map[string]interface{}{
		"location":                          "ignore",
		"subscription_id":                   "ignore",
		"os_type":                           constants.Target_Windows,
		"communicator":                      "winrm",
		"winrm_username":                    "ignore",
		"object_id":                         "ignored00",
		"tenant_id":                         "ignored00",
		"use_azure_cli_auth":                true,
		"image_publisher":                   "--image-publisher--",
		"image_offer":                       "--image-offer--",
		"image_sku":                         "--image-sku--",
		"image_version":                     "--version--",
		"managed_image_name":                "ManagedImageName",
		"managed_image_resource_group_name": "ManagedImageResourceGroupName",
		"deny_outbound_ip_addresses":        []string{"203.0.113.0/24", "198.51.100.10/32", "backend.example.com"},
	}

	var c Config
	_, err := c.Prepare(config, getPackerConfiguration())
	if err != nil {
		t.Fatal(err)
	}
	c.tmpKeyVaultName = "--keyvault-name--"

	builder, err := GetVirtualMachineTemplateBuilder(&c)
	if err != nil {
		t.Fatal(err)
	}

	bs, err := builder.ToJSON()
	if err != nil {
		t.Fatal(err)
	}
	jsonText := *bs
	for _, want := range []string{"198.51.100.10", "198.51.100.11", "203.0.113.0/24"} {
		if !strings.Contains(jsonText, want) {
			t.Fatalf("expected expanded destination %q in template", want)
		}
	}
}

func TestOutboundDenyAddressExpansion_FailsWholeInputOnMixedGoodAndBadEntries(t *testing.T) {
	defaultAddressLookup = fakeLookup(
		map[string][]net.IPAddr{
			"backend.example.com": {{IP: net.ParseIP("198.51.100.10")}},
		},
		map[string]error{
			"bad-backend.example.com": errors.New("nxdomain"),
		},
	)
	defer func() { defaultAddressLookup = net.DefaultResolver.LookupIPAddr }()

	config := map[string]interface{}{
		"location":                          "ignore",
		"subscription_id":                   "ignore",
		"os_type":                           constants.Target_Windows,
		"communicator":                      "winrm",
		"winrm_username":                    "ignore",
		"object_id":                         "ignored00",
		"tenant_id":                         "ignored00",
		"use_azure_cli_auth":                true,
		"image_publisher":                   "--image-publisher--",
		"image_offer":                       "--image-offer--",
		"image_sku":                         "--image-sku--",
		"image_version":                     "--version--",
		"managed_image_name":                "ManagedImageName",
		"managed_image_resource_group_name": "ManagedImageResourceGroupName",
		"deny_outbound_ip_addresses":        []string{"198.51.100.10/32", "backend.example.com", "bad-backend.example.com"},
	}

	var c Config
	_, err := c.Prepare(config, getPackerConfiguration())
	if err != nil {
		t.Fatal(err)
	}
	c.tmpKeyVaultName = "--keyvault-name--"

	_, err = GetVirtualMachineTemplateBuilder(&c)
	if err == nil {
		t.Fatal("expected outbound deny expansion to fail whole input")
	}
	if !strings.Contains(err.Error(), "bad-backend.example.com") {
		t.Fatalf("expected failing hostname in error, got %v", err)
	}
}
