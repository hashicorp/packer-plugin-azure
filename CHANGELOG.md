# Latest Release

Please refer to [releases](https://github.com/hashicorp/packer-plugin-azure/releases) for the latest CHANGELOG information.

---
## 2.6.1 (April 28, 2026)

## What's Changed
### Bug Fixes
* Align OIDC option names (`OIDCTokenRequestURL`, `OIDCTokenRequestToken`) with latest `hashicorp/go-azure-helpers` SDK [GH-608](https://github.com/hashicorp/packer-plugin-azure/pull/608)

### Other Changes
* Update Go to 1.25.9 and refresh module dependencies [GH-608](https://github.com/hashicorp/packer-plugin-azure/pull/608)
* Update `go-ntlmssp` to v0.1.1 for stability and security improvements [GH-609](https://github.com/hashicorp/packer-plugin-azure/pull/609)

**Full Changelog**: https://github.com/hashicorp/packer-plugin-azure/compare/v2.6.0...v2.6.1

## 2.6.0 (March 23, 2026)

## What's Changed
### Exciting New Features
* Add built-in LVM support to the `azure-chroot` builder, including the optional `lvm_root_device` override [GH-583](https://github.com/hashicorp/packer-plugin-azure/pull/583)
* Support sourcing Azure Compute Gallery images across subscriptions and preserve host availability zones for `azure-chroot` disks [GH-582](https://github.com/hashicorp/packer-plugin-azure/pull/582)
* Add `accelerated_networking` and `sas_token_duration` options to the `azure-arm` builder [GH-580](https://github.com/hashicorp/packer-plugin-azure/pull/580)
* Add `disk_controller_type` support to the `azure-arm` builder [GH-592](https://github.com/hashicorp/packer-plugin-azure/pull/592)
* Add `StandardSSD_LRS` support for `managed_image_storage_account_type` [GH-596](https://github.com/hashicorp/packer-plugin-azure/pull/596)

### Bug Fixes
* Allow `skip_create_image` without requiring capture destinations in the ARM and chroot builders [GH-579](https://github.com/hashicorp/packer-plugin-azure/pull/579)
* Relax resource group validation and VHD copy duration handling to avoid false failures [GH-567](https://github.com/hashicorp/packer-plugin-azure/pull/567)

### Other Changes
* Update dependencies to latest compatible versions [GH-599](https://github.com/hashicorp/packer-plugin-azure/pull/599)
* Bump `github.com/hashicorp/packer-plugin-sdk` from `0.6.4` to `0.6.6` [GH-570](https://github.com/hashicorp/packer-plugin-azure/pull/570), [GH-597](https://github.com/hashicorp/packer-plugin-azure/pull/597)
* Upgrade CI dependencies and release automation actions [GH-586](https://github.com/hashicorp/packer-plugin-azure/pull/586), [GH-587](https://github.com/hashicorp/packer-plugin-azure/pull/587), [GH-589](https://github.com/hashicorp/packer-plugin-azure/pull/589), [GH-590](https://github.com/hashicorp/packer-plugin-azure/pull/590), [GH-595](https://github.com/hashicorp/packer-plugin-azure/pull/595), [GH-598](https://github.com/hashicorp/packer-plugin-azure/pull/598)
* Prevent credential persistence in CI checkouts [GH-593](https://github.com/hashicorp/packer-plugin-azure/pull/593)
* Update `golangci-lint` to v2 [GH-594](https://github.com/hashicorp/packer-plugin-azure/pull/594)
* Improve Dependabot support for GitHub Actions updates [GH-569](https://github.com/hashicorp/packer-plugin-azure/pull/569)

**Full Changelog**: https://github.com/hashicorp/packer-plugin-azure/compare/v2.5.2...v2.6.0

## 2.5.2 (February 4, 2026)

## What's Changed
### Exciting New Features

### Bug Fixes
* BUG: Updates tests for permissions and polling [GH-572](https://github.com/hashicorp/packer-plugin-azure/pull/572)

### Other Changes
* Adds dynamic resource name suffixing for isolation of Acceptance tests [GH-575](https://github.com/hashicorp/packer-plugin-azure/pull/575)
* Updates Go version to 1.24.12 in module file [GH-573](https://github.com/hashicorp/packer-plugin-azure/pull/573)
* Fix various lint warnings. [GH-566](https://github.com/hashicorp/packer-plugin-azure/pull/566)
* Staticcheck/QF1003: Replace if-else with switch-case for OSType checks. [GH-565](https://github.com/hashicorp/packer-plugin-azure/pull/565)
* Replace deprecated ui.Message with ui.Say [GH-564](https://github.com/hashicorp/packer-plugin-azure/pull/564)

**Full Changelog**: https://github.com/hashicorp/packer-plugin-azure/compare/v2.5.1...v2.5.2

## 2.5.1 (December 18, 2025)

## What's Changed
### Exciting New Features
* Support setting custom_resource_build_prefix via environment variable [GH-541](https://github.com/hashicorp/packer-plugin-azure/pull/541)
* Add manual mount command option [GH-545](https://github.com/hashicorp/packer-plugin-azure/pull/545)

### Bug Fixes
* Fix disablePasswordAuthentication option [GH-550](https://github.com/hashicorp/packer-plugin-azure/pull/550)
* Fix: Update number of allowed resource tags [GH-552](https://github.com/hashicorp/packer-plugin-azure/pull/552)
* Increases RSA key size in tests to 2048 bits [GH-555](https://github.com/hashicorp/packer-plugin-azure/pull/555)

### Other Changes
* Add backport-assistant [GH-542](https://github.com/hashicorp/packer-plugin-azure/pull/542)
* Bump github.com/hashicorp/packer-plugin-sdk from 0.6.2 to 0.6.4 [GH-546](https://github.com/hashicorp/packer-plugin-azure/pull/546)
* [COMPLIANCE] Update Copyright Headers by @oss-core-libraries-dashboard[bot] [GH-551](https://github.com/hashicorp/packer-plugin-azure/pull/551)
* Bump x/crypto to v0.46.0 [GH-554](https://github.com/hashicorp/packer-plugin-azure/pull/554)

**Full Changelog**: https://github.com/hashicorp/packer-plugin-azure/compare/v2.5.0...v2.5.1

## 2.5.0 (September 3, 2025)

## What's Changed
### Breaking Changes
* VHDs are no longer built using unmanaged disks, this change was made because of the following deprecation notice from Microsoft https://learn.microsoft.com/en-us/azure/virtual-machines/unmanaged-disks-deprecation.  Users must now create the capture container in their storage account at the root of the container.  The plugin no longer relies on Azure system created capture containers.
### Exciting New Features
* Added Support for multiple artifact, (i.e. VHD, SharedImageGallery and Managed Image) creation in the same build in [GH-522](https://github.com/hashicorp/packer-plugin-azure/pull/522)

### Bug Fixes
* Move SIG Regex check to builder to fix validation failures in [GH-531](https://github.com/hashicorp/packer-plugin-azure/pull/531)
* Fixed Release Artifact Schema file name in [GH-532](https://github.com/hashicorp/packer-plugin-azure/pull/532)
* Fixed Unmanaged Disks in [GH-522](https://github.com/hashicorp/packer-plugin-azure/pull/522)
* VHD Migrations Fixes - VHD Acceptance Tests, and Disk Revoke Access on Failures in [GH-534](https://github.com/hashicorp/packer-plugin-azure/pull/534)

### Other Changes
* Packer Plugin SDK v0.6.1 => v0.6.2 and run make generate in [GH-525](https://github.com/hashicorp/packer-plugin-azure/pull/525)
* Remove unused constant causing linter failure in [GH-533](https://github.com/hashicorp/packer-plugin-azure/pull/533)
* Updated Module for Security Vulnerability in [GH-535](https://github.com/hashicorp/packer-plugin-azure/pull/535)

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

