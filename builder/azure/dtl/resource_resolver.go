// Copyright (c) HashiCorp, Inc.
// SPDX-License-Identifier: MPL-2.0

package dtl

// Code to resolve resources that are required by the API.  These resources
// can most likely be resolved without asking the user, thereby reducing the
// amount of configuration they need to provide.
//
// Resource resolver differs from config retriever because resource resolver
// requires a client to communicate with the Azure API.  A config retriever is
// used to determine values without use of a client.

import (
	"context"
	"fmt"
	"strings"

	"github.com/hashicorp/go-azure-helpers/resourcemanager/commonids"
	hashiImagesSDK "github.com/hashicorp/go-azure-sdk/resource-manager/compute/2022-03-01/images"
)

type resourceResolver struct {
	client                          *AzureClient
	findVirtualNetworkResourceGroup func(*AzureClient, string) (string, error)
	findVirtualNetworkSubnet        func(*AzureClient, string, string) (string, error)
}

func newResourceResolver(client *AzureClient) *resourceResolver {
	return &resourceResolver{
		client:                          client,
	}
}

func (s *resourceResolver) Resolve(c *Config) error {
	if s.shouldResolveManagedImageName(c) {
		image, err := findManagedImageByName(s.client, c.ClientConfig.SubscriptionID, c.CustomManagedImageName, c.CustomManagedImageResourceGroupName)
		if err != nil {
			return err
		}

		c.customManagedImageID = *image.Id
	}

	return nil
}

func (s *resourceResolver) shouldResolveManagedImageName(c *Config) bool {
	return c.CustomManagedImageName != ""
}

func getResourceGroupNameFromId(id string) string {
	// "/subscriptions/3f499422-dd76-4114-8859-86d526c9deb6/resourceGroups/packer-Resource-Group-yylnwsl30j/providers/...
	xs := strings.Split(id, "/")
	return xs[4]
}

func findManagedImageByName(client *AzureClient, name, subscriptionId, resourceGroupName string) (*hashiImagesSDK.Image, error) {
	id := commonids.NewResourceGroupID(subscriptionId, resourceGroupName)
	images, err := client.ImagesClient.ListByResourceGroupComplete(context.TODO(), id)
	if err != nil {
		return nil, err
	}

	for _, image := range images.Items {
		if strings.EqualFold(name, *image.Name) {
			return &image, nil
		}
	}

	return nil, fmt.Errorf("Cannot find an image named '%s' in the resource group '%s'", name, resourceGroupName)
}

