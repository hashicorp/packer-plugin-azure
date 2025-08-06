# Copyright (c) HashiCorp, Inc.
# SPDX-License-Identifier: MPL-2.0

variable "vault_name" {
  default = "${env("ARM_RESOURCE_PREFIX")}-pkr-test-vault"
  type    = string
}

variable "secret_name" {
  default = "test-secret"
  type    = string
}

variable "client_id" {
  type = string
}
variable "client_secret" {
  type = string
}
variable "tenant_id" {
  type = string
}
variable "subscription_id" {
  type = string
}

variable "resource_prefix" {
  default = "${env("ARM_RESOURCE_PREFIX")}"
  type    = string
}

data "azure-keyvaultsecret" "test" {
  vault_name   = var.vault_name
  secret_name  = var.secret_name

  subscription_id = var.subscription_id
  client_id       = var.client_id
  client_secret   = var.client_secret
  tenant_id       = var.tenant_id
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
