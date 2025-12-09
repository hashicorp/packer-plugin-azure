Type: `azure-dtl`
Artifact BuilderId: `Azure.ResourceManagement.VMImage`

The Azure DevTest Labs builder builds custom images and uploads them to an existing DevTest Lab image repository automatically.
For more information on crating an Azure DevTest Lab see the [Configuring a Lab How-to guide](https://docs.microsoft.com/en-us/azure/devtest-labs/devtest-lab-create-lab).

## Configuration Reference

There are many configuration options available for the builder. We'll start
with authentication parameters, then go over the Azure ARM builder specific
options. In addition to the options listed here, a [communicator](/packer/docs/templates/legacy_json_templates/communicator) can be configured for this builder.

## Azure DevTest Labs builder specific options

### Required:

<!-- Code generated from the comments of the Config struct in builder/azure/dtl/config.go; DO NOT EDIT MANUALLY -->

- `managed_image_resource_group_name` (string) - Specify the managed image resource group name where the result of the
  Packer build will be saved. The resource group must already exist. If
  this value is set, the value managed_image_name must also be set. See
  documentation to learn more about managed images.

- `managed_image_name` (string) - Specify the managed image name where the result of the Packer build will
  be saved. The image name must not exist ahead of time, and will not be
  overwritten. If this value is set, the value
  managed_image_resource_group_name must also be set. See documentation to
  learn more about managed images.

- `lab_name` (string) - Name of the existing lab where the virtual machine will be created.

- `lab_subnet_name` (string) - Name of the subnet being used in the lab, if not the default.

- `lab_resource_group_name` (string) - Name of the resource group where the lab exist.

<!-- End of code generated from the comments of the Config struct in builder/azure/dtl/config.go; -->


### Optional:

<!-- Code generated from the comments of the Config struct in builder/azure/dtl/config.go; DO NOT EDIT MANUALLY -->

- `capture_name_prefix` (string) - Capture

- `capture_container_name` (string) - Capture Container Name

- `shared_image_gallery` (SharedImageGallery) - Use a [Shared Gallery
  image](https://azure.microsoft.com/en-us/blog/announcing-the-public-preview-of-shared-image-gallery/)
  as the source for this build. *VHD targets are incompatible with this
  build type* - the target must be a *Managed Image*.

- `shared_image_gallery_destination` (SharedImageGalleryDestination) - The name of the Shared Image Gallery under which the managed image will be published as Shared Gallery Image version.
  
  Following is an example.

- `shared_image_gallery_timeout` (duration string | ex: "1h5m2s") - How long to wait for an image to be published to the shared image
  gallery before timing out. If your Packer build is failing on the
  Publishing to Shared Image Gallery step with the error `Original Error:
  context deadline exceeded`, but the image is present when you check your
  Azure dashboard, then you probably need to increase this timeout from
  its default of "60m" (valid time units include `s` for seconds, `m` for
  minutes, and `h` for hours.)

- `custom_image_capture_timeout` (duration string | ex: "1h5m2s") - How long to wait for an image to be captured before timing out
  If your Packer build is failing on the Capture Image step with the
  error `Original Error: context deadline exceeded`, but the image is
  present when you check your custom image repository, then you probably
  need to increase this timeout from its default of "30m" (valid time units
  include `s` for seconds, `m` for minutes, and `h` for hours.)

- `image_publisher` (string) - PublisherName for your base image. See
  [documentation](https://docs.microsoft.com/en-us/cli/azure/vm/image)
  for details.
  
  CLI example `az vm image list-publishers --location westus`

- `image_offer` (string) - Offer for your base image. See
  [documentation](https://docs.microsoft.com/en-us/cli/azure/vm/image)
  for details.
  
  CLI example
  `az vm image list-offers --location westus --publisher Canonical`

- `image_sku` (string) - SKU for your base image. See
  [documentation](https://docs.microsoft.com/en-us/cli/azure/vm/image)
  for details.
  
  CLI example
  `az vm image list-skus --location westus --publisher Canonical --offer UbuntuServer`

- `image_version` (string) - Specify a specific version of an OS to boot from.
  Defaults to `latest`. There may be a difference in versions available
  across regions due to image synchronization latency. To ensure a consistent
  version across regions set this value to one that is available in all
  regions where you are deploying.
  
  CLI example
  `az vm image list --location westus --publisher Canonical --offer UbuntuServer --sku 16.04.0-LTS --all`

- `image_url` (string) - Specify a custom VHD to use. If this value is set, do
  not set image_publisher, image_offer, image_sku, or image_version.

- `custom_managed_image_resource_group_name` (string) - Specify the source managed image's resource group used to use. If this
  value is set, do not set image\_publisher, image\_offer, image\_sku, or
  image\_version. If this value is set, the value
  `custom_managed_image_name` must also be set. See
  [documentation](https://docs.microsoft.com/en-us/azure/storage/storage-managed-disks-overview#images)
  to learn more about managed images.

- `custom_managed_image_name` (string) - Specify the source managed image's name to use. If this value is set, do
  not set image\_publisher, image\_offer, image\_sku, or image\_version.
  If this value is set, the value
  `custom_managed_image_resource_group_name` must also be set. See
  [documentation](https://docs.microsoft.com/en-us/azure/storage/storage-managed-disks-overview#images)
  to learn more about managed images.

- `location` (string) - Location

- `vm_size` (string) - Size of the VM used for building. This can be changed when you deploy a
  VM from your VHD. See
  [pricing](https://azure.microsoft.com/en-us/pricing/details/virtual-machines/)
  information. Defaults to `Standard_A1`.
  
  CLI example `az vm list-sizes --location westus`

- `managed_image_storage_account_type` (string) - Specify the storage account
  type for a managed image. Valid values are Standard_LRS and Premium_LRS.
  The default is Standard_LRS.

- `azure_tags` (map[string]string) - the user can define up to 50
  tags. Tag names cannot exceed 512 characters, and tag values cannot exceed
  256 characters. Tags are applied to every resource deployed by a Packer
  build, i.e. Resource Group, VM, NIC, VNET, Public IP, KeyVault, etc.

- `plan_id` (string) - Used for creating images from Marketplace images. Please refer to
  [Deploy an image with Marketplace
  terms](https://aka.ms/azuremarketplaceapideployment) for more details.
  Not all Marketplace images support programmatic deployment, and support
  is controlled by the image publisher.
  Plan_id is a string with unique identifier for the plan associated with images.
  Ex plan_id="1-12ab"

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

- `os_disk_size_gb` (int32) - Specify the size of the OS disk in GB
  (gigabytes). Values of zero or less than zero are ignored.

- `disk_additional_size` ([]int32) - For Managed build the final artifacts are included in the managed image.
  The additional disk will have the same storage account type as the OS
  disk, as specified with the `managed_image_storage_account_type`
  setting.

- `disk_caching_type` (string) - Specify the disk caching type. Valid values
  are None, ReadOnly, and ReadWrite. The default value is ReadWrite.

- `storage_type` (string) - DTL values

- `lab_virtual_network_name` (string) - Name of the virtual network used for communicating with the lab vms.

- `dtl_artifacts` ([]DtlArtifact) - One or more Artifacts that should be added to the VM at start.

- `vm_name` (string) - Name for the virtual machine within the DevTest lab.

- `disallow_public_ip` (bool) - DisallowPublicIPAddress - Indicates whether the virtual machine is to be created without a public IP address.

- `skip_sysprep` (bool) - SkipSysprep - Indicates whether SysPrep is to be requested to the DTL or if it should be skipped because it has already been applied. Defaults to false.

<!-- End of code generated from the comments of the Config struct in builder/azure/dtl/config.go; -->


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


#### DtlArtifact
<!-- Code generated from the comments of the DtlArtifact struct in provisioner/azure-dtlartifact/provisioner.go; DO NOT EDIT MANUALLY -->

- `artifact_name` (string) - Artifact Name

- `artifact_id` (string) - Artifact Id

- `parameters` ([]ArtifactParameter) - Parameters

<!-- End of code generated from the comments of the DtlArtifact struct in provisioner/azure-dtlartifact/provisioner.go; -->


#### ArtifactParmater
<!-- Code generated from the comments of the ArtifactParameter struct in provisioner/azure-dtlartifact/provisioner.go; DO NOT EDIT MANUALLY -->

- `name` (string) - Name

- `value` (string) - Value

- `type` (string) - Type

<!-- End of code generated from the comments of the ArtifactParameter struct in provisioner/azure-dtlartifact/provisioner.go; -->



## Basic Example

```hcl

variable "client_id" {
  type    = string
  default = "${env("ARM_CLIENT_ID")}"
}

variable "client_secret" {
  type    = string
  default = "${env("ARM_CLIENT_SECRET")}"
}

variable "subscription_id" {
  type    = string
  default = "${env("ARM_SUBSCRIPTION_ID")}"
}

locals { timestamp = regex_replace(timestamp(), "[- TZ:]", "") }

source "azure-dtl" "example" {
  subscription_id                   = "${var.subscription_id}"
  client_id          = "${var.client_id}"
  client_secret      = "${var.client_secret}"
  disallow_public_ip = true
  dtl_artifacts {
    artifact_name = "linux-apt-package"
    parameters {
      name  = "packages"
      value = "vim"
    }
    parameters {
      name  = "update"
      value = "true"
    }
    parameters {
      name  = "options"
      value = "--fix-broken"
    }
  }
  image_offer                       = "UbuntuServer"
  image_publisher                   = "Canonical"
  image_sku                         = "16.04-LTS"
  lab_name                          = "packer-test"
  lab_resource_group_name           = "packer-test"
  lab_virtual_network_name          = "dtlpacker-test"
  location                          = "South Central US"
  managed_image_name                = "ManagedDiskLinux-${local.timestamp}"
  managed_image_resource_group_name = "packer-test"
  os_type                           = "Linux"
  vm_size                           = "Standard_DS2_v2"
}

build {
  sources = ["source.azure-dtl.example"]

}
```
