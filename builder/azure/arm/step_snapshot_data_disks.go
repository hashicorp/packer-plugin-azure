// Copyright (c) HashiCorp, Inc.
// SPDX-License-Identifier: MPL-2.0

package arm

import (
	"context"
	"fmt"
	"strconv"

	"github.com/hashicorp/go-azure-sdk/resource-manager/compute/2022-03-02/snapshots"
	"github.com/hashicorp/packer-plugin-azure/builder/azure/common"
	"github.com/hashicorp/packer-plugin-azure/builder/azure/common/constants"
	"github.com/hashicorp/packer-plugin-sdk/multistep"
	packersdk "github.com/hashicorp/packer-plugin-sdk/packer"
)

type StepSnapshotDataDisks struct {
	client *AzureClient
	create func(ctx context.Context, subscriptionId string, resourceGroupName string, srcUriVhd string, location string, tags map[string]string, dstSnapshotName string) error
	say    func(message string)
	error  func(e error)
	enable func() bool
}

func NewStepSnapshotDataDisks(client *AzureClient, ui packersdk.Ui, config *Config) *StepSnapshotDataDisks {
	var step = &StepSnapshotDataDisks{
		client: client,
		say:    func(message string) { ui.Say(message) },
		error:  func(e error) { ui.Error(e.Error()) },
		enable: func() bool { return config.isManagedImage() && config.ManagedImageDataDiskSnapshotPrefix != "" },
	}

	step.create = step.createDataDiskSnapshot
	return step
}

func (s *StepSnapshotDataDisks) createDataDiskSnapshot(ctx context.Context, subscriptionId string, resourceGroupName string, srcUriVhd string, location string, tags map[string]string, dstSnapshotName string) error {
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
	pollingContext, cancel := context.WithTimeout(ctx, s.client.PollingDuration)
	defer cancel()
	id := snapshots.NewSnapshotID(subscriptionId, resourceGroupName, dstSnapshotName)
	err := s.client.SnapshotsClient.CreateOrUpdateThenPoll(pollingContext, id, srcVhdToSnapshot)

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

func (s *StepSnapshotDataDisks) Run(ctx context.Context, stateBag multistep.StateBag) multistep.StepAction {
	if !s.enable() {
		return multistep.ActionContinue
	}

	var resourceGroupName = stateBag.Get(constants.ArmManagedImageResourceGroupName).(string)
	var location = stateBag.Get(constants.ArmLocation).(string)
	var tags = stateBag.Get(constants.ArmTags).(map[string]string)
	var additionalDisks = stateBag.Get(constants.ArmAdditionalDiskVhds).([]string)
	var dstSnapshotPrefix = stateBag.Get(constants.ArmManagedImageDataDiskSnapshotPrefix).(string)
	var subscriptionId = stateBag.Get(constants.ArmSubscription).(string)

	s.say("Snapshotting data disk(s) ...")

	for i, disk := range additionalDisks {
		s.say(fmt.Sprintf(" -> Data Disk   : '%s'", disk))

		dstSnapshotName := dstSnapshotPrefix + strconv.Itoa(i)
		err := s.create(ctx, subscriptionId, resourceGroupName, disk, location, tags, dstSnapshotName)

		if err != nil {
			stateBag.Put(constants.Error, err)
			s.error(err)

			return multistep.ActionHalt
		}
	}

	return multistep.ActionContinue
}

func (*StepSnapshotDataDisks) Cleanup(multistep.StateBag) {
}
