variable "rgName" {
  type    = string
  default = "rg-acg-test"
}

variable "acgName" {
  type    = string
  default = "acgDemo"
}

variable "image_name" {
  type    = string
  default = "WindowsImage"
}

variable "build_key_vault_name" {
  type    = string
  default = "kv-demo"
}

variable "build_revision" {
  type    = string
  default = "001"
}

variable "disk_encryption_set_id" {
  type    = string
  default = "/subscriptions/<REPLACE_WITH_YOUR_SUBSCRIPTION_ID>/resourceGroups/<REPLACE_WITH_YOUR_RG_NAME>/providers/Microsoft.Compute/diskEncryptionSets/<DES_NAME>"
}

variable "image_offer" {
  type    = string
  default = "WindowsServer"
}

variable "image_publisher" {
  type    = string
  default = "MicrosoftWindowsServer"
}

variable "image_sku" {
  type    = string
  default = "2022-datacenter-g2"
}

variable "temp_os_disk_name" {
  type    = string
  default = "osDisk001"
}

variable "destination_image_version" {
  type    = string
  default = "1.0.0"
}

variable "location" {
  type    = string
  default = "westeurope"
}

variable "vmSize" {
  type    = string
  default = "Standard_DS3_V2"
}

variable "subscription_id" {
  type    = string
  default = "<REPLACE_WITH_YOUR_SUBSCRIPTION_ID>"
}

variable "tenant_id" {
  type    = string
  default = "<REPLACE_WITH_YOUR_TENANT_ID>"
}

variable "client_id" {
  type    = string
  default = "<REPLACE_WITH_YOUR_CLIENT_ID>"
}

variable "client_secret" {
  type    = string
  default = "<REPLACE_WITH_YOUR_CLIENT_SECRET>"
}

variable "Release" {
  type    = string
  default = "COOL"
}





