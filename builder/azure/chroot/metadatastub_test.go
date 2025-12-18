// Copyright IBM Corp. 2013, 2025
// SPDX-License-Identifier: MPL-2.0

package chroot

import "github.com/hashicorp/packer-plugin-azure/builder/azure/common/client"

func withMetadataStub(f func()) {
	mdc := client.DefaultMetadataClient
	defer func() { client.DefaultMetadataClient = mdc }()
	client.DefaultMetadataClient = client.MetadataClientStub{
		ComputeInfo: client.ComputeInfo{
			SubscriptionID:    "testSubscriptionID",
			ResourceGroupName: "testResourceGroup",
			Name:              "testVM",
			Location:          "testLocation",
		},
	}

	f()
}
