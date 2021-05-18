//go:generate packer-sdc struct-markdown
//go:generate packer-sdc mapstructure-to-hcl2 -type DatasourceOutput,Config
package keyvaultsecret

import (
	"context"
	"fmt"
	"log"
	"net/url"

	"github.com/Azure/azure-sdk-for-go/services/keyvault/v7.1/keyvault"
	"github.com/Azure/go-autorest/autorest"
	"github.com/hashicorp/hcl/v2/hcldec"
	"github.com/hashicorp/packer-plugin-azure/builder/azure/common/client"
	"github.com/hashicorp/packer-plugin-sdk/hcl2helper"
	"github.com/hashicorp/packer-plugin-sdk/template/config"
	"github.com/zclconf/go-cty/cty"

	packersdk "github.com/hashicorp/packer-plugin-sdk/packer"
)

type Config struct {
	// Specifies the name of the Key Vault Secret.
	Name string `mapstructure:"name" required:"true"`

	// Specifies the ID of the Key Vault instance where the Secret resides.
	KeyvaultId string `mapstructure:"keyvault_id" required:"true"`

	// Authentication via OAUTH
	ClientConfig client.Config `mapstructure:",squash"`
}

type Datasource struct {
	config Config
}

type DatasourceOutput struct {
	// The Key Vault Secret ID.
	Id string `mapstructure:"id"`
	// The value of the Key Vault Secret.
	Value string `mapstructure:"value"`
	// The content type for the Key Vault Secret.
	ContentType string `mapstructure:"content_type"`
	// Any tags assigned to this resource.
	Tags map[string]*string `mapstructure:"tags"`
}

func (d *Datasource) ConfigSpec() hcldec.ObjectSpec {
	return d.config.FlatMapstructure().HCL2Spec()
}

func (d *Datasource) Configure(raws ...interface{}) error {
	err := config.Decode(&d.config, nil, raws...)
	if err != nil {
		return err
	}

	var errs *packersdk.MultiError
	if d.config.Name == "" {
		errs = packersdk.MultiErrorAppend(errs, fmt.Errorf("a 'name' must be provided"))
	}
	if d.config.KeyvaultId == "" {
		errs = packersdk.MultiErrorAppend(errs, fmt.Errorf("a 'keyvault_id' must be provided"))
	}

	err = d.config.ClientConfig.SetDefaultValues()
	if err != nil {
		errs = packersdk.MultiErrorAppend(errs, err)
	}

	d.config.ClientConfig.Validate(errs)
	if errs != nil && len(errs.Errors) > 0 {
		return errs
	}

	return nil
}

func (d *Datasource) OutputSpec() hcldec.ObjectSpec {
	return (&DatasourceOutput{}).FlatMapstructure().HCL2Spec()
}

// We need to stub packersdk ui "say" function. UI is not available from this package, I guess this is the best we can do for now
func logSay(s string) {
	log.Println("[DEBUG] packer-datasource-azure-keyvaultsecret:", s)
}

func getVaultUrl(keyvaultURL *url.URL, vaultName string) string {
	return fmt.Sprintf("%s://%s.%s/", keyvaultURL.Scheme, vaultName, keyvaultURL.Host)
}

func (d *Datasource) Execute() (cty.Value, error) {
	err := d.config.ClientConfig.FillParameters()
	if err != nil {
		return cty.NullVal(cty.EmptyObject), err
	}

	keyVaultURL, err := url.Parse(d.config.ClientConfig.CloudEnvironment().KeyVaultEndpoint)
	if err != nil {
		return cty.NullVal(cty.EmptyObject), err
	}

	/* Get token from client configuration */
	_, servicePrincipalTokenVault, err := d.config.ClientConfig.GetServicePrincipalTokens(logSay)
	if err != nil {
		return cty.NullVal(cty.EmptyObject), err
	}

	/* Configure keyvault client */
	basicClient := keyvault.New()
	basicClient.Authorizer = autorest.NewBearerAuthorizer(servicePrincipalTokenVault)

	/* Get secret value */
	secret, err := basicClient.GetSecret(context.TODO(), getVaultUrl(keyVaultURL, d.config.KeyvaultId), d.config.Name, "")
	if err != nil {
		return cty.NullVal(cty.EmptyObject), err
	}

	/* Build output object */
	var output DatasourceOutput
	if secret.Value != nil {
		output.Value = *secret.Value
	}
	if secret.ID != nil {
		output.Id = *secret.ID
	}
	if secret.ContentType != nil {
		output.ContentType = *secret.ContentType
	}
	if secret.Tags != nil {
		output.Tags = secret.Tags
	}

	return hcl2helper.HCL2ValueFromConfig(output, d.OutputSpec()), nil
}
