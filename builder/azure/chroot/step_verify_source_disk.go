// Copyright (c) HashiCorp, Inc.
// SPDX-License-Identifier: MPL-2.0

package chroot

import (
	"context"
	"fmt"
	"log"
	"strings"

	"github.com/hashicorp/go-azure-sdk/resource-manager/compute/2022-03-02/disks"
	"github.com/hashicorp/packer-plugin-azure/builder/azure/common/client"
	"github.com/hashicorp/packer-plugin-sdk/multistep"
	packersdk "github.com/hashicorp/packer-plugin-sdk/packer"
)

type StepVerifySourceDisk struct {
	SourceDiskResourceID string
	Location             string

	get func(context.Context, client.AzureClientSet, disks.DiskId) (*disks.Disk, error)
}

func NewStepVerifySourceDisk(step *StepVerifySourceDisk) *StepVerifySourceDisk {
	step.get = step.getDisk
	return step
}

func (s StepVerifySourceDisk) Run(ctx context.Context, state multistep.StateBag) multistep.StepAction {
	azcli := state.Get("azureclient").(client.AzureClientSet)
	ui := state.Get("ui").(packersdk.Ui)

	ui.Say("Checking source disk location")
	resource, err := client.ParseResourceID(s.SourceDiskResourceID)
	if err != nil {
		log.Printf("StepVerifySourceDisk.Run: error: %+v", err)
		err := fmt.Errorf("Could not parse resource id %q: %s", s.SourceDiskResourceID, err)
		state.Put("error", err)
		ui.Error(err.Error())
		return multistep.ActionHalt
	}

	if !strings.EqualFold(resource.Subscription, azcli.SubscriptionID()) {
		err := fmt.Errorf("Source disk resource %q is in a different subscription than this VM (%q). "+
			"Packer does not know how to handle that.",
			s.SourceDiskResourceID, azcli.SubscriptionID())
		log.Printf("StepVerifySourceDisk.Run: error: %+v", err)
		state.Put("error", err)
		ui.Error(err.Error())
		return multistep.ActionHalt
	}

	if !(strings.EqualFold(resource.Provider, "Microsoft.Compute") && strings.EqualFold(resource.ResourceType.String(), "disks")) {
		err := fmt.Errorf("Resource ID %q is not a managed disk resource", s.SourceDiskResourceID)
		log.Printf("StepVerifySourceDisk.Run: error: %+v", err)
		state.Put("error", err)
		ui.Error(err.Error())
		return multistep.ActionHalt
	}

	diskId := disks.NewDiskID(azcli.SubscriptionID(), resource.ResourceGroup, resource.ResourceName.String())
	disk, err := s.get(ctx, azcli, diskId)
	if err != nil {
		err := fmt.Errorf("Unable to retrieve disk (%q): %s", s.SourceDiskResourceID, err)
		log.Printf("StepVerifySourceDisk.Run: error: %+v", err)
		state.Put("error", err)
		ui.Error(err.Error())
		return multistep.ActionHalt
	}

	location := disk.Location
	if !strings.EqualFold(location, s.Location) {
		err := fmt.Errorf("Source disk resource %q is in a different location (%q) than this VM (%q). "+
			"Packer does not know how to handle that.",
			s.SourceDiskResourceID,
			location,
			s.Location)
		log.Printf("StepVerifySourceDisk.Run: error: %+v", err)
		state.Put("error", err)
		ui.Error(err.Error())
		return multistep.ActionHalt
	}

	return multistep.ActionContinue
}

func (s StepVerifySourceDisk) getDisk(ctx context.Context, azcli client.AzureClientSet, id disks.DiskId) (*disks.Disk, error) {
	diskResult, err := azcli.DisksClient().Get(ctx, id)
	if err != nil {
		return nil, err
	}
	if diskResult.Model == nil {
		return nil, fmt.Errorf("SDK returned empty disk")
	}
	return diskResult.Model, nil
}

func (s StepVerifySourceDisk) Cleanup(state multistep.StateBag) {}
