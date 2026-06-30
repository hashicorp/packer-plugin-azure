// Copyright IBM Corp. 2013, 2025
// SPDX-License-Identifier: MPL-2.0

package chroot

import (
	"context"
	"fmt"
	"strings"

	"github.com/hashicorp/packer-plugin-azure/builder/azure/common/log"

	"github.com/hashicorp/go-azure-sdk/resource-manager/compute/2023-04-02/snapshots"
	"github.com/hashicorp/packer-plugin-azure/builder/azure/common"
	"github.com/hashicorp/packer-plugin-azure/builder/azure/common/client"
	"github.com/hashicorp/packer-plugin-sdk/multistep"
	packersdk "github.com/hashicorp/packer-plugin-sdk/packer"
)

var _ multistep.Step = &StepCreateSnapshotset{}

type StepCreateSnapshotset struct {
	OSDiskSnapshotID         string
	DataDiskSnapshotIDPrefix string
	Location                 string

	SkipCleanup bool

	snapshots Diskset

	create func(context.Context, client.AzureClientSet, snapshots.SnapshotId, snapshots.Snapshot) error
}

func NewStepCreateSnapshotset(step *StepCreateSnapshotset) *StepCreateSnapshotset {
	step.create = step.createSnapshot
	return step
}

func (s *StepCreateSnapshotset) Run(ctx context.Context, state multistep.StateBag) multistep.StepAction {
	azcli := state.Get("azureclient").(client.AzureClientSet)
	ui := state.Get("ui").(packersdk.Ui)
	diskset := state.Get(stateBagKey_Diskset).(Diskset)

	s.snapshots = make(Diskset)

	errorMessage := func(format string, params ...interface{}) multistep.StepAction {
		err := fmt.Errorf("StepCreateSnapshotset.Run: error: "+format, params...)
		state.Put("error", err)
		ui.Error(err.Error())
		return multistep.ActionHalt
	}

	for lun, resource := range diskset {
		snapshotID := fmt.Sprintf("%s%d", s.DataDiskSnapshotIDPrefix, lun)
		if lun == -1 {
			snapshotID = s.OSDiskSnapshotID
		}
		ssr, err := client.ParseResourceID(snapshotID)
		if err != nil {
			errorMessage("Could not create a valid resource id, tried %q: %v", snapshotID, err)
		}
		if !strings.EqualFold(ssr.Provider, "Microsoft.Compute") ||
			!strings.EqualFold(ssr.ResourceType.String(), "snapshots") {
			return errorMessage("Resource %q is not of type Microsoft.Compute/snapshots", snapshotID)
		}
		s.snapshots[lun] = ssr
		state.Put(stateBagKey_Snapshotset, s.snapshots)

		ui.Say(fmt.Sprintf("Creating snapshot %q", ssr))

		resourceID := resource.String()
		snapshot := snapshots.Snapshot{
			Location: s.Location,
			Properties: &snapshots.SnapshotProperties{
				CreationData: snapshots.CreationData{
					CreateOption:     snapshots.DiskCreateOptionCopy,
					SourceResourceId: &resourceID,
				},
				Incremental: common.BoolPtr(false),
			},
		}
		snapshotSDKID := snapshots.NewSnapshotID(azcli.SubscriptionID(), ssr.ResourceGroup, ssr.ResourceName.String())
		err = s.create(ctx, azcli, snapshotSDKID, snapshot)
		if err != nil {
			return errorMessage("error initiating snapshot %q: %v", ssr, err)
		}

	}

	return multistep.ActionContinue
}

func (s *StepCreateSnapshotset) createSnapshot(ctx context.Context, azcli client.AzureClientSet, id snapshots.SnapshotId, snapshot snapshots.Snapshot) error {
	pollingContext, cancel := context.WithTimeout(ctx, azcli.PollingDuration())
	defer cancel()
	return azcli.SnapshotsClient().CreateOrUpdateThenPoll(pollingContext, id, snapshot)
}

func (s *StepCreateSnapshotset) Cleanup(state multistep.StateBag) {
	if !s.SkipCleanup {
		azcli := state.Get("azureclient").(client.AzureClientSet)
		ui := state.Get("ui").(packersdk.Ui)

		for _, resource := range s.snapshots {

			snapshotID := snapshots.NewSnapshotID(azcli.SubscriptionID(), resource.ResourceGroup, resource.ResourceName.String())
			ui.Say(fmt.Sprintf("Removing any active SAS for snapshot %q", resource))
			{
				pollingContext, cancel := context.WithTimeout(context.TODO(), azcli.PollingDuration())
				defer cancel()
				err := azcli.SnapshotsClient().RevokeAccessThenPoll(pollingContext, snapshotID)
				if err != nil {
					log.Printf("StepCreateSnapshotset.Cleanup: error: %+v", err)
					ui.Error(fmt.Sprintf("error deleting snapshot %q: %v.", resource, err))
				}
			}

			ui.Say(fmt.Sprintf("Deleting snapshot %q", resource))
			{
				pollingContext, cancel := context.WithTimeout(context.TODO(), azcli.PollingDuration())
				defer cancel()
				err := azcli.SnapshotsClient().DeleteThenPoll(pollingContext, snapshotID)
				if err != nil {
					log.Printf("StepCreateSnapshotset.Cleanup: error: %+v", err)
					ui.Error(fmt.Sprintf("error deleting snapshot %q: %v.", resource, err))
				}
			}
		}
	}
}
