# Copyright (c) HashiCorp, Inc.
# SPDX-License-Identifier: MPL-2.0

locals { timestamp = regex_replace(timestamp(), "[- TZ:]", "") }

variable "resource_group_name" {
  default = "${env("ARM_RESOURCE_GROUP_NAME")}"
  type    = string
}
variable "resource_prefix" {
  default = "${env("ARM_RESOURCE_PREFIX")}"
  type    = string
}
source "azure-arm" "windows-sig" {
  communicator       = "winrm"
  winrm_timeout      = "5m"
  winrm_use_ssl      = true
  winrm_insecure     = true
  winrm_username     = "packer"
  use_azure_cli_auth = true
  public_ip_sku      = "Standard"
  shared_image_gallery_destination {
    image_name     = "${var.resource_prefix}-windows-sig"
    gallery_name   = "${var.resource_prefix}_acctestgallery"
    image_version  = "1.0.0"
    resource_group = var.resource_group_name
  }
  managed_image_name                = "packer-test-windows-sig-${local.timestamp}"
  managed_image_resource_group_name = var.resource_group_name

  os_type         = "Windows"
  image_publisher = "MicrosoftWindowsServer"
  image_offer     = "WindowsServer"
  image_sku       = "2022-datacenter"

  location = "South Central US"
  vm_size  = "Standard_DS2_v2"
}

build {
  sources = ["source.azure-arm.windows-sig"]
}

