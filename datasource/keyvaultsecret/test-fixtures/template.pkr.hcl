
variable "keyvault_id" {
    type=string
}
variable "client_id" {
    type=string
}
variable "client_secret" {
    type=string
}
variable "tenant_id" {
    type=string
}

data "azure-keyvault-secret" "basic-example" {
  name            = "packer-datasource-keyvault-test-secret"
  keyvault_id     = "${var.keyvault_id}"
  client_id       = "${var.client_id}"
  client_secret   = "${var.client_secret}"
  tenant_id       = "${var.tenant_id}"
}

locals {
  value         = data.azure-keyvault-secret.basic-example.value
  id            = data.azure-keyvault-secret.basic-example.id
  content_type  = data.azure-keyvault-secret.basic-example.content_type
  environment   = data.azure-keyvault-secret.basic-example.tags.environment
}

source "null" "basic-example" {
  communicator = "none"
}

build {
  sources = [
    "source.null.basic-example"
  ]

  provisioner "shell-local" {
    inline = [
      "echo secret value: ${local.value}",
      "echo secret id: ${local.id}",
      "echo secret content_type: ${local.content_type}",
      "echo secret environment: ${local.environment}"
    ]
  }
}