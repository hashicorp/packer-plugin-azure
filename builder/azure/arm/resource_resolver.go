// Copyright (c) HashiCorp, Inc.
// SPDX-License-Identifier: MPL-2.0
package arm

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
	"github.com/hashicorp/go-azure-sdk/resource-manager/compute/2022-03-01/images"
	"github.com/hashicorp/go-azure-sdk/resource-manager/network/2022-09-01/subnets"
)

type resourceResolver struct {
	client                          *AzureClient
	findVirtualNetworkResourceGroup func(*AzureClient, string, string) (string, error)
	findVirtualNetworkSubnet        func(*AzureClient, string, string, string) (string, error)
}

func newResourceResolver(client *AzureClient) *resourceResolver {
	return &resourceResolver{
		client:                          client,
		findVirtualNetworkResourceGroup: findVirtualNetworkResourceGroup,
		findVirtualNetworkSubnet:        findVirtualNetworkSubnet,
	}
}

func (s *resourceResolver) Resolve(c *Config) error {
	if s.shouldResolveResourceGroup(c) {
		resourceGroupName, err := s.findVirtualNetworkResourceGroup(s.client, c.ClientConfig.SubscriptionID, c.VirtualNetworkName)
		if err != nil {
			return err
		}

		subnetName, err := s.findVirtualNetworkSubnet(s.client, c.ClientConfig.SubscriptionID, resourceGroupName, c.VirtualNetworkName)
		if err != nil {
			return err
		}

		c.VirtualNetworkResourceGroupName = resourceGroupName
		c.VirtualNetworkSubnetName = subnetName
	}

	if s.shouldResolveManagedImageName(c) {
		image, err := findManagedImageByName(s.client, c.CustomManagedImageName, c.ClientConfig.SubscriptionID, c.CustomManagedImageResourceGroupName)
		if err != nil {
			return err
		}

		c.customManagedImageID = *image.Id
	}

	return nil
}

func (s *resourceResolver) shouldResolveResourceGroup(c *Config) bool {
	return c.VirtualNetworkName != "" && c.VirtualNetworkResourceGroupName == ""
}

func (s *resourceResolver) shouldResolveManagedImageName(c *Config) bool {
	return c.CustomManagedImageName != ""
}

func getResourceGroupNameFromId(id string) string {
	// "/subscriptions/3f499422-dd76-4114-8859-86d526c9deb6/resourceGroups/packer-Resource-Group-yylnwsl30j/providers/...
	xs := strings.Split(id, "/")
	return xs[4]
}

func findManagedImageByName(client *AzureClient, name, subscriptionId, resourceGroupName string) (*images.Image, error) {
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

func findVirtualNetworkResourceGroup(client *AzureClient, subscriptionId, name string) (string, error) {
	vnetListContext, cancel := context.WithTimeout(context.TODO(), client.PollingDuration)
	defer cancel()
	virtualNetworks, err := client.NetworkMetaClient.VirtualNetworks.ListAllComplete(vnetListContext, commonids.NewSubscriptionID(subscriptionId))
	if err != nil {
		return "", err
	}

	resourceGroupNames := make([]string, 0)
	for _, virtualNetwork := range virtualNetworks.Items {
		if strings.EqualFold(name, *virtualNetwork.Name) {
			rgn := getResourceGroupNameFromId(*virtualNetwork.Id)
			resourceGroupNames = append(resourceGroupNames, rgn)
		}
	}

	if len(resourceGroupNames) == 0 {
		return "", fmt.Errorf("Cannot find a resource group with a virtual network called %q", name)
	}

	if len(resourceGroupNames) > 1 {
		return "", fmt.Errorf("Found multiple resource groups with a virtual network called %q, please use virtual_network_subnet_name and virtual_network_resource_group_name to disambiguate", name)
	}

	return resourceGroupNames[0], nil
}

func findVirtualNetworkSubnet(client *AzureClient, subscriptionId string, resourceGroupName string, name string) (string, error) {

	subnetListContext, cancel := context.WithTimeout(context.TODO(), client.PollingDuration)
	defer cancel()
	subnets, err := client.NetworkMetaClient.Subnets.List(subnetListContext, subnets.NewVirtualNetworkID(subscriptionId, resourceGroupName, name))
	if err != nil {
		return "", err
	}

	subnetList := *subnets.Model

	if len(subnetList) == 0 {
		return "", fmt.Errorf("Cannot find a subnet in the resource group %q associated with the virtual network called %q", resourceGroupName, name)
	}

	if len(subnetList) > 1 {
		return "", fmt.Errorf("Found multiple subnets in the resource group %q associated with the virtual network called %q, please use virtual_network_subnet_name and virtual_network_resource_group_name to disambiguate", resourceGroupName, name)
	}

	subnet := subnetList[0]
	return *subnet.Name, nil
}
