# Copyright IBM Corp. 2013, 2025
# SPDX-License-Identifier: MPL-2.0

variable "resource_group_location" {
  type    = string
  default = "southcentralus"
}

variable "resource_group_name" {
  type    = string
}

variable "storage_account_name" {
  type    = string
}

variable "tenant_id" {
  type = string
}

variable "object_id" {
  type = string
}

// Variable applied to resources that have uniqueness constraints at a subscription level
// For example you can't have two shared image galleries named `linux` in the same Subscription in different resource group
variable "resource_prefix" {
  type = string
}
