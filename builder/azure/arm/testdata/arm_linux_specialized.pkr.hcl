# Copyright (c) HashiCorp, Inc.
# SPDX-License-Identifier: MPL-2.0


locals { timestamp = regex_replace(timestamp(), "[- TZ:]", "") }

source "azure-arm" "linux-sig" {
  image_offer        = "0001-com-ubuntu-server-jammy"
  image_publisher    = "canonical"
  image_sku          = "22_04-lts-arm64"
  use_azure_cli_auth = true
  location           = "South Central US"
  vm_size            = "Standard_D4ps_v5"

  shared_image_gallery_destination {
    image_name     = "arm-linux-specialized-sig"
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

