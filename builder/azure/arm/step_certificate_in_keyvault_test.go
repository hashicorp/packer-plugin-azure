// Copyright (c) HashiCorp, Inc.
// SPDX-License-Identifier: MPL-2.0

package arm

import (
	"bytes"
	"context"
	"testing"

	azcommon "github.com/hashicorp/packer-plugin-azure/builder/azure/common"
	"github.com/hashicorp/packer-plugin-azure/builder/azure/common/constants"
	"github.com/hashicorp/packer-plugin-sdk/multistep"
	packersdk "github.com/hashicorp/packer-plugin-sdk/packer"
)

func TestNewStepCertificateInKeyVault(t *testing.T) {
	cli := azcommon.MockAZVaultClient{}
	ui := &packersdk.BasicUi{
		Reader: new(bytes.Buffer),
		Writer: new(bytes.Buffer),
	}
	state := new(multistep.BasicStateBag)
	state.Put(constants.ArmKeyVaultName, "testKeyVaultName")

	config := &Config{
		winrmCertificate: "testCertificateString",
	}

	certKVStep := NewStepCertificateInKeyVault(&cli, ui, config, config.winrmCertificate)
	stepAction := certKVStep.Run(context.TODO(), state)

	if stepAction == multistep.ActionHalt {
		t.Fatalf("step should have succeeded.")
	}
	if !cli.SetSecretCalled {
		t.Fatalf("Step should have called SetSecret on Azure client.")
	}
	if cli.SetSecretCert != "testCertificateString" {
		t.Fatalf("Step should have read cert from winRMCertificate field on config.")
	}
	if cli.SetSecretVaultName != "testKeyVaultName" {
		t.Fatalf("step should have read keyvault name from state.")
	}
}

func TestNewStepCertificateInKeyVault_error(t *testing.T) {
	// Tell mock to return an error
	cli := azcommon.MockAZVaultClient{}
	cli.IsError = true

	ui := &packersdk.BasicUi{
		Reader: new(bytes.Buffer),
		Writer: new(bytes.Buffer),
	}
	state := new(multistep.BasicStateBag)
	state.Put(constants.ArmKeyVaultName, "testKeyVaultName")

	config := &Config{
		winrmCertificate: "testCertificateString",
	}

	certKVStep := NewStepCertificateInKeyVault(&cli, ui, config, config.winrmCertificate)
	stepAction := certKVStep.Run(context.TODO(), state)

	if stepAction != multistep.ActionHalt {
		t.Fatalf("step should have failed.")
	}
}
