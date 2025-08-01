// Copyright (c) HashiCorp, Inc.
// SPDX-License-Identifier: MPL-2.0

//go:generate packer-sdc struct-markdown
//go:generate packer-sdc mapstructure-to-hcl2 -type Config,DatasourceOutput
 
package keyvaultsecret

import (
	"context"
	"encoding/json"
	"errors"
	"fmt"
	"os"

	"github.com/hashicorp/hcl/v2/hcldec"
	"github.com/hashicorp/packer-plugin-azure/builder/azure/common/log"
	"github.com/hashicorp/packer-plugin-sdk/common"
	"github.com/hashicorp/packer-plugin-sdk/hcl2helper"
	packersdk "github.com/hashicorp/packer-plugin-sdk/packer"
	"github.com/hashicorp/packer-plugin-sdk/template/config"

	"github.com/zclconf/go-cty/cty"

	"github.com/Azure/azure-sdk-for-go/sdk/azcore"
	"github.com/Azure/azure-sdk-for-go/sdk/azidentity"
	"github.com/Azure/azure-sdk-for-go/sdk/keyvault/azsecrets"
)

type Config struct {
	common.PackerConfig `mapstructure:",squash"`

	// The name of the Azure Key Vault.
	VaultName string `mapstructure:"vault_name" required:"true"`
	// The name of the secret to fetch from the Azure Key Vault.
	SecretName string `mapstructure:"secret_name" required:"true"`
	// The version of the secret to fetch. If not provided, the latest version will be used.
	Version string `mapstructure:"version"`

	// Optional fields for authentication.
	// If not provided, the DefaultAzureCredential will be used.
	// This includes environment variables, managed identity, etc.

	TenantID     string `mapstructure:"tenant_id"`
	ClientID     string `mapstructure:"client_id"`
	ClientSecret string `mapstructure:"client_secret"`
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

	var errs *packersdk.MultiError

	if d.config.VaultName == "" {
		errs = packersdk.MultiErrorAppend(errs, errors.New("a 'vault_name' must be specified"))
	}
	if d.config.SecretName == "" {
		errs = packersdk.MultiErrorAppend(errs, errors.New("a 'secret_name' must be specified"))
	}

	if d.config.TenantID == "" {
		d.config.TenantID = os.Getenv("AZURE_TENANT_ID")
	}
	if d.config.ClientID == "" {
		d.config.ClientID = os.Getenv("AZURE_CLIENT_ID")
	}
	if d.config.ClientSecret == "" {
		d.config.ClientSecret = os.Getenv("AZURE_CLIENT_SECRET")
	}

	if errs != nil && len(errs.Errors) > 0 {
		return errs
	}
	return nil
}

func (d *Datasource) Execute() (cty.Value, error) {

	var cred azcore.TokenCredential
	var err error

	if d.config.TenantID != "" && d.config.ClientID == "" && d.config.ClientSecret == "" {
		log.Printf("Using ClientSecretCredential for vault %q", d.config.VaultName)
		cred, err = azidentity.NewClientSecretCredential(d.config.TenantID, d.config.ClientID, d.config.ClientSecret, nil)
	} else {
		log.Printf("Using DefaultAzureCredential for vault %q", d.config.VaultName)
		cred, err = azidentity.NewDefaultAzureCredential(nil)
	}
	if err != nil {
		log.Printf("failed to obtain a credential: %v", err)
		return cty.NullVal(cty.EmptyObject), fmt.Errorf("failed to obtain a credential for vault %q: %w", d.config.VaultName, err)
	}

	vaultURI := fmt.Sprintf("https://%s.vault.azure.net", d.config.VaultName)
	// Establish a connection to the Key Vault client
	client, err := azsecrets.NewClient(vaultURI, cred, nil)
	if err != nil {
		log.Printf("failed to create a Key Vault client: %v", err)
		return cty.NullVal(cty.EmptyObject), fmt.Errorf("failed to create Key Vault client for vault %q: %w", d.config.VaultName, err)
	}

	resp, err := client.GetSecret(context.TODO(), d.config.SecretName, d.config.Version, nil)
	if err != nil {
		log.Printf("failed to get the secret: %v", err)
		return cty.NullVal(cty.EmptyObject), fmt.Errorf("failed to get secret %q from vault %q: %w", d.config.SecretName, d.config.VaultName, err)
	}

	jsonResp, err := json.Marshal(resp.SecretBundle)
	if err != nil {
		log.Printf("failed to marshal secret bundle: %v", err)
		return cty.NullVal(cty.EmptyObject), fmt.Errorf("failed to marshal secret bundle for secret %q in vault %q: %w", d.config.SecretName, d.config.VaultName, err)
	}

	output := DatasourceOutput{
		Response: string(jsonResp),
		Value:    *resp.Value,
	}
	return hcl2helper.HCL2ValueFromConfig(output, d.OutputSpec()), nil
}
