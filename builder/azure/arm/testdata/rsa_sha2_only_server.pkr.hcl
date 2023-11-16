# Copyright (c) HashiCorp, Inc.
# SPDX-License-Identifier: MPL-2.0

/*
OpenSSH migrated the ssh-rsa key type, which historically used the ssh-rsa
signature algorithm based on SHA-1, to the new rsa-sha2-256 and rsa-sha2-512 signature algorithms.
Golang issues: https://github.com/golang/go/issues/49952
See plugin issue: https://github.com/hashicorp/packer-plugin-azure/issues/191
*/

variables {
  subscription_id = env("ARM_SUBSCRIPTION_ID")
  client_id       = env("ARM_CLIENT_ID")
  client_secret   = env("ARM_CLIENT_SECRET")
  resource_group  = env("ARM_RESOURCE_GROUP_NAME")
}
locals { timestamp = regex_replace(timestamp(), "[- TZ:]", "") }

source "azure-arm" "ubuntu2204" {
  subscription_id = var.subscription_id
  client_id       = var.client_id

  client_secret = var.client_secret

  managed_image_resource_group_name = var.resource_group
  managed_image_name                = "ubuntu-jammay-server-test-${local.timestamp}"

  os_type         = "Linux"
  image_publisher = "canonical"
  image_offer     = "0001-com-ubuntu-server-jammy-daily"
  image_sku       = "22_04-daily-lts"

  location = "West US2"
  vm_size  = "Standard_DS2_v2"
}

build {
  sources = ["source.azure-arm.ubuntu2204"]
  provisioner "shell" {
    inline = ["uname -a"]
  }
}

