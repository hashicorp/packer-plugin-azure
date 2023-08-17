The Azure plugin can be used with HashiCorp Packer to create custom images on Azure.
To do so, the plugin exposes multiple builders, among which you can choose the one most adapted to your workflow.

## Installation

To install this plugin, copy and paste this code into your Packer configuration, then run [`packer init`](https://www.packer.io/docs/commands/init).

```hcl
packer {
  required_plugins {
    azure = {
      source  = "github.com/hashicorp/azure"
      version = "~> 2"
    }
  }
}
```

Alternatively, you can use `packer plugins install` to manage installation of this plugin.

```sh
$ packer plugins install github.com/hashicorp/azure
```

## Components

Packer can create Azure virtual machine images through variety of ways depending on the strategy that you want to use for building the images.

### Builders

- [azure-arm](/packer/integrations/hashicorp/azure/latest/components/builder/arm) - The Azure ARM builder supports building Virtual Hard Disks (VHDs) and
  Managed Images in Azure Resource Manager.
- [azure-chroot](/packer/integrations/hashicorp/azure/latest/components/builder/chroot) - The Azure chroot builder supports building a managed disk image without
  launching a new Azure VM for every build, but instead use an already-running Azure VM.
- [azure-dtl](/packer/integrations/hashicorp/azure/latest/components/builder/dtl) - The Azure DevTest Labs builder builds custom images and uploads them to DevTest Lab image repository automatically.

### Provisioners

- [azure-dtlartifact](/packer/integrations/hashicorp/azure/latest/components/provisioner/dtlartifact) - The Azure DevTest Labs provisioner can be used to apply an artifact to a VM - Refer to [Add an artifact to a VM](https://docs.microsoft.com/en-us/azure/devtest-labs/add-artifact-vm)

## Authentication

<!-- Code generated from the comments of the Config struct in builder/azure/common/client/config.go; DO NOT EDIT MANUALLY -->

Config allows for various ways to authenticate Azure clients.  When
`client_id` and `subscription_id` are specified in addition to one and only
one of the following: `client_secret`, `client_jwt`, `client_cert_path` --
Packer will use the specified Azure Active Directory (AAD) Service Principal
(SP).
If none ofthese options are specified, Packer will attempt to use the Managed Identity
and subscription of the VM that Packer is running on.  This will only work if
Packer is running on an Azure VM with either a System Assigned Managed
Identity or User Assigned Managed Identity.

<!-- End of code generated from the comments of the Config struct in builder/azure/common/client/config.go; -->


### Managed Identity

If you're running Packer on an Azure VM with a [managed
identity](https://packer.io/docs/builders/azure#azure-managed-identity) you
don't need to specify any additional configuration options. As Packer will
attempt to use the Managed Identity and subscription of the VM that Packer is
running on.

You can use a different subscription if you set `subscription_id`.  If your VM
has multiple user assigned managed identities you will need to set `client_id`
too.

### Interactive User Authentication

To use interactive user authentication, you should specify
`use_interactive_auth` only.  Packer will use cached credentials or redirect you
to a website to log in.

### Service Principal

To use a [service principal](https://packer.io/docs/builders/azure#azure-active-directory-service-principal)
you should specify `subscription_id`, `client_id` and one of `client_secret`,
`client_cert_path` or `client_jwt`.

- `subscription_id` (string) - Subscription under which the build will be
  performed. **The service principal specified in `client_id` must have full
  access to this subscription, unless build_resource_group_name option is
  specified in which case it needs to have owner access to the existing
  resource group specified in build_resource_group_name parameter.**

- `client_id` (string) - The Active Directory service principal associated with
  your builder.

- `client_secret` (string) - The password or secret for your service principal.

- `client_cert_path` (string) - The location of a PEM file containing a
  certificate and private key for service principal.

- `client_cert_token_timeout` (duration string | ex: "1h30m12s") - How long to set the expire time on the token created when using
  `client_cert_path`.

- `client_jwt` (string) - The bearer JWT assertion signed using a certificate
  associated with your service principal principal. See [Azure Active
  Directory docs](https://docs.microsoft.com/en-us/azure/active-directory/develop/active-directory-certificate-credentials)
  for more information.
