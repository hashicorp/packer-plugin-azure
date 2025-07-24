# Copyright (c) HashiCorp, Inc.
# SPDX-License-Identifier: MPL-2.0

variable "vault_name" {
  default = "packer-test-vault"
  type    = string
}

variable "secret_name" {
  default = "test-secret"
  type    = string
}

data "azure-keyvaultsecret" "test" {
  vault_name = var.vault_name
  secret_name   = var.secret_name
}

locals {
  value = data.azure-keyvaultsecret.test.value
  response = data.azure-keyvaultsecret.test.response
}

source "null" "basic-example" {
  communicator = "none"
}

build {
  sources = [
    "source.null.basic-example"
  ]

  provisioner "shell-local" {
    inline = [
      "echo secret value: ${local.value}",
      "echo secret response: ${local.response}",
    ]
  }
}
