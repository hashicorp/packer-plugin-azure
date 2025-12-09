# Copyright IBM Corp. 2013, 2025
# SPDX-License-Identifier: MPL-2.0

source "azure-arm" "centos" {
  client_id                         = "${var.client_id}"
  client_secret                     = "${var.client_secret}"
  image_offer                       = "CentOS"
  image_publisher                   = "OpenLogic"
  image_sku                         = "7.3"
  image_version                     = "latest"
  location                          = "South Central US"
  managed_image_name                = "MyCentOSImage"
  managed_image_resource_group_name = "${var.resource_group}"
  os_type                           = "Linux"
  ssh_password                      = "${var.ssh_pass}"
  ssh_pty                           = "true"
  ssh_username                      = "${var.ssh_user}"
  subscription_id                   = "${var.subscription_id}"
  tenant_id                         = "${var.tenant_id}"
  vm_size                           = "Standard_DS2_v2"
}

build {
  sources = ["source.azure-arm.centos"]

  provisioner "shell" {
    execute_command = "echo '${var.ssh_pass}' | {{ .Vars }} sudo -S -E sh '{{ .Path }}'"
    inline          = ["yum update -y", "/usr/sbin/waagent -force -deprovision+user && export HISTSIZE=0 && sync"]
    inline_shebang  = "/bin/sh -x"
    skip_clean      = true
  }

}
