// Copyright (c) HashiCorp, Inc.
// SPDX-License-Identifier: MPL-2.0

package client

import (
	"context"
	"fmt"
	"regexp"
	"strings"
	"time"

	"github.com/hashicorp/packer-plugin-sdk/useragent"

	"github.com/hashicorp/go-azure-sdk/resource-manager/compute/2022-03-01/images"
	"github.com/hashicorp/go-azure-sdk/resource-manager/compute/2022-03-01/virtualmachineimages"
	"github.com/hashicorp/go-azure-sdk/resource-manager/compute/2022-03-01/virtualmachines"
	"github.com/hashicorp/go-azure-sdk/resource-manager/compute/2022-03-02/disks"
	"github.com/hashicorp/go-azure-sdk/resource-manager/compute/2022-03-02/snapshots"
	"github.com/hashicorp/go-azure-sdk/resource-manager/compute/2023-07-03/galleryimages"
	"github.com/hashicorp/go-azure-sdk/resource-manager/compute/2023-07-03/galleryimageversions"
	"github.com/hashicorp/go-azure-sdk/sdk/auth"
	version "github.com/hashicorp/packer-plugin-azure/version"
)

type AzureClientSet interface {
	MetadataClient() MetadataClientAPI

	DisksClient() disks.DisksClient
	SnapshotsClient() snapshots.SnapshotsClient
	ImagesClient() images.ImagesClient

	GalleryImagesClient() galleryimages.GalleryImagesClient
	GalleryImageVersionsClient() galleryimageversions.GalleryImageVersionsClient

	VirtualMachinesClient() virtualmachines.VirtualMachinesClient
	VirtualMachineImagesClient() virtualmachineimages.VirtualMachineImagesClient

	// SubscriptionID returns the subscription ID that this client set was created for
	SubscriptionID() string

	PollingDuration() time.Duration
}

// AzureClientSet is used for API requests on the chroot builder
var _ AzureClientSet = &azureClientSet{}

type azureClientSet struct {
	authorizer                 auth.Authorizer
	subscriptionID             string
	PollingDelay               time.Duration
	pollingDuration            time.Duration
	ResourceManagerEndpoint    string
	disksClient                disks.DisksClient
	snapshotsClient            snapshots.SnapshotsClient
	imagesClient               images.ImagesClient
	virtualMachinesClient      virtualmachines.VirtualMachinesClient
	virtualMachineImagesClient virtualmachineimages.VirtualMachineImagesClient
	galleryImagesClient        galleryimages.GalleryImagesClient
	galleryImageVersionsClient galleryimageversions.GalleryImageVersionsClient
}

func New(c Config, say func(string)) (AzureClientSet, error) {
	return new(c, say)
}

func new(c Config, say func(string)) (*azureClientSet, error) {
	// Pass in relevant auth information for hashicorp/go-azure-sdk
	authOptions := AzureAuthOptions{
		AuthType:           c.AuthType(),
		ClientID:           c.ClientID,
		ClientSecret:       c.ClientSecret,
		ClientJWT:          c.ClientJWT,
		ClientCertPath:     c.ClientCertPath,
		ClientCertPassword: c.ClientCertPassword,
		TenantID:           c.TenantID,
		SubscriptionID:     c.SubscriptionID,
	}
	cloudEnv := c.cloudEnvironment
	resourceManagerEndpoint, _ := cloudEnv.ResourceManager.Endpoint()
	authorizerContext, cancel := context.WithTimeout(context.Background(), time.Minute*15)
	defer cancel()
	authorizer, err := BuildResourceManagerAuthorizer(authorizerContext, authOptions, *cloudEnv)
	if err != nil {
		return nil, err
	}
	imagesClient, err := images.NewImagesClientWithBaseURI(cloudEnv.ResourceManager)
	if err != nil {
		return nil, err
	}
	imagesClient.Client.Authorizer = authorizer
	imagesClient.Client.UserAgent = fmt.Sprintf("%s %s", useragent.String(version.AzurePluginVersion.FormattedVersion()), imagesClient.Client.UserAgent)

	galleryImageVersionsClient, err := galleryimageversions.NewGalleryImageVersionsClientWithBaseURI(cloudEnv.ResourceManager)
	if err != nil {
		return nil, err
	}
	galleryImageVersionsClient.Client.Authorizer = authorizer
	galleryImageVersionsClient.Client.UserAgent = fmt.Sprintf("%s %s", useragent.String(version.AzurePluginVersion.FormattedVersion()), galleryImageVersionsClient.Client.UserAgent)

	galleryImagesClient, err := galleryimages.NewGalleryImagesClientWithBaseURI(cloudEnv.ResourceManager)
	if err != nil {
		return nil, err
	}
	galleryImagesClient.Client.Authorizer = authorizer
	galleryImagesClient.Client.UserAgent = fmt.Sprintf("%s %s", useragent.String(version.AzurePluginVersion.FormattedVersion()), galleryImagesClient.Client.UserAgent)

	disksClient, err := disks.NewDisksClientWithBaseURI(cloudEnv.ResourceManager)
	if err != nil {
		return nil, err
	}
	disksClient.Client.Authorizer = authorizer
	disksClient.Client.UserAgent = useragent.String(version.AzurePluginVersion.FormattedVersion())

	snapshotsClient, err := snapshots.NewSnapshotsClientWithBaseURI(cloudEnv.ResourceManager)
	if err != nil {
		return nil, err
	}
	snapshotsClient.Client.Authorizer = authorizer
	snapshotsClient.Client.UserAgent = fmt.Sprintf("%s %s", useragent.String(version.AzurePluginVersion.FormattedVersion()), snapshotsClient.Client.UserAgent)

	virtualMachinesClient, err := virtualmachines.NewVirtualMachinesClientWithBaseURI(cloudEnv.ResourceManager)
	if err != nil {
		return nil, err
	}
	virtualMachinesClient.Client.Authorizer = authorizer
	virtualMachinesClient.Client.UserAgent = fmt.Sprintf("%s %s", useragent.String(version.AzurePluginVersion.FormattedVersion()), virtualMachinesClient.Client.UserAgent)

	virtualMachineImagesClient, err := virtualmachineimages.NewVirtualMachineImagesClientWithBaseURI(cloudEnv.ResourceManager)
	if err != nil {
		return nil, err
	}
	virtualMachineImagesClient.Client.Authorizer = authorizer
	virtualMachineImagesClient.Client.UserAgent = fmt.Sprintf("%s %s", useragent.String(version.AzurePluginVersion.FormattedVersion()), virtualMachinesClient.Client.UserAgent)

	return &azureClientSet{
		authorizer:                 authorizer,
		subscriptionID:             c.SubscriptionID,
		PollingDelay:               time.Second,
		imagesClient:               *imagesClient,
		galleryImagesClient:        *galleryImagesClient,
		galleryImageVersionsClient: *galleryImageVersionsClient,
		disksClient:                *disksClient,
		virtualMachinesClient:      *virtualMachinesClient,
		virtualMachineImagesClient: *virtualMachineImagesClient,
		snapshotsClient:            *snapshotsClient,
		pollingDuration:            time.Minute * 15,
		ResourceManagerEndpoint:    *resourceManagerEndpoint,
	}, nil
}

func (s azureClientSet) SubscriptionID() string {
	return s.subscriptionID
}

func (s azureClientSet) PollingDuration() time.Duration {
	return s.pollingDuration
}

func (s azureClientSet) MetadataClient() MetadataClientAPI {
	return metadataClient{}
}

func (s azureClientSet) DisksClient() disks.DisksClient {
	return s.disksClient
}

func (s azureClientSet) SnapshotsClient() snapshots.SnapshotsClient {
	return s.snapshotsClient
}

func (s azureClientSet) ImagesClient() images.ImagesClient {
	return s.imagesClient
}

func (s azureClientSet) VirtualMachinesClient() virtualmachines.VirtualMachinesClient {
	return s.virtualMachinesClient
}

func (s azureClientSet) VirtualMachineImagesClient() virtualmachineimages.VirtualMachineImagesClient {
	return s.virtualMachineImagesClient
}

func (s azureClientSet) GalleryImagesClient() galleryimages.GalleryImagesClient {
	return s.galleryImagesClient
}

func (s azureClientSet) GalleryImageVersionsClient() galleryimageversions.GalleryImageVersionsClient {
	return s.galleryImageVersionsClient

}

func ParsePlatformImageURN(urn string) (image *PlatformImage, err error) {
	if !platformImageRegex.Match([]byte(urn)) {
		return nil, fmt.Errorf("%q is not a valid platform image specifier", urn)
	}
	parts := strings.Split(urn, ":")
	return &PlatformImage{parts[0], parts[1], parts[2], parts[3]}, nil
}

var platformImageRegex = regexp.MustCompile(`^[-_.a-zA-Z0-9]+:[-_.a-zA-Z0-9]+:[-_.a-zA-Z0-9]+:[-_.a-zA-Z0-9]+$`)

type PlatformImage struct {
	Publisher, Offer, Sku, Version string
}

func (pi PlatformImage) URN() string {
	return fmt.Sprintf("%s:%s:%s:%s",
		pi.Publisher,
		pi.Offer,
		pi.Sku,
		pi.Version)
}
