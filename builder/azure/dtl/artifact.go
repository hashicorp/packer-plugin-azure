// Copyright IBM Corp. 2013, 2025
// SPDX-License-Identifier: MPL-2.0

package dtl

import (
	"bytes"
	"fmt"
)

const (
	BuilderId = "Azure.ResourceManagement.VMImage"
)

type AdditionalDiskArtifact struct {
	AdditionalDiskUri            string
	AdditionalDiskUriReadOnlySas string
}

type Artifact struct {
	// OS type: Linux, Windows
	OSType string

	// Managed Image
	ManagedImageResourceGroupName string
	ManagedImageName              string
	ManagedImageLocation          string
	ManagedImageId                string

	// Shared Image Gallery
	// ARM resource id for Shared Image Gallery
	ManagedImageSharedImageGalleryId string
	SharedImageGalleryLocation       string

	// Additional Disks
	AdditionalDisks *[]AdditionalDiskArtifact
}

func NewManagedImageArtifact(osType, resourceGroup, name, location, id string) (*Artifact, error) {
	return &Artifact{
		ManagedImageResourceGroupName: resourceGroup,
		ManagedImageName:              name,
		ManagedImageLocation:          location,
		ManagedImageId:                id,
		OSType:                        osType,
	}, nil
}

func NewManagedImageArtifactWithSIGAsDestination(osType, resourceGroup, name, location, id, destinationSharedImageGalleryId string) (*Artifact, error) {
	return &Artifact{
		ManagedImageResourceGroupName:    resourceGroup,
		ManagedImageName:                 name,
		ManagedImageLocation:             location,
		ManagedImageId:                   id,
		OSType:                           osType,
		ManagedImageSharedImageGalleryId: destinationSharedImageGalleryId,
	}, nil
}

func (*Artifact) BuilderId() string {
	return BuilderId
}

func (*Artifact) Files() []string {
	return []string{}
}

func (a *Artifact) Id() string {
	return a.ManagedImageId
}

func (a *Artifact) State(name string) interface{} {
	switch name {
	default:
		return nil
	}
}

func (a *Artifact) String() string {
	var buf bytes.Buffer

	buf.WriteString(fmt.Sprintf("%s:\n\n", a.BuilderId()))
	buf.WriteString(fmt.Sprintf("OSType: %s\n", a.OSType))
	buf.WriteString(fmt.Sprintf("ManagedImageResourceGroupName: %s\n", a.ManagedImageResourceGroupName))
	buf.WriteString(fmt.Sprintf("ManagedImageName: %s\n", a.ManagedImageName))
	buf.WriteString(fmt.Sprintf("ManagedImageId: %s\n", a.ManagedImageId))
	buf.WriteString(fmt.Sprintf("ManagedImageLocation: %s\n", a.ManagedImageLocation))
	if a.ManagedImageSharedImageGalleryId != "" {
		buf.WriteString(fmt.Sprintf("ManagedImageSharedImageGalleryId: %s\n", a.ManagedImageSharedImageGalleryId))
	}
	return buf.String()
}

func (*Artifact) Destroy() error {
	return nil
}
