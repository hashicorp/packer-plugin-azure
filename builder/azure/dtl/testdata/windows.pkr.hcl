# Copyright (c) HashiCorp, Inc.
# SPDX-License-Identifier: MPL-2.0

variable "client_id" {
  type    = string
  default = "${env("ARM_CLIENT_ID")}"
}

variable "client_secret" {
  type    = string
  default = "${env("ARM_CLIENT_SECRET")}"
}

variable "resource_group_name" {
  type    = string
  default = "${env("ARM_RESOURCE_GROUP_NAME")}"
}

variable "subscription_id" {
  type    = string
  default = "${env("ARM_SUBSCRIPTION_ID")}"
}

variable "resource_prefix" {
  type    = string
  default = "${env("ARM_RESOURCE_PREFIX")}"
}
locals { timestamp = regex_replace(timestamp(), "[- TZ:]", "") }

source "azure-dtl" "autogenerated_1" {
  client_id                         = "${var.client_id}"
  client_secret                     = "${var.client_secret}"
  communicator                      = "winrm"
  image_offer                       = "WindowsServer"
  image_publisher                   = "MicrosoftWindowsServer"
  image_sku                         = "2022-datacenter"
  lab_name                          = "${var.resource_prefix}-packer-acceptance-test"
  lab_resource_group_name           = "${var.resource_group_name}"
  lab_virtual_network_name          = "dtlpacker-acceptance-test"
  location                          = "South Central US"
  managed_image_name                = "testBuilderAccManagedDiskWindows-${local.timestamp}"
  managed_image_resource_group_name = "${var.resource_group_name}"
  os_type                           = "Windows"
  polling_duration_timeout          = "25m"
  subscription_id                   = "${var.subscription_id}"
  vm_size                           = "Standard_DS2_v2"
  winrm_insecure                    = "true"
  winrm_timeout                     = "10m"
  winrm_use_ssl                     = "true"
  winrm_username                    = "packer"
}

build {
  sources = ["source.azure-dtl.autogenerated_1"]

}
