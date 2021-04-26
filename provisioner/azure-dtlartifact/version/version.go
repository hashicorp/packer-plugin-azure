package version

import (
	"github.com/hashicorp/packer-plugin-sdk/version"
	packerVersion "github.com/hashicorp/packer-plugin-azure/version"
)

var AzureDTLPluginVersion *version.PluginVersion

func init() {
	AzureDTLPluginVersion = version.InitializePluginVersion(
		packerVersion.Version, packerVersion.VersionPrerelease)
}
