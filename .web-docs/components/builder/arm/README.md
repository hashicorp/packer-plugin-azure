Type: `azure-arm`
Artifact BuilderId: `Azure.ResourceManagement.VMImage`

Packer supports building Virtual Hard Disks (VHDs) and Managed Images in [Azure Resource
Manager](https://azure.microsoft.com/en-us/documentation/articles/resource-group-overview/).
Azure provides new users a [`$200` credit for the first 30
days](https://azure.microsoft.com/en-us/free/); after which you will incur
costs for VMs built and stored using Packer.

Azure uses a combination of OAuth and Active Directory to authorize requests to
the ARM API. Learn how to [authorize access to
ARM](https://packer.io/docs/builder/azure#authentication-for-azure).

The documentation below references command output from the [Azure
CLI](https://azure.microsoft.com/en-us/documentation/articles/xplat-cli-install/).

## Configuration Reference

There are many configuration options available for the builder. We'll start
with authentication parameters, then go over the Azure ARM builder specific
options. In addition to the options listed here, a [communicator](https://packer.io/docs/templates/legacy_json_templates/communicator) can be configured for this builder.

### Azure ARM builder specific options

The Azure builder can create a VHD, a managed image, and a Shared Image Gallery version.
All builds must start from a managed image source, such as an Azure Marketplace image or another custom managed image, regardless of the desired output artifact.

### Required:

<!-- Code generated from the comments of the Config struct in builder/azure/arm/config.go; DO NOT EDIT MANUALLY -->

- `image_publisher` (string) - Name of the publisher to use for your base image (Azure Marketplace Images only). See
  [documentation](https://docs.microsoft.com/en-us/cli/azure/vm/image)
  for details.
  
  CLI example `az vm image list-publishers --location westus`

- `image_offer` (string) - Name of the publisher's offer to use for your base image (Azure Marketplace Images only). See
  [documentation](https://docs.microsoft.com/en-us/cli/azure/vm/image)
  for details.
  
  CLI example
  `az vm image list-offers --location westus --publisher Canonical`

- `image_sku` (string) - SKU of the image offer to use for your base image (Azure Marketplace Images only). See
  [documentation](https://docs.microsoft.com/en-us/cli/azure/vm/image)
  for details.
  
  CLI example
  `az vm image list-skus --location westus --publisher Canonical --offer UbuntuServer`

- `image_url` (string) - URL to a custom VHD to use for your base image. If this value is set,
  image_publisher, image_offer, image_sku, or image_version should not be set.

- `custom_managed_image_name` (string) - Name of a custom managed image to use for your base image. If this value is set, do
  not set image_publisher, image_offer, image_sku, or image_version.
  If this value is set, the option
  `custom_managed_image_resource_group_name` must also be set. See
  [documentation](https://docs.microsoft.com/en-us/azure/storage/storage-managed-disks-overview#images)
  to learn more about managed images.

- `custom_managed_image_resource_group_name` (string) - Name of a custom managed image's resource group to use for your base image. If this
  value is set, image_publisher, image_offer, image_sku, or image_version should not be set.
  If this value is set, the option
  `custom_managed_image_name` must also be set. See
  [documentation](https://docs.microsoft.com/en-us/azure/storage/storage-managed-disks-overview#images)
  to learn more about managed images.

<!-- End of code generated from the comments of the Config struct in builder/azure/arm/config.go; -->


When creating a VHD the following additional options are required:

- `capture_container_name` (string) - Destination container name. Essentially
  the "directory" where your VHD will be organized in Azure. The captured
  VHD's URL will be
  `https://<storage_account>.blob.core.windows.net/<capture_container_name>/<capture_name_prefix><os_disk_name>.vhd`.

- `capture_name_prefix` (string) - VHD prefix. The final artifacts will be
  named `<PREFIX><osDisk>.vhd`.

- `resource_group_name` (string) - Resource group under which the final
  artifact will be stored.

- `storage_account` (string) - Storage account under which the final artifact
  will be stored.

When creating a managed image the following additional options are required:

- `managed_image_name` (string) - Specify the managed image name where the
  result of the Packer build will be saved. The image name must not exist
  ahead of time, and will not be overwritten. If this value is set, the value
  `managed_image_resource_group_name` must also be set. See
  [documentation](https://docs.microsoft.com/en-us/azure/storage/storage-managed-disks-overview#images)
  to learn more about managed images.

- `managed_image_resource_group_name` (string) - Specify the managed image
  resource group name where the result of the Packer build will be saved. The
  resource group must already exist. If this value is set, the value
  `managed_image_name` must also be set. See
  [documentation](https://docs.microsoft.com/en-us/azure/storage/storage-managed-disks-overview#images)
  to learn more about managed images.

Creating a managed image using a [Shared Gallery image](https://azure.microsoft.com/en-us/blog/announcing-the-public-preview-of-shared-image-gallery/) as the source can be achieved by specifying the [shared_image_gallery](#shared-image-gallery) configuration option.

#### Resource Group Usage

The Azure builder can either provision resources into a new resource group that
it controls (default) or an existing one. The advantage of using a packer
defined resource group is that failed resource cleanup is easier because you
can simply remove the entire resource group, however this means that the
provided credentials must have permission to create and remove resource groups.
By using an existing resource group you can scope the provided credentials to
just this group, however failed builds are more likely to leave unused
artifacts.

To have Packer create a resource group you **must** provide:

- `location` (string) Azure datacenter in which your VM will build.

  CLI example `az account list-locations`

and optionally:

- `temp_resource_group_name` (string) name assigned to the temporary resource
  group created during the build. If this value is not set, a random value
  will be assigned. This resource group is deleted at the end of the build.

To use an existing resource group you **must** provide:

- `build_resource_group_name` (string) - Specify an existing resource group
  to run the build in.

Providing `temp_resource_group_name` or `location` in combination with
`build_resource_group_name` is not allowed.

### Optional:

<!-- Code generated from the comments of the Config struct in builder/azure/arm/config.go; DO NOT EDIT MANUALLY -->

- `user_assigned_managed_identities` ([]string) - A list of one or more fully-qualified resource IDs of user assigned
  managed identities to be configured on the VM.
  See [documentation](https://docs.microsoft.com/en-us/azure/active-directory/managed-identities-azure-resources/how-to-use-vm-token)
  for how to acquire tokens within the VM.
  To assign a user assigned managed identity to a VM, the provided account or service principal must have [Managed Identity Operator](https://docs.microsoft.com/en-us/azure/role-based-access-control/built-in-roles#managed-identity-operator)
  and [Virtual Machine Contributor](https://docs.microsoft.com/en-us/azure/role-based-access-control/built-in-roles#virtual-machine-contributor) role assignments.

- `capture_name_prefix` (string) - VHD prefix.

- `capture_container_name` (string) - Destination container name. This must be created before the build in the storage account

- `shared_image_gallery` (SharedImageGallery) - Use a [Shared Gallery
  image](https://azure.microsoft.com/en-us/blog/announcing-the-public-preview-of-shared-image-gallery/)
  as the source for this build.
  *VHD targets are incompatible with this build type*
  When using shared_image_gallery as a source, image_publisher, image_offer, image_sku, image_version, and custom_managed_image_name should not be set.
  
  In JSON
  ```json
  "shared_image_gallery": {
      "subscription": "00000000-0000-0000-0000-00000000000",
      "resource_group": "ResourceGroup",
      "gallery_name": "GalleryName",
      "image_name": "ImageName",
      "image_version": "1.0.0",
  }
  ```
  In HCL2
  ```hcl
  shared_image_gallery {
      subscription = "00000000-0000-0000-0000-00000000000"
      resource_group = "ResourceGroup"
      gallery_name = "GalleryName"
      image_name = "ImageName"
      image_version = "1.0.0"
  }
  ```

- `shared_image_gallery_destination` (SharedImageGalleryDestination) - The name of the Shared Image Gallery under which the managed image will be published as Shared Gallery Image version.
  A managed image target can also be set when using a shared image gallery destination

- `shared_image_gallery_timeout` (duration string | ex: "1h5m2s") - How long to wait for an image to be published to the shared image
  gallery before timing out. If your Packer build is failing on the
  Publishing to Shared Image Gallery step with the error `Original Error:
  context deadline exceeded`, but the image is present when you check your
  Azure dashboard, then you probably need to increase this timeout from
  its default of "60m" (valid time units include `s` for seconds, `m` for
  minutes, and `h` for hours.)

- `shared_gallery_image_version_end_of_life_date` (string) - The end of life date (2006-01-02T15:04:05.99Z) of the gallery Image Version. This property
  can be used for decommissioning purposes.

- `shared_image_gallery_replica_count` (int64) - The number of replicas of the Image Version to be created per region defined in `replication_regions`.
  Users using `target_region` blocks can specify individual replica counts per region using the `replicas` field.

- `shared_gallery_image_version_exclude_from_latest` (bool) - If set to true, Virtual Machines deployed from the latest version of the
  Image Definition won't use this Image Version.
  
  In JSON
  ```json
  "shared_image_gallery_destination": {
      "subscription": "00000000-0000-0000-0000-00000000000",
      "resource_group": "ResourceGroup",
      "gallery_name": "GalleryName",
      "image_name": "ImageName",
      "image_version": "1.0.0",
      "replication_regions": ["regionA", "regionB", "regionC"],
      "storage_account_type": "Standard_LRS"
  },
  "shared_image_gallery_timeout": "60m",
  "shared_gallery_image_version_end_of_life_date": "2006-01-02T15:04:05.99Z",
  "shared_gallery_image_version_replica_count": 1,
  "shared_gallery_image_version_exclude_from_latest": true
  ```
  
  In HCL2
  ```hcl
  shared_image_gallery_destination {
      subscription = "00000000-0000-0000-0000-00000000000"
      resource_group = "ResourceGroup"
      gallery_name = "GalleryName"
      image_name = "ImageName"
      image_version = "1.0.0"
      storage_account_type = "Standard_LRS"
      target_region {
        name = "regionA"
      }
      target_region {
        name = "regionB"
      }
      target_region {
        name = "regionC"
      }
  }
  shared_image_gallery_timeout = "60m"
  shared_gallery_image_version_end_of_life_date = "2006-01-02T15:04:05.99Z"
  shared_gallery_image_version_replica_count = 1
  shared_gallery_image_version_exclude_from_latest = true
  ```

- `image_version` (string) - Specify a specific version of an OS to boot from.
  Defaults to `latest`. There may be a difference in versions available
  across regions due to image synchronization latency. To ensure a consistent
  version across regions set this value to one that is available in all
  regions where you are deploying.
  
  CLI example
  `az vm image list --location westus --publisher Canonical --offer UbuntuServer --sku 16.04.0-LTS --all`

- `location` (string) - Azure datacenter in which your VM will build.

- `vm_size` (string) - Size of the VM used for building. This can be changed when you deploy a
  VM from your VHD. See
  [pricing](https://azure.microsoft.com/en-us/pricing/details/virtual-machines/)
  information. Defaults to `Standard_A1`.
  
  CLI example `az vm list-sizes --location westus`

- `spot` (Spot) - If set use a spot instance during build; spot configuration settings only apply to the virtual machine launched by Packer and will not be persisted on the resulting image artifact.
  
  Following is an example.
  
  In JSON
  
  ```json
  "spot": {
      "eviction_policy": "Delete",
  	   "max_price": "0.4",
  }
  ```
  
  In HCL2
  
  ```hcl
  spot {
      eviction_policy = "Delete"
      max_price = "0.4"
  }
  ```

- `managed_image_resource_group_name` (string) - Specify the managed image resource group name where the result of the
  Packer build will be saved. The resource group must already exist. If
  this value is set, the value managed_image_name must also be set. See
  documentation to learn more about managed images.

- `managed_image_name` (string) - Specify the managed image name where the result of the Packer build will
  be saved. The image name must not exist ahead of time, and will not be
  overwritten. If this value is set, the value
  managed_image_resource_group_name must also be set. See documentation to
  learn more about managed images.

- `managed_image_storage_account_type` (string) - Specify the storage account
  type for a managed image. Valid values are Standard_LRS and Premium_LRS.
  The default is Standard_LRS.

- `managed_image_os_disk_snapshot_name` (string) - If
  managed_image_os_disk_snapshot_name is set, a snapshot of the OS disk
  is created with the same name as this value before the VM is captured.

- `managed_image_data_disk_snapshot_prefix` (string) - If
  managed_image_data_disk_snapshot_prefix is set, snapshot of the data
  disk(s) is created with the same prefix as this value before the VM is
  captured.

- `keep_os_disk` (bool) - If
  keep_os_disk is set, the OS disk is not deleted.
  The default is false.

- `managed_image_zone_resilient` (bool) - Store the image in zone-resilient storage. You need to create it in a
  region that supports [availability
  zones](https://docs.microsoft.com/en-us/azure/availability-zones/az-overview).

- `azure_tags` (map[string]string) - Name/value pair tags to apply to every resource deployed i.e. Resource
  Group, VM, NIC, VNET, Public IP, KeyVault, etc. The user can define up
  to 50 tags. Tag names cannot exceed 512 characters, and tag values
  cannot exceed 256 characters.

- `azure_tag` ([]{name string, value string}) - Same as [`azure_tags`](#azure_tags) but defined as a singular repeatable block
  containing a `name` and a `value` field. In HCL2 mode the
  [`dynamic_block`](/packer/docs/templates/hcl_templates/expressions#dynamic-blocks)
  will allow you to create those programmatically.

- `resource_group_name` (string) - Resource group under which the final artifact will be stored.

- `storage_account` (string) - Storage account under which the final artifact will be stored.

- `temp_compute_name` (string) - temporary name assigned to the VM. If this
  value is not set, a random value will be assigned. Knowing the resource
  group and VM name allows one to execute commands to update the VM during a
  Packer build, e.g. attach a resource disk to the VM.

- `temp_nic_name` (string) - temporary name assigned to the Nic. If this
  value is not set, a random value will be assigned. Being able to assign a custom
  nicname could ease deployment if naming conventions are used.

- `temp_resource_group_name` (string) - name assigned to the temporary resource group created during the build.
  If this value is not set, a random value will be assigned. This resource
  group is deleted at the end of the build.

- `build_resource_group_name` (string) - Specify an existing resource group to run the build in.

- `build_key_vault_name` (string) - Specify an existing key vault to use for uploading the certificate for the
  instance to connect.

- `build_key_vault_secret_name` (string) - Specify the secret name to use for the certificate created in the key vault.

- `build_key_vault_sku` (string) - Specify the KeyVault SKU to create during the build. Valid values are
  standard or premium. The default value is standard.

- `skip_create_build_key_vault` (bool) - Skip creating the build key vault during Windows build.
  This is useful for cases when a subscription has policy restrictions on key vault resources.
  If true, you have to provide an alternate method to setup WinRM.
  You can find examples of this in the `example/windows_skip_key_vault` directory.
  These examples provide a minimal setup needed to get a working solution with the `skip_create_build_key_vault` flag.
  This script may require changes depending on the version of Windows used,
  or if any changes are made to the existing versions of Windows that impact creation of a WinRM listener.
  For more information about custom scripts or user data, please refer to the docs:
  * [Custom Script documentation](https://docs.microsoft.com/en-us/azure/virtual-machines/extensions/custom-script-windows)
  * [User Data for Azure VM documentation](https://learn.microsoft.com/en-us/azure/virtual-machines/user-data)

- `disk_encryption_set_id` (string) - Specify the Disk Encryption Set ID to use to encrypt the OS and data disks created with the VM during the build
  Only supported when publishing to Shared Image Galleries, without a managed image
  The disk encryption set ID can be found in the properties tab of a disk encryption set on the Azure Portal, and is labeled as its resource ID
  https://learn.microsoft.com/en-us/azure/virtual-machines/image-version-encryption

- `private_virtual_network_with_public_ip` (bool) - This value allows you to
  set a virtual_network_name and obtain a public IP. If this value is not
  set and virtual_network_name is defined Packer is only allowed to be
  executed from a host on the same subnet / virtual network.

- `virtual_network_name` (string) - Use a pre-existing virtual network for the
  VM. This option enables private communication with the VM, no public IP
  address is used or provisioned (unless you set
  private_virtual_network_with_public_ip).

- `virtual_network_subnet_name` (string) - If virtual_network_name is set,
  this value may also be set. If virtual_network_name is set, and this
  value is not set the builder attempts to determine the subnet to use with
  the virtual network. If the subnet cannot be found, or it cannot be
  disambiguated, this value should be set.

- `virtual_network_resource_group_name` (string) - If virtual_network_name is
  set, this value may also be set. If virtual_network_name is set, and
  this value is not set the builder attempts to determine the resource group
  containing the virtual network. If the resource group cannot be found, or
  it cannot be disambiguated, this value should be set.

- `custom_data_file` (string) - Specify a file containing custom data to inject into the cloud-init
  process. The contents of the file are read and injected into the ARM
  template. The custom data will be passed to cloud-init for processing at
  the time of provisioning. See
  [documentation](http://cloudinit.readthedocs.io/en/latest/topics/examples.html)
  to learn more about custom data, and how it can be used to influence the
  provisioning process.

- `custom_data` (string) - Specify a Base64-encode custom data to apply when launching the instance.
  Note that you need to be careful about escaping characters due to the templates being JSON.
  The custom data will be passed to cloud-init for processing at
  the time of provisioning. See
  [documentation](http://cloudinit.readthedocs.io/en/latest/topics/examples.html)
  to learn more about custom data, and how it can be used to influence the
  provisioning process.

- `user_data_file` (string) - Specify a file containing user data to inject into the cloud-init
  process. The contents of the file are read and injected into the ARM
  template. The user data will be available from the provision until the vm is
  deleted. Any application on the virtual machine can access the user data
  from the Azure Instance Metadata Service (IMDS) after provision.
  See [documentation](https://docs.microsoft.com/en-us/azure/virtual-machines/user-data)
  to learn more about user data.

- `user_data` (string) - Specify a Base64-encode user data to apply
  Note that you need to be careful about escaping characters due to the templates being JSON.
  The user data will be available from the provision until the vm is
  deleted. Any application on the virtual machine can access the user data
  from the Azure Instance Metadata Service (IMDS) after provision.
  See [documentation](https://docs.microsoft.com/en-us/azure/virtual-machines/user-data)
  to learn more about user data.

- `custom_script` (string) - Used for running a script on VM provision during the image build
  The following example executes the contents of the file specified by `user_data_file`:
   ```hcl2
   custom_script   = "powershell -ExecutionPolicy Unrestricted -NoProfile -NonInteractive -Command \"$userData = (Invoke-RestMethod -Headers @{Metadata=$true} -Method GET -Uri http://169.254.169.254/metadata/instance/compute/userData?api-version=2021-01-01$([char]38)format=text); $contents = [System.Text.Encoding]::UTF8.GetString([Convert]::FromBase64String($userData)); set-content -path c:\\Windows\\Temp\\userdata.ps1 -value $contents; . c:\\Windows\\Temp\\userdata.ps1;\""
   user_data_file  = "./scripts/userdata.ps1"
   ```
  Specify a command to inject into the CustomScriptExtension, to run on startup
  on Windows builds, before the communicator attempts to connect
  See [documentation](https://docs.microsoft.com/en-us/azure/virtual-machines/extensions/custom-script-windows)
  to learn more.

- `plan_info` (PlanInformation) - Used for creating images from Marketplace images. Please refer to
  [Deploy an image with Marketplace
  terms](https://aka.ms/azuremarketplaceapideployment) for more details.
  Not all Marketplace images support programmatic deployment, and support
  is controlled by the image publisher.
  
  An example plan\_info object is defined below.
  
  ```json
  {
    "plan_info": {
        "plan_name": "rabbitmq",
        "plan_product": "rabbitmq",
        "plan_publisher": "bitnami"
    }
  }
  ```
  
  `plan_name` (string) - The plan name, required. `plan_product` (string) -
  The plan product, required. `plan_publisher` (string) - The plan publisher,
  required. `plan_promotion_code` (string) - Some images accept a promotion
  code, optional.
  
  Images created from the Marketplace with `plan_info` **must** specify
  `plan_info` whenever the image is deployed. The builder automatically adds
  tags to the image to ensure this information is not lost. The following
  tags are added.
  
  ```text
  1.  PlanName
  2.  PlanProduct
  3.  PlanPublisher
  4.  PlanPromotionCode
  ```

- `polling_duration_timeout` (duration string | ex: "1h5m2s") - The default PollingDuration for azure is 15mins, this property will override
  that value.
  If your Packer build is failing on the
  ARM deployment step with the error `Original Error:
  context deadline exceeded`, then you probably need to increase this timeout from
  its default of "15m" (valid time units include `s` for seconds, `m` for
  minutes, and `h` for hours.)

- `os_type` (string) - If either Linux or Windows is specified Packer will
  automatically configure authentication credentials for the provisioned
  machine. For Linux this configures an SSH authorized key. For Windows
  this configures a WinRM certificate.

- `winrm_expiration_time` (duration string | ex: "1h5m2s") - A time duration with which to set the WinRM certificate to expire
  This only works for Windows builds (valid time units include `s` for seconds, `m` for
  minutes, and `h` for hours.)

- `temp_os_disk_name` (string) - temporary name assigned to the OSDisk. If this
  value is not set, a random value will be assigned. Being able to assign a custom
  osDiskName could ease deployment if naming conventions are used.

- `os_disk_size_gb` (int32) - Specify the size of the OS disk in GB
  (gigabytes). Values of zero or less than zero are ignored.

- `disk_additional_size` ([]int32) - The size(s) of any additional hard disks for the VM in gigabytes. If
  this is not specified then the VM will only contain an OS disk. The
  number of additional disks and maximum size of a disk depends on the
  configuration of your VM. See
  [Windows](https://docs.microsoft.com/en-us/azure/virtual-machines/windows/about-disks-and-vhds)
  or
  [Linux](https://docs.microsoft.com/en-us/azure/virtual-machines/linux/about-disks-and-vhds)
  for more information.
  
  For VHD builds the final artifacts will be named
  `<PREFIX><dataDisk>-<n>.vhd` and stored in the specified capture
  container along-side the OS disk.
  
  For Managed build the final artifacts are included in the managed image.
  The additional disk will have the same storage account type as the OS
  disk, as specified with the `managed_image_storage_account_type`
  setting.

- `disk_caching_type` (string) - Specify the disk caching type. Valid values
  are None, ReadOnly, and ReadWrite. The default value is ReadWrite.

- `allowed_inbound_ip_addresses` ([]string) - Specify the list of IP addresses and CIDR blocks that should be
  allowed access to the VM. If provided, an Azure Network Security
  Group will be created with corresponding rules and be bound to
  the subnet of the VM.
  Providing `allowed_inbound_ip_addresses` in combination with
  `virtual_network_name` is not allowed.

- `boot_diag_storage_account` (string) - Specify storage to store Boot Diagnostics -- Enabling this option
  will create 2 Files in the specified storage account. (serial console log & screenshot file)
  once the build is completed, it has to be removed manually.
  see [here](https://docs.microsoft.com/en-us/azure/virtual-machines/troubleshooting/boot-diagnostics) for more info

- `custom_resource_build_prefix` (string) - specify custom azure resource names during build limited to max 10 characters
  this will set the prefix for the resources. The actual resource names will be
  `custom_resource_build_prefix` + resourcetype + 5 character random alphanumeric string
  
  You can also set this via the environment variable `PACKER_AZURE_CUSTOM_RESOURCE_BUILD_PREFIX`.
  If both the config field and the environment variable are present, the config field takes precedence.

- `license_type` (string) - Specify a license type for the build VM to enable Azure Hybrid Benefit. If not set, Pay-As-You-Go license
  model (default) will be used. Valid values are:
  
  For Windows:
  - `Windows_Client`
  - `Windows_Server`
  
  For Linux:
  - `RHEL_BYOS`
  - `SLES_BYOS`
  
  Refer to the following documentation for more information about Hybrid Benefit:
  [Windows](https://learn.microsoft.com/en-us/azure/virtual-machines/windows/hybrid-use-benefit-licensing)
  or
  [Linux](https://learn.microsoft.com/en-us/azure/virtual-machines/linux/azure-hybrid-benefit-linux)

- `secure_boot_enabled` (bool) - Specifies if Secure Boot is enabled for the Virtual Machine. For Trusted Launch or Confidential VMs, Secure Boot must be enabled.

- `encryption_at_host` (\*bool) - Specifies if Encryption at host is enabled for the Virtual Machine.
  Requires enabling encryption at host in the Subscription read more [here](https://learn.microsoft.com/en-us/azure/virtual-machines/disks-enable-host-based-encryption-portal?tabs=azure-powershell)

- `public_ip_sku` (string) - Specify the Public IP Address SKU for the public IP used to connect to the build Virtual machine.
  Valid values are `Basic` and `Standard`. The default value is `Standard`.
  On 31 March 2025 Azure will remove the ability to create `Basic` SKU public IPs, we recommend upgrading as soon as possible
  You can learn more about public IP SKUs [here](https://learn.microsoft.com/en-us/azure/virtual-network/ip-services/public-ip-addresses#sku)

- `vtpm_enabled` (bool) - Specifies if vTPM (virtual Trusted Platform Module) is enabled for the Virtual Machine. For Trusted Launch or Confidential VMs, vTPM must be enabled.

- `security_type` (string) - Specifies the type of security to use for the VM. "TrustedLaunch" or "ConfidentialVM"

- `security_encryption_type` (string) - Specifies the encryption type to use for the Confidential VM. "DiskWithVMGuestState" or "VMGuestStateOnly"

- `async_resourcegroup_delete` (bool) - If you want packer to delete the
  temporary resource group asynchronously set this value. It's a boolean
  value and defaults to false. Important Setting this true means that
  your builds are faster, however any failed deletes are not reported.

<!-- End of code generated from the comments of the Config struct in builder/azure/arm/config.go; -->


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

- `client_jwt` (string) - The ID token when authenticating using OpenID Connect (OIDC).

- `object_id` (string) - The object ID for the AAD SP. Optional, will be derived from the oAuth token if left empty.

- `tenant_id` (string) - The Active Directory tenant identifier with which your `client_id` and
  `subscription_id` are associated. If not specified, `tenant_id` will be
  looked up using `subscription_id`.

- `subscription_id` (string) - The subscription to use.

- `oidc_request_token` (string) - OIDC Request Token is used for GitHub Actions OIDC, this token is used with oidc_request_url to fetch access tokens to Azure
  Value in GitHub Actions can be extracted from the `ACTIONS_ID_TOKEN_REQUEST_TOKEN` variable
  Refer to [Configure a federated identity credential on an app](https://learn.microsoft.com/en-us/entra/workload-id/workload-identity-federation-create-trust?pivots=identity-wif-apps-methods-azp#github-actions) for details on how setup GitHub Actions OIDC authentication

- `oidc_request_url` (string) - OIDC Request URL is used for GitHub Actions OIDC, this token is used with oidc_request_url to fetch access tokens to Azure
  Value in GitHub Actions can be extracted from the `ACTIONS_ID_TOKEN_REQUEST_URL` variable

- `use_azure_cli_auth` (bool) - Flag to use Azure CLI authentication. Defaults to false.
  CLI auth will use the information from an active `az login` session to connect to Azure and set the subscription id and tenant id associated to the signed in account.
  If enabled, it will use the authentication provided by the `az` CLI.
  Azure CLI authentication will use the credential marked as `isDefault` and can be verified using `az account show`.
  Works with normal authentication (`az login`) and service principals (`az login --service-principal --username APP_ID --password PASSWORD --tenant TENANT_ID`).
  Ignores all other configurations if enabled.

<!-- End of code generated from the comments of the Config struct in builder/azure/common/client/config.go; -->


<!-- Code generated from the comments of the Config struct in builder/azure/common/config.go; DO NOT EDIT MANUALLY -->

- `skip_create_image` (bool) - Skip creating the image.
  Useful for setting to `true` during a build test stage.
  Defaults to `false`.

<!-- End of code generated from the comments of the Config struct in builder/azure/common/config.go; -->



### Shared Image Gallery

The shared_image_gallery block is available for building a new image from a private or [community shared imaged gallery](https://docs.microsoft.com/en-us/azure/virtual-machines/azure-compute-gallery#community-gallery-preview) owned gallery.

<!-- Code generated from the comments of the SharedImageGallery struct in builder/azure/arm/config.go; DO NOT EDIT MANUALLY -->

- `id` (string) - ID of the Shared Image Gallery used as a base image in the build, this field is useful when using HCP Packer ancestry
  If this field is set, no other fields in the SharedImageGallery block can be set
  As those other fields simply build the reference ID

- `subscription` (string) - Subscription

- `resource_group` (string) - Resource Group

- `gallery_name` (string) - Gallery Name

- `image_name` (string) - Image Name

- `image_version` (string) - Specify a specific version of an OS to boot from.
  Defaults to latest. There may be a difference in versions available
  across regions due to image synchronization latency. To ensure a consistent
  version across regions set this value to one that is available in all
  regions where you are deploying.

- `community_gallery_image_id` (string) - Id of the community gallery image : /CommunityGalleries/{galleryUniqueName}/Images/{img}[/Versions/{}] (Versions part is optional)

- `direct_shared_gallery_image_id` (string) - Id of the direct shared gallery image : /sharedGalleries/{galleryUniqueName}/Images/{img}[/Versions/{}] (Versions part is optional)

<!-- End of code generated from the comments of the SharedImageGallery struct in builder/azure/arm/config.go; -->



### Shared Image Gallery Destination

The shared_image_gallery_destination block is available for publishing a new image version to an existing shared image gallery.

<!-- Code generated from the comments of the SharedImageGalleryDestination struct in builder/azure/arm/config.go; DO NOT EDIT MANUALLY -->

- `subscription` (string) - Sig Destination Subscription

- `resource_group` (string) - Sig Destination Resource Group

- `gallery_name` (string) - Sig Destination Gallery Name

- `image_name` (string) - Sig Destination Image Name

- `image_version` (string) - Sig Destination Image Version

- `replication_regions` ([]string) - A list of regions to replicate the image version in, by default the build location will be used as a replication region (the build location is either set in the location field, or the location of the resource group used in `build_resource_group_name` will be included.
  Can not contain any region but the build region when using shallow replication

- `target_region` ([]TargetRegion) - A target region to store the image version in. The attribute supersedes `replication_regions` which is now considered deprecated.
  One or more target_region blocks can be specified for storing an imager version to various regions. In addition to specifying a region,
  a DiskEncryptionSetId can be specified for each target region to support multi-region disk encryption.
  At a minimum their must be one target region entry for the primary build region where the image version will be stored.
  Target region must only contain one entry matching the build region when using shallow replication.
  See the `shared_image_gallery_destination` block description for an example of using this field

- `storage_account_type` (string) - Specify a storage account type for the Shared Image Gallery Image Version.
  Defaults to `Standard_LRS`. Accepted values are `Standard_LRS`, `Standard_ZRS` and `Premium_LRS`

- `specialized` (bool) - Set to true if publishing to a Specialized Gallery, this skips a call to set the build VM's OS state as Generalized

- `use_shallow_replication` (bool) - Setting a `shared_image_gallery_replica_count` or any `replication_regions` is unnecessary for shallow builds, as they can only replicate to the build region and must have a replica count of 1
  Refer to [Shallow Replication](https://learn.microsoft.com/en-us/azure/virtual-machines/shared-image-galleries?tabs=azure-cli#shallow-replication) for details on when to use shallow replication mode.

- `confidential_vm_image_encryption_type` (string) - The ConfidentialVM Image Encryption Type for the Shared Image Gallery Destination. This can be either "EncryptedVMGuestStateOnlyWithPmk", "EncryptedWithPmk", or "EncryptedWithCmk" (encrypted with DES). This option is used to publish a VM image to a Shared Image Gallery as a confidential VM image.

<!-- End of code generated from the comments of the SharedImageGalleryDestination struct in builder/azure/arm/config.go; -->


### Target Regions

The `target_regions` block is available inside the `shared_image_gallery_destination` block for setting replica regions and the replica community

<!-- Code generated from the comments of the TargetRegion struct in builder/azure/arm/config.go; DO NOT EDIT MANUALLY -->

- `disk_encryption_set_id` (string) - DiskEncryptionSetId for Disk Encryption Set in Region. Needed for supporting
  the replication of encrypted disks across regions. CMKs must
  already exist within the target regions.

- `replicas` (int64) - The number of replicas of the Image Version to be created within the region. Defaults to 1.
  Replica count must be between 1 and 100, but 50 replicas should be sufficient for most use cases.
  When using shallow replication `use_shallow_replication=true` the value can only be 1 for the primary build region.

<!-- End of code generated from the comments of the TargetRegion struct in builder/azure/arm/config.go; -->


### Spot

The `spot` block is available to use a spot instance during build.

<!-- Code generated from the comments of the Spot struct in builder/azure/arm/config.go; DO NOT EDIT MANUALLY -->

- `eviction_policy` (virtualmachines.VirtualMachineEvictionPolicyTypes) - Specify eviction policy for spot instance: "Deallocate" or "Delete". If this is set, a spot instance will be used.

- `max_price` (float32) - How much should the VM cost maximally per hour. Specify -1 (or do not specify) to not evict based on price.

<!-- End of code generated from the comments of the Spot struct in builder/azure/arm/config.go; -->



## Build Shared Information Variables

This builder generates data that are shared with provisioner and post-processor via build function of [template engine](https://packer.io/docs/templates/legacy_json_templates/engine) for JSON and [contextual variables](https://packer.io/docs/templates/hcl_templates/contextual-variables) for HCL2.

The generated variables available for this builder are:

- `SourceImageName` - The full name of the source image used in the deployment. When using
shared images the resulting name will point to the actual source used to create the said version.
  building the AMI.

- `SubscriptionID` - The ID of the Azure Subscription where the build takes place.  

- `TenantID` - The ID of the Azure Tenant where the build takes place.

All of the following variables are temporary resource names that the plugin uses to build resources that are generally deleted at the end of a build.

The presence of these variables does not mean these resources are created, they are just the names that are used for any temporary resources the builder creates. They are all randomly specified, but most can be overridden by builder fields.

- `TempComputeName` - Virtual machine name
- `TempNicName` - Network interface name
- `TempOSDiskName` - OS Disk Name
- `TempDataDiskName` - Data Disk Name
- `TempDeploymentName` - ARM Deployment name
- `TempVirtualNetworkName` - Virtual Network name
- `TempKeyVaultName` - Key vault name
- `TempResourceGroupName` - Resource Group Name, will be unset if a build resource group name is provided.
- `TempNsgName` - Network security group name
- `TempSubnetName` - Subnet name
- `TempPublicIPAddressName` - Public IP Address name

Usage example:

**HCL2**

```hcl
// When accessing one of these variables from inside the builder, you need to
// use the golang templating syntax. This is due to an architectural quirk that
// won't be easily resolvable until legacy json templates are deprecated:

{
source "azure-arm" "basic-example" {
  os_type = "Linux"
  image_publisher = "Canonical"
  image_offer = "UbuntuServer"
  image_sku = "14.04.4-LTS"
}

// when accessing one of the variables from a provisioner or post-processor, use
// hcl-syntax
post-processor "manifest" {
    output = "manifest.json"
    strip_path = true
    custom_data = {
        source_image_name = "${build.SourceImageName}"
	tenant_id = "${build.TenantID}"
        subscription_id = "${build.SubscriptionID}"
        temp_deployment_name = "${build.TempDeploymentName}"
        temp_compute_name = "${build.TempComputeName}"
        temp_nic_name = "${build.TempNicName}"
        temp_os_disk_name = "${build.TempOSDiskName}"
        temp_data_disk_name = "${build.TempDataDiskName}"
        temp_resource_group_name = "${build.TempResourceGroupName}"
        temp_nsg_name = "${build.TempNsgName}"
        temp_key_vault_name = "${build.TempKeyVaultName}"
        temp_subnet_name = "${build.TempSubnetName}"
        temp_virtual_network_name = "${build.TempVirtualNetworkName}"
        temp_public_ip_address_name = "${build.TempPublicIPAddressName}"
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
    	"source_image_name": "{{ build `SourceImageName` }}",
    	"tenant_id": "{{ build `TenantID` }}",
    	"subscription_id": "{{ build `SubscriptionID` }}",
    	"temp_deployment_name": "{{ build `TempDeploymentName`}}",
	"temp_compute_name": "{{ build `TempComputeName`}}",
	"temp_nic_name": "{{ build `TempNicName`}}",
	"temp_os_disk_name": "{{ build `TempOSDiskName`}}",
	"temp_data_disk_name": "{{ build `TempDataDiskName`}}",
	"temp_resource_group_name": "{{ build `TempResourceGroupName`}}",
	"temp_nsg_name": "{{ build `TempNsgName`}}",
	"temp_key_vault_name": "{{ build `TempKeyVaultName`}}",
	"temp_subnet_name": "{{ build `TempSubnetName`}}",
	"temp_virtual_network_name": "{{ build `TempVirtualNetworkName`}}",
	"temp_public_ip_address_name": "{{ build `TempPublicIPAddressName`}}"
    }
  }
]
```


### Communicator Config

In addition to the builder options, a communicator may also be defined:

<!-- Code generated from the comments of the Config struct in communicator/config.go; DO NOT EDIT MANUALLY -->

- `communicator` (string) - Packer currently supports three kinds of communicators:
  
  -   `none` - No communicator will be used. If this is set, most
      provisioners also can't be used.
  
  -   `ssh` - An SSH connection will be established to the machine. This
      is usually the default.
  
  -   `winrm` - A WinRM connection will be established.
  
  In addition to the above, some builders have custom communicators they
  can use. For example, the Docker builder has a "docker" communicator
  that uses `docker exec` and `docker cp` to execute scripts and copy
  files.

- `pause_before_connecting` (duration string | ex: "1h5m2s") - We recommend that you enable SSH or WinRM as the very last step in your
  guest's bootstrap script, but sometimes you may have a race condition
  where you need Packer to wait before attempting to connect to your
  guest.
  
  If you end up in this situation, you can use the template option
  `pause_before_connecting`. By default, there is no pause. For example if
  you set `pause_before_connecting` to `10m` Packer will check whether it
  can connect, as normal. But once a connection attempt is successful, it
  will disconnect and then wait 10 minutes before connecting to the guest
  and beginning provisioning.

<!-- End of code generated from the comments of the Config struct in communicator/config.go; -->


<!-- Code generated from the comments of the SSH struct in communicator/config.go; DO NOT EDIT MANUALLY -->

- `ssh_host` (string) - The address to SSH to. This usually is automatically configured by the
  builder.

- `ssh_port` (int) - The port to connect to SSH. This defaults to `22`.

- `ssh_username` (string) - The username to connect to SSH with. Required if using SSH.

- `ssh_password` (string) - A plaintext password to use to authenticate with SSH.

- `ssh_ciphers` ([]string) - This overrides the value of ciphers supported by default by Golang.
  The default value is [
    "aes128-gcm@openssh.com",
    "chacha20-poly1305@openssh.com",
    "aes128-ctr", "aes192-ctr", "aes256-ctr",
  ]
  
  Valid options for ciphers include:
  "aes128-ctr", "aes192-ctr", "aes256-ctr", "aes128-gcm@openssh.com",
  "chacha20-poly1305@openssh.com",
  "arcfour256", "arcfour128", "arcfour", "aes128-cbc", "3des-cbc",

- `ssh_clear_authorized_keys` (bool) - If true, Packer will attempt to remove its temporary key from
  `~/.ssh/authorized_keys` and `/root/.ssh/authorized_keys`. This is a
  mostly cosmetic option, since Packer will delete the temporary private
  key from the host system regardless of whether this is set to true
  (unless the user has set the `-debug` flag). Defaults to "false";
  currently only works on guests with `sed` installed.

- `ssh_key_exchange_algorithms` ([]string) - If set, Packer will override the value of key exchange (kex) algorithms
  supported by default by Golang. Acceptable values include:
  "curve25519-sha256@libssh.org", "ecdh-sha2-nistp256",
  "ecdh-sha2-nistp384", "ecdh-sha2-nistp521",
  "diffie-hellman-group14-sha1", and "diffie-hellman-group1-sha1".

- `ssh_certificate_file` (string) - Path to user certificate used to authenticate with SSH.
  The `~` can be used in path and will be expanded to the
  home directory of current user.

- `ssh_pty` (bool) - If `true`, a PTY will be requested for the SSH connection. This defaults
  to `false`.

- `ssh_timeout` (duration string | ex: "1h5m2s") - The time to wait for SSH to become available. Packer uses this to
  determine when the machine has booted so this is usually quite long.
  Example value: `10m`.
  This defaults to `5m`, unless `ssh_handshake_attempts` is set.

- `ssh_disable_agent_forwarding` (bool) - If true, SSH agent forwarding will be disabled. Defaults to `false`.

- `ssh_handshake_attempts` (int) - The number of handshakes to attempt with SSH once it can connect.
  This defaults to `10`, unless a `ssh_timeout` is set.

- `ssh_bastion_host` (string) - A bastion host to use for the actual SSH connection.

- `ssh_bastion_port` (int) - The port of the bastion host. Defaults to `22`.

- `ssh_bastion_agent_auth` (bool) - If `true`, the local SSH agent will be used to authenticate with the
  bastion host. Defaults to `false`.

- `ssh_bastion_username` (string) - The username to connect to the bastion host.

- `ssh_bastion_password` (string) - The password to use to authenticate with the bastion host.

- `ssh_bastion_interactive` (bool) - If `true`, the keyboard-interactive used to authenticate with bastion host.

- `ssh_bastion_private_key_file` (string) - Path to a PEM encoded private key file to use to authenticate with the
  bastion host. The `~` can be used in path and will be expanded to the
  home directory of current user.

- `ssh_bastion_certificate_file` (string) - Path to user certificate used to authenticate with bastion host.
  The `~` can be used in path and will be expanded to the
  home directory of current user.

- `ssh_file_transfer_method` (string) - `scp` or `sftp` - How to transfer files, Secure copy (default) or SSH
  File Transfer Protocol.
  
  **NOTE**: Guests using Windows with Win32-OpenSSH v9.1.0.0p1-Beta, scp
  (the default protocol for copying data) returns a a non-zero error code since the MOTW
  cannot be set, which cause any file transfer to fail. As a workaround you can override the transfer protocol
  with SFTP instead `ssh_file_transfer_method = "sftp"`.

- `ssh_proxy_host` (string) - A SOCKS proxy host to use for SSH connection

- `ssh_proxy_port` (int) - A port of the SOCKS proxy. Defaults to `1080`.

- `ssh_proxy_username` (string) - The optional username to authenticate with the proxy server.

- `ssh_proxy_password` (string) - The optional password to use to authenticate with the proxy server.

- `ssh_keep_alive_interval` (duration string | ex: "1h5m2s") - How often to send "keep alive" messages to the server. Set to a negative
  value (`-1s`) to disable. Example value: `10s`. Defaults to `5s`.

- `ssh_read_write_timeout` (duration string | ex: "1h5m2s") - The amount of time to wait for a remote command to end. This might be
  useful if, for example, packer hangs on a connection after a reboot.
  Example: `5m`. Disabled by default.

- `ssh_remote_tunnels` ([]string) - Remote tunnels forward a port from your local machine to the instance.
  Format: ["REMOTE_PORT:LOCAL_HOST:LOCAL_PORT"]
  Example: "9090:localhost:80" forwards localhost:9090 on your machine to port 80 on the instance.

- `ssh_local_tunnels` ([]string) - Local tunnels forward a port from the instance to your local machine.
  Format: ["LOCAL_PORT:REMOTE_HOST:REMOTE_PORT"]
  Example: "8080:localhost:3000" allows the instance to access your local machine’s port 3000 via localhost:8080.

<!-- End of code generated from the comments of the SSH struct in communicator/config.go; -->


- `ssh_private_key_file` (string) - Path to a PEM encoded private key file to use to authenticate with SSH.
  The `~` can be used in path and will be expanded to the home directory
  of current user.


## Basic Example

Here is a basic example for Azure.

**HCL2**

```hcl
source "azure-arm" "basic-example" {
  client_id = "fe354398-d7sf-4dc9-87fd-c432cd8a7e09"
  client_secret = "keepitsecret&#*$"
  resource_group_name = "packerdemo"
  storage_account = "virtualmachines"
  subscription_id = "44cae533-4247-4093-42cf-897ded6e7823"
  tenant_id = "de39842a-caba-497e-a798-7896aea43218"

  capture_container_name = "images"
  capture_name_prefix = "packer"

  os_type = "Linux"
  image_publisher = "Canonical"
  image_offer = "UbuntuServer"
  image_sku = "14.04.4-LTS"

  azure_tags = {
    dept = "engineering"
  }

  location = "West US"
  vm_size = "Standard_A2"
}

build {
  sources = ["sources.azure-arm.basic-example"]
}
```

**JSON**

```json
{
  "type": "azure-arm",

  "client_id": "fe354398-d7sf-4dc9-87fd-c432cd8a7e09",
  "client_secret": "keepitsecret&#*$",
  "resource_group_name": "packerdemo",
  "storage_account": "virtualmachines",
  "subscription_id": "44cae533-4247-4093-42cf-897ded6e7823",
  "tenant_id": "de39842a-caba-497e-a798-7896aea43218",

  "capture_container_name": "images",
  "capture_name_prefix": "packer",

  "os_type": "Linux",
  "image_publisher": "Canonical",
  "image_offer": "UbuntuServer",
  "image_sku": "14.04.4-LTS",

  "azure_tags": {
    "dept": "engineering"
  },

  "location": "West US",
  "vm_size": "Standard_A2"
}
```


## Deprovision

Azure VMs should be deprovisioned at the end of every build. For Windows this
means executing sysprep, and for Linux this means executing the waagent
deprovision process.

Please refer to the Azure
[example](https://github.com/hashicorp/packer-plugin-azure/tree/main/example) folder for
complete examples showing the deprovision process.

### Windows

The following provisioner snippet shows how to sysprep a Windows VM.
Deprovision should be the last operation executed by a build. The code below
will wait for sysprep to write the image status in the registry and will exit
after that. The possible states, in case you want to wait for another state,
[are documented
here](https://technet.microsoft.com/en-us/library/hh824815.aspx)

**JSON**

```json
{
  "provisioners": [
    {
      "type": "powershell",
      "inline": [
        "# If Guest Agent services are installed, make sure that they have started.",
        "foreach ($service in Get-Service -Name RdAgent, WindowsAzureTelemetryService, WindowsAzureGuestAgent -ErrorAction SilentlyContinue) { while ((Get-Service $service.Name).Status -ne 'Running') { Start-Sleep -s 5 } }",

        "& $env:SystemRoot\\System32\\Sysprep\\Sysprep.exe /oobe /generalize /quiet /quit /mode:vm",
        "while($true) { $imageState = Get-ItemProperty HKLM:\\SOFTWARE\\Microsoft\\Windows\\CurrentVersion\\Setup\\State | Select ImageState; if($imageState.ImageState -ne 'IMAGE_STATE_GENERALIZE_RESEAL_TO_OOBE') { Write-Output $imageState.ImageState; Start-Sleep -s 10  } else { break } }"
      ]
    }
  ]
}
```

**HCL2**

```hcl
provisioner "powershell" {
   inline = [
        "# If Guest Agent services are installed, make sure that they have started.",
        "foreach ($service in Get-Service -Name RdAgent, WindowsAzureTelemetryService, WindowsAzureGuestAgent -ErrorAction SilentlyContinue) { while ((Get-Service $service.Name).Status -ne 'Running') { Start-Sleep -s 5 } }",

        "& $env:SystemRoot\\System32\\Sysprep\\Sysprep.exe /oobe /generalize /quiet /quit /mode:vm",
        "while($true) { $imageState = Get-ItemProperty HKLM:\\SOFTWARE\\Microsoft\\Windows\\CurrentVersion\\Setup\\State | Select ImageState; if($imageState.ImageState -ne 'IMAGE_STATE_GENERALIZE_RESEAL_TO_OOBE') { Write-Output $imageState.ImageState; Start-Sleep -s 10  } else { break } }"
   ]
}
```


The Windows Guest Agent participates in the Sysprep process. The agent must be
fully installed before the VM can be sysprep'ed. To ensure this is true all
agent services must be running before executing sysprep.exe. The above JSON
snippet shows one way to do this in the PowerShell provisioner. This snippet is
**only** required if the VM is configured to install the agent, which is the
default. To learn more about disabling the Windows Guest Agent please see
[Install the VM
Agent](https://docs.microsoft.com/en-us/azure/virtual-machines/extensions/agent-windows#install-the-vm-agent).

Please note that sysprep can get stuck in infinite loops if it is not configured
correctly -- for example, if it is waiting for a reboot that you never perform.

### Linux

The following provisioner snippet shows how to deprovision a Linux VM.
Deprovision should be the last operation executed by a build.

**JSON**

```json
{
  "provisioners": [
    {
      "execute_command": "chmod +x {{ .Path }}; {{ .Vars }} sudo -E sh '{{ .Path }}'",
      "inline": [
        "/usr/sbin/waagent -force -deprovision+user && export HISTSIZE=0 && sync"
      ],
      "inline_shebang": "/bin/sh -x",
      "type": "shell"
    }
  ]
}
```

**HCL2**

```hcl
provisioner "shell" {
   execute_command = "chmod +x {{ .Path }}; {{ .Vars }} sudo -E sh '{{ .Path }}'"
   inline = [
        "/usr/sbin/waagent -force -deprovision+user && export HISTSIZE=0 && sync"
   ]
   inline_shebang = "/bin/sh -x"
}
```


To learn more about the Linux deprovision process please see WALinuxAgent's
[README](https://github.com/Azure/WALinuxAgent/blob/master/README.md).

#### skip_clean

Customers have reported issues with the deprovision process where the builder
hangs. The error message is similar to the following.

    Build 'azure-arm' errored: Retryable error: Error removing temporary script at /tmp/script_9899.sh: ssh: handshake failed: EOF

One solution is to set skip_clean to true in the provisioner. This prevents
Packer from cleaning up any helper scripts uploaded to the VM during the build.

## Defaults

The Azure builder attempts to pick default values that provide for a just works
experience. These values can be changed by the user to more suitable values.

- The default user name is Packer not root as in other builders. Most distros
  on Azure do not allow root to SSH to a VM hence the need for a non-root
  default user. Set the ssh_username option to override the default value.
- The default VM size is Standard_A1. Set the vm_size option to override
  the default value.
- The default image version is latest. Set the image_version option to
  override the default value.
- By default a temporary resource group will be created and destroyed as part
  of the build. If you do not have permissions to do so, use
  `build_resource_group_name` to specify an existing resource group to run
  the build in.

## Implementation

~> **Warning!** This is an advanced topic. You do not need to understand
the implementation to use the Azure builder.

The Azure builder uses ARM
[templates](https://azure.microsoft.com/en-us/documentation/articles/resource-group-authoring-templates/)
to deploy resources. ARM templates allow you to express the what without having
to express the how.

The Azure builder works under the assumption that it creates everything it
needs to execute a build. When the build has completed it simply deletes the
resource group to cleanup any runtime resources. Resource groups are named
using the form `packer-Resource-Group-<random>`. The value `<random>` is a
random value that is generated at every invocation of packer. The `<random>`
value is re-used as much as possible when naming resources, so users can better
identify and group these transient resources when seen in their subscription.

> The VHD is created on a user specified storage account, not a random one
> created at runtime. When a virtual machine is captured the resulting VHD is
> stored on the same storage account as the source VHD. The VHD created by
> Packer must persist after a build is complete, which is why the storage
> account is set by the user.

The basic steps for a build are:

1.  Create a resource group.
2.  Validate and deploy a VM template.
3.  Execute provision - defined by the user; typically shell commands.
4.  Power off and capture the VM.
5.  Delete the resource group.
6.  Delete the temporary VM's OS disk.

The templates used for a build are currently fixed in the code. There is a
template for Linux, Windows, and KeyVault. The templates are themselves
templated with place holders for names, passwords, SSH keys, certificates, etc.

### What's Randomized?

The Azure builder creates the following random values at runtime.

- Administrator Password: a random 32-character value using the _password
  alphabet_.
- Certificate: a 2,048-bit certificate used to secure WinRM communication.
  The certificate is valid for 24-hours, which starts roughly at invocation
  time.
- Certificate Password: a random 32-character value using the _password
  alphabet_ used to protect the private key of the certificate.
- Compute Name: a random 15-character name prefixed with pkrvm; the name of
  the VM.
- Deployment Name: a random 15-character name prefixed with pkfdp; the name
  of the deployment.
- KeyVault Name: a random 15-character name prefixed with pkrkv.
- NIC Name: a random 15-character name prefixed with pkrni.
- Public IP Name: a random 15-character name prefixed with pkrip.
- OS Disk Name: a random 15-character name prefixed with pkros.
- Data Disk Name: a random 15-character name prefixed with pkrdd.
- Resource Group Name: a random 33-character name prefixed with
  packer-Resource-Group-.
- Subnet Name: a random 15-character name prefixed with pkrsn.
- SSH Key Pair: a 2,048-bit asymmetric key pair; can be overridden by the
  user.
- Virtual Network Name: a random 15-character name prefixed with pkrvn.

The default alphabet used for random values is
**0123456789bcdfghjklmnpqrstvwxyz**. The alphabet was reduced (no vowels) to
prevent running afoul of Azure decency controls.

The password alphabet used for random values is
**0123456789abcdefghijklmnopqrstuvwxyzABCDEFGHIJKLMNOPQRSTUVWXYZ**.

### Windows

The Windows implementation is very similar to the Linux build, with the
exception that it deploys a template to configure KeyVault. Packer communicates
with a Windows VM using the WinRM protocol. Windows VMs on Azure default to
using both password and certificate based authentication for WinRM. The
password is easily set via the VM ARM template, but the certificate requires an
intermediary. The intermediary for Azure is KeyVault. The certificate is
uploaded to a new KeyVault provisioned in the same resource group as the VM.
When the Windows VM is deployed, it links to the certificate in KeyVault, and
Azure will ensure the certificate is injected as part of deployment.

The basic steps for a Windows build are:

1.  Create a resource group.
2.  Validate and deploy a KeyVault template.
3.  Validate and deploy a VM template.
4.  Execute provision - defined by the user; typically shell commands.
5.  Power off and capture the VM.
6.  Delete the resource group.
7.  Delete the temporary VM's OS disk.

A Windows build requires two templates and two deployments. Unfortunately, the
KeyVault and VM cannot be deployed at the same time hence the need for two
templates and deployments. The time required to deploy a KeyVault template is
minimal, so overall impact is small.

See the
[example](https://github.com/hashicorp/packer-plugin-azure/tree/main/example)
folder in the Packer project for more examples.
