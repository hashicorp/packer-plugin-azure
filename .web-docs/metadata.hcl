# For full specification on the configuration of this file visit:
# https://github.com/hashicorp/integration-template#metadata-configuration
integration {
  name = "Azure"
  description = "Packer can create Azure virtual machine images through variety of ways depending on the strategy that you want to use for building the images."
  identifier = "packer/hashicorp/azure"
  flags = ["hcp-ready"]
  component {
    type = "builder"
    name = "ARM"
    slug = "arm"
  }
  component {
    type = "builder"
    name = "chroot"
    slug = "chroot"
  }
  component {
    type = "builder"
    name = "DTL"
    slug = "dtl"
  }
  component {
    type = "provisioner"
    name = "DTL Artifact"
    slug = "dtlartifact"
  }
}
