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

func TestStepSnapshotDataDisksShouldFailIfSnapshotFails(t *testing.T) {
	var testSubject = &StepSnapshotDataDisks{
		create: func(context.Context, string, string, string, string, map[string]string, string) error {
			return fmt.Errorf("!! Unit Test FAIL !!")
		},
		say:    func(message string) {},
		error:  func(e error) {},
		enable: func() bool { return true },
	}

	stateBag := createTestStateBagStepSnapshotDataDisks()

	var result = testSubject.Run(context.Background(), stateBag)
	if result != multistep.ActionHalt {
		t.Fatalf("Expected the step to return 'ActionHalt', but got '%d'.", result)
	}

	if _, ok := stateBag.GetOk(constants.Error); ok == false {
		t.Fatalf("Expected the step to set stateBag['%s'], but it was not.", constants.Error)
	}
}

func TestStepSnapshotDataDisksShouldNotExecute(t *testing.T) {
	var testSubject = &StepSnapshotDataDisks{
		create: func(context.Context, string, string, string, string, map[string]string, string) error {
			return fmt.Errorf("!! Unit Test FAIL !!")
		},
		say:    func(message string) {},
		error:  func(e error) {},
		enable: func() bool { return false },
	}

	var result = testSubject.Run(context.Background(), nil)
	if result != multistep.ActionContinue {
		t.Fatalf("Expected the step to return 'ActionContinue', but got '%d'.", result)
	}
}

func TestStepSnapshotDataDisksShouldPassIfSnapshotPasses(t *testing.T) {
	var testSubject = &StepSnapshotDataDisks{
		create: func(context.Context, string, string, string, string, map[string]string, string) error {
			return nil
		},
		say:    func(message string) {},
		error:  func(e error) {},
		enable: func() bool { return true },
	}

	stateBag := createTestStateBagStepSnapshotDataDisks()

	var result = testSubject.Run(context.Background(), stateBag)
	if result != multistep.ActionContinue {
		t.Fatalf("Expected the step to return 'ActionContinue', but got '%d'.", result)
	}

	if _, ok := stateBag.GetOk(constants.Error); ok == true {
		t.Fatalf("Expected the step to not set stateBag['%s'], but it was.", constants.Error)
	}
}

func TestStepSnapshotDataDisksShouldAddLunTagAndPreserveOriginalTags(t *testing.T) {
	capturedTags := map[string]map[string]string{}
	var testSubject = &StepSnapshotDataDisks{
		create: func(ctx context.Context, subscriptionId string, resourceGroupName string, srcUriVhd string, location string, tags map[string]string, dstSnapshotName string) error {
			// Copy the tags so any later per-disk mutation does not affect the captured value.
			captured := make(map[string]string, len(tags))
			for k, v := range tags {
				captured[k] = v
			}
			capturedTags[srcUriVhd] = captured
			return nil
		},
		say:    func(message string) {},
		error:  func(e error) {},
		enable: func() bool { return true },
	}

	stateBag := new(multistep.BasicStateBag)
	stateBag.Put(constants.ArmManagedImageResourceGroupName, "Unit Test: ResourceGroupName")
	stateBag.Put(constants.ArmLocation, "Unit Test: Location")
	originalTags := map[string]string{"env": "test"}
	stateBag.Put(constants.ArmTags, originalTags)
	stateBag.Put(constants.ArmAdditionalDiskVhds, []DataDiskInfo{
		{Lun: 0, ManagedDiskID: "disk-lun-0"},
		{Lun: 3, ManagedDiskID: "disk-lun-3"},
	})
	stateBag.Put(constants.ArmManagedImageDataDiskSnapshotPrefix, "snap")
	stateBag.Put(constants.ArmSubscription, "Unit Test: SubscriptionId")

	if result := testSubject.Run(context.Background(), stateBag); result != multistep.ActionContinue {
		t.Fatalf("Expected the step to return 'ActionContinue', but got '%d'.", result)
	}

	expectedLunTag := map[string]string{
		"disk-lun-0": "0",
		"disk-lun-3": "3",
	}
	for diskID, wantLun := range expectedLunTag {
		tags, ok := capturedTags[diskID]
		if !ok {
			t.Fatalf("Expected create to be called for disk '%s', but it was not.", diskID)
		}
		if tags["packer:lun"] != wantLun {
			t.Fatalf("Expected 'packer:lun' tag for disk '%s' to be '%s', but got '%s'.", diskID, wantLun, tags["packer:lun"])
		}
		if tags["env"] != "test" {
			t.Fatalf("Expected original tag 'env=test' to be preserved for disk '%s', but got '%s'.", diskID, tags["env"])
		}
	}

	// The shared tags map from the state bag must not be mutated with per-disk LUN values.
	if _, ok := originalTags["packer:lun"]; ok {
		t.Fatalf("Expected the original tags map to not be mutated with a 'packer:lun' key, but it was.")
	}
	if len(originalTags) != 1 {
		t.Fatalf("Expected the original tags map to remain unchanged with 1 entry, but it had %d.", len(originalTags))
	}
}

func createTestStateBagStepSnapshotDataDisks() multistep.StateBag {
	stateBag := new(multistep.BasicStateBag)

	stateBag.Put(constants.ArmManagedImageResourceGroupName, "Unit Test: ResourceGroupName")
	stateBag.Put(constants.ArmLocation, "Unit Test: Location")

	value := "Unit Test: Tags"
	tags := map[string]string{
		"tag02:": value,
	}
	stateBag.Put(constants.ArmTags, tags)

	stateBag.Put(constants.ArmAdditionalDiskVhds, []DataDiskInfo{
		{Lun: 0, ManagedDiskID: "subscriptions/123-456-789/resourceGroups/existingresourcegroup/providers/Microsoft.Compute/disks/osdisk"},
	})
	stateBag.Put(constants.ArmManagedImageDataDiskSnapshotPrefix, "Unit Test: ManagedImageDataDiskSnapshotPrefix")
	stateBag.Put(constants.ArmSubscription, "Unit Test: SubscriptionId")

	return stateBag
}
