# Copyright (c) HashiCorp, Inc.
# SPDX-License-Identifier: MPL-2.0

resource "azurerm_resource_group" "rg" {
  location = var.resource_group_location
  name     = var.resource_group_name
}

## ARM Builder Resources
resource "azurerm_storage_account" "storage-account" {
  name                     = var.storage_account_name
  resource_group_name      = azurerm_resource_group.rg.name
  location                 = azurerm_resource_group.rg.location
  account_tier             = "Standard"
  account_replication_type = "GRS"
}

resource "azurerm_shared_image_gallery" "gallery" {
  name                = "${var.resource_prefix}_acctestgallery"
  resource_group_name = azurerm_resource_group.rg.name
  location            = azurerm_resource_group.rg.location
}

resource "azurerm_shared_image" "windows-sig" {
  name                = "${var.resource_prefix}-windows-sig"
  gallery_name        = azurerm_shared_image_gallery.gallery.name
  resource_group_name = azurerm_resource_group.rg.name
  location            = azurerm_resource_group.rg.location
  os_type             = "Windows"
  hyper_v_generation  = "V1"
  architecture        = "x64"
  identifier {
    publisher = "MicrosoftWindowsServer"
    offer     = "WindowsServer"
    sku       = "2022-datacenter"
  }
}

resource "azurerm_shared_image" "linux-sig" {
  name                = "${var.resource_prefix}-arm-linux-specialized-sig"
  gallery_name        = azurerm_shared_image_gallery.gallery.name
  resource_group_name = azurerm_resource_group.rg.name
  location            = azurerm_resource_group.rg.location
  os_type             = "Linux"
  specialized         = true
  architecture        = "Arm64"
  hyper_v_generation  = "V2"
  identifier {
    publisher = "canonical"
    offer     = "0001-com-ubuntu-server-jammy"
    sku       = "22_04-lts-arm64"
  }
}

resource "azurerm_key_vault" "vault" {
  name                        = "packer-acctest-vault"
  location                    = azurerm_resource_group.rg.location
  resource_group_name         = azurerm_resource_group.rg.name
  enabled_for_disk_encryption = true
  tenant_id                   = var.tenant_id
  soft_delete_retention_days  = 7
  purge_protection_enabled    = false

  sku_name = "standard"

  access_policy {
    tenant_id = var.tenant_id
    object_id = var.object_id

    secret_permissions = ["Get", "Set", "Delete"]
  }
}

/*
## DTL Builder Resources - disabled

resource "azurerm_dev_test_lab" "dtl" {
  name                = "${var.resource_prefix}-packer-acceptance-test"
  location            = azurerm_resource_group.rg.location
  resource_group_name = azurerm_resource_group.rg.name
}

resource "azurerm_dev_test_virtual_network" "vnet" {
  name                = "vnet"
  lab_name            = azurerm_dev_test_lab.dtl.name
  resource_group_name = azurerm_resource_group.rg.name
  subnet {
    use_in_virtual_machine_creation = "Allow"
  }
}
*/
