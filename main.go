// Copyright (c) HashiCorp, Inc.
// SPDX-License-Identifier: MPL-2.0

package main

import (
	"fmt"
	"os"

	azurearm "github.com/hashicorp/packer-plugin-azure/builder/azure/arm"
	azurechroot "github.com/hashicorp/packer-plugin-azure/builder/azure/chroot"
	azuredtl "github.com/hashicorp/packer-plugin-azure/builder/azure/dtl"
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
	pps.SetVersion(version.AzurePluginVersion)
	err := pps.Run()
	if err != nil {
		fmt.Fprintln(os.Stderr, err.Error())
		os.Exit(1)
	}
}
