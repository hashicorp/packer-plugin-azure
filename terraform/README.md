# Terraform Acceptance Test Infrastructure

This repo's acceptance tests require a bit of setup to run, a subscription, and app registration, and several resources must be created. To make this process easier to manage in CI and easier for developers to quickly test changes, this directory contains terraform configuration to create the required resources.

## Creating Azure Resources

First you need an Azure Subscription, it is also recommended to also have an app registration created with client/secret authentication setup, as this is required for the acceptance tests themselves.

Authenticate to Azure using the Azure CLI for a service principal

The default resource group is named `packer-acceptance-test` with a storage account named `packeracctest`, however you can use variables TF `resource_group_name` and `storage_account_name` to change that to anything. Resource names are automatically suffixed to avoid conflicts between concurrent runs; you can supply `resource_suffix` to control the suffix.

For example
```
terraform apply -var "resource_group_name=cool-group" -var "storage_account_name=coolblobstore"
```

Note that Azure storage account names must not contain special characters.

## Terraform Outputs

The Terraform program exposes outputs for the resource names that the ARM acceptance tests reference through environment variables. These outputs are intended for debugging and local test setup.

They help with two common problems:
- confirming that the GitHub Actions workflow is computing the same names that Terraform actually created
- quickly exporting or inspecting the exact resource names when running acceptance tests locally

Current outputs:
- `resource_group_name`
- `storage_account_name`
- `storage_container_name`
- `resource_prefix`
- `resource_suffix`
- `virtual_network_name`
- `virtual_network_subnet_name`

You can inspect them with:
```
terraform output
```

Or fetch a single value with:
```
terraform output -raw virtual_network_name
```

These outputs only cover resources created by this Terraform program. For example, `ARM_TEMP_RESOURCE_GROUP_NAME` is still configured outside Terraform and is therefore not exposed as an output.
