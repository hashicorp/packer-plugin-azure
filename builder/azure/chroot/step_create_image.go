// Copyright (c) HashiCorp, Inc.
// SPDX-License-Identifier: MPL-2.0

package chroot

import (
	"context"
	"fmt"
	"log"
	"sort"

	"github.com/hashicorp/go-azure-sdk/resource-manager/compute/2022-03-01/images"
	"github.com/hashicorp/packer-plugin-azure/builder/azure/common"
	"github.com/hashicorp/packer-plugin-azure/builder/azure/common/client"
	"github.com/hashicorp/packer-plugin-sdk/multistep"
	packersdk "github.com/hashicorp/packer-plugin-sdk/packer"
)

var _ multistep.Step = &StepCreateImage{}

type StepCreateImage struct {
	ImageResourceID            string
	ImageOSState               string
	OSDiskStorageAccountType   string
	OSDiskCacheType            string
	DataDiskStorageAccountType string
	DataDiskCacheType          string
	Location                   string

	create func(ctx context.Context, client client.AzureClientSet, id images.ImageId, image images.Image) error
}

func NewStepCreateImage(step *StepCreateImage) *StepCreateImage {
	step.create = step.createImage
	return step
}

func (s *StepCreateImage) Run(ctx context.Context, state multistep.StateBag) multistep.StepAction {
	azcli := state.Get("azureclient").(client.AzureClientSet)
	ui := state.Get("ui").(packersdk.Ui)
	diskset := state.Get(stateBagKey_Diskset).(Diskset)
	diskResourceID := diskset.OS().String()

	ui.Say(fmt.Sprintf("Creating image %s\n   using %s for os disk.",
		s.ImageResourceID,
		diskResourceID))

	imageResource, err := client.ParseResourceID(s.ImageResourceID)

	if err != nil {
		log.Printf("StepCreateImage.Run: error: %+v", err)
		err := fmt.Errorf(
			"error parsing image resource id '%s': %v", s.ImageResourceID, err)
		state.Put("error", err)
		ui.Error(err.Error())
		return multistep.ActionHalt
	}

	storageAccountType := images.StorageAccountTypes(s.OSDiskStorageAccountType)
	cacheingType := images.CachingTypes(s.OSDiskCacheType)
	image := images.Image{
		Location: s.Location,
		Properties: &images.ImageProperties{
			StorageProfile: &images.ImageStorageProfile{
				OsDisk: &images.ImageOSDisk{
					OsState: images.OperatingSystemStateTypes(s.ImageOSState),
					OsType:  images.OperatingSystemTypesLinux,
					ManagedDisk: &images.SubResource{
						Id: &diskResourceID,
					},
					StorageAccountType: &storageAccountType,
					Caching:            &cacheingType,
				},
			},
		},
	}

	var datadisks []images.ImageDataDisk
	if len(diskset) > 0 {
		storageAccountType = images.StorageAccountTypes(s.DataDiskStorageAccountType)
		cacheingType = images.CachingTypes(s.DataDiskStorageAccountType)
	}
	for lun, resource := range diskset {
		if lun != -1 {
			ui.Say(fmt.Sprintf("   using %q for data disk (lun %d).", resource, lun))

			datadisks = append(datadisks, images.ImageDataDisk{
				Lun:                lun,
				ManagedDisk:        &images.SubResource{Id: common.StringPtr(resource.String())},
				StorageAccountType: &storageAccountType,
				Caching:            &cacheingType,
			})
		}
	}
	if datadisks != nil {
		sort.Slice(datadisks, func(i, j int) bool {
			return datadisks[i].Lun < datadisks[j].Lun
		})
		image.Properties.StorageProfile.DataDisks = &datadisks
	}

	id := images.NewImageID(azcli.SubscriptionID(), imageResource.ResourceGroup, imageResource.ResourceName.String())
	err = s.create(
		ctx,
		azcli,
		id,
		image)
	if err != nil {
		log.Printf("StepCreateImage.Run: error: %+v", err)
		err := fmt.Errorf(
			"error creating image '%s': %v", s.ImageResourceID, err)
		state.Put("error", err)
		ui.Error(err.Error())
		return multistep.ActionHalt
	}
	log.Printf("Image creation complete")

	return multistep.ActionContinue
}

func (s *StepCreateImage) createImage(ctx context.Context, client client.AzureClientSet, id images.ImageId, image images.Image) error {
	pollingContext, cancel := context.WithTimeout(ctx, client.PollingDelay())
	defer cancel()
	return client.ImagesClient().CreateOrUpdateThenPoll(pollingContext, id, image)
}

func (*StepCreateImage) Cleanup(bag multistep.StateBag) {} // this is the final artifact, don't delete
