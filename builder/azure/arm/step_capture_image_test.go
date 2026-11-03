// Copyright IBM Corp. 2013, 2026
// SPDX-License-Identifier: MPL-2.0

package arm

import (
	"context"
	"fmt"
	"testing"

	"github.com/hashicorp/go-azure-sdk/resource-manager/compute/2022-03-01/images"
	"github.com/hashicorp/go-azure-sdk/resource-manager/compute/2022-03-01/virtualmachines"
	"github.com/hashicorp/packer-plugin-azure/builder/azure/common"
	"github.com/hashicorp/packer-plugin-azure/builder/azure/common/constants"
	"github.com/hashicorp/packer-plugin-sdk/multistep"
)

func TestStepCaptureImageShouldFailIfGetVMIDFails(t *testing.T) {
	var testSubject = &StepCaptureImage{
		config: &Config{
			tmpOSDiskName:        "tmpOSDiskName",
			CaptureContainerName: "CaptureContainerName",
			CaptureNamePrefix:    "CaptureNamePrefix",
		},
		getVMInternalID: func(context.Context, virtualmachines.VirtualMachineId) (string, error) {
			return "", fmt.Errorf("!! Unit Test FAIL !!")
		},
		generalizeVM: func(context.Context, virtualmachines.VirtualMachineId) error {
			return nil
		},
		say:   func(message string) {},
		error: func(e error) {},
	}

	stateBag := createTestStateBagStepCaptureImage()

	var result = testSubject.Run(context.Background(), stateBag)
	if result != multistep.ActionHalt {
		t.Fatalf("Expected the step to return 'ActionHalt', but got '%d'.", result)
	}

	if _, ok := stateBag.GetOk(constants.Error); ok == false {
		t.Fatalf("Expected the step to set stateBag['%s'], but it was not.", constants.Error)
	}
}

func TestStepCaptureImageShouldFailIfGrantAccessFails(t *testing.T) {
	var testSubject = &StepCaptureImage{
		config: &Config{
			tmpOSDiskName:        "tmpOSDiskName",
			CaptureContainerName: "CaptureContainerName",
			CaptureNamePrefix:    "CaptureNamePrefix",
		},
		grantAccess: func(ctx context.Context, subscriptionId string, resourceGroupName string, osDiskName string) (string, error) {
			return "", fmt.Errorf("!! Unit Test FAIL !!")
		},
		getVMInternalID: func(context.Context, virtualmachines.VirtualMachineId) (string, error) {
			return "id", nil
		},
		generalizeVM: func(context.Context, virtualmachines.VirtualMachineId) error {
			return nil
		},
		say:   func(message string) {},
		error: func(e error) {},
	}

	stateBag := createTestStateBagStepCaptureImage()

	var result = testSubject.Run(context.Background(), stateBag)
	if result != multistep.ActionHalt {
		t.Fatalf("Expected the step to return 'ActionHalt', but got '%d'.", result)
	}

	if _, ok := stateBag.GetOk(constants.Error); ok == false {
		t.Fatalf("Expected the step to set stateBag['%s'], but it was not.", constants.Error)
	}
}

func TestStepCaptureImageShouldFailIfCopyToStorageFails(t *testing.T) {
	var testSubject = &StepCaptureImage{
		config: &Config{
			tmpOSDiskName:        "tmpOSDiskName",
			CaptureContainerName: "CaptureContainerName",
			CaptureNamePrefix:    "CaptureNamePrefix",
		},
		grantAccess: func(ctx context.Context, subscriptionId string, resourceGroupName string, osDiskName string) (string, error) {
			return "accessuri", nil
		},
		copyToStorage: func(ctx context.Context, storageContainerName string, captureNamePrefix string, osDiskName string, accessUri string) error {
			return fmt.Errorf("!! Unit Test FAIL !!")
		},
		getVMInternalID: func(context.Context, virtualmachines.VirtualMachineId) (string, error) {
			return "id", nil
		},
		generalizeVM: func(context.Context, virtualmachines.VirtualMachineId) error {
			return nil
		},
		say:   func(message string) {},
		error: func(e error) {},
	}

	stateBag := createTestStateBagStepCaptureImage()

	var result = testSubject.Run(context.Background(), stateBag)
	if result != multistep.ActionHalt {
		t.Fatalf("Expected the step to return 'ActionHalt', but got '%d'.", result)
	}

	if _, ok := stateBag.GetOk(constants.Error); ok == false {
		t.Fatalf("Expected the step to set stateBag['%s'], but it was not.", constants.Error)
	}
}

func TestStepCaptureImageShouldRevokeAccessOnCleanup(t *testing.T) {
	revokeAccessCalled := false
	revokeAccessPtr := &revokeAccessCalled
	var testSubject = &StepCaptureImage{
		config: &Config{
			CaptureContainerName: "CaptureContainerName",
			CaptureNamePrefix:    "CaptureNamePrefix",
		},
		revokeAccess: func(ctx context.Context, subscriptionId string, resourceGroupName string, osDiskName string) error {
			revokeAccessPtr = common.BoolPtr(true)
			return nil
		},
		say:   func(message string) {},
		error: func(e error) {},
	}

	stateBag := createTestStateBagStepCaptureImage()

	testSubject.Cleanup(stateBag)
	if *revokeAccessPtr {
		t.Fatal("Revoke Access should not be called on TestCaptureImage.Cleanup when no disk names are tracked, but was")
	}
	testSubject.diskNamesToRevokeAccessTo = []string{"tmpOSDiskName"}
	testSubject.Cleanup(stateBag)
	if !*revokeAccessPtr {
		t.Fatal("Revoke Access should be called on TestCaptureImage.Cleanup when disk names are tracked, but wasn't")
	}
}

func TestStepCaptureImageShouldPassIfCapturePasses(t *testing.T) {
	var testSubject = &StepCaptureImage{
		config: &Config{
			tmpOSDiskName:        "tmpOSDiskName",
			CaptureContainerName: "CaptureContainerName",
			CaptureNamePrefix:    "CaptureNamePrefix",
		},
		grantAccess: func(ctx context.Context, subscriptionId string, resourceGroupName string, osDiskName string) (string, error) {
			return "accessuri", nil
		},
		copyToStorage: func(ctx context.Context, storageContainerName string, captureNamePrefix string, osDiskName string, accessUri string) error {
			return nil
		},
		revokeAccess: func(ctx context.Context, subscriptionId string, resourceGroupName string, osDiskName string) error {
			return nil
		},
		generalizeVM: func(context.Context, virtualmachines.VirtualMachineId) error {
			return nil
		},
		getVMInternalID: func(ctx context.Context, vmId virtualmachines.VirtualMachineId) (string, error) {
			return "id", nil
		},
		say:   func(message string) {},
		error: func(e error) {},
	}

	stateBag := createTestStateBagStepCaptureImage()
	stateBag.Put(constants.ArmIsSIGImage, true)
	var result = testSubject.Run(context.Background(), stateBag)
	if result != multistep.ActionContinue {
		t.Fatalf("Expected the step to return 'ActionContinue', but got '%d'.", result)
	}

	if _, ok := stateBag.GetOk(constants.Error); ok == true {
		t.Fatalf("Expected the step to not set stateBag['%s'], but it was.", constants.Error)
	}
}

func TestStepCaptureImageUsesManagedImageStorageAccountType(t *testing.T) {
	storageAccountTypes := []virtualmachines.StorageAccountTypes{
		virtualmachines.StorageAccountTypesStandardLRS,
		virtualmachines.StorageAccountTypesStandardSSDLRS,
		virtualmachines.StorageAccountTypesPremiumLRS,
	}

	for _, osDiskStorageAccountType := range storageAccountTypes {
		for _, managedImageStorageAccountType := range storageAccountTypes {
			t.Run(fmt.Sprintf("OS disk %s, managed image %s", osDiskStorageAccountType, managedImageStorageAccountType), func(t *testing.T) {
				var capturedStorageAccountType *images.StorageAccountTypes
				testSubject := &StepCaptureImage{
					config: &Config{
						managedImageStorageAccountType: managedImageStorageAccountType,
						osDiskStorageAccountType:      osDiskStorageAccountType,
					},
					generalizeVM: func(context.Context, virtualmachines.VirtualMachineId) error {
						return nil
					},
					captureManagedImage: func(_ context.Context, _, _, _ string, parameters *images.Image) error {
						capturedStorageAccountType = parameters.Properties.StorageProfile.OsDisk.StorageAccountType
						return nil
					},
					say:   func(string) {},
					error: func(error) {},
				}

				stateBag := createTestStateBagStepCaptureImage()
				stateBag.Put(constants.ArmIsManagedImage, true)
				stateBag.Put(constants.ArmIsVHDSaveToStorage, false)
				stateBag.Put(constants.ArmManagedImageResourceGroupName, "image-resource-group")
				stateBag.Put(constants.ArmManagedImageName, "image-name")
				stateBag.Put(constants.ArmImageParameters, &images.Image{
					Properties: &images.ImageProperties{
						StorageProfile: &images.ImageStorageProfile{},
					},
				})

				if result := testSubject.Run(context.Background(), stateBag); result != multistep.ActionContinue {
					t.Fatalf("expected the step to return ActionContinue, got %d", result)
				}
			if capturedStorageAccountType == nil {
				t.Fatal("expected managed image capture to set the OS disk storage account type")
			}
			if got := *capturedStorageAccountType; got != images.StorageAccountTypes(managedImageStorageAccountType) {
				t.Fatalf("expected managed image storage account type %s, got %s", managedImageStorageAccountType, got)
			}
			})
		}
	}
}

func TestStepCaptureImageShouldCallGeneralizeIfSpecializedIsFalse(t *testing.T) {
	generalizeCount := 0
	var testSubject = &StepCaptureImage{
		generalizeVM: func(context.Context, virtualmachines.VirtualMachineId) error {
			generalizeCount++
			return nil
		},
		say:   func(message string) {},
		error: func(e error) {},
	}

	stateBag := createTestStateBagStepCaptureImage()
	stateBag.Put(constants.ArmIsManagedImage, false)
	stateBag.Put(constants.ArmIsVHDSaveToStorage, false)
	stateBag.Put(constants.ArmIsSIGImage, true)
	stateBag.Put(constants.ArmSharedImageGalleryDestinationSpecialized, false)
	var result = testSubject.Run(context.Background(), stateBag)
	if result != multistep.ActionContinue {
		t.Fatalf("Expected the step to return 'ActionContinue', but got '%d'.", result)
	}

	if _, ok := stateBag.GetOk(constants.Error); ok == true {
		t.Fatalf("Expected the step to not set stateBag['%s'], but it was.", constants.Error)
	}
	if generalizeCount != 1 {
		t.Fatalf("Expected generalize to be called 1, was called %d times", generalizeCount)
	}
}

func TestStepCaptureImageShouldNotCallGeneralizeIfSpecializedIsTrue(t *testing.T) {
	generalizeCount := 0
	var testSubject = &StepCaptureImage{
		generalizeVM: func(context.Context, virtualmachines.VirtualMachineId) error {
			generalizeCount++
			return nil
		},
		say:   func(message string) {},
		error: func(e error) {},
	}

	stateBag := createTestStateBagStepCaptureImage()
	stateBag.Put(constants.ArmSharedImageGalleryDestinationSpecialized, true)
	var result = testSubject.Run(context.Background(), stateBag)
	if result != multistep.ActionContinue {
		t.Fatalf("Expected the step to return 'ActionContinue', but got '%d'.", result)
	}

	if _, ok := stateBag.GetOk(constants.Error); ok == true {
		t.Fatalf("Expected the step to not set stateBag['%s'], but it was.", constants.Error)
	}
	if generalizeCount != 0 {
		t.Fatalf("Expected generalize to not be called, was called %d times", generalizeCount)
	}
}

func TestStepCaptureImageShouldTakeStepArgumentsFromStateBag(t *testing.T) {
	cancelCh := make(chan<- struct{})
	defer close(cancelCh)

	var actualResourceGroupName string
	expectedVirtualMachineID := "id"
	var testSubject = &StepCaptureImage{
		config: &Config{
			tmpOSDiskName:        "tmpOSDiskName",
			CaptureContainerName: "CaptureContainerName",
			CaptureNamePrefix:    "CaptureNamePrefix",
		},
		grantAccess: func(ctx context.Context, subscriptionId string, resourceGroupName string, osDiskName string) (string, error) {
			actualResourceGroupName = resourceGroupName

			return "accessuri", nil
		},
		copyToStorage: func(ctx context.Context, storageContainerName string, captureNamePrefix string, osDiskName string, accessUri string) error {
			return nil
		},
		revokeAccess: func(ctx context.Context, subscriptionId string, resourceGroupName string, osDiskName string) error {
			return nil
		},
		getVMInternalID: func(ctx context.Context, vmId virtualmachines.VirtualMachineId) (string, error) {
			return expectedVirtualMachineID, nil
		},
		generalizeVM: func(context.Context, virtualmachines.VirtualMachineId) error {
			return nil
		},
		say:   func(message string) {},
		error: func(e error) {},
	}

	stateBag := createTestStateBagStepCaptureImage()
	var result = testSubject.Run(context.Background(), stateBag)

	if result != multistep.ActionContinue {
		t.Fatalf("Expected the step to return 'ActionContinue', but got '%d'.", result)
	}

	var expectedResourceGroupName = stateBag.Get(constants.ArmResourceGroupName).(string)

	actualVirtualMachineID := stateBag.Get(constants.ArmBuildVMInternalId).(string)
	if actualVirtualMachineID != expectedVirtualMachineID {
		t.Fatalf("Expected StepCaptureImage to set 'constants.ArmBuildVMInternalId' to the state bag to %s, but it was set to %s.", expectedVirtualMachineID, actualVirtualMachineID)
	}

	if actualResourceGroupName != expectedResourceGroupName {
		t.Fatal("Expected StepCaptureImage to source 'constants.ArmResourceGroupName' from the state bag, but it did not.")
	}
}

func createTestStateBagStepCaptureImage() multistep.StateBag {
	stateBag := new(multistep.BasicStateBag)

	stateBag.Put(constants.ArmLocation, "localhost")
	stateBag.Put(constants.ArmComputeName, "Unit Test: ComputeName")
	stateBag.Put(constants.ArmResourceGroupName, "Unit Test: ResourceGroupName")
	stateBag.Put(constants.ArmSubscription, "Unit Test: SubscriptionId")
	stateBag.Put(constants.ArmVirtualMachineCaptureParameters, &virtualmachines.VirtualMachineCaptureParameters{})

	stateBag.Put(constants.ArmIsVHDSaveToStorage, true)
	stateBag.Put(constants.ArmIsManagedImage, false)
	stateBag.Put(constants.ArmManagedImageResourceGroupName, "")
	stateBag.Put(constants.ArmManagedImageName, "")
	stateBag.Put(constants.ArmImageParameters, &images.Image{})
	stateBag.Put(constants.ArmIsSIGImage, false)
	stateBag.Put(constants.ArmSharedImageGalleryDestinationSpecialized, false)

	return stateBag
}
