// Copyright (c) HashiCorp, Inc.
// SPDX-License-Identifier: MPL-2.0

package chroot

import (
	"context"
	"fmt"
	"strings"

	"github.com/hashicorp/packer-plugin-azure/builder/azure/common/log"

	"github.com/hashicorp/go-azure-helpers/resourcemanager/commonids"
	"github.com/hashicorp/packer-plugin-azure/builder/azure/common/client"
	"github.com/hashicorp/packer-plugin-sdk/multistep"
	packersdk "github.com/hashicorp/packer-plugin-sdk/packer"

	"github.com/hashicorp/go-azure-sdk/resource-manager/compute/2022-03-02/disks"
	"github.com/hashicorp/go-azure-sdk/resource-manager/compute/2022-03-03/galleryimageversions"
)

var _ multistep.Step = &StepCreateNewDiskset{}

type StepCreateNewDiskset struct {
	OSDiskID                   string // Disk ID
	OSDiskSizeGB               int64  // optional, ignored if 0
	OSDiskStorageAccountType   string // from compute.DiskStorageAccountTypes
	DataDiskStorageAccountType string // from compute.DiskStorageAccountTypes

	DataDiskIDPrefix string

	disks Diskset

	HyperVGeneration string // For OS disk

	// Copy another disk
	SourceOSDiskResourceID string

	// Extract from platform image
	SourcePlatformImage *client.PlatformImage
	// Extract from shared image
	SourceImageResourceID string
	// Location is needed for platform and shared images
	Location string

	SkipCleanup bool

	getVersion func(context.Context, client.AzureClientSet, galleryimageversions.ImageVersionId) (*galleryimageversions.GalleryImageVersion, error)
	create     func(context.Context, client.AzureClientSet, commonids.ManagedDiskId, disks.Disk) error
}

func NewStepCreateNewDiskset(step *StepCreateNewDiskset) *StepCreateNewDiskset {
	step.getVersion = step.getSharedImageGalleryVersion
	step.create = step.createDiskset
	return step
}
func (s *StepCreateNewDiskset) Run(ctx context.Context, state multistep.StateBag) multistep.StepAction {
	azcli := state.Get("azureclient").(client.AzureClientSet)
	ui := state.Get("ui").(packersdk.Ui)

	s.disks = make(Diskset)

	errorMessage := func(format string, params ...interface{}) multistep.StepAction {
		err := fmt.Errorf("StepCreateNewDiskset.Run: error: "+format, params...)
		state.Put("error", err)
		ui.Error(err.Error())
		return multistep.ActionHalt
	}

	// we always have an OS disk
	osDisk, err := client.ParseResourceID(s.OSDiskID)
	if err != nil {
		return errorMessage("error parsing resource id '%s': %v", s.OSDiskID, err)
	}
	if !strings.EqualFold(osDisk.Provider, "Microsoft.Compute") ||
		!strings.EqualFold(osDisk.ResourceType.String(), "disks") {
		return errorMessage("Resource %q is not of type Microsoft.Compute/disks", s.OSDiskID)
	}

	// transform step config to disk model
	disk := s.getOSDiskDefinition(azcli.SubscriptionID())

	// Initiate disk creation
	diskId := commonids.NewManagedDiskID(azcli.SubscriptionID(), osDisk.ResourceGroup, osDisk.ResourceName.String())
	ui.Say(fmt.Sprintf("Creating disk %q", s.OSDiskID))
	err = s.create(ctx, azcli, diskId, disk)
	if err != nil {
		return errorMessage("Failed to create disk: %q", osDisk)
	}
	ui.Say(fmt.Sprintf("Disk created: %q", s.OSDiskID))
	s.disks[-1] = osDisk                    // save the resoure we just create in our disk set
	state.Put(stateBagKey_Diskset, s.disks) // update the statebag

	if s.SourceImageResourceID != "" {
		// retrieve image to see if there are any datadisks
		imageID, err := client.ParseResourceID(s.SourceImageResourceID)
		if err != nil {
			return errorMessage("could not parse source image id %q: %v", s.SourceImageResourceID, err)
		}

		if !strings.EqualFold(imageID.Provider+"/"+imageID.ResourceType.String(),
			"Microsoft.Compute/galleries/images/versions") {
			return errorMessage("source image id is not a shared image version %q, expected type 'Microsoft.Compute/galleries/images/versions'", imageID)
		}
		galleryImageVersionId := galleryimageversions.NewImageVersionID(azcli.SubscriptionID(),
			imageID.ResourceGroup, imageID.ResourceName[0], imageID.ResourceName[1], imageID.ResourceName[2])
		image, err := s.getVersion(ctx, azcli, galleryImageVersionId)
		if err != nil {
			return errorMessage("error retrieving source image %q: %v", imageID, err)
		}
		if image.Properties != nil &&
			image.Properties.StorageProfile.DataDiskImages != nil {
			for _, ddi := range *image.Properties.StorageProfile.DataDiskImages {
				datadiskID, err := client.ParseResourceID(fmt.Sprintf("%s%d", s.DataDiskIDPrefix, ddi.Lun))
				if err != nil {
					return errorMessage("unable to construct resource id for datadisk: %v", err)
				}

				disk := s.getDatadiskDefinitionFromImage(ddi.Lun)
				// Initiate disk creation
				diskId := commonids.NewManagedDiskID(azcli.SubscriptionID(), datadiskID.ResourceGroup, datadiskID.ResourceName.String())
				ui.Say(fmt.Sprintf("Creating disk %q", datadiskID))
				err = s.create(ctx, azcli, diskId, disk)
				if err != nil {
					return errorMessage("Failed to create disk: %q", datadiskID)
				}
				ui.Say(fmt.Sprintf("Disk created %q", datadiskID))
				s.disks[ddi.Lun] = datadiskID           // save the resoure we just create in our disk set
				state.Put(stateBagKey_Diskset, s.disks) // update the statebag

			}
		}
	}
	return multistep.ActionContinue
}

func (s StepCreateNewDiskset) getOSDiskDefinition(subscriptionID string) disks.Disk {
	osType := disks.OperatingSystemTypesLinux
	disk := disks.Disk{
		Location: s.Location,
		Properties: &disks.DiskProperties{
			OsType:       &osType,
			CreationData: disks.CreationData{},
		},
	}

	if s.OSDiskStorageAccountType != "" {
		hashiDiskSkuName := disks.DiskStorageAccountTypes(s.OSDiskStorageAccountType)
		disk.Sku = &disks.DiskSku{
			Name: &hashiDiskSkuName,
		}
	}

	if s.HyperVGeneration != "" {
		hyperVGeneration := disks.HyperVGeneration(s.HyperVGeneration)
		disk.Properties.HyperVGeneration = &hyperVGeneration
	}

	if s.OSDiskSizeGB > 0 {
		disk.Properties.DiskSizeGB = &s.OSDiskSizeGB
	}

	switch {
	case s.SourcePlatformImage != nil:
		imageID := fmt.Sprintf(
			"/subscriptions/%s/providers/Microsoft.Compute/locations/%s/publishers/%s/artifacttypes/vmimage/offers/%s/skus/%s/versions/%s", subscriptionID, s.Location,
			s.SourcePlatformImage.Publisher, s.SourcePlatformImage.Offer, s.SourcePlatformImage.Sku, s.SourcePlatformImage.Version)
		disk.Properties.CreationData.CreateOption = disks.DiskCreateOptionFromImage
		disk.Properties.CreationData.ImageReference = &disks.ImageDiskReference{
			Id: &imageID,
		}
	case s.SourceOSDiskResourceID != "":
		disk.Properties.CreationData.CreateOption = disks.DiskCreateOptionCopy
		disk.Properties.CreationData.SourceResourceId = &s.SourceOSDiskResourceID
	case s.SourceImageResourceID != "":
		disk.Properties.CreationData.CreateOption = disks.DiskCreateOptionFromImage
		disk.Properties.CreationData.GalleryImageReference = &disks.ImageDiskReference{
			Id: &s.SourceImageResourceID,
		}
	default:
		disk.Properties.CreationData.CreateOption = disks.DiskCreateOptionEmpty
	}
	return disk
}

func (s StepCreateNewDiskset) getDatadiskDefinitionFromImage(lun int64) disks.Disk {
	disk := disks.Disk{
		Location: s.Location,
		Properties: &disks.DiskProperties{
			CreationData: disks.CreationData{},
		},
	}

	disk.Properties.CreationData.CreateOption = disks.DiskCreateOptionFromImage
	disk.Properties.CreationData.GalleryImageReference = &disks.ImageDiskReference{
		Id:  &s.SourceImageResourceID,
		Lun: &lun,
	}

	diskSkuName := disks.DiskStorageAccountTypes(s.DataDiskStorageAccountType)
	if s.DataDiskStorageAccountType != "" {
		disk.Sku = &disks.DiskSku{
			Name: &diskSkuName,
		}
	}
	return disk
}

func (s *StepCreateNewDiskset) createDiskset(ctx context.Context, azcli client.AzureClientSet, id commonids.ManagedDiskId, disk disks.Disk) error {
	pollingContext, cancel := context.WithTimeout(ctx, azcli.PollingDuration())
	defer cancel()

	err := azcli.DisksClient().CreateOrUpdateThenPoll(pollingContext, id, disk)
	if err != nil {
		return err
	}
	return nil
}

func (s *StepCreateNewDiskset) getSharedImageGalleryVersion(ctx context.Context, azclient client.AzureClientSet, id galleryimageversions.ImageVersionId) (*galleryimageversions.GalleryImageVersion, error) {
	pollingContext, cancel := context.WithTimeout(ctx, azclient.PollingDuration())
	defer cancel()

	imageVersionResult, err := azclient.GalleryImageVersionsClient().Get(pollingContext, id, galleryimageversions.DefaultGetOperationOptions())
	if err != nil {
		return nil, err
	}
	if imageVersionResult.Model == nil {
		return nil, client.NullModelSDKErr
	}
	return imageVersionResult.Model, nil
}

func (s *StepCreateNewDiskset) Cleanup(state multistep.StateBag) {
	if !s.SkipCleanup {
		azcli := state.Get("azureclient").(client.AzureClientSet)
		ui := state.Get("ui").(packersdk.Ui)

		for _, d := range s.disks {

			ui.Say(fmt.Sprintf("Waiting for disk %q detach to complete", d))
			detatchDisk, detatchDiskCancel := context.WithTimeout(context.Background(), azcli.PollingDuration())
			defer detatchDiskCancel()
			err := NewDiskAttacher(azcli, ui).WaitForDetach(detatchDisk, d.String())
			if err != nil {
				ui.Error(fmt.Sprintf("error detaching disk %q: %s", d, err))
			}

			ui.Say(fmt.Sprintf("Deleting disk %q", d))

			diskID := commonids.NewManagedDiskID(azcli.SubscriptionID(), d.ResourceGroup, d.ResourceName.String())
			pollingContext, cancel := context.WithTimeout(context.Background(), azcli.PollingDuration())
			defer cancel()
			err = azcli.DisksClient().DeleteThenPoll(pollingContext, diskID)
			if err != nil {
				log.Printf("StepCreateNewDiskset.Cleanup: error: %+v", err)
				ui.Error(fmt.Sprintf("error deleting disk '%s': %v.", d, err))
			}
		}
	}
}
