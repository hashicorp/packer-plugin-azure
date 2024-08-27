# Copyright (c) HashiCorp, Inc.
# SPDX-License-Identifier: MPL-2.0

locals { timestamp = regex_replace(timestamp(), "[- TZ:]", "") }

variable "resource_group_name" {
  default = "${env("ARM_RESOURCE_GROUP_NAME")}"
  type    = string
}
variable "resource_prefix" {
  default = "${env("ARM_RESOURCE_PREFIX")}"
  type    = string
}
variable "subscription_id" {
  default = "${env("ARM_SUBSCRIPTION_ID")}"
  type    = string
}
source "azure-arm" "windows-sig" {
  communicator       = "winrm"
  winrm_timeout      = "5m"
  winrm_use_ssl      = true
  winrm_insecure     = true
  winrm_username     = "packer"
  use_azure_cli_auth = true
  shared_image_gallery_replica_count = 4
  shared_image_gallery {
    resource_group = var.resource_group_name
    image_name     = "${var.resource_prefix}-windows-sig"
    gallery_name   = "${var.resource_prefix}_acctestgallery"
    image_version  = "1.0.1"
    subscription = var.subscription_id
  }
  shared_image_gallery_destination {
    image_name     = "${var.resource_prefix}-windows-sig"
    gallery_name   = "${var.resource_prefix}_acctestgallery"
    image_version  = "1.0.7"
    resource_group = var.resource_group_name
    target_region  {
      name = "East US2"
      replicas = 1
    }
  }
  managed_image_name                = "packer-test-windows-sig-${local.timestamp}"
  managed_image_resource_group_name = var.resource_group_name

  os_type         = "Windows"
  #image_publisher = "MicrosoftWindowsServer"
  #image_offer     = "WindowsServer"
  #image_sku       = "2012-R2-Datacenter"

  location = "East US2"
  vm_size  = "Standard_DS2_v2"
}

build {
provisioner "powershell" {
   inline = [
        "# If Guest Agent services are installed, make sure that they have started.",
        "foreach ($service in Get-Service -Name RdAgent, WindowsAzureTelemetryService, WindowsAzureGuestAgent -ErrorAction SilentlyContinue) { while ((Get-Service $service.Name).Status -ne 'Running') { Start-Sleep -s 5 } }",

        "& $env:SystemRoot\\System32\\Sysprep\\Sysprep.exe /oobe /generalize /quiet /quit /mode:vm",
        "while($true) { $imageState = Get-ItemProperty HKLM:\\SOFTWARE\\Microsoft\\Windows\\CurrentVersion\\Setup\\State | Select ImageState; if($imageState.ImageState -ne 'IMAGE_STATE_GENERALIZE_RESEAL_TO_OOBE') { Write-Output $imageState.ImageState; Start-Sleep -s 10  } else { break } }"
   ]
}
  sources = ["source.azure-arm.windows-sig"]
}

