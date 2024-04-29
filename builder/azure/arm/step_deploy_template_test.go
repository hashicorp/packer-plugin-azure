// Copyright (c) HashiCorp, Inc.
// SPDX-License-Identifier: MPL-2.0

package arm

import (
	"context"
	"fmt"
	"testing"
	"time"

	"github.com/google/go-cmp/cmp"
	"github.com/hashicorp/go-azure-helpers/resourcemanager/commonids"
	"github.com/hashicorp/go-azure-sdk/resource-manager/compute/2022-03-01/virtualmachines"
	"github.com/hashicorp/go-azure-sdk/resource-manager/resources/2022-09-01/deploymentoperations"
	"github.com/hashicorp/packer-plugin-azure/builder/azure/common"
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
	mockTrackers := mockTrackers{
		deleteDiskCounter: common.IntPtr(0),
	}
	var testSubject = createTestStepDeployTemplateDeleteOSImage(t, &mockTrackers, VirtualMachineTemplate, virtualMachineDeploymentOperations())

	stateBag := createTestStateBagStepDeployTemplate()
	stateBag.Put(constants.ArmIsManagedImage, true)
	stateBag.Put(constants.ArmIsSIGImage, false)
	stateBag.Put(constants.ArmIsExistingResourceGroup, true)
	stateBag.Put(constants.ArmIsResourceGroupCreated, true)
	stateBag.Put(constants.ArmKeepOSDisk, false)
	stateBag.Put("ui", packersdk.TestUi(t))

	testSubject.Cleanup(stateBag)
	if *mockTrackers.deleteDiskCounter != 1 {
		t.Fatalf("Expected DeployTemplate Cleanup to invoke deleteDisk 1 time, but invoked %d times", *mockTrackers.deleteDiskCounter)
	}
	if mockTrackers.vmDeleteCalled == nil || *mockTrackers.vmDeleteCalled == false {
		t.Fatalf("Expected DeployTemplate cleanup to delete call deleteVM to delete the VirtualMachine but didn't")
	}
}

func TestStepDeployTemplateCleanupShouldDeleteManagedOSImageInTemporaryResourceGroup(t *testing.T) {
	mockTrackers := mockTrackers{
		deleteDiskCounter: common.IntPtr(0),
	}
	var testSubject = createTestStepDeployTemplateDeleteOSImage(t, &mockTrackers, VirtualMachineTemplate, virtualMachineDeploymentOperations())

	stateBag := createTestStateBagStepDeployTemplate()
	stateBag.Put(constants.ArmIsManagedImage, true)
	stateBag.Put(constants.ArmIsSIGImage, false)
	stateBag.Put(constants.ArmIsExistingResourceGroup, false)
	stateBag.Put(constants.ArmIsResourceGroupCreated, true)
	stateBag.Put(constants.ArmKeepOSDisk, false)
	stateBag.Put("ui", packersdk.TestUi(t))

	testSubject.Cleanup(stateBag)
	if *mockTrackers.deleteDiskCounter != 1 {
		t.Fatalf("Expected DeployTemplate Cleanup to invoke deleteDisk 1 time, but invoked %d times", *mockTrackers.deleteDiskCounter)
	}
	if mockTrackers.vmDeleteCalled == nil || *mockTrackers.vmDeleteCalled == false {
		t.Fatalf("Expected DeployTemplate cleanup to delete call deleteVM to delete the VirtualMachine but didn't")
	}
}

func TestStepDeployTemplateCleanupShouldDeleteVHDOSImageInExistingResourceGroup(t *testing.T) {
	mockTrackers := mockTrackers{
		deleteDiskCounter: common.IntPtr(0),
	}
	var testSubject = createTestStepDeployTemplateDeleteOSImage(t, &mockTrackers, VirtualMachineTemplate, virtualMachineDeploymentOperations())

	stateBag := createTestStateBagStepDeployTemplate()
	stateBag.Put(constants.ArmIsManagedImage, false)
	stateBag.Put(constants.ArmIsSIGImage, false)
	stateBag.Put(constants.ArmIsExistingResourceGroup, true)
	stateBag.Put(constants.ArmIsResourceGroupCreated, true)
	stateBag.Put(constants.ArmKeepOSDisk, false)
	stateBag.Put("ui", packersdk.TestUi(t))

	testSubject.Cleanup(stateBag)
	if *mockTrackers.deleteDiskCounter != 1 {
		t.Fatalf("Expected DeployTemplate Cleanup to invoke deleteDisk 1 time, but invoked %d times", *mockTrackers.deleteDiskCounter)
	}
	if mockTrackers.vmDeleteCalled == nil || *mockTrackers.vmDeleteCalled == false {
		t.Fatalf("Expected DeployTemplate cleanup to delete call deleteVM to delete the VirtualMachine but didn't")
	}
}

func TestStepDeployTemplateCleanupShouldVHDOSImageInTemporaryResourceGroup(t *testing.T) {
	mockTrackers := mockTrackers{
		deleteDiskCounter: common.IntPtr(0),
	}
	var testSubject = createTestStepDeployTemplateDeleteOSImage(t, &mockTrackers, VirtualMachineTemplate, virtualMachineDeploymentOperations())

	stateBag := createTestStateBagStepDeployTemplate()
	stateBag.Put(constants.ArmIsManagedImage, false)
	stateBag.Put(constants.ArmIsSIGImage, false)
	stateBag.Put(constants.ArmIsExistingResourceGroup, false)
	stateBag.Put(constants.ArmIsResourceGroupCreated, true)
	stateBag.Put(constants.ArmKeepOSDisk, false)
	stateBag.Put("ui", packersdk.TestUi(t))

	testSubject.Cleanup(stateBag)
	if *mockTrackers.deleteDiskCounter != 1 {
		t.Fatalf("Expected DeployTemplate Cleanup to invoke deleteDisk 1 times, but invoked %d times", *mockTrackers.deleteDiskCounter)
	}
	if mockTrackers.vmDeleteCalled == nil || *mockTrackers.vmDeleteCalled == false {
		t.Fatalf("Expected DeployTemplate cleanup to delete call deleteVM to delete the VirtualMachine but didn't")
	}
}
func TestStepDeployTemplateCleanupShouldDeleteVirtualMachineAndNetworkResourcesInOrderToAvoidConflicts(t *testing.T) {
	mockTrackers := mockTrackers{
		actualNetworkResources: nil,
	}
	var testSubject = createTestStepDeployTemplateDeleteOSImage(t, &mockTrackers, VirtualMachineTemplate, virtualMachineDeploymentOperations())
	testSubject.listDeploymentOps = func(ctx context.Context, id deploymentoperations.ResourceGroupDeploymentId) ([]deploymentoperations.DeploymentOperation, error) {
		return virtualMachineWithNetworkingDeploymentOperations(), nil
	}
	testSubject.deleteDetatchedResources = func(ctx context.Context, subscriptionId, resourceGroupName string, resources map[string]string) {
		if mockTrackers.vmDeleteCalled == nil || *mockTrackers.vmDeleteCalled == false {
			t.Fatal("deleteNetworkResources called before deleting VM, this will lead to deletion conflicts")
		}
		if mockTrackers.deleteNicCalled == nil || *mockTrackers.deleteNicCalled == false {
			t.Fatal("deleteNetworkResources called before deleting NIC, this will lead to deletion conflicts")
		}

		if len(resources) != 0 {
			mockTrackers.actualNetworkResources = &resources
		}
	}

	stateBag := createTestStateBagStepDeployTemplate()
	stateBag.Put(constants.ArmIsManagedImage, false)
	stateBag.Put(constants.ArmIsSIGImage, false)
	stateBag.Put(constants.ArmIsExistingResourceGroup, true)
	stateBag.Put(constants.ArmIsResourceGroupCreated, true)
	stateBag.Put(constants.ArmKeepOSDisk, false)
	stateBag.Put("ui", packersdk.TestUi(t))

	testSubject.Cleanup(stateBag)
	if mockTrackers.vmDeleteCalled == nil || *mockTrackers.vmDeleteCalled == false {
		t.Fatalf("Expected DeployTemplate cleanup to call deleteVM to delete the VirtualMachine but it didn't")
	}
	if mockTrackers.deleteNicCalled == nil || *mockTrackers.deleteNicCalled == false {
		t.Fatalf("Expected DeployTemplate cleanup to call delete network interface but it didn't")
	}
	if mockTrackers.actualNetworkResources == nil {
		t.Fatalf("Expected DeployTemplate to call delete network resources but it didn't")
	} else {
		expectedResources := map[string]string{"Microsoft.Network/publicIPAddresses": "ip", "Microsoft.Network/virtualNetworks": "vnet"}
		if diff := cmp.Diff(expectedResources, *mockTrackers.actualNetworkResources); diff != "" {
			t.Fatalf("Unexpected difference in expected parameter deleteNetworkResources.resources %s", diff)
		}
	}
}
func TestStepDeployTemplateCleanupShouldDeleteKeyVault(t *testing.T) {
	mockTrackers := mockTrackers{
		keyVaultDeleteCalled: common.BoolPtr(false),
	}
	// This step lacks any methods not required to delete the key vault
	// As such it validates that during the deletion of the key vault, no other endpoints are called, such as delete Disk
	var testSubject = createTestStepDeployTemplateKeyVault(&mockTrackers, KeyVaultTemplate, keyVaultDeploymentOperations())

	stateBag := createTestStateBagStepDeployTemplate()
	stateBag.Put(constants.ArmIsManagedImage, false)
	stateBag.Put(constants.ArmIsSIGImage, false)
	stateBag.Put(constants.ArmIsExistingResourceGroup, true)
	stateBag.Put(constants.ArmIsResourceGroupCreated, true)
	stateBag.Put(constants.ArmKeepOSDisk, false)
	stateBag.Put("ui", packersdk.TestUi(t))

	testSubject.Cleanup(stateBag)
	if mockTrackers.keyVaultDeleteCalled == nil || *mockTrackers.keyVaultDeleteCalled == false {
		t.Fatalf("Expected DeployTemplate cleanup to delete call deleteKV to delete the KeyVault but didn't")
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

func virtualMachineDeploymentOperations() []deploymentoperations.DeploymentOperation {
	return []deploymentoperations.DeploymentOperation{
		{
			Properties: &deploymentoperations.DeploymentOperationProperties{
				TargetResource: &deploymentoperations.TargetResource{
					ResourceName: common.StringPtr("virtualmachine"),
					ResourceType: common.StringPtr("Microsoft.Compute/virtualMachines"),
				},
			},
		},
	}
}

func virtualMachineWithNetworkingDeploymentOperations() []deploymentoperations.DeploymentOperation {
	return []deploymentoperations.DeploymentOperation{
		{
			Properties: &deploymentoperations.DeploymentOperationProperties{
				TargetResource: &deploymentoperations.TargetResource{
					ResourceName: common.StringPtr("virtualmachine"),
					ResourceType: common.StringPtr("Microsoft.Compute/virtualMachines"),
				},
			},
		},
		{
			Properties: &deploymentoperations.DeploymentOperationProperties{
				TargetResource: &deploymentoperations.TargetResource{
					ResourceName: common.StringPtr("coolnic"),
					ResourceType: common.StringPtr("Microsoft.Network/networkInterfaces"),
				},
			},
		},
		{
			Properties: &deploymentoperations.DeploymentOperationProperties{
				TargetResource: &deploymentoperations.TargetResource{
					ResourceName: common.StringPtr("vnet"),
					ResourceType: common.StringPtr("Microsoft.Network/virtualNetworks"),
				},
			},
		},
		{
			Properties: &deploymentoperations.DeploymentOperationProperties{
				TargetResource: &deploymentoperations.TargetResource{
					ResourceName: common.StringPtr("ip"),
					ResourceType: common.StringPtr("Microsoft.Network/publicIPAddresses"),
				},
			},
		},
	}
}

func keyVaultDeploymentOperations() []deploymentoperations.DeploymentOperation {
	return []deploymentoperations.DeploymentOperation{
		{
			Properties: &deploymentoperations.DeploymentOperationProperties{
				TargetResource: &deploymentoperations.TargetResource{
					ResourceName: common.StringPtr("vault3-mojave"),
					ResourceType: common.StringPtr("Microsoft.KeyVault/vaults"),
				},
			},
		},
	}
}

type mockTrackers struct {
	deleteDiskCounter      *int
	deleteNicCalled        *bool
	keyVaultDeleteCalled   *bool
	vmDeleteCalled         *bool
	actualNetworkResources *map[string]string
}

func createTestStepDeployTemplateDeleteOSImage(t *testing.T, trackers *mockTrackers, templateType DeploymentTemplateType, deploymentOperations []deploymentoperations.DeploymentOperation) *StepDeployTemplate {
	if trackers.deleteDiskCounter == nil {
		trackers.deleteDiskCounter = common.IntPtr(0)
	}
	return &StepDeployTemplate{
		deploy: func(context.Context, string, string, string) error { return nil },
		say:    func(message string) {},
		error:  func(e error) {},
		deleteDisk: func(ctx context.Context, imageName string, resourceGroupName string, isManagedDisk bool, subscriptionId string, storageAccountName string) error {
			*trackers.deleteDiskCounter++
			return nil
		},
		getDisk: func(ctx context.Context, subscriptionId, resourceGroupName, computeName string) (string, string, error) {
			return "Microsoft.Compute/disks", "", nil
		},
		deleteNic: func(ctx context.Context, networkInterfacesId commonids.NetworkInterfaceId) error {
			if trackers.vmDeleteCalled == nil || *trackers.vmDeleteCalled == false {
				t.Fatal("Unexpectedly deleteNic before deleting VM")
			}
			trackers.deleteNicCalled = common.BoolPtr(true)
			return nil
		},
		deleteDetatchedResources: func(ctx context.Context, subscriptionId string, resourceGroupName string, resources map[string]string) {
			if len(resources) != 0 {
				trackers.actualNetworkResources = &resources
			}
		},
		deleteDeployment: func(ctx context.Context, state multistep.StateBag) error {
			return nil
		},
		listDeploymentOps: func(ctx context.Context, id deploymentoperations.ResourceGroupDeploymentId) ([]deploymentoperations.DeploymentOperation, error) {
			return deploymentOperations, nil
		},
		deleteVM: func(ctx context.Context, virtualMachineId virtualmachines.VirtualMachineId) error {
			trackers.vmDeleteCalled = common.BoolPtr(true)
			return nil
		},
		templateType: templateType,
	}
}

func createTestStepDeployTemplateKeyVault(trackers *mockTrackers, templateType DeploymentTemplateType, deploymentOperations []deploymentoperations.DeploymentOperation) *StepDeployTemplate {
	return &StepDeployTemplate{
		deploy: func(context.Context, string, string, string) error { return nil },
		say:    func(message string) {},
		error:  func(e error) {},
		listDeploymentOps: func(ctx context.Context, id deploymentoperations.ResourceGroupDeploymentId) ([]deploymentoperations.DeploymentOperation, error) {
			return deploymentOperations, nil
		},
		deleteKV: func(ctx context.Context, id commonids.KeyVaultId) error {
			trackers.keyVaultDeleteCalled = common.BoolPtr(true)
			return nil
		},
		deleteDeployment: func(ctx context.Context, state multistep.StateBag) error {
			return nil
		},
		templateType: templateType,
	}
}
