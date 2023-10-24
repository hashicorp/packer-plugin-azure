# Copyright (c) HashiCorp, Inc.
# SPDX-License-Identifier: MPL-2.0
locals { timestamp = regex_replace(timestamp(), "[- TZ:]", "") }

variable "ssh_private_key_location" {
  default = "${env("ARM_SSH_PRIVATE_KEY_FILE")}"
  type    = string
}
variable "resource_group_name" {
  default = "${env("ARM_RESOURCE_GROUP_NAME")}"
  type    = string
}
source "azure-arm" "linux-sig" {
  image_offer          = "0001-com-ubuntu-server-jammy"
  image_publisher      = "canonical"
  image_sku            = "22_04-lts-arm64"
  use_azure_cli_auth   = true
  location             = "South Central US"
  vm_size              = "Standard_D4ps_v5"
  ssh_username         = "packer"
  ssh_private_key_file = var.ssh_private_key_location
  communicator         = "ssh"
  shared_image_gallery_destination {
    image_name              = "arm-linux-specialized-sig"
    gallery_name            = "acctestgallery"
    image_version           = "1.0.0"
    resource_group          = var.resource_group_name
    specialized             = true
    use_shallow_replication = true
  }

  os_type = "Linux"
}

build {
  sources = ["source.azure-arm.linux-sig"]
}

