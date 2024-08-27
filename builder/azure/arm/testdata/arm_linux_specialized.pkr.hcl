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
variable "resource_prefix" {
  default = "${env("ARM_RESOURCE_PREFIX")}"
  type    = string
}
source "azure-arm" "linux-sig" {
  image_offer          = "0001-com-ubuntu-server-jammy"
  image_publisher      = "canonical"
  image_sku            = "22_04-lts-arm64"
  use_azure_cli_auth   = true
  location             = "East US2"
  vm_size              = "Standard_D4ps_v5"
  ssh_username         = "packer"
  ssh_private_key_file = var.ssh_private_key_location
  communicator         = "ssh"
  managed_image_resource_group_name = var.resource_group_name
  managed_image_name = "packer-build-linux-rhel-4"
  shared_gallery_image_version_exclude_from_latest = true
  shared_image_gallery_destination {
    image_name              = "${var.resource_prefix}-arm-linux-specialized-sig"
    gallery_name            = "${var.resource_prefix}_acctestgallery"
    image_version           = "5.0.5"
    resource_group          = var.resource_group_name
    target_region  {
      name = "East US2"
      replicas = 2
    }
    target_region  {
      name = "West US"
      replicas = 1
    }
    target_region  {
      name = "East US"
      replicas = 3
    }
  }

  os_type = "Linux"
}

build {
  sources = ["source.azure-arm.linux-sig"]
}

