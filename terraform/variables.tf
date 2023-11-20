# Copyright (c) HashiCorp, Inc.
# SPDX-License-Identifier: MPL-2.0

variable "resource_group_location" {
  type    = string
  default = "southcentralus"
}

variable "resource_group_name" {
  type    = string
  default = "packer-acceptance-test"
}

variable "storage_account_name" {
  type    = string
  default = "packeracctest"
}

variable "dtl_name" {
  type = string
  default = "packer-acceptance-test"
}
