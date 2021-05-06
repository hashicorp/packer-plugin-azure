package main

import (
	"fmt"
	"os"

	azurearm "github.com/hashicorp/packer-plugin-azure/builder/azure/arm"
	azurechroot "github.com/hashicorp/packer-plugin-azure/builder/azure/chroot"
	azuredtl "github.com/hashicorp/packer-plugin-azure/builder/azure/dtl"
	azuredtlartifact "github.com/hashicorp/packer-plugin-azure/provisioner/azure-dtlartifact"
	internalPluginVersion "github.com/hashicorp/packer-plugin-azure/version"
	"github.com/hashicorp/packer-plugin-sdk/plugin"
	"github.com/hashicorp/packer-plugin-sdk/version"
)

var (
	// Version is the main version number that is being run at the moment.
	Version = "0.0.1"

	// VersionPrerelease is A pre-release marker for the Version. If this is ""
	// (empty string) then it means that it is a final release. Otherwise, this
	// is a pre-release such as "dev" (in development), "beta", "rc1", etc.
	VersionPrerelease = "dev"

	// PluginVersion is used by the plugin set to allow Packer to recognize
	// what version this plugin is.
	PluginVersion = version.InitializePluginVersion(Version, VersionPrerelease)
)

func main() {
	pps := plugin.NewSet()
	pps.RegisterBuilder("arm", new(azurearm.Builder))
	pps.RegisterBuilder("chroot", new(azurechroot.Builder))
	pps.RegisterBuilder("dtl", new(azuredtl.Builder))
	pps.RegisterProvisioner("dtlartifact", new(azuredtlartifact.Provisioner))
	pps.SetVersion(PluginVersion)
	internalPluginVersion.AzurePluginVersion = PluginVersion
	err := pps.Run()
	if err != nil {
		fmt.Fprintln(os.Stderr, err.Error())
		os.Exit(1)
	}
}
