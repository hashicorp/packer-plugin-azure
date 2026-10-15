// Copyright IBM Corp. 2013, 2025
// SPDX-License-Identifier: MPL-2.0

package arm

import (
	"context"
	"fmt"
	"testing"
	"time"

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

func TestStepCertificateInKeyVault_CleanupDeletesWhenConfigured(t *testing.T) {
	state := new(multistep.BasicStateBag)
	state.Put(constants.ArmKeyVaultName, "testKeyVaultName")
	state.Put(constants.ArmKeyVaultSecretName, "testKeyVaultSecretName")

	deleted := false
	var gotVault, gotSecret string

	certKVStep := &StepCertificateInKeyVault{
		say:   func(message string) {},
		error: func(e error) { t.Fatalf("unexpected error: %v", e) },
		delete: func(ctx context.Context, vaultName, secretName string) error {
			deleted = true
			gotVault = vaultName
			gotSecret = secretName
			return nil
		},
		config: &Config{
			BuildKeyVaultName:         "testKeyVaultName",
			BuildKeyVaultSecretDelete: true,
		},
		client: &AzureClient{PollingDuration: time.Minute},
	}

	certKVStep.Cleanup(state)

	if !deleted {
		t.Fatalf("expected secret delete to be called")
	}
	if gotVault != "testKeyVaultName" || gotSecret != "testKeyVaultSecretName" {
		t.Fatalf("unexpected delete args: vault=%q secret=%q", gotVault, gotSecret)
	}
}

func TestStepCertificateInKeyVault_CleanupSkippedWithoutFlag(t *testing.T) {
	state := new(multistep.BasicStateBag)
	state.Put(constants.ArmKeyVaultName, "testKeyVaultName")
	state.Put(constants.ArmKeyVaultSecretName, "testKeyVaultSecretName")

	deleted := false
	certKVStep := &StepCertificateInKeyVault{
		say:   func(message string) {},
		error: func(e error) { t.Fatalf("unexpected error: %v", e) },
		delete: func(ctx context.Context, vaultName, secretName string) error {
			deleted = true
			return nil
		},
		config: &Config{
			BuildKeyVaultName:         "testKeyVaultName",
			BuildKeyVaultSecretDelete: false,
		},
		client: &AzureClient{PollingDuration: time.Minute},
	}

	certKVStep.Cleanup(state)

	if deleted {
		t.Fatalf("did not expect secret delete when flag is false")
	}
}

func TestStepCertificateInKeyVault_CleanupSkippedWithoutExternalVault(t *testing.T) {
	state := new(multistep.BasicStateBag)
	state.Put(constants.ArmKeyVaultName, "testKeyVaultName")
	state.Put(constants.ArmKeyVaultSecretName, "testKeyVaultSecretName")

	deleted := false
	certKVStep := &StepCertificateInKeyVault{
		say:   func(message string) {},
		error: func(e error) { t.Fatalf("unexpected error: %v", e) },
		delete: func(ctx context.Context, vaultName, secretName string) error {
			deleted = true
			return nil
		},
		config: &Config{
			BuildKeyVaultName:         "",
			BuildKeyVaultSecretDelete: true,
		},
		client: &AzureClient{PollingDuration: time.Minute},
	}

	certKVStep.Cleanup(state)

	if deleted {
		t.Fatalf("did not expect secret delete without build_key_vault_name")
	}
}
