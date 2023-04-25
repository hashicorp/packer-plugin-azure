Type: `azure-chroot`
Artifact BuilderId: `azure.chroot`

The `azure-chroot` builder is able to build Azure managed disk (MD) images. For
more information on managed disks, see [Azure Managed Disks Overview](https://docs.microsoft.com/en-us/azure/virtual-machines/windows/managed-disks-overview).

The difference between this builder and the `azure-arm` builder is that this
builder is able to build a managed disk image without launching a new Azure VM
for every build, but instead use an already-running Azure VM. This can
dramatically speed up image builds. It also allows for more deterministic image
content and enables some capabilities that are not possible with the
`azure-arm` builder.

> **This is an advanced builder** If you're just getting started with Packer,
> it is recommend to start with the [azure-arm builder](https://packer.io/docs/builder/azure-arm),
> which is much easier to use.

## How Does it Work?

This builder works by creating a new MD from either an existing source or from
scratch and attaching it to the (already existing) Azure VM where Packer is
running. Once attached, a [chroot](https://en.wikipedia.org/wiki/Chroot) is set
up and made available to the [provisioners](https://packer.io/docs/provisioners).
After provisioning, the MD is detached, snapshotted and a MD image is created.

Using this process, minutes can be shaved off the image creation process
because Packer does not need to launch a VM instance.

There are some restrictions however:

- The host system must be a similar system (generally the same OS version,
  kernel versions, etc.) as the image being built.
- If the source is a managed disk, it must be made available in the same
  region as the host system.
- The host system SKU has to allow for all of the specified disks to be
  attached.

## Configuration Reference

There are many configuration options available for the builder. We'll start
with authentication parameters, then go over the Azure chroot builder specific
options.

### Authentication options

None of the authentication options are required, but depending on which
ones are specified a different authentication method may be used. See the
[shared Azure builders documentation](https://packer.io/docs/builder/azure) for more
information.

<!-- Code generated from the comments of the Config struct in builder/azure/common/client/config.go; DO NOT EDIT MANUALLY -->

- `cloud_environment_name` (string) - One of Public, China, or
  USGovernment. Defaults to Public. Long forms such as
  USGovernmentCloud and AzureUSGovernmentCloud are also supported.

- `metadata_host` (string) - The Hostname of the Azure Metadata Service
  (for example management.azure.com), used to obtain the Cloud Environment
  when using a Custom Azure Environment. This can also be sourced from the
  ARM_METADATA_HOST Environment Variable.
  Note: CloudEnvironmentName must be set to the requested environment
  name in the list of available environments held in the metadata_host.

- `client_id` (string) - The application ID of the AAD Service Principal.
  Requires either `client_secret`, `client_cert_path` or `client_jwt` to be set as well.

- `client_secret` (string) - A password/secret registered for the AAD SP.

- `client_cert_path` (string) - The path to a PKCS#12 bundle (.pfx file) to be used as the client certificate
  that will be used to authenticate as the specified AAD SP.

- `client_cert_password` (string) - The password for decrypting the client certificate bundle.

- `client_jwt` (string) - A JWT bearer token for client auth (RFC 7523, Sec. 2.2) that will be used
  to authenticate the AAD SP. Provides more control over token the expiration
  when using certificate authentication than when using `client_cert_path`.

- `object_id` (string) - The object ID for the AAD SP. Optional, will be derived from the oAuth token if left empty.

- `tenant_id` (string) - The Active Directory tenant identifier with which your `client_id` and
  `subscription_id` are associated. If not specified, `tenant_id` will be
  looked up using `subscription_id`.

- `subscription_id` (string) - The subscription to use.

- `use_azure_cli_auth` (bool) - Flag to use Azure CLI authentication. Defaults to false.
  CLI auth will use the information from an active `az login` session to connect to Azure and set the subscription id and tenant id associated to the signed in account.
  If enabled, it will use the authentication provided by the `az` CLI.
  Azure CLI authentication will use the credential marked as `isDefault` and can be verified using `az account show`.
  Works with normal authentication (`az login`) and service principals (`az login --service-principal --username APP_ID --password PASSWORD --tenant TENANT_ID`).
  Ignores all other configurations if enabled.

<!-- End of code generated from the comments of the Config struct in builder/azure/common/client/config.go; -->


### Azure chroot builder specific options

#### Required:

<!-- Code generated from the comments of the Config struct in builder/azure/chroot/builder.go; DO NOT EDIT MANUALLY -->

- `source` (string) - One of the following can be used as a source for an image:
  - a shared image version resource ID
  - a managed disk resource ID
  - a publisher:offer:sku:version specifier for plaform image sources.

<!-- End of code generated from the comments of the Config struct in builder/azure/chroot/builder.go; -->


#### Optional:

<!-- Code generated from the comments of the Config struct in builder/azure/chroot/builder.go; DO NOT EDIT MANUALLY -->

- `from_scratch` (bool) - When set to `true`, starts with an empty, unpartitioned disk. Defaults to `false`.

- `command_wrapper` (string) - How to run shell commands. This may be useful to set environment variables or perhaps run
  a command with sudo or so on. This is a configuration template where the `.Command` variable
  is replaced with the command to be run. Defaults to `{{.Command}}`.

- `pre_mount_commands` ([]string) - A series of commands to execute after attaching the root volume and before mounting the chroot.
  This is not required unless using `from_scratch`. If so, this should include any partitioning
  and filesystem creation commands. The path to the device is provided by `{{.Device}}`.

- `mount_options` ([]string) - Options to supply the `mount` command when mounting devices. Each option will be prefixed with
  `-o` and supplied to the `mount` command ran by Packer. Because this command is ran in a shell,
  user discretion is advised. See this manual page for the `mount` command for valid file system specific options.

- `mount_partition` (string) - The partition number containing the / partition. By default this is the first partition of the volume.

- `mount_path` (string) - The path where the volume will be mounted. This is where the chroot environment will be. This defaults
  to `/mnt/packer-amazon-chroot-volumes/{{.Device}}`. This is a configuration template where the `.Device`
  variable is replaced with the name of the device where the volume is attached.

- `post_mount_commands` ([]string) - As `pre_mount_commands`, but the commands are executed after mounting the root device and before the
  extra mount and copy steps. The device and mount path are provided by `{{.Device}}` and `{{.MountPath}}`.

- `chroot_mounts` ([][]string) - This is a list of devices to mount into the chroot environment. This configuration parameter requires
  some additional documentation which is in the "Chroot Mounts" section below. Please read that section
  for more information on how to use this.

- `copy_files` ([]string) - Paths to files on the running Azure instance that will be copied into the chroot environment prior to
  provisioning. Defaults to `/etc/resolv.conf` so that DNS lookups work. Pass an empty list to skip copying
  `/etc/resolv.conf`. You may need to do this if you're building an image that uses systemd.

- `os_disk_size_gb` (int64) - Try to resize the OS disk to this size on the first copy. Disks can only be englarged. If not specified,
  the disk will keep its original size. Required when using `from_scratch`

- `os_disk_storage_account_type` (string) - The [storage SKU](https://docs.microsoft.com/en-us/rest/api/compute/disks/createorupdate#diskstorageaccounttypes)
  to use for the OS Disk. Defaults to `Standard_LRS`.

- `os_disk_cache_type` (string) - The [cache type](https://docs.microsoft.com/en-us/rest/api/compute/images/createorupdate#cachingtypes)
  specified in the resulting image and for attaching it to the Packer VM. Defaults to `ReadOnly`

- `data_disk_storage_account_type` (string) - The [storage SKU](https://docs.microsoft.com/en-us/rest/api/compute/disks/createorupdate#diskstorageaccounttypes)
  to use for datadisks. Defaults to `Standard_LRS`.

- `data_disk_cache_type` (string) - The [cache type](https://docs.microsoft.com/en-us/rest/api/compute/images/createorupdate#cachingtypes)
  specified in the resulting image and for attaching it to the Packer VM. Defaults to `ReadOnly`

- `image_hyperv_generation` (string) - The [Hyper-V generation type](https://docs.microsoft.com/en-us/rest/api/compute/images/createorupdate#hypervgenerationtypes) for Managed Image output.
  Defaults to `V1`.

- `temporary_os_disk_id` (string) - The id of the temporary OS disk that will be created. Will be generated if not set.

- `temporary_os_disk_snapshot_id` (string) - The id of the temporary OS disk snapshot that will be created. Will be generated if not set.

- `temporary_data_disk_id_prefix` (string) - The prefix for the resource ids of the temporary data disks that will be created. The disks will be suffixed with a number. Will be generated if not set.

- `temporary_data_disk_snapshot_id` (string) - The prefix for the resource ids of the temporary data disk snapshots that will be created. The snapshots will be suffixed with a number. Will be generated if not set.

- `skip_cleanup` (bool) - If set to `true`, leaves the temporary disks and snapshots behind in the Packer VM resource group. Defaults to `false`

- `image_resource_id` (string) - The managed image to create using this build.

- `shared_image_destination` (SharedImageGalleryDestination) - The shared image to create using this build.

<!-- End of code generated from the comments of the Config struct in builder/azure/chroot/builder.go; -->


<!-- Code generated from the comments of the Config struct in builder/azure/common/config.go; DO NOT EDIT MANUALLY -->

- `skip_create_image` (bool) - Skip creating the image.
  Useful for setting to `true` during a build test stage.
  Defaults to `false`.

<!-- End of code generated from the comments of the Config struct in builder/azure/common/config.go; -->


#### Output options:

At least one of these options needs to be specified:

- `image_resource_id` (string) - The managed image to create using this build.

- `shared_image_destination` (object) - The shared image to create using this build.

Where `shared_image_destination` is an object with the following properties:

<!-- Code generated from the comments of the SharedImageGalleryDestination struct in builder/azure/chroot/shared_image_gallery_destination.go; DO NOT EDIT MANUALLY -->

- `resource_group` (string) - Resource Group

- `gallery_name` (string) - Gallery Name

- `image_name` (string) - Image Name

- `image_version` (string) - Image Version

<!-- End of code generated from the comments of the SharedImageGalleryDestination struct in builder/azure/chroot/shared_image_gallery_destination.go; -->


<!-- Code generated from the comments of the SharedImageGalleryDestination struct in builder/azure/chroot/shared_image_gallery_destination.go; DO NOT EDIT MANUALLY -->

- `target_regions` ([]TargetRegion) - Target Regions

- `exclude_from_latest` (bool) - Exclude From Latest

<!-- End of code generated from the comments of the SharedImageGalleryDestination struct in builder/azure/chroot/shared_image_gallery_destination.go; -->


And `target_regions` is an array of objects with the following properties:

<!-- Code generated from the comments of the TargetRegion struct in builder/azure/chroot/shared_image_gallery_destination.go; DO NOT EDIT MANUALLY -->

- `name` (string) - Name of the Azure region

<!-- End of code generated from the comments of the TargetRegion struct in builder/azure/chroot/shared_image_gallery_destination.go; -->


<!-- Code generated from the comments of the TargetRegion struct in builder/azure/chroot/shared_image_gallery_destination.go; DO NOT EDIT MANUALLY -->

- `replicas` (int64) - Number of replicas in this region. Default: 1

- `storage_account_type` (string) - Storage account type: Standard_LRS or Standard_ZRS. Default: Standard_ZRS

<!-- End of code generated from the comments of the TargetRegion struct in builder/azure/chroot/shared_image_gallery_destination.go; -->


## Chroot Mounts

The `chroot_mounts` configuration can be used to mount specific devices within
the chroot. By default, the following additional mounts are added into the
chroot by Packer:

- `/proc` (proc)
- `/sys` (sysfs)
- `/dev` (bind to real `/dev`)
- `/dev/pts` (devpts)
- `/proc/sys/fs/binfmt_misc` (binfmt_misc)

These default mounts are usually good enough for anyone and are sane defaults.
However, if you want to change or add the mount points, you may using the
`chroot_mounts` configuration. Here is an example configuration which only
mounts `/prod` and `/dev`:

```json
{
  "chroot_mounts": [
    ["proc", "proc", "/proc"],
    ["bind", "/dev", "/dev"]
  ]
}
```

`chroot_mounts` is a list of a 3-tuples of strings. The three components of the
3-tuple, in order, are:

- The filesystem type. If this is "bind", then Packer will properly bind the
  filesystem to another mount point.

- The source device.

- The mount directory.

## Additional template function

Because this builder runs on an Azure VM, there is an additional template function
available called `vm`, which returns the following VM metadata:

- name
- subscription_id
- resource_group
- location
- resource_id

This function can be used in the configuration templates, for example, use

```text
"{{ vm `subscription_id` }}"
```

to fill in the subscription ID of the VM in any of the configuration options.

## Build Shared Information Variables

This builder generates data that are shared with provisioner and post-processor via build function of [template engine](https://packer.io/docs/templates/legacy_json_templates/engine) for JSON and [contextual variables](https://packer.io/docs/templates/hcl_templates/contextual-variables) for HCL2.

The generated variables available for this builder are:

- `SourceImageName` - The full name of the source image used in the deployment. When using
shared images the resulting name will point to the actual source used to create the said version.
  building the AMI.

Usage example:

**HCL2**

```hcl
// When accessing one of these variables from inside the builder, you need to
// use the golang templating syntax. This is due to an architectural quirk that
// won't be easily resolvable until legacy json templates are deprecated:

{
source "amazon-arm" "basic-example" {
  source = "credativ:Debian:9:latest"
}

// when accessing one of the variables from a provisioner or post-processor, use
// hcl-syntax
post-processor "manifest" {
    output = "manifest.json"
    strip_path = true
    custom_data = {
        source_image_name = "${build.SourceImageName}"
    }
}
```
**JSON**

```json
"post-processors": [
  {
    "type": "manifest",
    "output": "manifest.json",
    "strip_path": true,
    "custom_data": {
      "source_image_name": "{{ build `SourceImageName` }}"
    }
  }
]
```



## Examples

Here are some examples using this builder.
This builder requires privileged actions, such as mounting disks, running
`chroot` and other admin commands. Usually it needs to be run with root
permissions, for example:

```shell-session
$ sudo -E packer build example.pkr.json
```

### Using a VM with a Managed Identity

On a VM with a system-assigned managed identity that has the contributor role
on its own resource group, the following config can be used to create an
updated Debian image:


**HCL2**

```hcl
source "azure-chroot" "example" {
  image_resource_id = "/subscriptions/{{vm `subscription_id`}}/resourceGroups/{{vm `resource_group`}}/providers/Microsoft.Compute/images/MyDebianOSImage-{{timestamp}}"
  source            = "credativ:Debian:9:latest"
}

build {
  sources = ["source.azure-chroot.example"]

  provisioner "shell" {
    inline         = ["apt-get update", "apt-get upgrade -y"]
    inline_shebang = "/bin/sh -x"
  }
}
```


**JSON**

```json
{
  "builders": [
    {
      "type": "azure-chroot",

      "image_resource_id": "/subscriptions/{{vm `subscription_id`}}/resourceGroups/{{vm `resource_group`}}/providers/Microsoft.Compute/images/MyDebianOSImage-{{timestamp}}",
      "source": "credativ:Debian:9:latest"
    }
  ],
  "provisioners": [
    {
      "inline": ["apt-get update", "apt-get upgrade -y"],
      "inline_shebang": "/bin/sh -x",
      "type": "shell"
    }
  ]
}
```


### Using a Service Principal

Here is an example that creates a Debian image with updated packages. Specify
all environment variables (`ARM_CLIENT_ID`, `ARM_CLIENT_SECRET`,
`ARM_SUBSCRIPTION_ID`) to use a service principal.
The identity you choose should have permission to create disks and images and also
to update your VM.
Set the `ARM_IMAGE_RESOURCEGROUP_ID` variable to an existing resource group in the
subscription where the resulting image will be created.

**HCL2**

```hcl
variable "client_id" {
  type = string
}
variable "client_secret" {
  type = string
}
variable "subscription_id" {
  type = string
} variable "resource_group" {
  type = string
}

source "azure-chroot" "basic-example" {
  client_id = var.client_id
  client_secret = var.client_secret
  subscription_id = var.subscription_id

  image_resource_id = "/subscriptions/${var.subscription_id}/resourceGroups/${var.resource_group}/providers/Microsoft.Compute/images/MyDebianOSImage-{{timestamp}}"

  source = "credativ:Debian:9:latest"
}

build {
  sources = ["sources.azure-chroot.basic-example"]

  provisioner "shell" {
    inline = ["apt-get update", "apt-get upgrade -y"]
    inline_shebang = "/bin/sh -x"
  }
}
```

**JSON**

```json
{
  "variables": {
    "client_id": "{{env `ARM_CLIENT_ID`}}",
    "client_secret": "{{env `ARM_CLIENT_SECRET`}}",
    "subscription_id": "{{env `ARM_SUBSCRIPTION_ID`}}",
    "resource_group": "{{env `ARM_IMAGE_RESOURCEGROUP_ID`}}"
  },
  "builders": [
    {
      "type": "azure-chroot",

      "client_id": "{{user `client_id`}}",
      "client_secret": "{{user `client_secret`}}",
      "subscription_id": "{{user `subscription_id`}}",

      "image_resource_id": "/subscriptions/{{user `subscription_id`}}/resourceGroups/{{user `resource_group`}}/providers/Microsoft.Compute/images/MyDebianOSImage-{{timestamp}}",

      "source": "credativ:Debian:9:latest"
    }
  ],
  "provisioners": [
    {
      "inline": ["apt-get update", "apt-get upgrade -y"],
      "inline_shebang": "/bin/sh -x",
      "type": "shell"
    }
  ]
}
```
