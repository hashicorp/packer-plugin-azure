Type: `azure-dtlartifact`

The Azure DevTest Labs provisioner can be used to apply an artifact to a VM - See [Add an artifact to a VM](https://docs.microsoft.com/en-us/azure/devtest-labs/add-artifact-vm)

## Azure DevTest Labs provisioner specific options

### Required:

<!-- Code generated from the comments of the Config struct in provisioner/azure-dtlartifact/provisioner.go; DO NOT EDIT MANUALLY -->

- `dtl_artifacts` ([]DtlArtifact) - Dtl Artifacts

- `lab_name` (string) - Name of the existing lab where the virtual machine exist.

- `lab_resource_group_name` (string) - Name of the resource group where the lab exist.

- `vm_name` (string) - Name of the virtual machine within the DevTest lab.

<!-- End of code generated from the comments of the Config struct in provisioner/azure-dtlartifact/provisioner.go; -->


### Optional:

<!-- Code generated from the comments of the Config struct in provisioner/azure-dtlartifact/provisioner.go; DO NOT EDIT MANUALLY -->

- `polling_duration_timeout` (duration string | ex: "1h5m2s") - The default PollingDuration for azure is 15mins, this property will override
  that value.
  If your Packer build is failing on the
  ARM deployment step with the error `Original Error:
  context deadline exceeded`, then you probably need to increase this timeout from
  its default of "15m" (valid time units include `s` for seconds, `m` for
  minutes, and `h` for hours.)

- `azure_tags` (map[string]\*string) - Azure Tags

<!-- End of code generated from the comments of the Config struct in provisioner/azure-dtlartifact/provisioner.go; -->


#### DtlArtifact
<!-- Code generated from the comments of the DtlArtifact struct in provisioner/azure-dtlartifact/provisioner.go; DO NOT EDIT MANUALLY -->

- `artifact_name` (string) - Artifact Name

- `artifact_id` (string) - Artifact Id

- `parameters` ([]ArtifactParameter) - Parameters

<!-- End of code generated from the comments of the DtlArtifact struct in provisioner/azure-dtlartifact/provisioner.go; -->


#### ArtifactParmater
<!-- Code generated from the comments of the ArtifactParameter struct in provisioner/azure-dtlartifact/provisioner.go; DO NOT EDIT MANUALLY -->

- `name` (string) - Name

- `value` (string) - Value

- `type` (string) - Type

<!-- End of code generated from the comments of the ArtifactParameter struct in provisioner/azure-dtlartifact/provisioner.go; -->


## Basic Example

```hcl
source "null" "example" {
  communicator = "none"
}

build {
  sources = ["source.null.example"]

  provisioner "azure-dtlartifact" {
    lab_name                          = "packer-test"
    lab_resource_group_name           = "packer-test"
    vm_name                          = "packer-test-vm"
    dtl_artifacts {
        artifact_name = "linux-apt-package"
        parameters {
          name  = "packages"
          value = "vim"
        }
        parameters {
          name  = "update"
          value = "true"
        }
        parameters {
          name  = "options"
          value = "--fix-broken"
        }
    }
  }
}
```
