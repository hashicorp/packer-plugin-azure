The Key Vault Secret data source provides information about an Azure Key Vault's secret,
including its value and metadata.

-> **Note:** Data sources is a feature exclusively available to HCL2 templates.

Basic examples of usage:

```hcl
data "azure-keyvaultsecret" "basic-example" {
  vault_name = "packer-test-vault"
  secret_name = "test-secret"
}

# usage example of the data source output
locals {
  value = data.azure-keyvaultsecret.basic-example.value
  payload = data.azure-keyvaultsecret.basic-example.payload
}
```

Reading key-value pairs from JSON back into a native Packer map can be accomplished
with the [jsondecode() function](/packer/docs/templates/hcl_templates/functions/encoding/jsondecode).

## Configuration Reference

### Required

<!-- Code generated from the comments of the Config struct in datasource/keyvaultsecret/data.go; DO NOT EDIT MANUALLY -->

- `vault_name` (string) - The name of the Azure Key Vault.

- `secret_name` (string) - The name of the secret to fetch from the Azure Key Vault.

<!-- End of code generated from the comments of the Config struct in datasource/keyvaultsecret/data.go; -->


### Optional

<!-- Code generated from the comments of the Config struct in datasource/keyvaultsecret/data.go; DO NOT EDIT MANUALLY -->

- `version` (string) - The version of the secret to fetch. If not provided, the latest version will be used.

<!-- End of code generated from the comments of the Config struct in datasource/keyvaultsecret/data.go; -->


## Output Data

<!-- Code generated from the comments of the DatasourceOutput struct in datasource/keyvaultsecret/data.go; DO NOT EDIT MANUALLY -->

- `response` (string) - The raw string response of the secret version.

- `value` (string) - The value extracted using the 'key', if provided.

<!-- End of code generated from the comments of the DatasourceOutput struct in datasource/keyvaultsecret/data.go; -->


## Authentication

To authenticate with Azure Key Vault, this data-source supports everything the plugin does.
To get more information on this, refer to the plugin's description page, under
the [authentication](/packer/integrations/hashicorp/azure#authentication) section.
