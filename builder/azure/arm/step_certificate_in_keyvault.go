// Copyright IBM Corp. 2013, 2025
// SPDX-License-Identifier: MPL-2.0

package arm

import (
	"context"
	"fmt"
	"net/http"
	"time"

	"github.com/hashicorp/go-azure-sdk/resource-manager/keyvault/2023-07-01/secrets"
	sdkClient "github.com/hashicorp/go-azure-sdk/sdk/client"
	"github.com/hashicorp/go-azure-sdk/sdk/client/resourcemanager"
	"github.com/hashicorp/go-azure-sdk/sdk/environments"
	"github.com/hashicorp/packer-plugin-azure/builder/azure/common/client"
	"github.com/hashicorp/packer-plugin-azure/builder/azure/common/constants"
	"github.com/hashicorp/packer-plugin-sdk/multistep"
	packersdk "github.com/hashicorp/packer-plugin-sdk/packer"
)

const keyVaultDataPlaneAPIVersion = "7.4"

type StepCertificateInKeyVault struct {
	config         *Config
	client         *AzureClient
	set            func(ctx context.Context, id secrets.SecretId) error
	delete         func(ctx context.Context, vaultName, secretName string) error
	say            func(message string)
	error          func(e error)
	certificate    string
	expirationTime time.Duration
}

func NewStepCertificateInKeyVault(azureClient *AzureClient, ui packersdk.Ui, config *Config, certificate string, expirationTime time.Duration) *StepCertificateInKeyVault {
	var step = &StepCertificateInKeyVault{
		client:         azureClient,
		config:         config,
		say:            func(message string) { ui.Say(message) },
		error:          func(e error) { ui.Error(e.Error()) },
		certificate:    certificate,
		expirationTime: expirationTime,
	}

	step.set = step.setCertificate
	step.delete = step.deleteCertificate
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

// deleteCertificate removes the secret via the Key Vault data-plane API.
// https://learn.microsoft.com/en-us/rest/api/keyvault/secrets/delete-secret/delete-secret
func (s *StepCertificateInKeyVault) deleteCertificate(ctx context.Context, vaultName, secretName string) error {
	vaultURI := fmt.Sprintf("https://%s.vault.azure.net", vaultName)
	endpoint := environments.NewApiEndpoint("KeyVault", vaultURI, nil)
	kvClient, err := resourcemanager.NewClient(endpoint, "secrets", keyVaultDataPlaneAPIVersion)
	if err != nil {
		return fmt.Errorf("instantiating Key Vault secrets client: %w", err)
	}

	authOptions := client.AzureAuthOptions{
		AuthType:           s.config.ClientConfig.AuthType(),
		ClientID:           s.config.ClientConfig.ClientID,
		ClientSecret:       s.config.ClientConfig.ClientSecret,
		ClientJWT:          s.config.ClientConfig.ClientJWT,
		ClientCertPath:     s.config.ClientConfig.ClientCertPath,
		ClientCertPassword: s.config.ClientConfig.ClientCertPassword,
		TenantID:           s.config.ClientConfig.TenantID,
		SubscriptionID:     s.config.ClientConfig.SubscriptionID,
		OidcRequestUrl:     s.config.ClientConfig.OidcRequestURL,
		OidcRequestToken:   s.config.ClientConfig.OidcRequestToken,
	}

	cloud := s.config.ClientConfig.CloudEnvironment()
	if cloud == nil {
		return fmt.Errorf("azure cloud environment is not configured")
	}

	authorizer, err := client.BuildKeyVaultAuthorizer(ctx, authOptions, *cloud)
	if err != nil {
		return fmt.Errorf("building Key Vault authorizer: %w", err)
	}
	kvClient.SetAuthorizer(authorizer)

	opts := sdkClient.RequestOptions{
		ContentType: "application/json; charset=utf-8",
		ExpectedStatusCodes: []int{
			http.StatusOK,
			http.StatusNotFound,
		},
		HttpMethod: http.MethodDelete,
		Path:       fmt.Sprintf("/secrets/%s", secretName),
	}

	req, err := kvClient.NewRequest(ctx, opts)
	if err != nil {
		return err
	}

	_, err = req.Execute(ctx)
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

func (s *StepCertificateInKeyVault) Cleanup(state multistep.StateBag) {
	if s.config == nil || !s.config.BuildKeyVaultSecretDelete {
		return
	}
	if s.config.BuildKeyVaultName == "" {
		return
	}
	if s.delete == nil {
		return
	}

	keyVaultName, ok := state.GetOk(constants.ArmKeyVaultName)
	if !ok {
		return
	}
	keyVaultSecretName, ok := state.GetOk(constants.ArmKeyVaultSecretName)
	if !ok {
		return
	}

	s.say(fmt.Sprintf("Deleting Key Vault secret '%s' from '%s'...", keyVaultSecretName, keyVaultName))
	ctx, cancel := context.WithTimeout(context.Background(), s.client.PollingDuration)
	defer cancel()

	if err := s.delete(ctx, keyVaultName.(string), keyVaultSecretName.(string)); err != nil {
		s.error(fmt.Errorf("Error deleting Key Vault secret: %s", err))
	}
}
