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

