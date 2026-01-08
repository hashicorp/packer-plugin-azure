// Copyright IBM Corp. 2013, 2025
// SPDX-License-Identifier: MPL-2.0

package client

import (
	"time"

	"github.com/hashicorp/go-azure-sdk/resource-manager/compute/2022-03-01/virtualmachineimages"
	"github.com/hashicorp/go-azure-sdk/resource-manager/compute/2022-03-01/virtualmachines"
	"github.com/hashicorp/go-azure-sdk/resource-manager/compute/2022-03-02/disks"
	"github.com/hashicorp/go-azure-sdk/resource-manager/compute/2022-03-02/snapshots"
	"github.com/hashicorp/go-azure-sdk/resource-manager/compute/2023-07-03/galleryimages"
	"github.com/hashicorp/go-azure-sdk/resource-manager/compute/2023-07-03/galleryimageversions"
	"github.com/hashicorp/go-azure-sdk/resource-manager/compute/2024-03-01/images"
	"github.com/hashicorp/go-azure-sdk/sdk/auth"
)

var _ AzureClientSet = &AzureClientSetMock{}

// AzureClientSetMock provides a generic mock for AzureClientSet
type AzureClientSetMock struct {
	DisksClientMock                disks.DisksClient
	SnapshotsClientMock            snapshots.SnapshotsClient
	ImagesClientMock               images.ImagesClient
	VirtualMachinesClientMock      virtualmachines.VirtualMachinesClient
	VirtualMachineImagesClientMock virtualmachineimages.VirtualMachineImagesClient
	GalleryImagesClientMock        galleryimages.GalleryImagesClient
	GalleryImageVersionsClientMock galleryimageversions.GalleryImageVersionsClient
	MetadataClientMock             MetadataClientAPI
	SubscriptionIDMock             string
	PollingDurationMock            time.Duration
	AuthorizerMock                 auth.Authorizer
}

// DisksClient returns a DisksClient
func (m *AzureClientSetMock) DisksClient() disks.DisksClient {
	return m.DisksClientMock
}

// SnapshotsClient returns a SnapshotsClient
func (m *AzureClientSetMock) SnapshotsClient() snapshots.SnapshotsClient {
	return m.SnapshotsClientMock
}

// ImagesClient returns a ImagesClient
func (m *AzureClientSetMock) ImagesClient() images.ImagesClient {
	return m.ImagesClientMock
}

// VirtualMachineImagesClient returns a VirtualMachineImagesClient
func (m *AzureClientSetMock) VirtualMachineImagesClient() virtualmachineimages.VirtualMachineImagesClient {
	return m.VirtualMachineImagesClientMock
}

// VirtualMachinesClient returns a VirtualMachinesClient
func (m *AzureClientSetMock) VirtualMachinesClient() virtualmachines.VirtualMachinesClient {
	return m.VirtualMachinesClientMock
}

// GalleryImagesClient returns a GalleryImagesClient
func (m *AzureClientSetMock) GalleryImagesClient() galleryimages.GalleryImagesClient {
	return m.GalleryImagesClientMock
}

// GalleryImageVersionsClient returns a GalleryImageVersionsClient
func (m *AzureClientSetMock) GalleryImageVersionsClient() galleryimageversions.GalleryImageVersionsClient {
	return m.GalleryImageVersionsClientMock
}

// MetadataClient returns a MetadataClient
func (m *AzureClientSetMock) MetadataClient() MetadataClientAPI {
	return m.MetadataClientMock
}

// SubscriptionID returns SubscriptionIDMock
func (m *AzureClientSetMock) SubscriptionID() string {
	return m.SubscriptionIDMock
}

func (m *AzureClientSetMock) PollingDuration() time.Duration {
	return m.PollingDurationMock
}

func (m *AzureClientSetMock) TokenAuthorizer() auth.Authorizer {
	return m.AuthorizerMock
}
