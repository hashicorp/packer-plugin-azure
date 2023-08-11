// Copyright (c) HashiCorp, Inc.
// SPDX-License-Identifier: MPL-2.0

package arm

// these tests require the following variables to be set,
// although some test will only use a subset:
//
// * ARM_CLIENT_ID
// * ARM_CLIENT_SECRET
// * ARM_SUBSCRIPTION_ID
// * ARM_STORAGE_ACCOUNT
// * ARM_SSH_PRIVATE_KEY_FILE - the file location of a PEM encoded RSA SSH Private Key (ed25519 is not supported by Azure),
//
// The subscription in question should have a resource group
// called "packer-acceptance-test" in "South Central US" region. The
// storage account referred to in the above variable should
// be inside this resource group and in "South Central US" as well.
//
// There should be a shared image gallery inside of the resource group
// it should be called `acctestgallery` in "South Central US" as well.
//
// In addition, the PACKER_ACC variable should also be set to
// a non-empty value to enable Packer acceptance tests and the
// options "-v -timeout 90m" should be provided to the test
// command, e.g.:
//   go test -v -timeout 90m -run TestBuilderAcc_.*

import (
	"bytes"
	"context"
	_ "embed"
	"fmt"
	"io/ioutil"
	"os"
	"os/exec"
	"testing"
	"time"

	"github.com/hashicorp/go-azure-sdk/resource-manager/compute/2022-03-03/galleryimages"
	"github.com/hashicorp/go-azure-sdk/resource-manager/compute/2022-03-03/galleryimageversions"
	commonclient "github.com/hashicorp/packer-plugin-azure/builder/azure/common/client"
	"github.com/hashicorp/packer-plugin-sdk/acctest"
	packersdk "github.com/hashicorp/packer-plugin-sdk/packer"
	"github.com/hashicorp/packer-plugin-sdk/retry"
)

const DeviceLoginAcceptanceTest = "DEVICELOGIN_TEST"

// This test builds two images,
// First a parent Specialized ARM 64 Linux VM to a Shared Image Gallery/Compute Gallery
// Then a second Specialized ARM64 Linux VM that uses the first as its source/parent image
func TestBuilderAcc_SharedImageGallery_ARM64SpecializedLinuxSIG_WithChildImage(t *testing.T) {
	t.Parallel()

	if os.Getenv("PACKER_ACC") == "" {
		t.Skip("Skipping acceptance test as environment variable `PACKER_ACC` is not set")
		return
	}

	if os.Getenv("AZURE_CLI_AUTH") == "" {
		t.Fatalf("Azure CLI Acceptance tests require 'AZURE_CLI_AUTH' is set, and an active `az login` session has been established")
		return
	}

	if os.Getenv("ARM_SSH_PRIVATE_KEY_FILE") == "" {
		t.Fatalf("To run this test set a valid ssh private key location in ARM_SSH_PRIVATE_KEY_FILE")
		return
	}

	subscriptionID := os.Getenv("ARM_SUBSCRIPTION_ID")

	createSharedImageGalleryDefinition(t, CreateSharedImageGalleryDefinitionParameters{
		galleryImageName: "arm-linux-specialized-sig",
		imageSku:         "22_04-lts-arm64",
		subscriptionId:   subscriptionID,
		imageOffer:       "0001-com-ubuntu-server-jammy",
		imagePublisher:   "canonical",
		isX64:            false,
		isWindows:        false,
		useGenTwoVM:      true,
		specialized:      true,
	})

	defer deleteSharedImageGalleryDefinition(t, subscriptionID, "arm-linux-specialized-sig", []string{"1.0.0", "1.0.1"})
	// Create parent specialized shared gallery image
	acctest.TestPlugin(t, &acctest.PluginTestCase{
		Name:     "test-specialized-linux-sig",
		Type:     "azure-arm",
		Template: string(armLinuxSpecialziedSIGTemplate),
		Setup: func() error {
			createSharedImageGalleryDefinition(t, CreateSharedImageGalleryDefinitionParameters{
				galleryImageName: "arm-linux-specialized-sig",
				subscriptionId:   os.Getenv("ARM_SUBSCRIPTION_ID"),
				imageSku:         "22_04-lts-arm64",
				imageOffer:       "0001-com-ubuntu-server-jammy",
				imagePublisher:   "canonical",
				isX64:            false,
				isWindows:        false,
				useGenTwoVM:      true,
				specialized:      true,
			})
			return nil
		},
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
		Name:     "test-specialized-linux-sig-child",
		Type:     "azure-arm",
		Template: string(armLinuxChildFromSpecializedParent),
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
	if os.Getenv("PACKER_ACC") == "" {
		t.Skip("Skipping acceptance test as environment variable `PACKER_ACC` is not set")
		return
	}
	if os.Getenv("AZURE_CLI_AUTH") == "" {
		t.Fatalf("Azure CLI Acceptance tests require 'AZURE_CLI_AUTH' is set, and an active `az login` session has been established")
		return
	}

	subscriptionID := os.Getenv("ARM_SUBSCRIPTION_ID")
	createSharedImageGalleryDefinition(t, CreateSharedImageGalleryDefinitionParameters{
		galleryImageName: "windows-sig",
		imageSku:         "2012-R2-Datacenter",
		imageOffer:       "WindowsServer",
		imagePublisher:   "MicrosoftWindowsServer",
		isX64:            true,
		subscriptionId:   subscriptionID,
		isWindows:        true,
	})
	defer deleteSharedImageGalleryDefinition(t, subscriptionID, "windows-sig", []string{"1.0.0"})

	acctest.TestPlugin(t, &acctest.PluginTestCase{
		Name:     "test-windows-sig",
		Type:     "azure-arm",
		Template: string(windowsSIGTemplate),
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
	if os.Getenv("PACKER_ACC") == "" {
		t.Skip("Skipping acceptance test as environment variable `PACKER_ACC` is not set")
		return
	}
	if os.Getenv("AZURE_CLI_AUTH") == "" {
		t.Fatalf("Azure CLI Acceptance tests require 'AZURE_CLI_AUTH' is set, and an active `az login` session has been established")
		return
	}

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
	tmpfile, err := ioutil.TempFile("", "userdata")
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

type CreateSharedImageGalleryDefinitionParameters struct {
	galleryImageName string
	subscriptionId   string
	imageSku         string
	imageOffer       string
	imagePublisher   string
	isX64            bool
	isWindows        bool
	useGenTwoVM      bool
	specialized      bool
}

func createTestAzureClient(t *testing.T) AzureClient {
	b := Builder{}
	_, _, _ = b.Prepare()
	ui := testUi()
	// Use CLI auth for our test client
	b.config.ClientConfig.UseAzureCLIAuth = true
	_ = b.config.ClientConfig.FillParameters()
	authOptions := commonclient.AzureAuthOptions{
		AuthType:       b.config.ClientConfig.AuthType(),
		ClientID:       b.config.ClientConfig.ClientID,
		ClientSecret:   b.config.ClientConfig.ClientSecret,
		TenantID:       b.config.ClientConfig.TenantID,
		SubscriptionID: b.config.ClientConfig.SubscriptionID,
	}
	ui.Message("Creating test Azure Resource Manager (ARM) client ...")
	azureClient, err := NewAzureClient(
		context.TODO(),
		true,
		b.config.ClientConfig.CloudEnvironment(),
		b.config.SharedGalleryTimeout,
		b.config.PollingDurationTimeout,
		authOptions)
	if err != nil {
		t.Fatalf("failed to create test azure client: %s", err)
	}
	return *azureClient
}

func createSharedImageGalleryDefinition(t *testing.T, params CreateSharedImageGalleryDefinitionParameters) {
	azureClient := createTestAzureClient(t)
	osType := galleryimages.OperatingSystemTypesLinux
	if params.isWindows {
		osType = galleryimages.OperatingSystemTypesWindows
	}
	osState := galleryimages.OperatingSystemStateTypesGeneralized
	if params.specialized {
		osState = galleryimages.OperatingSystemStateTypesSpecialized
	}
	osArch := galleryimages.ArchitectureArmSixFour
	if params.isX64 {
		osArch = galleryimages.ArchitectureXSixFour
	}
	hyperVGeneration := galleryimages.HyperVGenerationVOne
	if params.useGenTwoVM {
		hyperVGeneration = galleryimages.HyperVGenerationVTwo
	}
	location := "southcentralus"
	galleryId := galleryimages.NewGalleryImageID(params.subscriptionId, "packer-acceptance-test", "acctestgallery", params.galleryImageName)
	_, err := azureClient.GalleryImagesClient.CreateOrUpdate(context.TODO(), galleryId, galleryimages.GalleryImage{
		Properties: &galleryimages.GalleryImageProperties{
			OsType:           osType,
			OsState:          osState,
			Architecture:     &osArch,
			HyperVGeneration: &hyperVGeneration,
			Identifier: galleryimages.GalleryImageIdentifier{
				Publisher: params.imagePublisher,
				Offer:     params.imageOffer,
				Sku:       params.imageSku,
			},
		},
		Location: location,
	})

	if err != nil {
		t.Fatalf("failed to create Gallery %s: %s", params.galleryImageName, err)
	}
}

func deleteSharedImageGalleryDefinition(t *testing.T, subscriptionID string, galleryImageName string, imageVersions []string) {
	azureClient := createTestAzureClient(t)
	for _, imageVersion := range imageVersions {
		// If we fail to delete a gallery version we should still try to delete other versions and the gallery
		// Its possible a build was canceled or failed mid test that would leave any of the builds incomplete
		// We still want to try and delete the Gallery to not leave behind orphaned resources to manually clean up
		id := galleryimageversions.NewImageVersionID(subscriptionID, "packer-acceptance-test", "acctestgallery", galleryImageName, imageVersion)
		err := azureClient.GalleryImageVersionsClient.DeleteThenPoll(context.TODO(), id)
		if err != nil {
			t.Logf("failed to delete Gallery Image Version %s:%s %s", galleryImageName, imageVersion, err)
		}
	}
	retryConfig := retry.Config{
		Tries:      5,
		RetryDelay: (&retry.Backoff{InitialBackoff: 2 * time.Second, MaxBackoff: 30 * time.Second, Multiplier: 2}).Linear,
	}
	err := retryConfig.Run(context.TODO(), func(ctx context.Context) error {
		id := galleryimages.NewGalleryImageID(subscriptionID, "packer-acceptance-test", "acctestgallery", galleryImageName)
		err := azureClient.GalleryImagesClient.DeleteThenPoll(context.TODO(), id)
		if err != nil {
			return err
		}
		return nil
	})
	if err != nil {
		t.Fatalf("failed to delete Gallery %s: %s", galleryImageName, err)
	}
}

func testUi() *packersdk.BasicUi {
	return &packersdk.BasicUi{
		Reader:      new(bytes.Buffer),
		Writer:      new(bytes.Buffer),
		ErrorWriter: new(bytes.Buffer),
	}
}

func testBuilderUserDataLinux(userdata string) string {
	return fmt.Sprintf(`
{
	"variables": {
	  "client_id": "{{env `+"`ARM_CLIENT_ID`"+`}}",
	  "client_secret": "{{env `+"`ARM_CLIENT_SECRET`"+`}}",
	  "subscription_id": "{{env `+"`ARM_SUBSCRIPTION_ID`"+`}}",
	  "storage_account": "{{env `+"`ARM_STORAGE_ACCOUNT`"+`}}"
	},
	"builders": [{
	  "type": "azure-arm",

	  "client_id": "{{user `+"`client_id`"+`}}",
	  "client_secret": "{{user `+"`client_secret`"+`}}",
	  "subscription_id": "{{user `+"`subscription_id`"+`}}",

	  "storage_account": "{{user `+"`storage_account`"+`}}",
	  "resource_group_name": "packer-acceptance-test",
	  "capture_container_name": "test",
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
	  "subscription_id": "{{env ` + "`ARM_SUBSCRIPTION_ID`" + `}}"
	},
	"builders": [{
	  "type": "azure-arm",

	  "client_id": "{{user ` + "`client_id`" + `}}",
	  "client_secret": "{{user ` + "`client_secret`" + `}}",
	  "subscription_id": "{{user ` + "`subscription_id`" + `}}",

	  "managed_image_resource_group_name": "packer-acceptance-test",
	  "managed_image_name": "testBuilderAccManagedDiskWindows-{{timestamp}}",

	  "os_type": "Windows",
	  "image_publisher": "MicrosoftWindowsServer",
	  "image_offer": "WindowsServer",
	  "image_sku": "2012-R2-Datacenter",

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
	  "subscription_id": "{{env ` + "`ARM_SUBSCRIPTION_ID`" + `}}"
	},
	"builders": [{
	  "type": "azure-arm",

	  "client_id": "{{user ` + "`client_id`" + `}}",
	  "client_secret": "{{user ` + "`client_secret`" + `}}",
	  "subscription_id": "{{user ` + "`subscription_id`" + `}}",

	  "build_resource_group_name" : "packer-acceptance-test",
	  "managed_image_resource_group_name": "packer-acceptance-test",
	  "managed_image_name": "testBuilderAccManagedDiskWindowsBuildResourceGroup-{{timestamp}}",

	  "os_type": "Windows",
	  "image_publisher": "MicrosoftWindowsServer",
	  "image_offer": "WindowsServer",
	  "image_sku": "2012-R2-Datacenter",

	  "communicator": "winrm",
	  "winrm_use_ssl": "true",
	  "winrm_insecure": "true",
	  "winrm_timeout": "3m",
	  "winrm_username": "packer",
	  "async_resourcegroup_delete": "true",

	  "vm_size": "Standard_DS2_v2"
	}]
}
`

const testBuilderAccManagedDiskWindowsBuildResourceGroupAdditionalDisk = `
{
	"variables": {
	  "client_id": "{{env ` + "`ARM_CLIENT_ID`" + `}}",
	  "client_secret": "{{env ` + "`ARM_CLIENT_SECRET`" + `}}",
	  "subscription_id": "{{env ` + "`ARM_SUBSCRIPTION_ID`" + `}}"
	},
	"builders": [{
	  "type": "azure-arm",

	  "client_id": "{{user ` + "`client_id`" + `}}",
	  "client_secret": "{{user ` + "`client_secret`" + `}}",
	  "subscription_id": "{{user ` + "`subscription_id`" + `}}",

	  "build_resource_group_name" : "packer-acceptance-test",
	  "managed_image_resource_group_name": "packer-acceptance-test",
	  "managed_image_name": "testBuilderAccManagedDiskWindowsBuildResourceGroupAdditionDisk-{{timestamp}}",

	  "os_type": "Windows",
	  "image_publisher": "MicrosoftWindowsServer",
	  "image_offer": "WindowsServer",
	  "image_sku": "2012-R2-Datacenter",

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
	  "subscription_id": "{{env ` + "`ARM_SUBSCRIPTION_ID`" + `}}"
	},
	"builders": [{
	  "type": "azure-arm",

	  "client_id": "{{user ` + "`client_id`" + `}}",
	  "client_secret": "{{user ` + "`client_secret`" + `}}",
	  "subscription_id": "{{user ` + "`subscription_id`" + `}}",

	  "managed_image_resource_group_name": "packer-acceptance-test",
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
	  "storage_account": "{{env ` + "`ARM_STORAGE_ACCOUNT`" + `}}"
	},
	"builders": [{
	  "type": "azure-arm",

	  "client_id": "{{user ` + "`client_id`" + `}}",
	  "client_secret": "{{user ` + "`client_secret`" + `}}",
	  "subscription_id": "{{user ` + "`subscription_id`" + `}}",

	  "storage_account": "{{user ` + "`storage_account`" + `}}",
	  "resource_group_name": "packer-acceptance-test",
	  "capture_container_name": "azure-arm",
	  "capture_name_prefix": "testBuilderAccBlobWin",

	  "os_type": "Windows",
	  "image_publisher": "MicrosoftWindowsServer",
	  "image_offer": "WindowsServer",
	  "image_sku": "2012-R2-Datacenter",

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
	  "client_secret": "{{env ` + "`ARM_CLIENT_SECRET`" + `}}",
	  "subscription_id": "{{env ` + "`ARM_SUBSCRIPTION_ID`" + `}}",
	  "storage_account": "{{env ` + "`ARM_STORAGE_ACCOUNT`" + `}}"
	},
	"builders": [{
	  "type": "azure-arm",

	  "client_id": "{{user ` + "`client_id`" + `}}",
	  "client_secret": "{{user ` + "`client_secret`" + `}}",
	  "subscription_id": "{{user ` + "`subscription_id`" + `}}",

	  "storage_account": "{{user ` + "`storage_account`" + `}}",
	  "resource_group_name": "packer-acceptance-test",
	  "capture_container_name": "test",
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
	"builders": [{
	  "type": "azure-arm",

	  "use_azure_cli_auth": true,

	  "managed_image_resource_group_name": "packer-acceptance-test",
	  "managed_image_name": "testBuilderAccManagedDiskLinuxAzureCLI-{{timestamp}}",
	  "temp_resource_group_name": "packer-acceptance-test-managed-cli",

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
