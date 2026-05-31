# Copyright IBM Corp. 2013, 2026
# SPDX-License-Identifier: MPL-2.0

output "resource_group_name" {
  value = azurerm_resource_group.rg.name
}

output "storage_account_name" {
  value = azurerm_storage_account.storage-account.name
}

output "storage_container_name" {
  value = azurerm_storage_container.example.name
}

output "resource_prefix" {
  value = local.resource_prefix
}

output "resource_suffix" {
  value = local.resource_suffix
}

output "virtual_network_name" {
  value = azurerm_virtual_network.vnet.name
}
