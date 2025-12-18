// Copyright IBM Corp. 2013, 2025
// SPDX-License-Identifier: MPL-2.0

//go:generate packer-sdc struct-markdown
//go:generate packer-sdc mapstructure-to-hcl2 -type Config,DatasourceOutput

package keyvaultsecret

import (
	"context"
	"errors"
	"fmt"
	"io"
	"net/http"
	"time"

	"github.com/hashicorp/go-azure-sdk/resource-manager/keyvault/2023-07-01/secrets"
	sdkClient "github.com/hashicorp/go-azure-sdk/sdk/client"
	"github.com/hashicorp/go-azure-sdk/sdk/environments"
	"github.com/hashicorp/hcl/v2/hcldec"
	azclient "github.com/hashicorp/packer-plugin-azure/builder/azure/common/client"
	"github.com/hashicorp/packer-plugin-azure/builder/azure/common/log"
	"github.com/hashicorp/packer-plugin-sdk/common"
	"github.com/hashicorp/packer-plugin-sdk/hcl2helper"
	packersdk "github.com/hashicorp/packer-plugin-sdk/packer"
	"github.com/hashicorp/packer-plugin-sdk/template/config"

	"github.com/zclconf/go-cty/cty"
)

type Config struct {
	common.PackerConfig `mapstructure:",squash"`

	// The name of the Azure Key Vault.
	VaultName string `mapstructure:"vault_name" required:"true"`
	// The name of the secret to fetch from the Azure Key Vault.
	SecretName string `mapstructure:"secret_name" required:"true"`
	// The version of the secret to fetch. If not provided, the latest version will be used.
	Version string `mapstructure:"version"`

	azclient.Config `mapstructure:",squash"` // Embed ClientConfig to allow for common client configuration
}

type Datasource struct {
	config Config
}

type DatasourceOutput struct {
	// The raw string response of the secret version.
	Response string `mapstructure:"response"`

	// The value extracted using the 'key', if provided.
	Value string `mapstructure:"value"`
}

func (d *Datasource) ConfigSpec() hcldec.ObjectSpec {
	return d.config.FlatMapstructure().HCL2Spec()
}

func (d *Datasource) OutputSpec() hcldec.ObjectSpec {
	return (&DatasourceOutput{}).FlatMapstructure().HCL2Spec()
}

func (d *Datasource) Configure(raws ...interface{}) error {
	err := config.Decode(&d.config, nil, raws...)
	if err != nil {
		return err
	}

	errs := new(packersdk.MultiError)

	if d.config.VaultName == "" {
		errs = packersdk.MultiErrorAppend(errs, errors.New("a 'vault_name' must be specified"))
	}
	if d.config.SecretName == "" {
		errs = packersdk.MultiErrorAppend(errs, errors.New("a 'secret_name' must be specified"))
	}

	d.config.Validate(errs)

	err = d.config.SetDefaultValues()
	if err != nil {
		errs = packersdk.MultiErrorAppend(errs, fmt.Errorf("failed to set default values: %w", err))
	}

	if errs != nil && len(errs.Errors) > 0 {
		return errs
	}
	return nil
}

func (d *Datasource) Execute() (cty.Value, error) {

	err := d.config.FillParameters()
	if err != nil {
		return cty.NullVal(cty.EmptyObject), err
	}

	vaultURI := fmt.Sprintf("https://%s.vault.azure.net", d.config.VaultName)
	endpoint := environments.NewApiEndpoint("KeyVault", vaultURI, nil)
	client, err := NewSecretsClientWithBaseURI(endpoint)
	if err != nil {
		return cty.NullVal(cty.EmptyObject), fmt.Errorf("failed to create secrets client: %w", err)
	}

	ctx, cancel := context.WithTimeout(context.Background(), 1000*time.Second)
	defer cancel()

	authOptions := azclient.AzureAuthOptions{
		AuthType:           d.config.AuthType(),
		ClientID:           d.config.ClientID,
		ClientSecret:       d.config.ClientSecret,
		ClientJWT:          d.config.ClientJWT,
		ClientCertPath:     d.config.ClientCertPath,
		ClientCertPassword: d.config.ClientCertPassword,
		TenantID:           d.config.TenantID,
		SubscriptionID:     d.config.SubscriptionID,
		OidcRequestUrl:     d.config.OidcRequestURL,
		OidcRequestToken:   d.config.OidcRequestToken,
	}

	authorizer, err := azclient.BuildKeyVaultAuthorizer(ctx, authOptions, *d.config.CloudEnvironment())
	if err != nil {
		log.Printf("failed to create Key Vault authorizer: %v", err)
		return cty.NullVal(cty.EmptyObject), fmt.Errorf("failed to create Key Vault authorizer: %w", err)
	}

	client.Client.SetAuthorizer(authorizer)

	result, err := d.getSecret(ctx, client)
	if err != nil {
		log.Printf("failed to get secret: %v", err)
		return cty.NullVal(cty.EmptyObject), fmt.Errorf("failed to get secret %q from vault %q: %w", d.config.SecretName, d.config.VaultName, err)
	}

	bytes, err := io.ReadAll(result.HttpResponse.Body)
	if err != nil {
		return cty.NullVal(cty.EmptyObject), fmt.Errorf("failed to read response body: %w", err)
	}
	log.Printf("[DEBUG] Retrieved secret %q from vault %q", d.config.SecretName, d.config.VaultName)

	return hcl2helper.HCL2ValueFromConfig(DatasourceOutput{
		Response: string(bytes),
		Value:    result.Model.Value,
	}, d.OutputSpec()), nil
}

// We are using the SecretsClient from the secrets package, which is a wrapper around the resourcemanager.Client.
// This allows us to use the same client for both the SecretsClient and the resourcemanager.Client,
// while still providing the necessary functionality to interact with Azure Key Vault secrets.
//
// Using the SecretsClient directly for fetching secrets currently only allows us
// to get the secret's metadata, and not the actual secret value.
func (d *Datasource) getSecret(ctx context.Context, client *secrets.SecretsClient) (result GetOperationResponse, err error) {
	// Implementation for retrieving the secret goes here
	opts := sdkClient.RequestOptions{
		ContentType: "application/json; charset=utf-8",
		ExpectedStatusCodes: []int{
			http.StatusOK,
		},
		HttpMethod: http.MethodGet,
		Path:       fmt.Sprintf("/secrets/%s/%s", d.config.SecretName, d.config.Version),
	}

	req, err := client.Client.NewRequest(ctx, opts)
	if err != nil {
		return
	}

	var resp *sdkClient.Response
	resp, err = req.Execute(ctx)
	if resp != nil {
		result.OData = resp.OData
		result.HttpResponse = resp.Response
	}
	if err != nil {
		return
	}

	var model Secret
	result.Model = &model
	if err = resp.Unmarshal(result.Model); err != nil {
		return
	}

	return result, nil

}
