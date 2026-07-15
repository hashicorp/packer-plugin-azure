// Copyright IBM Corp. 2013, 2026
// SPDX-License-Identifier: MPL-2.0

package version

import (
	"github.com/hashicorp/packer-plugin-sdk/version"
)

var (
	Version            = "2.6.3"
	VersionPrerelease  = ""
	VersionMetadata    = ""
	AzurePluginVersion = version.NewPluginVersion(Version, VersionPrerelease, VersionMetadata)
)
