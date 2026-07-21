// Copyright IBM Corp. 2013, 2026
// SPDX-License-Identifier: MPL-2.0

package template

import (
	"reflect"
	"testing"

	approvaltests "github.com/approvals/go-approval-tests"
	compute "github.com/hashicorp/go-azure-sdk/resource-manager/compute/2022-03-01/virtualmachines"
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

	err = testSubject.SetAdditionalDisks([]int32{32, 64}, nil, nil, "datadisk", compute.CachingTypesReadWrite)
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

	err = testSubject.SetAdditionalDisks([]int32{32, 64}, nil, nil, "datadisk", compute.CachingTypesReadWrite)
	if err != nil {
		t.Fatal(err)
	}

	doc, err := testSubject.ToJSON()
	if err != nil {
		t.Fatal(err)
	}

	approvaltests.VerifyJSONBytes(t, []byte(*doc))
}

// Windows build where the source image already contains a data disk (LUN 0) and
// the user supplies additional disks without explicit LUNs. The additional disks
// must skip the source-image LUN and land on LUN 1 and 2, while the source disk
// stays on LUN 0.
func TestBuildWindowsSourceDataDisk00(t *testing.T) {
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

	err = testSubject.SetAdditionalDisks([]int32{32, 64}, nil, []int32{0}, "datadisk", compute.CachingTypesReadWrite)
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
	err = testSubject.SetAdditionalDisks([]int32{32, 64}, nil, nil, "datadisk", compute.CachingTypesReadWrite)
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

// Test with accelerated networking explicitly disabled
func TestAcceleratedNetworking00(t *testing.T) {
	testSubject, err := NewTemplateBuilder(BasicTemplate)
	if err != nil {
		t.Fatal(err)
	}

	if err = testSubject.BuildLinux("--test-ssh-authorized-key--", true); err != nil {
		t.Fatal(err)
	}

	f := false
	err = testSubject.SetAcceleratedNetworking(&f)
	if err != nil {
		t.Fatal(err)
	}

	doc, err := testSubject.ToJSON()
	if err != nil {
		t.Fatal(err)
	}

	approvaltests.VerifyJSONBytes(t, []byte(*doc))
}

// Test with accelerated networking enabled
func TestAcceleratedNetworking01(t *testing.T) {
	testSubject, err := NewTemplateBuilder(BasicTemplate)
	if err != nil {
		t.Fatal(err)
	}

	if err = testSubject.BuildLinux("--test-ssh-authorized-key--", true); err != nil {
		t.Fatal(err)
	}

	tr := true
	err = testSubject.SetAcceleratedNetworking(&tr)
	if err != nil {
		t.Fatal(err)
	}

	doc, err := testSubject.ToJSON()
	if err != nil {
		t.Fatal(err)
	}

	approvaltests.VerifyJSONBytes(t, []byte(*doc))
}

// Test with disk controller type set to SCSI
func TestDiskControllerType00(t *testing.T) {
	testSubject, err := NewTemplateBuilder(BasicTemplate)
	if err != nil {
		t.Fatal(err)
	}

	if err = testSubject.BuildLinux("--test-ssh-authorized-key--", true); err != nil {
		t.Fatal(err)
	}

	err = testSubject.SetDiskControllerType("SCSI")
	if err != nil {
		t.Fatal(err)
	}

	doc, err := testSubject.ToJSON()
	if err != nil {
		t.Fatal(err)
	}

	approvaltests.VerifyJSONBytes(t, []byte(*doc))
}

// Test with disk controller type set to NVMe
func TestDiskControllerType01(t *testing.T) {
	testSubject, err := NewTemplateBuilder(BasicTemplate)
	if err != nil {
		t.Fatal(err)
	}

	if err = testSubject.BuildLinux("--test-ssh-authorized-key--", true); err != nil {
		t.Fatal(err)
	}

	err = testSubject.SetDiskControllerType("NVMe")
	if err != nil {
		t.Fatal(err)
	}

	doc, err := testSubject.ToJSON()
	if err != nil {
		t.Fatal(err)
	}

	approvaltests.VerifyJSONBytes(t, []byte(*doc))
}

// dataDiskLuns is a test helper that returns the LUNs of the configured data
// disks, in order, for the virtual machine resource of the given builder.
func dataDiskLuns(t *testing.T, b *TemplateBuilder) []int {
	t.Helper()

	resource, err := b.getResourceByType(resourceVirtualMachine)
	if err != nil {
		t.Fatalf("could not find virtual machine resource: %s", err)
	}
	if resource.Properties.StorageProfile.DataDisks == nil {
		return nil
	}

	luns := make([]int, 0, len(*resource.Properties.StorageProfile.DataDisks))
	for _, d := range *resource.Properties.StorageProfile.DataDisks {
		if d.Lun == nil {
			t.Fatalf("expected every data disk to have a LUN, found one with a nil LUN")
		}
		luns = append(luns, *d.Lun)
	}
	return luns
}

// SetSourceImageDataDisks should create one FromImage data disk per requested
// LUN, preserving the requested LUN values.
func TestSetSourceImageDataDisks(t *testing.T) {
	testSubject, err := NewTemplateBuilder(BasicTemplate)
	if err != nil {
		t.Fatal(err)
	}

	if err = testSubject.BuildWindows("winrm", "--test-key-vault-name", "--test-winrm-certificate-url--", false); err != nil {
		t.Fatal(err)
	}

	if err = testSubject.SetSourceImageDataDisks([]int32{0, 2}); err != nil {
		t.Fatal(err)
	}

	resource, err := testSubject.getResourceByType(resourceVirtualMachine)
	if err != nil {
		t.Fatal(err)
	}

	disks := *resource.Properties.StorageProfile.DataDisks
	if len(disks) != 2 {
		t.Fatalf("expected 2 source data disks, got %d", len(disks))
	}
	for i, wantLun := range []int{0, 2} {
		if disks[i].Lun == nil || *disks[i].Lun != wantLun {
			t.Errorf("disk %d: expected LUN %d, got %v", i, wantLun, disks[i].Lun)
		}
		if disks[i].CreateOption != compute.DiskCreateOptionTypesFromImage {
			t.Errorf("disk %d: expected createOption FromImage, got %q", i, disks[i].CreateOption)
		}
	}
}

// SetAdditionalDisks assigns LUNs to the additional data disks, taking the
// source-image data disk LUNs into account. The cases below exercise both
// implicit LUN assignment (which must skip LUNs already used by the source
// image and fill the gaps starting from 0) and explicit LUN assignment
// (honoured verbatim, but rejected when it collides with a source LUN).
func TestSetAdditionalDisks_LunAssignment(t *testing.T) {
	tests := []struct {
		name                    string
		diskSizeGB              []int32
		additionalDataDiskLuns  []int32
		sourceImageDataDiskLuns []int32
		wantLuns                []int
		wantErr                 bool
	}{
		{
			// Source disks on LUN 1 and 3, three implicit additional disks fill
			// the gaps and land on 0, 2 and 4.
			name:                    "implicit luns fill non-contiguous gaps",
			diskSizeGB:              []int32{32, 64, 128},
			sourceImageDataDiskLuns: []int32{1, 3},
			wantLuns:                []int{1, 3, 0, 2, 4},
		},
		{
			// Explicit, non-conflicting LUNs are honoured verbatim.
			name:                    "explicit luns are honoured",
			diskSizeGB:              []int32{32, 64},
			additionalDataDiskLuns:  []int32{3, 4},
			sourceImageDataDiskLuns: []int32{0},
			wantLuns:                []int{0, 3, 4},
		},
		{
			// An explicit LUN that collides with a source LUN is an error.
			name:                    "explicit lun conflicting with source lun errors",
			diskSizeGB:              []int32{32},
			additionalDataDiskLuns:  []int32{0},
			sourceImageDataDiskLuns: []int32{0},
			wantErr:                 true,
		},
	}

	for _, tc := range tests {
		t.Run(tc.name, func(t *testing.T) {
			testSubject, err := NewTemplateBuilder(BasicTemplate)
			if err != nil {
				t.Fatal(err)
			}

			if err = testSubject.BuildWindows("winrm", "--test-key-vault-name", "--test-winrm-certificate-url--", false); err != nil {
				t.Fatal(err)
			}
			if err = testSubject.SetManagedMarketplaceImage("WindowsServer", "2012-R2-Datacenter", "latest", "2015-1", "Premium_LRS", compute.CachingTypesReadWrite); err != nil {
				t.Fatal(err)
			}

			// SetAdditionalDisks configures the source-image data disks itself
			// when passed a non-empty sourceImageDataDiskLuns, so we do not need
			// to call SetSourceImageDataDisks separately here.
			err = testSubject.SetAdditionalDisks(tc.diskSizeGB, tc.additionalDataDiskLuns, tc.sourceImageDataDiskLuns, "datadisk", compute.CachingTypesReadWrite)
			if tc.wantErr {
				if err == nil {
					t.Fatal("expected an error, got nil")
				}
				return
			}
			if err != nil {
				t.Fatal(err)
			}

			if got := dataDiskLuns(t, testSubject); !reflect.DeepEqual(got, tc.wantLuns) {
				t.Fatalf("expected LUNs %v, got %v", tc.wantLuns, got)
			}
		})
	}
}
