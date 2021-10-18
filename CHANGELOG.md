## 1.0.4 (October 18, 2021)

### IMPROVEMENTS:
* Add SourceImageName as shared builder information variable. [GH-160]
* Add SourceImageName to HCP Packer registry image metadata. [GH-160]
* Update packer-plugin-sdk to v0.2.7 [GH-159]

### BUG FIXES:
* builder/arm: Fix panic when running the cleanup step on a failed deployment. [GH-155]

## 1.0.3 (September 13, 2021)

### FEATURES:
* Add HCP Packer registry image metadata to builder artifacts. [GH-138] [GH-150]

### IMPROVEMENTS:
* Allow Premium_LRS as SIG storage account type. [GH-124]

### BUG FIXES:
* Update VaultClientDelete to pass correct Azure cloud environment endpoint.  [GH-137]

## 1.0.2 (August 19, 2021)

### IMPROVEMENTS:
* Add user_data_file to arm builder. [GH-123]

### BUG FIXES:
* Bump github.com/Azure/azure-sdk-for-go to fix vulnerability in plugin dependency. [GH-117]

## 1.0.0 (June 15, 2021)

* Update packer-plugin-sdk to v0.2.3 [GH-96]
* Add Go module retraction for v0.0.1

## 0.0.3 (May 14, 2021)
* Update packer-plugin-sdk to enable use of ntlm with WinRM.

## 0.0.2 (May 13, 2021)

### IMPROVEMENTS:

* builder/dtl: Add `disallow_public_ip` configuration to support private DevTest Lab VMs. [GH-85]

### BUG FIXES:

* Fixes a version string issue to support plugin vendoring from within Packer [hashicorp/packer#10979](https://github.com/hashicorp/packer/pull/10979).
  [GH-84]

## 0.0.1 (May 7, 2021)

* Azure Plugin break out from Packer core. Changes prior to break out can be found in [Packer's CHANGELOG](https://github.com/hashicorp/packer/blob/master/CHANGELOG.md)

