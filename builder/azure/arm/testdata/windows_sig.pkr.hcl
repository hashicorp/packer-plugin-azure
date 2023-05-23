# Copyright (c) HashiCorp, Inc.
# SPDX-License-Identifier: MPL-2.0

locals { timestamp = regex_replace(timestamp(), "[- TZ:]", "") }

source "azure-arm" "windows-sig" {
  communicator       = "winrm"
  winrm_timeout      = "5m"
  winrm_use_ssl      = true
  winrm_insecure     = true
  winrm_username     = "packer"
  use_azure_cli_auth = true
  shared_image_gallery_destination {
    image_name     = "windows-sig"
    gallery_name   = "acctestgallery"
    image_version  = "1.0.0"
    resource_group = "packer-acceptance-test"
  }
  managed_image_name                = "packer-test-windows-sig-${local.timestamp}"
  managed_image_resource_group_name = "packer-acceptance-test"

  os_type         = "Windows"
  image_publisher = "MicrosoftWindowsServer"
  image_offer     = "WindowsServer"
  image_sku       = "2012-R2-Datacenter"

  location = "South Central US"
  vm_size  = "Standard_DS2_v2"
}

build {
  sources = ["source.azure-arm.windows-sig"]
}

