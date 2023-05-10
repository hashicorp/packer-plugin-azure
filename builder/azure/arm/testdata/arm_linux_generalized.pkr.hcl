# Copyright (c) HashiCorp, Inc.
# SPDX-License-Identifier: MPL-2.0


variables {
  subscription_id = env("ARM_SUBSCRIPTION_ID")
  client_id       = env("ARM_CLIENT_ID")
  client_secret   = env("ARM_CLIENT_SECRET")
  resource_group  = env("ARM_RESOURCE_GROUP_NAME")
}
locals { timestamp = regex_replace(timestamp(), "[- TZ:]", "") }

source "azure-arm" "linux-sig" {
  subscription_id            = var.subscription_id
  client_id                  = var.client_id
  async_resourcegroup_delete = true
  client_secret              = var.client_secret
  image_offer                = "0001-com-ubuntu-server-jammy"
  image_publisher            = "canonical"
  image_sku                  = "22_04-lts-arm64"

  location = "South Central US"
  vm_size  = "Standard_D4ps_v5"

  shared_image_gallery_destination {
    image_name     = "arm-linux-generalized-sig"
    gallery_name   = "acctestgallery"
    image_version  = "1.0.0"
    resource_group = "packer-acceptance-test"
    specialized    = true
  }

  os_type = "Linux"
}

build {
  sources = ["source.azure-arm.linux-sig"]
}

