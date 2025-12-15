// Copyright IBM Corp. 2013, 2025
// SPDX-License-Identifier: MPL-2.0

package dtl

import (
	"testing"

	"github.com/hashicorp/packer-plugin-azure/builder/azure/common/constants"
)

func TestStateBagShouldBePopulatedExpectedValues(t *testing.T) {
	var testSubject = &Builder{}
	_, _, err := testSubject.Prepare(getDtlBuilderConfiguration(), getPackerConfiguration())
	if err != nil {
		t.Fatalf("failed to prepare: %s", err)
	}

	var expectedStateBagKeys = []string{
		constants.AuthorizedKey,

		constants.ArmTags,
		constants.ArmComputeName,
		constants.ArmDeploymentName,
		constants.ArmNicName,
		constants.ArmResourceGroupName,
		constants.ArmVirtualMachineCaptureParameters,
		constants.ArmPublicIPAddressName,
	}

	for _, v := range expectedStateBagKeys {
		if _, ok := testSubject.stateBag.GetOk(v); ok == false {
			t.Errorf("Expected the builder's state bag to contain '%s', but it did not.", v)
		}
	}
}
