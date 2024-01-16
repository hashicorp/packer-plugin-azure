// Copyright (c) HashiCorp, Inc.
// SPDX-License-Identifier: MPL-2.0

package client

import "strings"

// NormalizeLocation returns a normalized location string.
// Strings are converted to lower case and spaces are removed.
func NormalizeLocation(loc string) string {
	return strings.ReplaceAll(strings.ToLower(loc), " ", "")
}
