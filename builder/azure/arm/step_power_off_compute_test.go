// Copyright IBM Corp. 2013, 2025
// SPDX-License-Identifier: MPL-2.0

package arm

import (
	"context"
	"fmt"
	"testing"

	"github.com/hashicorp/packer-plugin-azure/builder/azure/common/constants"
	"github.com/hashicorp/packer-plugin-sdk/multistep"
)

func TestStepPowerOffComputeShouldFailIfPowerOffFails(t *testing.T) {
	var testSubject = &StepPowerOffCompute{
		powerOff: func(context.Context, string, string, string) error { return fmt.Errorf("!! Unit Test FAIL !!") },
		say:      func(message string) {},
		error:    func(e error) {},
	}

	stateBag := createTestStateBagStepPowerOffCompute()

	var result = testSubject.Run(context.Background(), stateBag)
	if result != multistep.ActionHalt {
		t.Fatalf("Expected the step to return 'ActionHalt', but got '%d'.", result)
	}

	if _, ok := stateBag.GetOk(constants.Error); ok == false {
		t.Fatalf("Expected the step to set stateBag['%s'], but it was not.", constants.Error)
	}
}

func TestStepPowerOffComputeShouldPassIfPowerOffPasses(t *testing.T) {
	var testSubject = &StepPowerOffCompute{
		powerOff: func(context.Context, string, string, string) error { return nil },
		say:      func(message string) {},
		error:    func(e error) {},
	}

	stateBag := createTestStateBagStepPowerOffCompute()

	var result = testSubject.Run(context.Background(), stateBag)
	if result != multistep.ActionContinue {
		t.Fatalf("Expected the step to return 'ActionContinue', but got '%d'.", result)
	}

	if _, ok := stateBag.GetOk(constants.Error); ok == true {
		t.Fatalf("Expected the step to not set stateBag['%s'], but it was.", constants.Error)
	}
}

func TestStepPowerOffComputeShouldTakeStepArgumentsFromStateBag(t *testing.T) {
	var actualResourceGroupName string
	var actualComputeName string
	var actualSubscriptionId string

	var testSubject = &StepPowerOffCompute{
		powerOff: func(ctx context.Context, subscriptionId string, resourceGroupName string, computeName string) error {
			actualResourceGroupName = resourceGroupName
			actualComputeName = computeName
			actualSubscriptionId = subscriptionId

			return nil
		},
		say:   func(message string) {},
		error: func(e error) {},
	}

	stateBag := createTestStateBagStepPowerOffCompute()
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
		t.Fatal("Expected the step to source 'constants.ArmSubscription' from the state bag, but it did not.")
	}
}

func createTestStateBagStepPowerOffCompute() multistep.StateBag {
	stateBag := new(multistep.BasicStateBag)

	stateBag.Put(constants.ArmComputeName, "Unit Test: ComputeName")
	stateBag.Put(constants.ArmResourceGroupName, "Unit Test: ResourceGroupName")
	stateBag.Put(constants.ArmSubscription, "UnitTest: SubscriptionId")
	return stateBag
}
