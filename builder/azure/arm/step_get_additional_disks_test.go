// Copyright (c) HashiCorp, Inc.
// SPDX-License-Identifier: MPL-2.0

package arm

import (
	"context"
	"fmt"
	"testing"

	"github.com/hashicorp/go-azure-sdk/resource-manager/compute/2022-03-01/virtualmachines"
	"github.com/hashicorp/packer-plugin-azure/builder/azure/common/constants"

	"github.com/hashicorp/packer-plugin-sdk/multistep"
)

func TestStepGetAdditionalDiskShouldFailIfGetFails(t *testing.T) {
	var testSubject = &StepGetDataDisk{
		query: func(context.Context, string, string, string) (*virtualmachines.VirtualMachine, error) {
			return createVirtualMachineWithDataDisksFromUri("test.vhd"), fmt.Errorf("!! Unit Test FAIL !!")
		},
		say:   func(message string) {},
		error: func(e error) {},
	}

	stateBag := createTestStateBagStepGetAdditionalDisks()

	var result = testSubject.Run(context.Background(), stateBag)
	if result != multistep.ActionHalt {
		t.Fatalf("Expected the step to return 'ActionHalt', but got '%d'.", result)
	}

	if _, ok := stateBag.GetOk(constants.Error); ok == false {
		t.Fatalf("Expected the step to set stateBag['%s'], but it was not.", constants.Error)
	}
}

func TestStepGetAdditionalDiskShouldPassIfGetPasses(t *testing.T) {
	var testSubject = &StepGetDataDisk{
		query: func(context.Context, string, string, string) (*virtualmachines.VirtualMachine, error) {
			return createVirtualMachineWithDataDisksFromUri("test.vhd"), nil
		},
		say:   func(message string) {},
		error: func(e error) {},
	}

	stateBag := createTestStateBagStepGetAdditionalDisks()

	var result = testSubject.Run(context.Background(), stateBag)
	if result != multistep.ActionContinue {
		t.Fatalf("Expected the step to return 'ActionContinue', but got '%d'.", result)
	}

	if _, ok := stateBag.GetOk(constants.Error); ok == true {
		t.Fatalf("Expected the step to not set stateBag['%s'], but it was.", constants.Error)
	}
}

func TestStepGetAdditionalDiskShouldTakeValidateArgumentsFromStateBag(t *testing.T) {
	var actualResourceGroupName string
	var actualComputeName string
	var actualSubscriptionId string
	var testSubject = &StepGetDataDisk{
		query: func(ctx context.Context, subscriptionId string, resourceGroupName string, computeName string) (*virtualmachines.VirtualMachine, error) {
			actualResourceGroupName = resourceGroupName
			actualComputeName = computeName
			actualSubscriptionId = subscriptionId
			return createVirtualMachineWithDataDisksFromUri("test.vhd"), nil
		},
		say:   func(message string) {},
		error: func(e error) {},
	}

	stateBag := createTestStateBagStepGetAdditionalDisks()
	var result = testSubject.Run(context.Background(), stateBag)

	if result != multistep.ActionContinue {
		t.Fatalf("Expected the step to return 'ActionContinue', but got '%d'.", result)
	}

	var expectedComputeName = stateBag.Get(constants.ArmComputeName).(string)
	var expectedResourceGroupName = stateBag.Get(constants.ArmResourceGroupName).(string)

	var expectedSubscriptionId = stateBag.Get(constants.ArmSubscription).(string)

	if actualComputeName != expectedComputeName {
		t.Fatal("Expected the step to source 'constants.ArmResourceGroupName' from the state bag, but it did not.")
	}

	if actualResourceGroupName != expectedResourceGroupName {
		t.Fatal("Expected the step to source 'constants.ArmResourceGroupName' from the state bag, but it did not.")
	}

	if actualSubscriptionId != expectedSubscriptionId {
		t.Fatalf("Expected the step to source 'constants.ArmSubscriptionId' from the state bag, but it did not. %s %s", actualSubscriptionId, expectedSubscriptionId)
	}
	expectedAdditionalDiskVhds, ok := stateBag.GetOk(constants.ArmAdditionalDiskVhds)
	if !ok {
		t.Fatalf("Expected the state bag to have a value for '%s', but it did not.", constants.ArmAdditionalDiskVhds)
	}

	expectedAdditionalDiskVhd := expectedAdditionalDiskVhds.([]string)
	if expectedAdditionalDiskVhd[0] != "test.vhd" {
		t.Fatalf("Expected the value of stateBag[%s] to be 'test.vhd', but got '%s'.", constants.ArmAdditionalDiskVhds, expectedAdditionalDiskVhd[0])
	}
}

func createTestStateBagStepGetAdditionalDisks() multistep.StateBag {
	stateBag := new(multistep.BasicStateBag)

	stateBag.Put(constants.ArmComputeName, "Unit Test: ComputeName")
	stateBag.Put(constants.ArmResourceGroupName, "Unit Test: ResourceGroupName")
	stateBag.Put(constants.ArmSubscription, "Unit Test: Subscription")

	return stateBag
}

func createVirtualMachineWithDataDisksFromUri(vhdUri string) *virtualmachines.VirtualMachine {
	vm := virtualmachines.VirtualMachine{
		Properties: &virtualmachines.VirtualMachineProperties{
			StorageProfile: &virtualmachines.StorageProfile{
				OsDisk: &virtualmachines.OSDisk{
					Vhd: &virtualmachines.VirtualHardDisk{
						Uri: &vhdUri,
					},
				},
				DataDisks: &[]virtualmachines.DataDisk{
					{
						Vhd: &virtualmachines.VirtualHardDisk{
							Uri: &vhdUri,
						},
					},
				},
			},
		},
	}

	return &vm
}
