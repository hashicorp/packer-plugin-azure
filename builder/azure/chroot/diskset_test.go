// Copyright (c) HashiCorp, Inc.
// SPDX-License-Identifier: MPL-2.0

package chroot

import "github.com/hashicorp/packer-plugin-azure/builder/azure/common/client"

// diskset easily creates a diskset for testing
func diskset(ids ...string) Diskset {
	diskset := make(Diskset)
	for i, id := range ids {
		r, err := client.ParseResourceID(id)
		if err != nil {
			panic(err)
		}
		diskset[int64(i-1)] = r
	}
	return diskset
}
