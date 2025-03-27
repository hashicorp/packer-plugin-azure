# Copyright (c) HashiCorp, Inc.
# SPDX-License-Identifier: MPL-2.0
variable "subscription" {
  default   = "${env("ARM_SUBSCRIPTION_ID")}"
  type      = string
  sensitive = true
}

variable "ssh_private_key_location" {
  default = "${env("ARM_SSH_PRIVATE_KEY_FILE")}"
  type    = string
}
variable "hcp_packer_bucket_name" {
  default = "${env("HCP_PACKER_BUCKET_NAME")}"
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

data "hcp-packer-version" "hardened-source" {
  bucket_name  = var.hcp_packer_bucket_name
  channel_name = "latest"
}

data "hcp-packer-artifact" "parent" {
  bucket_name  = var.hcp_packer_bucket_name
  version_fingerprint = "${data.hcp-packer-version.hardened-source.fingerprint}"
  platform            = "azure"
  region              = "South Central US"
}


source "azure-arm" "linux-sig" {
  use_azure_cli_auth   = true
  location             = "South Central US"
  vm_size              = "Standard_D4ps_v5"
  ssh_username         = "packer"
  ssh_private_key_file = var.ssh_private_key_location
  communicator         = "ssh"
  shared_image_gallery {
    id = data.hcp-packer-artifact.parent.external_identifier 
  }
  shared_image_gallery_destination {
    image_version           = "1.0.4"
    image_name              = "${var.resource_prefix}-arm-linux-specialized-sig"
    gallery_name            = "${var.resource_prefix}_acctestgallery"
    resource_group          = var.resource_group_name
    specialized             = true
    use_shallow_replication = true
  }

  os_type = "Linux"
}

hcp_packer_registry {
  bucket_name = "child"
}

build {
  sources = ["source.azure-arm.linux-sig"]
}

