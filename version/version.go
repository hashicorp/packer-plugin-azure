// Copyright IBM Corp. 2013, 2025
// SPDX-License-Identifier: MPL-2.0

package version

import (
	"github.com/hashicorp/packer-plugin-sdk/version"
)

var (
	Version            = "2.5.1"
	VersionPrerelease  = ""
	VersionMetadata    = ""
	AzurePluginVersion = version.NewPluginVersion(Version, VersionPrerelease, VersionMetadata)
)
