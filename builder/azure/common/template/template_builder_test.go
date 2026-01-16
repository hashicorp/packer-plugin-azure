// Copyright IBM Corp. 2013, 2025
// SPDX-License-Identifier: MPL-2.0

package template

import (
	"testing"

	approvaltests "github.com/approvals/go-approval-tests"
	compute "github.com/hashicorp/go-azure-sdk/resource-manager/compute/2024-03-01/virtualmachines"
	"github.com/hashicorp/packer-plugin-azure/builder/azure/common/constants"
)

// Ensure that a Linux template is configured as expected.
// Include SSH configuration: authorized key, and key path.
func TestBuildLinux00(t *testing.T) {
	testSubject, err := NewTemplateBuilder(BasicTemplate)
	if err != nil {
		t.Fatal(err)
	}

	err = testSubject.BuildLinux("--test-ssh-authorized-key--", true)
	if err != nil {
		t.Fatal(err)
	}

	err = testSubject.SetMarketPlaceImage("Canonical", "UbuntuServer", "16.04", "latest", compute.CachingTypesReadWrite)
	if err != nil {
		t.Fatal(err)
	}

	doc, err := testSubject.ToJSON()
	if err != nil {
		t.Fatal(err)
	}

	approvaltests.VerifyJSONBytes(t, []byte(*doc))
}

// Ensure that a user can specify a custom VHD when building a Linux template.
func TestBuildLinux01(t *testing.T) {
	testSubject, err := NewTemplateBuilder(BasicTemplate)
	if err != nil {
		t.Fatal(err)
	}

	err = testSubject.BuildLinux("--test-ssh-authorized-key--", false)
	if err != nil {
		t.Fatal(err)
	}

	err = testSubject.SetImageUrl("http://azure/custom.vhd", compute.OperatingSystemTypesLinux, compute.CachingTypesReadWrite)
	if err != nil {
		t.Fatal(err)
	}

	doc, err := testSubject.ToJSON()
	if err != nil {
		t.Fatal(err)
	}

	approvaltests.VerifyJSONBytes(t, []byte(*doc))
}

// Ensure that a user can specify an existing Virtual Network
func TestBuildLinux02(t *testing.T) {
	testSubject, err := NewTemplateBuilder(BasicTemplate)
	if err != nil {
		t.Fatal(err)
	}

	err = testSubject.BuildLinux("--test-ssh-authorized-key--", true)
	if err != nil {
		t.Fatal(err)
	}
	err = testSubject.SetImageUrl("http://azure/custom.vhd", compute.OperatingSystemTypesLinux, compute.CachingTypesReadWrite)
	if err != nil {
		t.Fatal(err)
	}
	err = testSubject.SetOSDiskSizeGB(100)
	if err != nil {
		t.Fatal(err)
	}

	err = testSubject.SetVirtualNetwork("--virtual-network-resource-group--", "--virtual-network--", "--subnet-name--")
	if err != nil {
		t.Fatal(err)
	}

	doc, err := testSubject.ToJSON()
	if err != nil {
		t.Fatal(err)
	}

	approvaltests.VerifyJSONBytes(t, []byte(*doc))
}

// Ensure that a Windows template is configured as expected.
// * Include WinRM configuration.
// * Include KeyVault configuration, which is needed for WinRM.
func TestBuildWindows00(t *testing.T) {
	testSubject, err := NewTemplateBuilder(BasicTemplate)
	if err != nil {
		t.Fatal(err)
	}

	err = testSubject.BuildWindows("winrm", "--test-key-vault-name", "--test-winrm-certificate-url--", false)
	if err != nil {
		t.Fatal(err)
	}

	err = testSubject.SetMarketPlaceImage("MicrosoftWindowsServer", "WindowsServer", "2012-R2-Datacenter", "latest", compute.CachingTypesReadWrite)
	if err != nil {
		t.Fatal(err)
	}

	doc, err := testSubject.ToJSON()
	if err != nil {
		t.Fatal(err)
	}

	approvaltests.VerifyJSONBytes(t, []byte(*doc))
}

// Windows build with additional disk for an managed build
func TestBuildWindows01(t *testing.T) {
	testSubject, err := NewTemplateBuilder(BasicTemplate)
	if err != nil {
		t.Fatal(err)
	}

	err = testSubject.BuildWindows("winrm", "--test-key-vault-name", "--test-winrm-certificate-url--", false)
	if err != nil {
		t.Fatal(err)
	}

	err = testSubject.SetManagedMarketplaceImage("WindowsServer", "2012-R2-Datacenter", "latest", "2015-1", "Premium_LRS", compute.CachingTypesReadWrite)
	if err != nil {
		t.Fatal(err)
	}

	err = testSubject.SetAdditionalDisks([]int32{32, 64}, "datadisk", compute.CachingTypesReadWrite)
	if err != nil {
		t.Fatal(err)
	}

	doc, err := testSubject.ToJSON()
	if err != nil {
		t.Fatal(err)
	}

	approvaltests.VerifyJSONBytes(t, []byte(*doc))
}

// Windows build with additional disk for an unmanaged build
func TestBuildWindows02(t *testing.T) {
	testSubject, err := NewTemplateBuilder(BasicTemplate)
	if err != nil {
		t.Fatal(err)
	}

	err = testSubject.BuildWindows("winrm", "--test-key-vault-name", "--test-winrm-certificate-url--", false)
	if err != nil {
		t.Fatal(err)
	}

	err = testSubject.SetAdditionalDisks([]int32{32, 64}, "datadisk", compute.CachingTypesReadWrite)
	if err != nil {
		t.Fatal(err)
	}

	doc, err := testSubject.ToJSON()
	if err != nil {
		t.Fatal(err)
	}

	approvaltests.VerifyJSONBytes(t, []byte(*doc))
}

// Ensure that a Windows template is configured as expected.
//   - Include SSH configuration.
func TestBuildWindows03(t *testing.T) {
	testSubject, err := NewTemplateBuilder(BasicTemplate)
	if err != nil {
		t.Fatal(err)
	}

	err = testSubject.BuildWindows("ssh", "--test-key-vault-name", "--test-ssh-certificate-url--", false)
	if err != nil {
		t.Fatal(err)
	}

	err = testSubject.SetMarketPlaceImage("MicrosoftWindowsServer", "WindowsServer", "2012-R2-Datacenter", "latest", compute.CachingTypesReadWrite)
	if err != nil {
		t.Fatal(err)
	}

	doc, err := testSubject.ToJSON()
	if err != nil {
		t.Fatal(err)
	}

	approvaltests.VerifyJSONBytes(t, []byte(*doc))
}

// Ensure that a Windows template is configured as expected.
//   - Include Winrm configuration.
func TestBuildWindows04(t *testing.T) {
	testSubject, err := NewTemplateBuilder(BasicTemplate)
	if err != nil {
		t.Fatal(err)
	}

	err = testSubject.BuildWindows("winrm", "--test-key-vault-name", "--test-ssh-certificate-url--", true)
	if err != nil {
		t.Fatal(err)
	}

	err = testSubject.SetMarketPlaceImage("MicrosoftWindowsServer", "WindowsServer", "2012-R2-Datacenter", "latest", compute.CachingTypesReadWrite)
	if err != nil {
		t.Fatal(err)
	}

	doc, err := testSubject.ToJSON()
	if err != nil {
		t.Fatal(err)
	}

	approvaltests.VerifyJSONBytes(t, []byte(*doc))
}

// Ensure that a Windows template is configured as expected when skip KV create flag is set.
//   - Include SSH configuration.
//   - Expect no regression for SSH.
func TestBuildWindows05(t *testing.T) {
	testSubject, err := NewTemplateBuilder(BasicTemplate)
	if err != nil {
		t.Fatal(err)
	}

	err = testSubject.BuildWindows("ssh", "--test-key-vault-name", "--test-ssh-certificate-url--", true)
	if err != nil {
		t.Fatal(err)
	}

	err = testSubject.SetMarketPlaceImage("MicrosoftWindowsServer", "WindowsServer", "2012-R2-Datacenter", "latest", compute.CachingTypesReadWrite)
	if err != nil {
		t.Fatal(err)
	}

	doc, err := testSubject.ToJSON()
	if err != nil {
		t.Fatal(err)
	}

	approvaltests.VerifyJSONBytes(t, []byte(*doc))
}

// Windows build with additional disk for an managed build
func TestBuildEncryptedWindows(t *testing.T) {
	testSubject, err := NewTemplateBuilder(BasicTemplate)
	if err != nil {
		t.Fatal(err)
	}

	err = testSubject.BuildWindows("winrm", "--test-key-vault-name", "--test-winrm-certificate-url--", false)
	if err != nil {
		t.Fatal(err)
	}

	err = testSubject.SetManagedMarketplaceImage("WindowsServer", "2012-R2-Datacenter", "latest", "2015-1", "Premium_LRS", compute.CachingTypesReadWrite)
	if err != nil {
		t.Fatal(err)
	}
	err = testSubject.SetDiskEncryptionSetID("encrypted", nil, nil)
	if err != nil {
		t.Fatal(err)
	}
	err = testSubject.SetAdditionalDisks([]int32{32, 64}, "datadisk", compute.CachingTypesReadWrite)
	if err != nil {
		t.Fatal(err)
	}

	doc, err := testSubject.ToJSON()
	if err != nil {
		t.Fatal(err)
	}

	approvaltests.VerifyJSONBytes(t, []byte(*doc))
}

// Shared Image Gallery Build
func TestSharedImageGallery00(t *testing.T) {
	testSubject, err := NewTemplateBuilder(BasicTemplate)
	if err != nil {
		t.Fatal(err)
	}

	err = testSubject.BuildLinux("--test-ssh-authorized-key--", false)
	if err != nil {
		t.Fatal(err)
	}

	imageID := "/subscriptions/ignore/resourceGroups/ignore/providers/Microsoft.Compute/galleries/ignore/images/ignore"
	err = testSubject.SetSharedGalleryImage("westcentralus", imageID, compute.CachingTypesReadOnly)
	if err != nil {
		t.Fatal(err)
	}

	doc, err := testSubject.ToJSON()
	if err != nil {
		t.Fatal(err)
	}

	approvaltests.VerifyJSONBytes(t, []byte(*doc))
}

// Community Shared Image Gallery Build
func TestCommunitySharedImageGallery00(t *testing.T) {
	testSubject, err := NewTemplateBuilder(BasicTemplate)
	if err != nil {
		t.Fatal(err)
	}

	err = testSubject.BuildLinux("--test-ssh-authorized-key--", false)
	if err != nil {
		t.Fatal(err)
	}

	imageID := "/communityGalleries/cg/Images/img/Versions/1.0.0"
	err = testSubject.SetCommunityGalleryImage("westcentralus", imageID, compute.CachingTypesReadOnly)
	if err != nil {
		t.Fatal(err)
	}

	doc, err := testSubject.ToJSON()
	if err != nil {
		t.Fatal(err)
	}

	approvaltests.VerifyJSONBytes(t, []byte(*doc))
}

// Community Shared Image Gallery Build
func TestDirectSharedImageGallery00(t *testing.T) {
	testSubject, err := NewTemplateBuilder(BasicTemplate)
	if err != nil {
		t.Fatal(err)
	}

	err = testSubject.BuildLinux("--test-ssh-authorized-key--", false)
	if err != nil {
		t.Fatal(err)
	}

	imageID := "/sharedGalleries/dsg/Images/img/Versions/1.0.0"
	err = testSubject.SetDirectSharedGalleryImage("westcentralus", imageID, compute.CachingTypesReadOnly)
	if err != nil {
		t.Fatal(err)
	}

	doc, err := testSubject.ToJSON()
	if err != nil {
		t.Fatal(err)
	}

	approvaltests.VerifyJSONBytes(t, []byte(*doc))
}

// Linux build with Network Security Group
func TestNetworkSecurityGroup00(t *testing.T) {
	testSubject, err := NewTemplateBuilder(BasicTemplate)
	if err != nil {
		t.Fatal(err)
	}

	err = testSubject.BuildLinux("--test-ssh-authorized-key--", false)
	if err != nil {
		t.Fatal(err)
	}

	err = testSubject.SetMarketPlaceImage("Canonical", "UbuntuServer", "16.04", "latest", compute.CachingTypesReadWrite)
	if err != nil {
		t.Fatal(err)
	}

	err = testSubject.SetNetworkSecurityGroup([]string{"127.0.0.1", "192.168.100.0/24"}, 123)
	if err != nil {
		t.Fatal(err)
	}

	doc, err := testSubject.ToJSON()
	if err != nil {
		t.Fatal(err)
	}

	approvaltests.VerifyJSONBytes(t, []byte(*doc))
}

// Linux with user assigned managed identity configured
func TestSetIdentity00(t *testing.T) {
	testSubject, err := NewTemplateBuilder(BasicTemplate)
	if err != nil {
		t.Fatal(err)
	}

	if err = testSubject.BuildLinux("--test-ssh-authorized-key--", true); err != nil {
		t.Fatal(err)
	}

	if err = testSubject.SetMarketPlaceImage("Canonical", "UbuntuServer", "16.04", "latest", compute.CachingTypesReadWrite); err != nil {
		t.Fatal(err)
	}

	if err = testSubject.SetIdentity([]string{"/subscriptions/00000000-0000-0000-0000-000000000000/resourceGroups/rg1/providers/Microsoft.ManagedIdentity/userAssignedIdentities/id"}); err != nil {
		t.Fatal(err)
	}

	doc, err := testSubject.ToJSON()
	if err != nil {
		t.Fatal(err)
	}

	approvaltests.VerifyJSONBytes(t, []byte(*doc))
}

// Test with no license type
func TestLicenseType00(t *testing.T) {
	testSubject, err := NewTemplateBuilder(BasicTemplate)
	if err != nil {
		t.Fatal(err)
	}

	if err = testSubject.BuildLinux("--test-ssh-authorized-key--", true); err != nil {
		t.Fatal(err)
	}

	doc, err := testSubject.ToJSON()
	if err != nil {
		t.Fatal(err)
	}

	approvaltests.VerifyJSONBytes(t, []byte(*doc))
}

// Test with specified license type
func TestLicenseType01(t *testing.T) {
	testSubject, err := NewTemplateBuilder(BasicTemplate)
	if err != nil {
		t.Fatal(err)
	}

	if err = testSubject.BuildLinux("--test-ssh-authorized-key--", true); err != nil {
		t.Fatal(err)
	}

	err = testSubject.SetLicenseType(constants.License_SUSE)
	if err != nil {
		t.Fatal(err)
	}

	doc, err := testSubject.ToJSON()
	if err != nil {
		t.Fatal(err)
	}

	approvaltests.VerifyJSONBytes(t, []byte(*doc))
}
