// Copyright (c) HashiCorp, Inc.
// SPDX-License-Identifier: MPL-2.0

package arm

import (
	"context"
	"fmt"
	"testing"

	"github.com/hashicorp/go-azure-sdk/resource-manager/compute/2022-03-01/images"
	"github.com/hashicorp/go-azure-sdk/resource-manager/compute/2022-03-01/virtualmachines"
	"github.com/hashicorp/packer-plugin-azure/builder/azure/common/constants"
	"github.com/hashicorp/packer-plugin-sdk/multistep"
)

func TestStepCaptureImageShouldFailIfCaptureFails(t *testing.T) {
	var testSubject = &StepCaptureImage{
		captureVhd: func(context.Context, virtualmachines.VirtualMachineId, *virtualmachines.VirtualMachineCaptureParameters) error {
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

func TestStepCaptureImageShouldFailIfGetVMIDFails(t *testing.T) {
	var testSubject = &StepCaptureImage{
		captureVhd: func(context.Context, virtualmachines.VirtualMachineId, *virtualmachines.VirtualMachineCaptureParameters) error {
			return nil
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
func TestStepCaptureImageShouldPassIfCapturePasses(t *testing.T) {
	var testSubject = &StepCaptureImage{
		captureVhd: func(ctx context.Context, vmId virtualmachines.VirtualMachineId, parameters *virtualmachines.VirtualMachineCaptureParameters) error {
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

func TestStepCaptureImageShouldCallGeneralizeIfSpecializedIsFalse(t *testing.T) {
	generalizeCount := 0
	var testSubject = &StepCaptureImage{
		captureVhd: func(context.Context, virtualmachines.VirtualMachineId, *virtualmachines.VirtualMachineCaptureParameters) error {
			return nil
		},
		generalizeVM: func(context.Context, virtualmachines.VirtualMachineId) error {
			generalizeCount++
			return nil
		},
		say:   func(message string) {},
		error: func(e error) {},
	}

	stateBag := createTestStateBagStepCaptureImage()
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
		captureVhd: func(context.Context, virtualmachines.VirtualMachineId, *virtualmachines.VirtualMachineCaptureParameters) error {
			return nil
		},
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
	var actualComputeName string
	var actualVirtualMachineCaptureParameters *virtualmachines.VirtualMachineCaptureParameters
	expectedVirtualMachineID := "id"
	var testSubject = &StepCaptureImage{
		captureVhd: func(ctx context.Context, id virtualmachines.VirtualMachineId, parameters *virtualmachines.VirtualMachineCaptureParameters) error {
			actualResourceGroupName = id.ResourceGroupName
			actualComputeName = id.VirtualMachineName
			actualVirtualMachineCaptureParameters = parameters

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

	var expectedComputeName = stateBag.Get(constants.ArmComputeName).(string)
	var expectedResourceGroupName = stateBag.Get(constants.ArmResourceGroupName).(string)
	var expectedVirtualMachineCaptureParameters = stateBag.Get(constants.ArmVirtualMachineCaptureParameters).(*virtualmachines.VirtualMachineCaptureParameters)

	actualVirtualMachineID := stateBag.Get(constants.ArmBuildVMInternalId).(string)
	if actualVirtualMachineID != expectedVirtualMachineID {
		t.Fatalf("Expected StepCaptureImage to set 'constants.ArmBuildVMInternalId' to the state bag to %s, but it was set to %s.", expectedVirtualMachineID, actualVirtualMachineID)
	}
	if actualComputeName != expectedComputeName {
		t.Fatal("Expected StepCaptureImage to source 'constants.ArmComputeName' from the state bag, but it did not.")
	}

	if actualResourceGroupName != expectedResourceGroupName {
		t.Fatal("Expected StepCaptureImage to source 'constants.ArmResourceGroupName' from the state bag, but it did not.")
	}

	if actualVirtualMachineCaptureParameters != expectedVirtualMachineCaptureParameters {
		t.Fatal("Expected StepCaptureImage to source 'constants.ArmVirtualMachineCaptureParameters' from the state bag, but it did not.")
	}

}

func createTestStateBagStepCaptureImage() multistep.StateBag {
	stateBag := new(multistep.BasicStateBag)

	stateBag.Put(constants.ArmLocation, "localhost")
	stateBag.Put(constants.ArmComputeName, "Unit Test: ComputeName")
	stateBag.Put(constants.ArmResourceGroupName, "Unit Test: ResourceGroupName")
	stateBag.Put(constants.ArmSubscription, "Unit Test: SubscriptionId")
	stateBag.Put(constants.ArmVirtualMachineCaptureParameters, &virtualmachines.VirtualMachineCaptureParameters{})

	stateBag.Put(constants.ArmIsManagedImage, false)
	stateBag.Put(constants.ArmManagedImageResourceGroupName, "")
	stateBag.Put(constants.ArmManagedImageName, "")
	stateBag.Put(constants.ArmImageParameters, &images.Image{})
	stateBag.Put(constants.ArmIsSIGImage, false)
	stateBag.Put(constants.ArmSharedImageGalleryDestinationSpecialized, false)

	return stateBag
}
