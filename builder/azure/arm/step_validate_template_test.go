// Copyright (c) HashiCorp, Inc.
// SPDX-License-Identifier: MPL-2.0

package arm

import (
	"context"
	"fmt"
	"testing"

	"github.com/hashicorp/packer-plugin-azure/builder/azure/common/constants"
	"github.com/hashicorp/packer-plugin-sdk/multistep"
)

func TestStepValidateTemplateShouldFailIfValidateFails(t *testing.T) {
	var testSubject = &StepValidateTemplate{
		validate: func(context.Context, string, string, string) error { return fmt.Errorf("!! Unit Test FAIL !!") },
		say:      func(message string) {},
		error:    func(e error) {},
	}

	stateBag := createTestStateBagStepValidateTemplate()

	var result = testSubject.Run(context.Background(), stateBag)
	if result != multistep.ActionHalt {
		t.Fatalf("Expected the step to return 'ActionHalt', but got '%d'.", result)
	}

	if _, ok := stateBag.GetOk(constants.Error); ok == false {
		t.Fatalf("Expected the step to set stateBag['%s'], but it was not.", constants.Error)
	}
}

func TestStepValidateTemplateShouldPassIfValidatePasses(t *testing.T) {
	var testSubject = &StepValidateTemplate{
		validate: func(context.Context, string, string, string) error { return nil },
		say:      func(message string) {},
		error:    func(e error) {},
	}

	stateBag := createTestStateBagStepValidateTemplate()

	var result = testSubject.Run(context.Background(), stateBag)
	if result != multistep.ActionContinue {
		t.Fatalf("Expected the step to return 'ActionContinue', but got '%d'.", result)
	}

	if _, ok := stateBag.GetOk(constants.Error); ok == true {
		t.Fatalf("Expected the step to not set stateBag['%s'], but it was.", constants.Error)
	}
}

func TestStepValidateTemplateShouldTakeResourceGroupNameArgumentFromStateBag(t *testing.T) {
	var actualResourceGroupName string
	var actualSubscriptionId string

	var testSubject = &StepValidateTemplate{
		validate: func(ctx context.Context, subscriptionId string, resourceGroupName string, deploymentName string) error {
			actualResourceGroupName = resourceGroupName
			actualSubscriptionId = subscriptionId
			return nil
		},
		say:   func(message string) {},
		error: func(e error) {},
	}

	stateBag := createTestStateBagStepValidateTemplate()
	var result = testSubject.Run(context.Background(), stateBag)

	if result != multistep.ActionContinue {
		t.Fatalf("Expected the step to return 'ActionContinue', but got '%d'.", result)
	}

	var expectedResourceGroupName = stateBag.Get(constants.ArmResourceGroupName).(string)
	var expectedSubscriptionId = stateBag.Get(constants.ArmSubscription).(string)

	if actualResourceGroupName != expectedResourceGroupName {
		t.Fatal("Expected the step to source 'constants.ArmResourceGroupName' from the state bag, but it did not.")
	}

	if actualSubscriptionId != expectedSubscriptionId {
		t.Fatal("Expected the step to source 'constants.ArmSubscription' from the state bag, but it did not.")
	}
}

func TestStepValidateTemplateShouldTakeDeploymentNameArgumentFromParam(t *testing.T) {
	var actualDeploymentName string
	var expectedDeploymentName = "Unit Test: DeploymentName"

	stateBag := createTestStateBagStepValidateTemplate()
	var testSubject = &StepValidateTemplate{
		validate: func(ctx context.Context, subscriptionId string, resourceGroupName string, deploymentName string) error {
			actualDeploymentName = deploymentName

			return nil
		},
		say:   func(message string) {},
		error: func(e error) {},
		name:  expectedDeploymentName,
	}

	var result = testSubject.Run(context.Background(), stateBag)

	if result != multistep.ActionContinue {
		t.Fatalf("Expected the step to return 'ActionContinue', but got '%d'.", result)
	}

	if actualDeploymentName != expectedDeploymentName {
		t.Fatal("Expected the step to source 'deploymentName' from parameter, but it did not.")
	}
}

func createTestStateBagStepValidateTemplate() multistep.StateBag {
	stateBag := new(multistep.BasicStateBag)

	stateBag.Put(constants.ArmDeploymentName, "Unit Test: DeploymentName")
	stateBag.Put(constants.ArmResourceGroupName, "Unit Test: ResourceGroupName")
	stateBag.Put(constants.ArmSubscription, "Unit Test: Subscription")

	return stateBag
}
