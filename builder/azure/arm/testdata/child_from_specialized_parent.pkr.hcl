# Copyright (c) HashiCorp, Inc.
# SPDX-License-Identifier: MPL-2.0
locals { timestamp = regex_replace(timestamp(), "[- TZ:]", "") }

variable "ssh_password" {
  type = string
  default = "${env("ARM_SSH_PASS")}"
  sensitive = true
}

variable "subscription" {
  default = "${env("ARM_SUBSCRIPTION_ID")}"
  type = string
  sensitive = true
}

variable "ssh_private_key_location" {
  default = "${env("ARM_SSH_PRIVATE_KEY_LOCATION")}"
  type = string
}

source "azure-arm" "linux-sig" {
  use_azure_cli_auth = true
  location           = "South Central US"
  vm_size            = "Standard_D4ps_v5"
  ssh_username       = "packer"
  ssh_password       = var.ssh_password
  ssh_private_key_file = var.ssh_private_key_location
  communicator       = "ssh"
  shared_image_gallery{
    subscription   = var.subscription
    image_name     = "arm-linux-specialized-sig"
    gallery_name   = "acctestgallery"
    image_version  = "1.0.0"
    resource_group = "packer-acceptance-test"
  }
  shared_image_gallery_destination {
    image_name     = "arm-linux-specialized-sig"
    gallery_name   = "acctestgallery"
    image_version  = "1.0.1"
    resource_group = "packer-acceptance-test"
    specialized    = true
  }

  os_type = "Linux"
}

build {
  sources = ["source.azure-arm.linux-sig"]
}

