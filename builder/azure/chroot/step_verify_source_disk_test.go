// Copyright (c) HashiCorp, Inc.
// SPDX-License-Identifier: MPL-2.0

package chroot

import (
	"context"
	"fmt"
	"reflect"
	"regexp"
	"testing"

	"github.com/hashicorp/go-azure-helpers/resourcemanager/commonids"
	"github.com/hashicorp/go-azure-sdk/resource-manager/compute/2022-03-02/disks"
	"github.com/hashicorp/packer-plugin-azure/builder/azure/common/client"
	"github.com/hashicorp/packer-plugin-sdk/multistep"
)

func Test_StepVerifySourceDisk_Run(t *testing.T) {
	type fields struct {
		SourceDiskResourceID string
		Location             string

		GetDiskError    error
		GetDiskResponse *disks.Disk
	}
	diskWithCorrectLocation := disks.Disk{
		Location: "westus2",
	}
	tests := []struct {
		name       string
		fields     fields
		want       multistep.StepAction
		errormatch string
	}{
		{
			name: "HappyPath",
			fields: fields{
				SourceDiskResourceID: "/subscriptions/subid1/resourcegroups/rg1/providers/Microsoft.Compute/disks/disk1",
				Location:             "westus2",

				GetDiskResponse: &diskWithCorrectLocation,
			},
			want: multistep.ActionContinue,
		},
		{
			name: "NotAResourceID",
			fields: fields{
				SourceDiskResourceID: "/other",
				Location:             "westus2",
			},
			want:       multistep.ActionHalt,
			errormatch: "Could not parse resource id",
		},
		{
			name: "DiskNotFound",
			fields: fields{
				SourceDiskResourceID: "/subscriptions/subid1/resourcegroups/rg1/providers/Microsoft.Compute/disks/disk1",
				Location:             "westus2",

				GetDiskError:    fmt.Errorf("404"),
				GetDiskResponse: nil,
			},
			want:       multistep.ActionHalt,
			errormatch: "Unable to retrieve",
		},
		{
			name: "NotADisk",
			fields: fields{
				SourceDiskResourceID: "/subscriptions/subid1/resourcegroups/rg1/providers/Microsoft.Compute/images/image1",
				Location:             "westus2",
			},
			want:       multistep.ActionHalt,
			errormatch: "not a managed disk",
		},
		{
			name: "OtherSubscription",
			fields: fields{
				SourceDiskResourceID: "/subscriptions/subid2/resourcegroups/rg1/providers/Microsoft.Compute/disks/disk1",
				Location:             "westus2",

				GetDiskResponse: &diskWithCorrectLocation,
			},
			want:       multistep.ActionHalt,
			errormatch: "different subscription",
		},
		{
			name: "OtherLocation",
			fields: fields{
				SourceDiskResourceID: "/subscriptions/subid1/resourcegroups/rg1/providers/Microsoft.Compute/disks/disk1",
				Location:             "eastus",

				GetDiskResponse: &diskWithCorrectLocation,
			},
			want:       multistep.ActionHalt,
			errormatch: "different location",
		},
	}
	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			s := StepVerifySourceDisk{
				SourceDiskResourceID: tt.fields.SourceDiskResourceID,
				Location:             tt.fields.Location,
				get: func(ctx context.Context, azcli client.AzureClientSet, id commonids.ManagedDiskId) (*disks.Disk, error) {
					if tt.fields.GetDiskError == nil && tt.fields.GetDiskResponse == nil {
						t.Fatalf("expected getDisk to not be called but it was")
					}
					return tt.fields.GetDiskResponse, tt.fields.GetDiskError
				},
			}

			ui, getErr := testUI()

			state := new(multistep.BasicStateBag)
			state.Put("azureclient", &client.AzureClientSetMock{
				SubscriptionIDMock: "subid1",
			})
			state.Put("ui", ui)

			got := s.Run(context.TODO(), state)
			if !reflect.DeepEqual(got, tt.want) {
				t.Errorf("StepVerifySourceDisk.Run() = %v, want %v", got, tt.want)
			}

			if tt.errormatch != "" {
				errs := getErr()
				if !regexp.MustCompile(tt.errormatch).MatchString(errs) {
					t.Errorf("Expected the error output (%q) to match %q", errs, tt.errormatch)
				}
			}

			if got == multistep.ActionHalt {
				if _, ok := state.GetOk("error"); !ok {
					t.Fatal("Expected 'error' to be set in statebag after failure")
				}
			}
		})
	}
}
