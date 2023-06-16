// Copyright (c) HashiCorp, Inc.
// SPDX-License-Identifier: MPL-2.0

package arm

import (
	"context"
	"fmt"

	hashiSecretsSDK "github.com/hashicorp/go-azure-sdk/resource-manager/keyvault/2023-02-01/secrets"
	"github.com/hashicorp/packer-plugin-azure/builder/azure/common/constants"
	"github.com/hashicorp/packer-plugin-sdk/multistep"
	packersdk "github.com/hashicorp/packer-plugin-sdk/packer"
)

type StepCertificateInKeyVault struct {
	config      *Config
	client      *AzureClient
	set         func(ctx context.Context, id hashiSecretsSDK.SecretId) error
	say         func(message string)
	error       func(e error)
	certificate string
}

func NewStepCertificateInKeyVault(client *AzureClient, ui packersdk.Ui, config *Config, certificate string) *StepCertificateInKeyVault {
	var step = &StepCertificateInKeyVault{
		client:      client,
		config:      config,
		say:         func(message string) { ui.Say(message) },
		error:       func(e error) { ui.Error(e.Error()) },
		certificate: certificate,
	}

	step.set = step.setCertificate
	return step
}

func (s *StepCertificateInKeyVault) setCertificate(ctx context.Context, id hashiSecretsSDK.SecretId) error {
	_, err := s.client.SecretsClient.CreateOrUpdate(ctx, id, hashiSecretsSDK.SecretCreateOrUpdateParameters{
		Properties: hashiSecretsSDK.SecretProperties{
			Value: &s.certificate,
		},
	})

	return err
}
func (s *StepCertificateInKeyVault) Run(ctx context.Context, state multistep.StateBag) multistep.StepAction {
	s.say("Setting the certificate in the KeyVault...")
	var keyVaultName = state.Get(constants.ArmKeyVaultName).(string)
	var subscriptionId = state.Get(constants.ArmSubscription).(string)
	var resourceGroupName = state.Get(constants.ArmResourceGroupName).(string)
	id := hashiSecretsSDK.NewSecretID(subscriptionId, resourceGroupName, keyVaultName, DefaultSecretName)
	err := s.set(ctx, id)
	if err != nil {
		s.error(fmt.Errorf("Error setting winrm cert in custom keyvault: %s", err))
		return multistep.ActionHalt
	}

	return multistep.ActionContinue
}

func (*StepCertificateInKeyVault) Cleanup(multistep.StateBag) {
}
