// Copyright (c) HashiCorp, Inc.
// SPDX-License-Identifier: MPL-2.0

package common

import (
	"context"
	"fmt"
	"log"
	"sort"
	"strings"

	"github.com/hashicorp/go-azure-sdk/resource-manager/compute/2022-03-01/images"
	"github.com/hashicorp/packer-plugin-azure/builder/azure/common/client"
	packersdk "github.com/hashicorp/packer-plugin-sdk/packer"
	registryimage "github.com/hashicorp/packer-plugin-sdk/packer/registry/image"
)

// Artifact is an artifact implementation that contains built Managed Images or Disks.
type Artifact struct {
	// Array of the Azure resource IDs that were created.
	Resources []string

	// BuilderId is the unique ID for the builder that created this AMI
	BuilderIdValue string

	// Azure client for performing API stuff.
	AzureClientSet client.AzureClientSet

	// StateData should store data such as GeneratedData
	// to be shared with post-processors
	StateData map[string]interface{}
}

func (a *Artifact) BuilderId() string {
	return a.BuilderIdValue
}

func (*Artifact) Files() []string {
	// We have no files
	return nil
}

func (a *Artifact) Id() string {
	parts := make([]string, 0, len(a.Resources))
	for _, resource := range a.Resources {
		parts = append(parts, strings.ToLower(resource))
	}

	sort.Strings(parts)
	return strings.Join(parts, ",")
}

func (a *Artifact) String() string {
	parts := make([]string, 0, len(a.Resources))
	for _, resource := range a.Resources {
		parts = append(parts, strings.ToLower(resource))
	}

	sort.Strings(parts)
	return fmt.Sprintf("Azure resources created:\n%s\n", strings.Join(parts, "\n"))
}

func (a *Artifact) State(name string) interface{} {
	if name == registryimage.ArtifactStateURI {
		return a.hcpPackerRegistryMetadata()
	}
	return a.StateData[name]
}

func (a *Artifact) Destroy() error {
	errs := make([]error, 0)

	for _, resource := range a.Resources {
		log.Printf("Deleting resource %s", resource)

		id, err := client.ParseResourceID(resource)
		if err != nil {
			return fmt.Errorf("Unable to parse resource id (%s): %v", resource, err)
		}

		ctx := context.TODO()
		restype := strings.ToLower(fmt.Sprintf("%s/%s", id.Provider, id.ResourceType))

		switch restype {
		case "microsoft.compute/images":
			imageID := images.NewImageID(a.AzureClientSet.SubscriptionID(), id.ResourceGroup, id.ResourceName.String())
			pollingContext, cancel := context.WithTimeout(ctx, a.AzureClientSet.PollingDuration())
			defer cancel()
			err := a.AzureClientSet.ImagesClient().DeleteThenPoll(pollingContext, imageID)
			if err != nil {
				errs = append(errs, fmt.Errorf("Unable to initiate deletion of resource (%s): %v", resource, err))
			}
		default:
			errs = append(errs, fmt.Errorf("Don't know how to delete resources of type %s (%s)", resource, restype))
		}

	}

	if len(errs) > 0 {
		if len(errs) == 1 {
			return errs[0]
		} else {
			return &packersdk.MultiError{Errors: errs}
		}
	}

	return nil
}

func (a *Artifact) hcpPackerRegistryMetadata() []*registryimage.Image {
	var generatedData map[string]interface{}

	if a.StateData != nil {
		generatedData = a.StateData["generated_data"].(map[string]interface{})
	}

	var sourceID string
	if sourceImage, ok := generatedData["SourceImageName"].(string); ok {
		sourceID = sourceImage
	}
	var images []*registryimage.Image
	for _, resource := range a.Resources {
		image, err := registryimage.FromArtifact(a,
			registryimage.WithProvider("azure"),
			registryimage.WithID(resource),
			registryimage.WithSourceID(sourceID),
		)

		if err != nil {
			continue
		}

		images = append(images, image)
	}

	return images
}
