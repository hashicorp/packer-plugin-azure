// Copyright IBM Corp. 2013, 2025
// SPDX-License-Identifier: MPL-2.0

package arm

// Below are the requirements for running the acceptance tests for the Packer Azure plugin ARM Builder
//
// * An Azure subscription, with a resource group, app registration based credentials and a few image galleries
// You can use the Terraform config in the terraform folder at the base of the repository
// It is recommended to set the required environment variables and then run the acceptance_test_setup.sh script in the terraform directory
//
// * The Azure CLI installed and logged in for testing CLI based authentication
// * Env Variables for Auth
// ** ARM_CLIENT_ID
// ** ARM_CLIENT_SECRET
// ** ARM_SUBSCRIPTION_ID
// *
// * Env Variables Defining Azure Resources for Packer templates
// ** ARM_RESOURCE_GROUP_NAME - Resource group
// ** ARM_STORAGE_ACCOUNT - a storage account located in above resource group
// ** ARM_RESOURCE_PREFIX - String prefix for resources unique name constraints
// ** ARM_STORAGE_CONTAINER_NAME - storage container name for blob-based tests
// ** ARM_TEMP_RESOURCE_GROUP_NAME - temp resource group name for CLI-based tests
// ** For example SIG gallery names must be unique not just within the resource group, but within a subscription, and a user may not have access to all SIGs in a subscription.
// * As well as the following misc env variables
// ** ARM_SSH_PRIVATE_KEY_FILE - the file location of a PEM encoded RSA SSH Private Key (ed25519 is not supported by Azure),
// ** PACKER_ACC - set to any non 0 value
//
// It is recommended to run the tests with the options "-v -timeout 90m"
// command, e.g.:
//   go test -v -timeout 90m -run TestBuilderAcc_.*
// This is to avoid hitting the default go test timeout, especially in the shared image gallery test

import (
	_ "embed"
	"fmt"
	"os"
	"os/exec"
	"testing"

	"github.com/hashicorp/packer-plugin-azure/builder/azure/common"
	"github.com/hashicorp/packer-plugin-sdk/acctest"
)

// This test builds two images,
// First a parent Specialized ARM 64 Linux VM to a Shared Image Gallery/Compute Gallery
// Then a second Specialized ARM64 Linux VM that uses the first as its source/parent image
func TestBuilderAcc_SharedImageGallery_ARM64SpecializedLinuxSIG_WithChildImage(t *testing.T) {
	t.Parallel()
	common.CheckAcceptanceTestEnvVars(t,
		common.CheckAcceptanceTestEnvVarsParams{
			CheckAzureCLI:          true,
			CheckSSHPrivateKeyFile: true,
		},
	)
	subscriptionID := os.Getenv("ARM_SUBSCRIPTION_ID")
	resourcePrefix := os.Getenv("ARM_RESOURCE_PREFIX")
	resourceGroupName := os.Getenv("ARM_RESOURCE_GROUP_NAME")

	// After test finishes try and delete the created versions
	defer deleteGalleryVersions(t, subscriptionID, resourceGroupName, fmt.Sprintf("%s_acctestgallery", resourcePrefix), fmt.Sprintf("%s-arm-linux-specialized-sig", resourcePrefix), []string{"1.0.0", "1.0.1"})
	// Create parent specialized shared gallery image
	acctest.TestPlugin(t, &acctest.PluginTestCase{
		Name: "test-specialized-linux-sig",
		Type: "azure-arm",
		// Run build with force to ignore previous test runs failed artifact deletions
		BuildExtraArgs: []string{"-force"},
		Template:       string(armLinuxSpecialziedSIGTemplate),
		Check: func(buildCommand *exec.Cmd, logfile string) error {
			if buildCommand.ProcessState != nil {
				if buildCommand.ProcessState.ExitCode() != 0 {
					return fmt.Errorf("Bad exit code. Logfile: %s", logfile)
				}
			}
			return nil
		},
	})

	// Create child image from a specialized parent
	acctest.TestPlugin(t, &acctest.PluginTestCase{
		Name:           "test-specialized-linux-sig-child",
		Type:           "azure-arm",
		BuildExtraArgs: []string{"-force"},
		Template:       string(armLinuxChildFromSpecializedParent),
		Check: func(buildCommand *exec.Cmd, logfile string) error {
			if buildCommand.ProcessState != nil {
				if buildCommand.ProcessState.ExitCode() != 0 {
					return fmt.Errorf("Bad exit code. Logfile: %s", logfile)
				}
			}
			return nil
		},
	})

}

func TestBuilderAcc_SharedImageGallery_WindowsSIG(t *testing.T) {
	t.Parallel()
	common.CheckAcceptanceTestEnvVars(t,
		common.CheckAcceptanceTestEnvVarsParams{
			CheckAzureCLI: true,
		},
	)

	subscriptionID := os.Getenv("ARM_SUBSCRIPTION_ID")
	resourceGroupName := os.Getenv("ARM_RESOURCE_GROUP_NAME")
	resourcePrefix := os.Getenv("ARM_RESOURCE_PREFIX")
	defer deleteGalleryVersions(t, subscriptionID, resourceGroupName, fmt.Sprintf("%s_acctestgallery", resourcePrefix), fmt.Sprintf("%s-windows-sig", resourcePrefix), []string{"1.0.0"})

	acctest.TestPlugin(t, &acctest.PluginTestCase{
		Name:           "test-windows-sig",
		Type:           "azure-arm",
		BuildExtraArgs: []string{"-force"},
		Template:       string(windowsSIGTemplate),
		Check: func(buildCommand *exec.Cmd, logfile string) error {
			if buildCommand.ProcessState != nil {
				if buildCommand.ProcessState.ExitCode() != 0 {
					return fmt.Errorf("Bad exit code. Logfile: %s", logfile)
				}
			}
			return nil
		},
	})
}

func TestBuilderAcc_ManagedDisk_Windows(t *testing.T) {
	t.Parallel()
	acctest.TestPlugin(t, &acctest.PluginTestCase{
		Name:     "test-azure-managedisk-windows",
		Type:     "azure-arm",
		Template: testBuilderAccManagedDiskWindows,
		Check: func(buildCommand *exec.Cmd, logfile string) error {
			if buildCommand.ProcessState != nil {
				if buildCommand.ProcessState.ExitCode() != 0 {
					return fmt.Errorf("Bad exit code. Logfile: %s", logfile)
				}
			}
			return nil
		},
	})
}

// TODO Implement this test to validate client cert auth
func TestBuilderAcc_ClientCertificateAuth(t *testing.T) {
	t.Skip("Unimplemented Client Cert Auth Acceptance test")
}

func TestBuilderAcc_ManagedDisk_Windows_Build_Resource_Group(t *testing.T) {
	t.Parallel()
	acctest.TestPlugin(t, &acctest.PluginTestCase{
		Name:     "test-azure-managedisk-windows-build-resource-group",
		Type:     "azure-arm",
		Template: testBuilderAccManagedDiskWindowsBuildResourceGroup,
		Check: func(buildCommand *exec.Cmd, logfile string) error {
			if buildCommand.ProcessState != nil {
				if buildCommand.ProcessState.ExitCode() != 0 {
					return fmt.Errorf("Bad exit code. Logfile: %s", logfile)
				}
			}
			return nil
		},
	})
}

func TestBuilderAcc_ManagedDisk_Windows_Build_Resource_Group_Additional_Disk(t *testing.T) {
	t.Parallel()
	acctest.TestPlugin(t, &acctest.PluginTestCase{
		Name:     "test-azure-managedisk-windows-build-resource-group-additional-disk",
		Type:     "azure-arm",
		Template: testBuilderAccManagedDiskWindowsBuildResourceGroupAdditionalDisk,
		Check: func(buildCommand *exec.Cmd, logfile string) error {
			if buildCommand.ProcessState != nil {
				if buildCommand.ProcessState.ExitCode() != 0 {
					return fmt.Errorf("Bad exit code. Logfile: %s", logfile)
				}
			}
			return nil
		},
	})
}

func TestBuilderAcc_ManagedDisk_Linux(t *testing.T) {
	t.Parallel()
	acctest.TestPlugin(t, &acctest.PluginTestCase{
		Name:     "test-azure-managedisk-linux",
		Type:     "azure-arm",
		Template: testBuilderAccManagedDiskLinux,
		Check: func(buildCommand *exec.Cmd, logfile string) error {
			if buildCommand.ProcessState != nil {
				if buildCommand.ProcessState.ExitCode() != 0 {
					return fmt.Errorf("Bad exit code. Logfile: %s", logfile)
				}
			}
			return nil
		},
	})
}

func TestBuilderAcc_ManagedDisk_Linux_AzureCLI(t *testing.T) {
	t.Parallel()
	common.CheckAcceptanceTestEnvVars(t, common.CheckAcceptanceTestEnvVarsParams{
		CheckAzureCLI: true,
	})
	acctest.TestPlugin(t, &acctest.PluginTestCase{
		Name:     "test-azure-managedisk-linux-azurecli",
		Type:     "azure-arm",
		Template: testBuilderAccManagedDiskLinuxAzureCLI,
		Check: func(buildCommand *exec.Cmd, logfile string) error {
			if buildCommand.ProcessState != nil {
				if buildCommand.ProcessState.ExitCode() != 0 {
					return fmt.Errorf("Bad exit code. Logfile: %s", logfile)
				}
			}
			return nil
		},
	})
}

func TestBuilderAcc_Blob_Windows(t *testing.T) {
	t.Parallel()
	acctest.TestPlugin(t, &acctest.PluginTestCase{
		Name:     "test-azure-blob-windows",
		Type:     "azure-arm",
		Template: testBuilderAccBlobWindows,
		Check: func(buildCommand *exec.Cmd, logfile string) error {
			if buildCommand.ProcessState != nil {
				if buildCommand.ProcessState.ExitCode() != 0 {
					return fmt.Errorf("Bad exit code. Logfile: %s", logfile)
				}
			}
			return nil
		},
	})
}

func TestBuilderAcc_Blob_Linux(t *testing.T) {
	t.Parallel()
	acctest.TestPlugin(t, &acctest.PluginTestCase{
		Name:     "test-azure-blob-linux",
		Type:     "azure-arm",
		Template: testBuilderAccBlobLinux,
		Check: func(buildCommand *exec.Cmd, logfile string) error {
			if buildCommand.ProcessState != nil {
				if buildCommand.ProcessState.ExitCode() != 0 {
					return fmt.Errorf("Bad exit code. Logfile: %s", logfile)
				}
			}
			return nil
		},
	})
}

func TestBuilderUserData_Linux(t *testing.T) {
	t.Parallel()
	tmpfile, err := os.CreateTemp("", "userdata")
	if err != nil {
		t.Fatalf("failed creating tempfile: %s", err)
	}

	defer os.Remove(tmpfile.Name())

	if _, err := tmpfile.WriteString(testBuilderCustomDataLinux); err != nil {
		t.Fatalf("failed writing userdata: %s", err)
	}
	if err := tmpfile.Close(); err != nil {
		t.Fatalf("failed closing file: %s", err)
	}

	acctest.TestPlugin(t, &acctest.PluginTestCase{
		Name:     "test-azure-userdata-linux",
		Type:     "azure-arm",
		Template: testBuilderUserDataLinux(tmpfile.Name()),
		Check: func(buildCommand *exec.Cmd, logfile string) error {
			if buildCommand.ProcessState != nil {
				if buildCommand.ProcessState.ExitCode() != 0 {
					return fmt.Errorf("Bad exit code. Logfile: %s", logfile)
				}
			}
			return nil
		},
	})
}

//go:embed testdata/rsa_sha2_only_server.pkr.hcl
var rsaSHA2OnlyTemplate []byte

//go:embed testdata/windows_sig.pkr.hcl
var windowsSIGTemplate []byte

//go:embed testdata/arm_linux_specialized.pkr.hcl
var armLinuxSpecialziedSIGTemplate []byte

//go:embed testdata/child_from_specialized_parent.pkr.hcl
var armLinuxChildFromSpecializedParent []byte

func TestBuilderAcc_rsaSHA2OnlyServer(t *testing.T) {
	t.Parallel()
	acctest.TestPlugin(t, &acctest.PluginTestCase{
		Name:     "test-azure-ubuntu-jammy-linux",
		Type:     "azure-arm",
		Template: string(rsaSHA2OnlyTemplate),
		Check: func(buildCommand *exec.Cmd, logfile string) error {
			if buildCommand.ProcessState != nil {
				if buildCommand.ProcessState.ExitCode() != 0 {
					return fmt.Errorf("Bad exit code. Logfile: %s", logfile)
				}
			}
			return nil
		},
	})
}

func deleteGalleryVersions(t *testing.T, subscriptionID string, resourceGroupName string, galleryName string, galleryImageName string, imageVersions []string) {
	for _, imageVersion := range imageVersions {
		// If we fail to delete a gallery version we should still try to delete other versions and the gallery
		// Its possible a build was canceled or failed mid test that would leave any of the builds incomplete
		// We still want to try and delete the Gallery to not leave behind orphaned resources to manually clean up
		deleteCommand := exec.Command(
			"az", "sig", "image-version", "delete",
			fmt.Sprintf("--gallery-image-definition=%s", galleryImageName),
			fmt.Sprintf("--gallery-image-version=%s", imageVersion),
			fmt.Sprintf("--gallery-name=%s", galleryName),
			fmt.Sprintf("-g=%s", resourceGroupName),
		)
		deleteStdout, err := deleteCommand.CombinedOutput()
		if err != nil {
			t.Logf("failed to delete Gallery Image Version %s:%s %s", galleryImageName, imageVersion, err)
			t.Logf("Failed command output \n%s", string(deleteStdout))
		}
	}
}

// TODO Move these templates to separate files inside the testdata directory rather than defined strings here
func testBuilderUserDataLinux(userdata string) string {
	return fmt.Sprintf(`
{
	"variables": {
	  "client_id": "{{env `+"`ARM_CLIENT_ID`"+`}}",
	  "client_secret": "{{env `+"`ARM_CLIENT_SECRET`"+`}}",
	  "subscription_id": "{{env `+"`ARM_SUBSCRIPTION_ID`"+`}}",
	  "storage_account": "{{env `+"`ARM_STORAGE_ACCOUNT`"+`}}",
	  "resource_group_name": "{{env `+"`ARM_RESOURCE_GROUP_NAME`"+`}}",
	  "capture_container_name": "{{env `+"`ARM_STORAGE_CONTAINER_NAME`"+`}}"
	},
	"builders": [{
	  "type": "azure-arm",

	  "client_id": "{{user `+"`client_id`"+`}}",
	  "client_secret": "{{user `+"`client_secret`"+`}}",
	  "subscription_id": "{{user `+"`subscription_id`"+`}}",

	  "storage_account": "{{user `+"`storage_account`"+`}}",
	  "resource_group_name": "{{user `+"`resource_group_name`"+`}}",
	  "capture_container_name": "{{user `+"`capture_container_name`"+`}}",
	  "capture_name_prefix": "testBuilderUserDataLinux",

	  "os_type": "Linux",
	  "image_publisher": "Canonical",
	  "image_offer": "UbuntuServer",
	  "image_sku": "16.04-LTS",
	  "user_data_file": "%s",

	  "location": "South Central US",
	  "vm_size": "Standard_DS2_v2"
	}]
}
`, userdata)
}

const testBuilderCustomDataLinux = `#cloud-config
growpart:
  mode: off
`

const testBuilderAccManagedDiskWindows = `
{
	"variables": {
	  "client_id": "{{env ` + "`ARM_CLIENT_ID`" + `}}",
	  "client_secret": "{{env ` + "`ARM_CLIENT_SECRET`" + `}}",
	  "resource_group_name": "{{env ` + "`ARM_RESOURCE_GROUP_NAME`" + `}}",
	  "subscription_id": "{{env ` + "`ARM_SUBSCRIPTION_ID`" + `}}"
	},
	"builders": [{
	  "type": "azure-arm",

	  "client_id": "{{user ` + "`client_id`" + `}}",
	  "client_secret": "{{user ` + "`client_secret`" + `}}",
	  "subscription_id": "{{user ` + "`subscription_id`" + `}}",

	  "managed_image_resource_group_name": "{{user ` + "`resource_group_name`" + `}}",
	  "managed_image_name": "testBuilderAccManagedDiskWindows-{{timestamp}}",

	  "os_type": "Windows",
	  "image_publisher": "MicrosoftWindowsServer",
	  "image_offer": "WindowsServer",
	  "image_sku": "2022-datacenter",

	  "communicator": "winrm",
	  "winrm_use_ssl": "true",
	  "winrm_insecure": "true",
	  "winrm_timeout": "3m",
	  "winrm_username": "packer",
	  "async_resourcegroup_delete": "true",

	  "location": "South Central US",
	  "vm_size": "Standard_DS2_v2"
	}]
}
`

const testBuilderAccManagedDiskWindowsBuildResourceGroup = `
{
	"variables": {
	  "client_id": "{{env ` + "`ARM_CLIENT_ID`" + `}}",
	  "client_secret": "{{env ` + "`ARM_CLIENT_SECRET`" + `}}",
	  "subscription_id": "{{env ` + "`ARM_SUBSCRIPTION_ID`" + `}}",
	  "resource_group_name": "{{env ` + "`ARM_RESOURCE_GROUP_NAME`" + `}}"
	},
	"builders": [{
	  "type": "azure-arm",

	  "client_id": "{{user ` + "`client_id`" + `}}",
	  "client_secret": "{{user ` + "`client_secret`" + `}}",
	  "subscription_id": "{{user ` + "`subscription_id`" + `}}",

	  "build_resource_group_name" : "{{user ` + "`resource_group_name`" + `}}",
	  "managed_image_resource_group_name": "{{user ` + "`resource_group_name`" + `}}",
	  "managed_image_name": "testBuilderAccManagedDiskWindowsBuildResourceGroup-{{timestamp}}",

	  "os_type": "Windows",
	  "image_publisher": "MicrosoftWindowsServer",
	  "image_offer": "WindowsServer",
	  "image_sku": "2022-datacenter",

	  "communicator": "winrm",
	  "winrm_use_ssl": "true",
	  "winrm_insecure": "true",
	  "winrm_timeout": "3m",
	  "winrm_username": "packer",

	  "vm_size": "Standard_DS2_v2"
	}]
}
`

const testBuilderAccManagedDiskWindowsBuildResourceGroupAdditionalDisk = `
{
	"variables": {
	  "client_id": "{{env ` + "`ARM_CLIENT_ID`" + `}}",
	  "client_secret": "{{env ` + "`ARM_CLIENT_SECRET`" + `}}",
	  "subscription_id": "{{env ` + "`ARM_SUBSCRIPTION_ID`" + `}}",
	  "resource_group_name": "{{env ` + "`ARM_RESOURCE_GROUP_NAME`" + `}}"
	},
	"builders": [{
	  "type": "azure-arm",

	  "client_id": "{{user ` + "`client_id`" + `}}",
	  "client_secret": "{{user ` + "`client_secret`" + `}}",
	  "subscription_id": "{{user ` + "`subscription_id`" + `}}",

	  "build_resource_group_name" : "{{user ` + "`resource_group_name`" + `}}",
	  "managed_image_resource_group_name": "{{user ` + "`resource_group_name`" + `}}",
	  "managed_image_name": "testBuilderAccManagedDiskWindowsBuildResourceGroupAdditionDisk-{{timestamp}}",

	  "os_type": "Windows",
	  "image_publisher": "MicrosoftWindowsServer",
	  "image_offer": "WindowsServer",
	  "image_sku": "2022-datacenter",

	  "communicator": "winrm",
	  "winrm_use_ssl": "true",
	  "winrm_insecure": "true",
	  "winrm_timeout": "3m",
	  "winrm_username": "packer",
	  "async_resourcegroup_delete": "true",

	  "vm_size": "Standard_DS2_v2",
	  "disk_additional_size": [10,15]
	}]
}
`

const testBuilderAccManagedDiskLinux = `
{
	"variables": {
	  "client_id": "{{env ` + "`ARM_CLIENT_ID`" + `}}",
	  "client_secret": "{{env ` + "`ARM_CLIENT_SECRET`" + `}}",
	  "subscription_id": "{{env ` + "`ARM_SUBSCRIPTION_ID`" + `}}",
	  "resource_group_name": "{{env ` + "`ARM_RESOURCE_GROUP_NAME`" + `}}"
	},
	"builders": [{
	  "type": "azure-arm",

	  "client_id": "{{user ` + "`client_id`" + `}}",
	  "client_secret": "{{user ` + "`client_secret`" + `}}",
	  "subscription_id": "{{user ` + "`subscription_id`" + `}}",

	  "managed_image_resource_group_name": "{{user ` + "`resource_group_name`" + `}}",
	  "managed_image_name": "testBuilderAccManagedDiskLinux-{{timestamp}}",

	  "os_type": "Linux",
	  "image_publisher": "Canonical",
	  "image_offer": "UbuntuServer",
	  "image_sku": "16.04-LTS",

	  "location": "South Central US",
	  "vm_size": "Standard_DS2_v2",
	  "azure_tags": {
	    "env": "testing",
	    "builder": "packer"
	   }
	}]
}
`

const testBuilderAccBlobWindows = `
{
	"variables": {
	  "client_id": "{{env ` + "`ARM_CLIENT_ID`" + `}}",
	  "client_secret": "{{env ` + "`ARM_CLIENT_SECRET`" + `}}",
	  "subscription_id": "{{env ` + "`ARM_SUBSCRIPTION_ID`" + `}}",
	  "storage_account": "{{env ` + "`ARM_STORAGE_ACCOUNT`" + `}}",
	  "resource_group_name": "{{env ` + "`ARM_RESOURCE_GROUP_NAME`" + `}}",
	  "capture_container_name": "{{env ` + "`ARM_STORAGE_CONTAINER_NAME`" + `}}"
	},
	"builders": [{
	  "type": "azure-arm",

	  "client_id": "{{user ` + "`client_id`" + `}}",
	  "client_secret": "{{user ` + "`client_secret`" + `}}",
	  "subscription_id": "{{user ` + "`subscription_id`" + `}}",

	  "storage_account": "{{user ` + "`storage_account`" + `}}",
	  "resource_group_name": "{{user ` + "`resource_group_name`" + `}}",
	  "capture_container_name": "{{user ` + "`capture_container_name`" + `}}",
	  "capture_name_prefix": "testBuilderAccBlobWin",

	  "os_type": "Windows",
	  "image_publisher": "MicrosoftWindowsServer",
	  "image_offer": "WindowsServer",
	  "image_sku": "2022-datacenter",

	  "communicator": "winrm",
	  "winrm_use_ssl": "true",
	  "winrm_insecure": "true",
	  "winrm_timeout": "3m",
	  "winrm_username": "packer",

	  "location": "South Central US",
	  "vm_size": "Standard_DS2_v2"
	}]
}
`

const testBuilderAccBlobLinux = `
{
	"variables": {
	  "client_id": "{{env ` + "`ARM_CLIENT_ID`" + `}}",
	  "resource_group_name": "{{env ` + "`ARM_RESOURCE_GROUP_NAME`" + `}}",
	  "client_secret": "{{env ` + "`ARM_CLIENT_SECRET`" + `}}",
	  "subscription_id": "{{env ` + "`ARM_SUBSCRIPTION_ID`" + `}}",
	  "storage_account": "{{env ` + "`ARM_STORAGE_ACCOUNT`" + `}}",
	  "capture_container_name": "{{env ` + "`ARM_STORAGE_CONTAINER_NAME`" + `}}"
	},
	"builders": [{
	  "type": "azure-arm",

	  "client_id": "{{user ` + "`client_id`" + `}}",
	  "client_secret": "{{user ` + "`client_secret`" + `}}",
	  "subscription_id": "{{user ` + "`subscription_id`" + `}}",

	  "storage_account": "{{user ` + "`storage_account`" + `}}",
	  "resource_group_name": "{{user ` + "`resource_group_name`" + `}}",
	  "capture_container_name": "{{user ` + "`capture_container_name`" + `}}",
	  "capture_name_prefix": "testBuilderAccBlobLinux",

	  "os_type": "Linux",
	  "image_publisher": "Canonical",
	  "image_offer": "UbuntuServer",
	  "image_sku": "16.04-LTS",

	  "location": "South Central US",
	  "vm_size": "Standard_DS2_v2"
	}]
}
`

const testBuilderAccManagedDiskLinuxAzureCLI = `
{
	"variables": {
	  "resource_group_name": "{{env ` + "`ARM_RESOURCE_GROUP_NAME`" + `}}",
	  "temp_resource_group_name": "{{env ` + "`ARM_TEMP_RESOURCE_GROUP_NAME`" + `}}"
	},
	"builders": [{
	  "type": "azure-arm",

	  "use_azure_cli_auth": true,

	  "managed_image_resource_group_name": "{{user ` + "`resource_group_name`" + `}}",
	  "managed_image_name": "testBuilderAccManagedDiskLinuxAzureCLI-{{timestamp}}",
	  "temp_resource_group_name": "{{user ` + "`temp_resource_group_name`" + `}}",

	  "os_type": "Linux",
	  "image_publisher": "Canonical",
	  "image_offer": "UbuntuServer",
	  "image_sku": "16.04-LTS",

	  "location": "South Central US",
	  "vm_size": "Standard_DS2_v2",
	  "azure_tags": {
	    "env": "testing",
	    "builder": "packer"
	   }
	}]
}
`
