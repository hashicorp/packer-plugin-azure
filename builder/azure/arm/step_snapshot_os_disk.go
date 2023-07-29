// Copyright (c) HashiCorp, Inc.
// SPDX-License-Identifier: MPL-2.0

package arm

import (
	"context"
	"fmt"

	"github.com/hashicorp/go-azure-sdk/resource-manager/compute/2022-03-02/snapshots"
	"github.com/hashicorp/packer-plugin-azure/builder/azure/common"
	"github.com/hashicorp/packer-plugin-azure/builder/azure/common/constants"
	"github.com/hashicorp/packer-plugin-sdk/multistep"
	packersdk "github.com/hashicorp/packer-plugin-sdk/packer"
)

type StepSnapshotOSDisk struct {
	client *AzureClient
	create func(ctx context.Context, subscriptionId string, resourceGroupName string, srcUriVhd string, location string, tags map[string]string, dstSnapshotName string) error
	say    func(message string)
	error  func(e error)
	enable func() bool
}

func NewStepSnapshotOSDisk(client *AzureClient, ui packersdk.Ui, config *Config) *StepSnapshotOSDisk {
	var step = &StepSnapshotOSDisk{
		client: client,
		say:    func(message string) { ui.Say(message) },
		error:  func(e error) { ui.Error(e.Error()) },
		enable: func() bool { return config.isManagedImage() && config.ManagedImageOSDiskSnapshotName != "" },
	}

	step.create = step.createSnapshot
	return step
}

func (s *StepSnapshotOSDisk) createSnapshot(ctx context.Context, subscriptionId string, resourceGroupName string, srcUriVhd string, location string, tags map[string]string, dstSnapshotName string) error {

	srcVhdToSnapshot := snapshots.Snapshot{
		Properties: &snapshots.SnapshotProperties{
			CreationData: snapshots.CreationData{
				CreateOption:     snapshots.DiskCreateOptionCopy,
				SourceResourceId: common.StringPtr(srcUriVhd),
			},
		},
		Location: *common.StringPtr(location),
		Tags:     &tags,
	}
	id := snapshots.NewSnapshotID(subscriptionId, resourceGroupName, dstSnapshotName)
	err := s.client.SnapshotsClient.CreateOrUpdateThenPoll(ctx, id, srcVhdToSnapshot)

	if err != nil {
		s.say(s.client.LastError.Error())
		return err
	}

	snapshot, err := s.client.SnapshotsClient.Get(ctx, id)
	if err != nil {
		s.say(s.client.LastError.Error())
		return err
	}

	s.say(fmt.Sprintf(" -> Snapshot ID : '%s'", *(snapshot.Model.Id)))
	return nil
}

func (s *StepSnapshotOSDisk) Run(ctx context.Context, stateBag multistep.StateBag) multistep.StepAction {
	if !s.enable() {
		return multistep.ActionContinue
	}

	s.say("Snapshotting OS disk ...")

	var resourceGroupName = stateBag.Get(constants.ArmManagedImageResourceGroupName).(string)
	var location = stateBag.Get(constants.ArmLocation).(string)
	var tags = stateBag.Get(constants.ArmNewSDKTags).(map[string]string)
	var srcUriVhd = stateBag.Get(constants.ArmOSDiskUri).(string)
	var dstSnapshotName = stateBag.Get(constants.ArmManagedImageOSDiskSnapshotName).(string)
	var subscriptionId = stateBag.Get(constants.ArmSubscription).(string)

	s.say(fmt.Sprintf(" -> OS Disk     : '%s'", srcUriVhd))
	err := s.create(ctx, subscriptionId, resourceGroupName, srcUriVhd, location, tags, dstSnapshotName)

	if err != nil {
		stateBag.Put(constants.Error, err)
		s.error(err)

		return multistep.ActionHalt
	}

	return multistep.ActionContinue
}

func (*StepSnapshotOSDisk) Cleanup(multistep.StateBag) {
}
