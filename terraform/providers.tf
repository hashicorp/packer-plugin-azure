terraform {
  required_providers {
    azurerm = {
      source  = "hashicorp/azurerm"
      version = "~>3.77"
    }
  }
}

provider "azurerm" {
   features {
     resource_group {
       prevent_deletion_if_contains_resources = false
     }
   }
  skip_provider_registration = true
}
