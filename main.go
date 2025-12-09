// Copyright IBM Corp. 2013, 2025
// SPDX-License-Identifier: MPL-2.0

package main

import (
	"fmt"
	"os"

	azurearm "github.com/hashicorp/packer-plugin-azure/builder/azure/arm"
	azurechroot "github.com/hashicorp/packer-plugin-azure/builder/azure/chroot"
	azuredtl "github.com/hashicorp/packer-plugin-azure/builder/azure/dtl"
	"github.com/hashicorp/packer-plugin-azure/datasource/keyvaultsecret"
	azuredtlartifact "github.com/hashicorp/packer-plugin-azure/provisioner/azure-dtlartifact"
	"github.com/hashicorp/packer-plugin-azure/version"

	"github.com/hashicorp/packer-plugin-sdk/plugin"
)

func main() {
	pps := plugin.NewSet()
	pps.RegisterBuilder("arm", new(azurearm.Builder))
	pps.RegisterBuilder("chroot", new(azurechroot.Builder))
	pps.RegisterBuilder("dtl", new(azuredtl.Builder))
	pps.RegisterProvisioner("dtlartifact", new(azuredtlartifact.Provisioner))
	pps.RegisterDatasource("keyvaultsecret", new(keyvaultsecret.Datasource))
	pps.SetVersion(version.AzurePluginVersion)
	err := pps.Run()
	if err != nil {
		fmt.Fprintln(os.Stderr, err.Error())
		os.Exit(1)
	}
}
