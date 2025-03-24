// Copyright (c) HashiCorp, Inc.
// SPDX-License-Identifier: MPL-2.0

package version

import (
	"github.com/hashicorp/packer-plugin-sdk/version"
)

var (
	Version            = "2.3.2"
	VersionPrerelease  = ""
	VersionMetadata    = ""
	AzurePluginVersion = version.NewPluginVersion(Version, VersionPrerelease, VersionMetadata)
)
