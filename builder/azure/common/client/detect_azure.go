// Copyright IBM Corp. 2013, 2025
// SPDX-License-Identifier: MPL-2.0

//go:build !linux
// +build !linux

package client

// IsAzure returns true if Packer is running on Azure (currently only works on Linux)
func IsAzure() bool {
	return false
}
