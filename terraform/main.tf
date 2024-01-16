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
  name                = "acctestgallery"
  resource_group_name = azurerm_resource_group.rg.name
  location            = azurerm_resource_group.rg.location
}

resource "azurerm_shared_image" "windows-sig" {
  name                = "windows-sig"
  gallery_name        = azurerm_shared_image_gallery.gallery.name
  resource_group_name = azurerm_resource_group.rg.name
  location            = azurerm_resource_group.rg.location
  os_type             = "Windows"
  hyper_v_generation  = "V1"
  architecture        = "x64"
  identifier {
    publisher = "MicrosoftWindowsServer"
    offer     = "WindowsServer"
    sku       = "2012-R2-Datacenter"
  }
}

resource "azurerm_shared_image" "linux-sig" {
  name                = "arm-linux-specialized-sig"
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

## DTL Builder Resources

resource "azurerm_dev_test_lab" "dtl" {
  name                = var.dtl_name
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
