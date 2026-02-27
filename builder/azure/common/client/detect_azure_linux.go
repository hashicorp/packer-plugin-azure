// Copyright IBM Corp. 2013, 2025
// SPDX-License-Identifier: MPL-2.0

package client

import (
	"bytes"
	"os"
)

var (
	smbiosAssetTagFile = "/sys/class/dmi/id/chassis_asset_tag"
	azureAssetTag      = []byte("7783-7084-3265-9085-8269-3286-77\n")
)

// IsAzure returns true if Packer is running on Azure
func IsAzure() bool {
	return isAzureAssetTag(smbiosAssetTagFile)
}

func isAzureAssetTag(filename string) bool {
	if d, err := os.ReadFile(filename); err == nil {
		return bytes.Equal(d, azureAssetTag)
	}
	return false
}
