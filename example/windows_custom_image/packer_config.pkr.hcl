# Copyright IBM Corp. 2013, 2025
# SPDX-License-Identifier: MPL-2.0

packer {
  required_plugins {
    azure = {
      version = ">= 1.4.2"
      source  = "github.com/hashicorp/azure"
    }
  }
}
