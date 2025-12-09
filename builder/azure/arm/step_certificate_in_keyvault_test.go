// Copyright IBM Corp. 2013, 2025
// SPDX-License-Identifier: MPL-2.0

package arm

import (
	"context"
	"fmt"
	"testing"

	"github.com/hashicorp/go-azure-sdk/resource-manager/keyvault/2023-07-01/secrets"
	"github.com/hashicorp/packer-plugin-azure/builder/azure/common/constants"
	"github.com/hashicorp/packer-plugin-sdk/multistep"
)

func TestNewStepCertificateInKeyVault(t *testing.T) {

	state := new(multistep.BasicStateBag)
	state.Put(constants.ArmKeyVaultName, "testKeyVaultName")
	state.Put(constants.ArmSubscription, "testSubscription")
	state.Put(constants.ArmResourceGroupName, "testResourceGroupName")
	state.Put(constants.ArmKeyVaultSecretName, "testKeyVaultSecretName")

	config := &Config{
		winrmCertificate: "testCertificateString",
	}

	certKVStep := &StepCertificateInKeyVault{
		say:         func(message string) {},
		error:       func(e error) {},
		set:         func(ctx context.Context, id secrets.SecretId) error { return nil },
		config:      config,
		certificate: config.winrmCertificate}

	stepAction := certKVStep.Run(context.TODO(), state)

	if stepAction == multistep.ActionHalt {
		t.Fatalf("step should have succeeded.")
	}

}

func TestNewStepCertificateInKeyVault_error(t *testing.T) {
	state := new(multistep.BasicStateBag)
	state.Put(constants.ArmKeyVaultName, "testKeyVaultName")
	state.Put(constants.ArmSubscription, "testSubscription")
	state.Put(constants.ArmResourceGroupName, "testResourceGroupName")
	state.Put(constants.ArmKeyVaultSecretName, "testKeyVaultSecretName")

	config := &Config{
		winrmCertificate: "testCertificateString",
	}

	certKVStep := &StepCertificateInKeyVault{
		say:         func(message string) {},
		error:       func(e error) {},
		set:         func(ctx context.Context, id secrets.SecretId) error { return fmt.Errorf("Unit test fail") },
		config:      config,
		certificate: config.winrmCertificate}

	stepAction := certKVStep.Run(context.TODO(), state)

	if stepAction != multistep.ActionHalt {
		t.Fatalf("step should have failed.")
	}
}
