This repo's acceptance tests require a bit of setup to run, a subscription, and app registration, and several resources must be created.  To make this process easier to manage in CI and easier for developers to quickly test changes, this directory contains terraform configuration to create the requires resources.

## Creating Azure Resources

First you need an Azure Subscription, it is also reccomended to also have an app registration created with client/secret authentication setup, as this is required for the acceptance tests themselves.

Authenticate to Azure either through the Azure CLI or with client/secret auth with the following environment variables `ARM_CLIENT_ID`, `ARM_CLIENT_SECRET`, `ARM_TENANT_ID`, and `ARM_SUBSCRIPTION_ID`.  The Tenant ID can also be fetched using CLI auth as a backup.

The default resource group is named `packer-acceptance-test` with a storage account named `packeracctest`, however you can use variables TF `resource_group_name` and `storage_account_name` to change that to anything

For example 
```
terraform apply -var "resource_group_name=cool-group" -var "storage_account_name=coolblobstore"
```

Note that Azure storage account names must not contain special characters

