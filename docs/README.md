# Azure Plugins
<!--
  Include a short overview about the plugin.

  This document is a great location for creating a table of contents for each
  of the components the plugin may provide. This document should load automatically
  when navigating to the docs directory for a plugin.

-->

## Installation

### Using pre-built releases

#### Using the `packer init` command

Starting from version 1.7, Packer supports a new `packer init` command allowing
automatic installation of Packer plugins. Read the
[Packer documentation](https://www.packer.io/docs/commands/init) for more information.

To install this plugin, copy and paste this code into your Packer configuration .
Then, run [`packer init`](https://www.packer.io/docs/commands/init).

```hcl
packer {
  required_plugins {
    azure = {
      version = ">= 1.4.2"
      source  = "github.com/hashicorp/azure"
    }
  }
}
```

#### Manual installation

You can find pre-built binary releases of the plugin [here](https://github.com/hashicorp/packer-plugin-azure/releases).
Once you have downloaded the latest archive corresponding to your target OS,
uncompress it to retrieve the plugin binary file corresponding to your platform.
To install the plugin, please follow the Packer documentation on
[installing a plugin](https://www.packer.io/docs/extending/plugins/#installing-plugins).


#### From Source

If you prefer to build the plugin from its source code, clone the GitHub
repository locally and run the command `go build` from the root
directory. Upon successful compilation, a `packer-plugin-azure` plugin
binary file can be found in the root directory.
To install the compiled plugin, please follow the official Packer documentation
on [installing a plugin](https://www.packer.io/docs/extending/plugins/#installing-plugins).


## Plugin Contents

Packer can create Azure virtual machine images through variety of ways depending on the strategy that you want to use for building the images.

### Builders

- [azure-arm](builders/arm.mdx) - The Azure ARM builder supports building Virtual Hard Disks (VHDs) and
  Managed Images in Azure Resource Manager.
- [azure-chroot](builders/chroot.mdx) - The Azure chroot builder supports building a managed disk image without
  launching a new Azure VM for every build, but instead use an already-running Azure VM.
- [azure-dtl](builders/dtl.mdx) - The Azure DevTest Labs builder builds custom images and uploads them to DevTest Lab image repository automatically.

### Provisioners

- [azure-dtlartifact](provisioners/dtlartifact.mdx) - The Azure DevTest Labs provisioner can be used to apply an artifact to a VM - See [Add an artifact to a VM](https://docs.microsoft.com/en-us/azure/devtest-labs/add-artifact-vm)

