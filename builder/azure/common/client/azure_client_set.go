// Copyright (c) HashiCorp, Inc.
// SPDX-License-Identifier: MPL-2.0

package client

import (
	"context"
	"fmt"
	"log"
	"net/http"
	"regexp"
	"strings"
	"time"

	"github.com/hashicorp/packer-plugin-sdk/useragent"

	"github.com/Azure/go-autorest/autorest"
	"github.com/hashicorp/go-azure-sdk/resource-manager/compute/2022-03-01/images"
	"github.com/hashicorp/go-azure-sdk/resource-manager/compute/2022-03-01/virtualmachineimages"
	"github.com/hashicorp/go-azure-sdk/resource-manager/compute/2022-03-01/virtualmachines"
	"github.com/hashicorp/go-azure-sdk/resource-manager/compute/2022-03-02/disks"
	"github.com/hashicorp/go-azure-sdk/resource-manager/compute/2022-03-02/snapshots"
	"github.com/hashicorp/go-azure-sdk/resource-manager/compute/2022-03-03/galleryimages"
	"github.com/hashicorp/go-azure-sdk/resource-manager/compute/2022-03-03/galleryimageversions"
	"github.com/hashicorp/go-azure-sdk/sdk/auth"
	authWrapper "github.com/hashicorp/go-azure-sdk/sdk/auth/autorest"
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

	PollingDelay() time.Duration
}

var _ AzureClientSet = &azureClientSet{}

type azureClientSet struct {
	sender                  autorest.Sender
	authorizer              auth.Authorizer
	subscriptionID          string
	pollingDelay            time.Duration
	ResourceManagerEndpoint string
}

func New(c Config, say func(string)) (AzureClientSet, error) {
	return new(c, say)
}

func new(c Config, say func(string)) (*azureClientSet, error) {
	// Pass in relevant auth information for hashicorp/go-azure-sdk
	authOptions := AzureAuthOptions{
		AuthType:       c.AuthType(),
		ClientID:       c.ClientID,
		ClientSecret:   c.ClientSecret,
		ClientJWT:      c.ClientJWT,
		ClientCertPath: c.ClientCertPath,
		TenantID:       c.TenantID,
		SubscriptionID: c.SubscriptionID,
	}
	cloudEnv := c.cloudEnvironment
	resourceManagerEndpoint, _ := cloudEnv.ResourceManager.Endpoint()
	authorizer, err := BuildResourceManagerAuthorizer(context.TODO(), authOptions, *cloudEnv)
	if err != nil {
		return nil, err
	}
	return &azureClientSet{
		authorizer:              authorizer,
		subscriptionID:          c.SubscriptionID,
		sender:                  http.DefaultClient,
		pollingDelay:            time.Second,
		ResourceManagerEndpoint: *resourceManagerEndpoint,
	}, nil
}

func (s azureClientSet) SubscriptionID() string {
	return s.subscriptionID
}

func (s azureClientSet) PollingDelay() time.Duration {
	return s.pollingDelay
}

func (s azureClientSet) configureTrack1Client(c *autorest.Client) {
	err := c.AddToUserAgent(useragent.String(version.AzurePluginVersion.FormattedVersion()))
	if err != nil {
		log.Printf("Error appending Packer plugin version to user agent.")
	}
	c.Authorizer = authWrapper.AutorestAuthorizer(s.authorizer)
	c.Sender = s.sender
}

func (s azureClientSet) MetadataClient() MetadataClientAPI {
	return metadataClient{
		s.sender,
		useragent.String(version.AzurePluginVersion.FormattedVersion()),
	}
}

func (s azureClientSet) DisksClient() disks.DisksClient {
	c := disks.NewDisksClientWithBaseURI(s.ResourceManagerEndpoint)
	s.configureTrack1Client(&c.Client)
	c.Client.PollingDelay = s.pollingDelay
	return c
}

func (s azureClientSet) SnapshotsClient() snapshots.SnapshotsClient {
	c := snapshots.NewSnapshotsClientWithBaseURI(s.ResourceManagerEndpoint)
	s.configureTrack1Client(&c.Client)
	c.Client.PollingDelay = s.pollingDelay
	return c
}

func (s azureClientSet) ImagesClient() images.ImagesClient {
	c := images.NewImagesClientWithBaseURI(s.ResourceManagerEndpoint)
	s.configureTrack1Client(&c.Client)
	c.Client.PollingDelay = s.pollingDelay
	return c
}

func (s azureClientSet) VirtualMachinesClient() virtualmachines.VirtualMachinesClient {
	c := virtualmachines.NewVirtualMachinesClientWithBaseURI(s.ResourceManagerEndpoint)
	s.configureTrack1Client(&c.Client)
	c.Client.PollingDelay = s.pollingDelay
	return c
}

func (s azureClientSet) VirtualMachineImagesClient() virtualmachineimages.VirtualMachineImagesClient {
	c := virtualmachineimages.NewVirtualMachineImagesClientWithBaseURI(s.ResourceManagerEndpoint)
	s.configureTrack1Client(&c.Client)
	c.Client.PollingDelay = s.pollingDelay
	return c
}

func (s azureClientSet) GalleryImagesClient() galleryimages.GalleryImagesClient {
	c := galleryimages.NewGalleryImagesClientWithBaseURI(s.ResourceManagerEndpoint)
	s.configureTrack1Client(&c.Client)
	c.Client.PollingDelay = s.pollingDelay
	return c
}

func (s azureClientSet) GalleryImageVersionsClient() galleryimageversions.GalleryImageVersionsClient {
	c := galleryimageversions.NewGalleryImageVersionsClientWithBaseURI(s.ResourceManagerEndpoint)
	s.configureTrack1Client(&c.Client)
	c.Client.PollingDelay = s.pollingDelay
	return c
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
