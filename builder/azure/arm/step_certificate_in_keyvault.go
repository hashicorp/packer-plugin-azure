// Copyright (c) HashiCorp, Inc.
// SPDX-License-Identifier: MPL-2.0

package arm

import (
	"context"
	"fmt"
	"time"

	"github.com/hashicorp/go-azure-sdk/resource-manager/keyvault/2023-07-01/secrets"
	"github.com/hashicorp/packer-plugin-azure/builder/azure/common/constants"
	"github.com/hashicorp/packer-plugin-sdk/multistep"
	packersdk "github.com/hashicorp/packer-plugin-sdk/packer"
)

type StepCertificateInKeyVault struct {
	config         *Config
	client         *AzureClient
	set            func(ctx context.Context, id secrets.SecretId) error
	say            func(message string)
	error          func(e error)
	certificate    string
	expirationTime time.Duration
}

func NewStepCertificateInKeyVault(client *AzureClient, ui packersdk.Ui, config *Config, certificate string, expirationTime time.Duration) *StepCertificateInKeyVault {
	var step = &StepCertificateInKeyVault{
		client:         client,
		config:         config,
		say:            func(message string) { ui.Say(message) },
		error:          func(e error) { ui.Error(e.Error()) },
		certificate:    certificate,
		expirationTime: expirationTime,
	}

	step.set = step.setCertificate
	return step
}

func (s *StepCertificateInKeyVault) setCertificate(ctx context.Context, id secrets.SecretId) error {
	secret := secrets.SecretCreateOrUpdateParameters{
		Properties: secrets.SecretProperties{
			Value: &s.certificate,
		},
	}
	if s.expirationTime != 0 {
		// Secrets API expects expiration time in seconds since the start of the unix epoch
		// https://learn.microsoft.com/en-us/azure/templates/microsoft.keyvault/vaults/secrets?pivots=deployment-language-bicep#secretattributes
		expirationTimeUnix := time.Now().Add(s.expirationTime).Unix()
		secret.Properties.Attributes = &secrets.Attributes{
			Exp: &expirationTimeUnix,
		}
	}
	pollingContext, cancel := context.WithTimeout(ctx, s.client.PollingDuration)
	defer cancel()

	_, err := s.client.SecretsClient.CreateOrUpdate(pollingContext, id, secret)

	return err
}
func (s *StepCertificateInKeyVault) Run(ctx context.Context, state multistep.StateBag) multistep.StepAction {
	s.say("Setting the certificate in the KeyVault...")
	var keyVaultName = state.Get(constants.ArmKeyVaultName).(string)
	var subscriptionId = state.Get(constants.ArmSubscription).(string)
	var resourceGroupName = state.Get(constants.ArmResourceGroupName).(string)
	var keyVaultSecretName = state.Get(constants.ArmKeyVaultSecretName).(string)
	id := secrets.NewSecretID(subscriptionId, resourceGroupName, keyVaultName, keyVaultSecretName)
	err := s.set(ctx, id)
	if err != nil {
		s.error(fmt.Errorf("Error setting winrm cert in custom keyvault: %s", err))
		return multistep.ActionHalt
	}

	return multistep.ActionContinue
}

func (*StepCertificateInKeyVault) Cleanup(multistep.StateBag) {
}
