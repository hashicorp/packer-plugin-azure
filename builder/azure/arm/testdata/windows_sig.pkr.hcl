# Copyright (c) HashiCorp, Inc.
# SPDX-License-Identifier: MPL-2.0


variables {
  subscription_id = env("ARM_SUBSCRIPTION_ID")
  client_id       = env("ARM_CLIENT_ID")
  client_secret   = env("ARM_CLIENT_SECRET")
  resource_group  = env("ARM_RESOURCE_GROUP_NAME")
}
locals { timestamp = regex_replace(timestamp(), "[- TZ:]", "") }

source "azure-arm" "windows-sig" {
  subscription_id            = var.subscription_id
  client_id                  = var.client_id
  communicator               = "winrm"
  winrm_timeout              = "5m"
  winrm_use_ssl              = true
  winrm_insecure             = true
  winrm_username             = "packer"
  async_resourcegroup_delete = true
  client_secret              = var.client_secret

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

