# Latest Release

Please refer to [releases](https://github.com/hashicorp/packer-plugin-azure/releases) for the latest CHANGELOG information.

---
## 2.4.0 (August 6, 2025)

## What's Changed

### Exciting New Features
* Allow parent SIG images to referenced by their ID in [GH-482](https://github.com/hashicorp/packer-plugin-azure/pull/482)
* Added KeyVaultSecret Datasource in [GH-516](https://github.com/hashicorp/packer-plugin-azure/pull/516)

### Bug Fixes
* Updating ARM Builder Disk Steps order in [GH-505](https://github.com/hashicorp/packer-plugin-azure/pull/505)
* Prevents snapshot overwrite conflicts in managed images in [GH-509](https://github.com/hashicorp/packer-plugin-azure/pull/509)
* Fetching the Blob Endpoint for Deletion from Account in [GH-508](https://github.com/hashicorp/packer-plugin-azure/pull/508)
* fixes subscription ID while validating image in [GH-473](https://github.com/hashicorp/packer-plugin-azure/pull/473)

### Other Changes
* Update Golang-JWT to v5.2.2 in [GH-493](https://github.com/hashicorp/packer-plugin-azure/pull/493)
* Enforce /x/net 0.38 in [GH-494](https://github.com/hashicorp/packer-plugin-azure/pull/494)
* Updated SharedImageGallery param docs examples in [GH-519](https://github.com/hashicorp/packer-plugin-azure/pull/519)
* CRT Migration Changes in [GH-517](https://github.com/hashicorp/packer-plugin-azure/pull/517)

## 1.0.4 (October 18, 2021)

### NOTES:
Support for the HCP Packer registry is currently in beta and requires
Packer v1.7.7 [GH-160]

### IMPROVEMENTS:
* Add `SourceImageName` as shared builder information variable. [GH-160]
* Add `SourceImageName` to HCP Packer registry image metadata. [GH-160]
* Update packer-plugin-sdk to v0.2.7 [GH-159]

### BUG FIXES:
* builder/arm: Fix panic when running the cleanup step on a failed deployment. [GH-155]

## 1.0.3 (September 13, 2021)

### NOTES:
HCP Packer private beta support requires Packer version 1.7.5 or 1.7.6 [GH-150]

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

