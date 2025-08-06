# Copyright (c) HashiCorp, Inc.
# SPDX-License-Identifier: MPL-2.0

terraform {
  required_providers {
    azurerm = {
      source  = "hashicorp/azurerm"
      version = "~>4.14"
    }
  }
}

provider "azurerm" {
   features {
     resource_group {
       prevent_deletion_if_contains_resources = false
     }
     key_vault {
       purge_soft_delete_on_destroy    = true
       recover_soft_deleted_key_vaults = true
    }
   }
}
