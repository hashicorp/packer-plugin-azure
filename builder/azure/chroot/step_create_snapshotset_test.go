// Copyright IBM Corp. 2013, 2025
// SPDX-License-Identifier: MPL-2.0

package chroot

import (
	"context"
	"reflect"
	"sort"
	"testing"

	"github.com/google/go-cmp/cmp"
	"github.com/hashicorp/go-azure-sdk/resource-manager/compute/2023-04-02/snapshots"
	"github.com/hashicorp/packer-plugin-azure/builder/azure/common"
	"github.com/hashicorp/packer-plugin-azure/builder/azure/common/client"
	"github.com/hashicorp/packer-plugin-sdk/multistep"
	packersdk "github.com/hashicorp/packer-plugin-sdk/packer"
)

func TestStepCreateSnapshot_Run(t *testing.T) {
	type fields struct {
		OSDiskSnapshotID         string
		DataDiskSnapshotIDPrefix string
		Location                 string
	}
	tests := []struct {
		name              string
		fields            fields
		diskset           Diskset
		want              multistep.StepAction
		wantSnapshotset   Diskset
		expectedSnapshots []snapshots.Snapshot
	}{
		{
			name: "happy path",
			fields: fields{
				OSDiskSnapshotID: "/subscriptions/1234/resourceGroups/rg/providers/Microsoft.Compute/snapshots/osdisk-snap",
				Location:         "region1",
			},
			diskset: diskset("/subscriptions/12345/resourceGroups/group1/providers/Microsoft.Compute/disks/disk1"),
			expectedSnapshots: []snapshots.Snapshot{
				{
					Location: "region1",
					Properties: &snapshots.SnapshotProperties{
						CreationData: snapshots.CreationData{
							SourceResourceId: common.StringPtr("/subscriptions/12345/resourceGroups/group1/providers/Microsoft.Compute/disks/disk1"),
							CreateOption:     "Copy",
						},
						Incremental: common.BoolPtr(false),
					},
				},
			},
			wantSnapshotset: diskset("/subscriptions/1234/resourceGroups/rg/providers/Microsoft.Compute/snapshots/osdisk-snap"),
		},
		{
			name: "multi disk",
			fields: fields{
				OSDiskSnapshotID:         "/subscriptions/1234/resourceGroups/rg/providers/Microsoft.Compute/snapshots/osdisk-snap",
				DataDiskSnapshotIDPrefix: "/subscriptions/1234/resourceGroups/rg/providers/Microsoft.Compute/snapshots/datadisk-snap",
				Location:                 "region1",
			},
			diskset: diskset(
				"/subscriptions/12345/resourceGroups/group1/providers/Microsoft.Compute/disks/osdisk",
				"/subscriptions/12345/resourceGroups/group1/providers/Microsoft.Compute/disks/datadisk1",
				"/subscriptions/12345/resourceGroups/group1/providers/Microsoft.Compute/disks/datadisk2",
				"/subscriptions/12345/resourceGroups/group1/providers/Microsoft.Compute/disks/datadisk3"),
			wantSnapshotset: diskset(
				"/subscriptions/1234/resourceGroups/rg/providers/Microsoft.Compute/snapshots/osdisk-snap",
				"/subscriptions/1234/resourceGroups/rg/providers/Microsoft.Compute/snapshots/datadisk-snap0",
				"/subscriptions/1234/resourceGroups/rg/providers/Microsoft.Compute/snapshots/datadisk-snap1",
				"/subscriptions/1234/resourceGroups/rg/providers/Microsoft.Compute/snapshots/datadisk-snap2",
			),
			expectedSnapshots: []snapshots.Snapshot{
				{
					Location: "region1",
					Properties: &snapshots.SnapshotProperties{
						CreationData: snapshots.CreationData{
							SourceResourceId: common.StringPtr("/subscriptions/12345/resourceGroups/group1/providers/Microsoft.Compute/disks/osdisk"),
							CreateOption:     "Copy",
						},
						Incremental: common.BoolPtr(false),
					},
				},
				{
					Location: "region1",
					Properties: &snapshots.SnapshotProperties{
						CreationData: snapshots.CreationData{
							SourceResourceId: common.StringPtr("/subscriptions/12345/resourceGroups/group1/providers/Microsoft.Compute/disks/datadisk1"),
							CreateOption:     "Copy",
						},
						Incremental: common.BoolPtr(false),
					},
				},
				{
					Location: "region1",
					Properties: &snapshots.SnapshotProperties{
						CreationData: snapshots.CreationData{
							SourceResourceId: common.StringPtr("/subscriptions/12345/resourceGroups/group1/providers/Microsoft.Compute/disks/datadisk2"),
							CreateOption:     "Copy",
						},
						Incremental: common.BoolPtr(false),
					},
				},
				{
					Location: "region1",
					Properties: &snapshots.SnapshotProperties{
						CreationData: snapshots.CreationData{
							SourceResourceId: common.StringPtr("/subscriptions/12345/resourceGroups/group1/providers/Microsoft.Compute/disks/datadisk3"),
							CreateOption:     "Copy",
						},
						Incremental: common.BoolPtr(false),
					},
				},
			},
		},
		{
			name: "invalid ResourceID",
			fields: fields{
				OSDiskSnapshotID: "notaresourceid",
				Location:         "region1",
			},
			diskset: diskset("/subscriptions/12345/resourceGroups/group1/providers/Microsoft.Compute/disks/disk1"),
			want:    multistep.ActionHalt,
		},
	}
	for _, tt := range tests {
		state := new(multistep.BasicStateBag)
		state.Put("azureclient", &client.AzureClientSetMock{})
		state.Put("ui", packersdk.TestUi(t))
		state.Put(stateBagKey_Diskset, tt.diskset)

		t.Run(tt.name, func(t *testing.T) {
			actualSnapshots := []snapshots.Snapshot{}
			s := &StepCreateSnapshotset{
				OSDiskSnapshotID:         tt.fields.OSDiskSnapshotID,
				DataDiskSnapshotIDPrefix: tt.fields.DataDiskSnapshotIDPrefix,
				Location:                 tt.fields.Location,
				create: func(ctx context.Context, azcli client.AzureClientSet, id snapshots.SnapshotId, snapshot snapshots.Snapshot) error {
					actualSnapshots = append(actualSnapshots, snapshot)
					return nil
				},
			}
			if got := s.Run(context.TODO(), state); !reflect.DeepEqual(got, tt.want) {
				t.Errorf("StepCreateSnapshot.Run() = %v, want %v", got, tt.want)
			}

			if len(tt.wantSnapshotset) > 0 {
				got := state.Get(stateBagKey_Snapshotset).(Diskset)
				if !reflect.DeepEqual(got, tt.wantSnapshotset) {
					t.Errorf("Snapshotset = %v, want %v", got, tt.wantSnapshotset)
				}
			}

			if len(tt.expectedSnapshots) > 0 {
				sort.Slice(tt.expectedSnapshots, func(i, j int) bool {
					return *tt.expectedSnapshots[i].Properties.CreationData.SourceResourceId < *tt.expectedSnapshots[j].Properties.CreationData.SourceResourceId
				})
				sort.Slice(actualSnapshots, func(i, j int) bool {
					return *actualSnapshots[i].Properties.CreationData.SourceResourceId < *actualSnapshots[j].Properties.CreationData.SourceResourceId
				})
				if diff := cmp.Diff(tt.expectedSnapshots, actualSnapshots); diff != "" {
					t.Fatal(diff)
				}
			}
		})
	}
}

func TestStepCreateSnapshot_Cleanup_skipped(t *testing.T) {
	state := new(multistep.BasicStateBag)
	state.Put("azureclient", &client.AzureClientSetMock{})
	state.Put("ui", packersdk.TestUi(t))

	s := &StepCreateSnapshotset{
		SkipCleanup: true,
	}
	s.Cleanup(state)
}
