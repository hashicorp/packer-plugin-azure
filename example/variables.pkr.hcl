# Copyright IBM Corp. 2013, 2025
# SPDX-License-Identifier: MPL-2.0

variable "client_id" {
  type    = string
  default = "${env("ARM_CLIENT_ID")}"
}

variable "client_secret" {
  type    = string
  default = "${env("ARM_CLIENT_SECRET")}"
}

variable "resource_group" {
  type    = string
  default = "${env("ARM_RESOURCE_GROUP_NAME")}"
}

variable "ssh_pass" {
  type    = string
  default = "${env("ARM_SSH_PASS")}"
}

variable "ssh_user" {
  type    = string
  default = "centos"
}

variable "subscription_id" {
  type    = string
  default = "${env("ARM_SUBSCRIPTION_ID")}"
}

variable "tenant_id" {
  type    = string
  default = "${env("ARM_TENANT_ID")}"
}

