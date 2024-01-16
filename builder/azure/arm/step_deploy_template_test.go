// Copyright (c) HashiCorp, Inc.
// SPDX-License-Identifier: MPL-2.0

package arm

import (
	"context"
	"fmt"
	"testing"
	"time"

	"github.com/hashicorp/packer-plugin-azure/builder/azure/common/constants"
	"github.com/hashicorp/packer-plugin-sdk/multistep"
	packersdk "github.com/hashicorp/packer-plugin-sdk/packer"
)

func TestStepDeployTemplateShouldFailIfDeployFails(t *testing.T) {
	var testSubject = &StepDeployTemplate{
		deploy: func(context.Context, string, string, string) error {
			return fmt.Errorf("!! Unit Test FAIL !!")
		},
		say:   func(message string) {},
		error: func(e error) {},
	}

	stateBag := createTestStateBagStepDeployTemplate()

	var result = testSubject.Run(context.Background(), stateBag)

	if result != multistep.ActionHalt {
		t.Fatalf("Expected the step to return 'ActionHalt', but got '%d'.", result)
	}

	if _, ok := stateBag.GetOk(constants.Error); ok == false {
		t.Fatalf("Expected the step to set stateBag['%s'], but it was not.", constants.Error)
	}
}

func TestStepDeployTemplateShouldPassIfDeployPasses(t *testing.T) {
	var testSubject = &StepDeployTemplate{
		deploy: func(context.Context, string, string, string) error { return nil },
		say:    func(message string) {},
		error:  func(e error) {},
	}

	stateBag := createTestStateBagStepDeployTemplate()

	var result = testSubject.Run(context.Background(), stateBag)
	if result != multistep.ActionContinue {
		t.Fatalf("Expected the step to return 'ActionContinue', but got '%d'.", result)
	}

	if _, ok := stateBag.GetOk(constants.Error); ok == true {
		t.Fatalf("Expected the step to not set stateBag['%s'], but it was.", constants.Error)
	}
}

func TestStepDeployTemplateShouldTakeStepArgumentsFromStateBag(t *testing.T) {
	var actualResourceGroupName string
	var actualDeploymentName string
	var actualSubscriptionId string

	var testSubject = &StepDeployTemplate{
		deploy: func(ctx context.Context, subscriptionId string, resourceGroupName string, deploymentName string) error {
			actualResourceGroupName = resourceGroupName
			actualDeploymentName = deploymentName
			actualSubscriptionId = subscriptionId
			return nil
		},
		say:          func(message string) {},
		error:        func(e error) {},
		name:         "--deployment-name--",
		templateType: VirtualMachineTemplate,
	}

	stateBag := createTestStateBagStepValidateTemplate()
	var result = testSubject.Run(context.Background(), stateBag)

	if result != multistep.ActionContinue {
		t.Fatalf("Expected the step to return 'ActionContinue', but got '%d'.", result)
	}

	var expectedResourceGroupName = stateBag.Get(constants.ArmResourceGroupName).(string)
	var expectedSubscriptionId = stateBag.Get(constants.ArmSubscription).(string)

	if actualDeploymentName != "--deployment-name--" {
		t.Fatal("Expected StepValidateTemplate to source 'constants.ArmDeploymentName' from the state bag, but it did not.")
	}

	if actualResourceGroupName != expectedResourceGroupName {
		t.Fatal("Expected the step to source 'constants.ArmResourceGroupName' from the state bag, but it did not.")
	}

	if actualSubscriptionId != expectedSubscriptionId {
		t.Fatal("Expected the step to source 'constants.ArmSubscription' from the state bag, but it did not.")
	}
}

func TestStepDeployTemplateDeleteImageShouldFailWhenImageUrlCannotBeParsed(t *testing.T) {
	var testSubject = &StepDeployTemplate{
		say:          func(message string) {},
		error:        func(e error) {},
		name:         "--deployment-name--",
		templateType: VirtualMachineTemplate,
		client:       &AzureClient{PollingDuration: time.Minute * 5},
	}
	// Invalid URL per https://golang.org/src/net/url/url_test.go
	err := testSubject.deleteImage(context.TODO(), "http://[fe80::1%en0]/", "Unit Test: ResourceGroupName", false, "subscriptionId", "")
	if err == nil {
		t.Fatal("Expected a failure because of the failed image name")
	}
}

func TestStepDeployTemplateDeleteImageShouldFailWithInvalidImage(t *testing.T) {
	var testSubject = &StepDeployTemplate{
		say:          func(message string) {},
		error:        func(e error) {},
		client:       &AzureClient{PollingDuration: time.Minute * 5},
		name:         "--deployment-name--",
		templateType: VirtualMachineTemplate,
	}
	err := testSubject.deleteImage(context.TODO(), "storage.blob.core.windows.net/abc", "Unit Test: ResourceGroupName", false, "subscriptionId", "")
	if err == nil {
		t.Fatal("Expected a failure because of the failed image name")
	}
}

func TestStepDeployTemplateCleanupShouldDeleteManagedOSImageInExistingResourceGroup(t *testing.T) {
	var deleteDiskCounter = 0
	var testSubject = createTestStepDeployTemplateDeleteOSImage(&deleteDiskCounter, VirtualMachineTemplate)

	stateBag := createTestStateBagStepDeployTemplate()
	stateBag.Put(constants.ArmIsManagedImage, true)
	stateBag.Put(constants.ArmIsSIGImage, false)
	stateBag.Put(constants.ArmIsExistingResourceGroup, true)
	stateBag.Put(constants.ArmIsResourceGroupCreated, true)
	stateBag.Put(constants.ArmKeepOSDisk, false)
	stateBag.Put("ui", packersdk.TestUi(t))

	testSubject.Cleanup(stateBag)
	if deleteDiskCounter != 1 {
		t.Fatalf("Expected DeployTemplate Cleanup to invoke deleteDisk 1 time, but invoked %d times", deleteDiskCounter)
	}
}

func TestStepDeployTemplateCleanupShouldDeleteManagedOSImageInTemporaryResourceGroup(t *testing.T) {
	var deleteDiskCounter = 0
	var testSubject = createTestStepDeployTemplateDeleteOSImage(&deleteDiskCounter, VirtualMachineTemplate)

	stateBag := createTestStateBagStepDeployTemplate()
	stateBag.Put(constants.ArmIsManagedImage, true)
	stateBag.Put(constants.ArmIsSIGImage, false)
	stateBag.Put(constants.ArmIsExistingResourceGroup, false)
	stateBag.Put(constants.ArmIsResourceGroupCreated, true)
	stateBag.Put(constants.ArmKeepOSDisk, false)
	stateBag.Put("ui", packersdk.TestUi(t))

	testSubject.Cleanup(stateBag)
	if deleteDiskCounter != 1 {
		t.Fatalf("Expected DeployTemplate Cleanup to invoke deleteDisk 1 times, but invoked %d times", deleteDiskCounter)
	}
}

func TestStepDeployTemplateCleanupShouldDeleteVHDOSImageInExistingResourceGroup(t *testing.T) {
	var deleteDiskCounter = 0
	var testSubject = createTestStepDeployTemplateDeleteOSImage(&deleteDiskCounter, VirtualMachineTemplate)

	stateBag := createTestStateBagStepDeployTemplate()
	stateBag.Put(constants.ArmIsManagedImage, false)
	stateBag.Put(constants.ArmIsSIGImage, false)
	stateBag.Put(constants.ArmIsExistingResourceGroup, true)
	stateBag.Put(constants.ArmIsResourceGroupCreated, true)
	stateBag.Put(constants.ArmKeepOSDisk, false)
	stateBag.Put("ui", packersdk.TestUi(t))

	testSubject.Cleanup(stateBag)
	if deleteDiskCounter != 1 {
		t.Fatalf("Expected DeployTemplate Cleanup to invoke deleteDisk 1 time, but invoked %d times", deleteDiskCounter)
	}
}

func TestStepDeployTemplateCleanupShouldVHDOSImageInTemporaryResourceGroup(t *testing.T) {
	var deleteDiskCounter = 0
	var testSubject = createTestStepDeployTemplateDeleteOSImage(&deleteDiskCounter, VirtualMachineTemplate)

	stateBag := createTestStateBagStepDeployTemplate()
	stateBag.Put(constants.ArmIsManagedImage, false)
	stateBag.Put(constants.ArmIsSIGImage, false)
	stateBag.Put(constants.ArmIsExistingResourceGroup, false)
	stateBag.Put(constants.ArmIsResourceGroupCreated, true)
	stateBag.Put(constants.ArmKeepOSDisk, false)
	stateBag.Put("ui", packersdk.TestUi(t))

	testSubject.Cleanup(stateBag)
	if deleteDiskCounter != 1 {
		t.Fatalf("Expected DeployTemplate Cleanup to invoke deleteDisk 1 times, but invoked %d times", deleteDiskCounter)
	}
}

func TestStepDeployTemplateCleanupShouldNotDeleteDiskForKeyVaultDeployments(t *testing.T) {
	var deleteDiskCounter = 0
	var testSubject = createTestStepDeployTemplateDeleteOSImage(&deleteDiskCounter, KeyVaultTemplate)

	stateBag := createTestStateBagStepDeployTemplate()
	stateBag.Put(constants.ArmIsManagedImage, false)
	stateBag.Put(constants.ArmIsSIGImage, false)
	stateBag.Put(constants.ArmIsExistingResourceGroup, true)
	stateBag.Put(constants.ArmIsResourceGroupCreated, true)
	stateBag.Put(constants.ArmKeepOSDisk, false)
	stateBag.Put("ui", packersdk.TestUi(t))

	testSubject.Cleanup(stateBag)
	if deleteDiskCounter != 0 {
		t.Fatalf("Expected DeployTemplate Cleanup to not invoke deleteDisk, but invoked %d times", deleteDiskCounter)
	}
}

func createTestStateBagStepDeployTemplate() multistep.StateBag {
	stateBag := new(multistep.BasicStateBag)

	stateBag.Put(constants.ArmDeploymentName, "Unit Test: DeploymentName")
	stateBag.Put(constants.ArmStorageAccountName, "Unit Test: StorageAccountName")
	stateBag.Put(constants.ArmResourceGroupName, "Unit Test: ResourceGroupName")
	stateBag.Put(constants.ArmComputeName, "Unit Test: ComputeName")
	stateBag.Put(constants.ArmSubscription, "Unit Test: Subscription")

	return stateBag
}

func createTestStepDeployTemplateDeleteOSImage(deleteDiskCounter *int, templateType DeploymentTemplateType) *StepDeployTemplate {
	return &StepDeployTemplate{
		deploy: func(context.Context, string, string, string) error { return nil },
		say:    func(message string) {},
		error:  func(e error) {},
		deleteDisk: func(ctx context.Context, imageName string, resourceGroupName string, isManagedDisk bool, subscriptionId string, storageAccountName string) error {
			*deleteDiskCounter++
			return nil
		},
		disk: func(ctx context.Context, subscriptionId, resourceGroupName, computeName string) (string, string, error) {
			return "Microsoft.Compute/disks", "", nil
		},
		delete: func(ctx context.Context, subscriptionId, deploymentName, resourceGroupName string) error {
			return nil
		},
		deleteDeployment: func(ctx context.Context, state multistep.StateBag) error {
			return nil
		},
		templateType: templateType,
	}
}
