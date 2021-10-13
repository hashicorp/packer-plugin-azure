
variable "client_id" {
  type    = string
  default = "${env("ARM_CLIENT_ID")}"
}

variable "client_secret" {
  type    = string
  default = "${env("ARM_CLIENT_SECRET")}"
}

variable "resource_group_name" {
  type    = string
  default = "${env("ARM_RESOURCE_GROUP_NAME")}"
}

variable "subscription_id" {
  type    = string
  default = "${env("ARM_SUBSCRIPTION_ID")}"
}

source "azure-arm" "windows-test" {
  build_resource_group_name         = "${var.resource_group_name}"
  client_id                         = "${var.client_id}"
  client_secret                     = "${var.client_secret}"
  subscription_id                   = "${var.subscription_id}"
  communicator                      = "none"
  image_offer                       = "windowsserver"
  image_publisher                   = "microsoftwindowsserver"
  image_sku                         = "2019-datacenter"
  managed_image_name                = "PackerImageWindowsTest"
  managed_image_resource_group_name = "${var.resource_group_name}"
  os_type                           = "Windows"
  vm_size                           = "Standard_D8s_v3"
}

build {
  sources = ["source.azure-arm.windows-test"]

}
