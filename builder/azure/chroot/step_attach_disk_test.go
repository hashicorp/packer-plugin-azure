// Copyright (c) HashiCorp, Inc.
// SPDX-License-Identifier: MPL-2.0

package chroot

import (
	"context"
	"errors"
	"io/ioutil"
	"net/http"
	"reflect"
	"strings"
	"testing"

	"github.com/Azure/azure-sdk-for-go/services/compute/mgmt/2019-12-01/compute"
	"github.com/Azure/go-autorest/autorest"
	"github.com/hashicorp/packer-plugin-azure/builder/azure/common/client"
	"github.com/hashicorp/packer-plugin-sdk/multistep"
	packersdk "github.com/hashicorp/packer-plugin-sdk/packer"
)

func TestStepAttachDisk_Run(t *testing.T) {
	type fields struct {
		GetDiskResponseCode int
		GetDiskResponseBody string

		attachError        error
		waitForDeviceError error
	}
	tests := []struct {
		name   string
		fields fields
		want   multistep.StepAction
	}{
		{
			name: "HappyPath",
			want: multistep.ActionContinue,
		},
		{
			name: "AttachError",
			fields: fields{
				attachError: errors.New("unit test"),
			},
			want: multistep.ActionHalt,
		},
		{
			name: "WaitForDeviceError",
			fields: fields{
				waitForDeviceError: errors.New("unit test"),
			},
			want: multistep.ActionHalt,
		},
	}
	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			s := &StepAttachDisk{}

			errorBuffer := &strings.Builder{}
			ui := &packersdk.BasicUi{
				Reader:      strings.NewReader(""),
				Writer:      ioutil.Discard,
				ErrorWriter: errorBuffer,
			}

			NewDiskAttacher = func(azcli client.AzureClientSet, ui packersdk.Ui) DiskAttacher {
				return &fakeDiskAttacher{
					attachError:        tt.fields.attachError,
					waitForDeviceError: tt.fields.waitForDeviceError,
					ui:                 ui,
				}
			}

			dm := compute.NewDisksClient("subscriptionId")
			dm.Sender = autorest.SenderFunc(func(r *http.Request) (*http.Response, error) {
				return &http.Response{
					Request:    r,
					Body:       ioutil.NopCloser(strings.NewReader(tt.fields.GetDiskResponseBody)),
					StatusCode: tt.fields.GetDiskResponseCode,
				}, nil
			})

			state := new(multistep.BasicStateBag)
			state.Put("azureclient", &client.AzureClientSetMock{})
			state.Put("ui", ui)
			state.Put(stateBagKey_Diskset, diskset("/subscriptions/12345/resourceGroups/group1/providers/Microsoft.Compute/disks/disk1"))

			got := s.Run(context.TODO(), state)
			if !reflect.DeepEqual(got, tt.want) {
				t.Errorf("StepAttachDisk.Run() = %v, want %v", got, tt.want)
			}

			if got == multistep.ActionHalt {
				if _, ok := state.GetOk("error"); !ok {
					t.Fatal("Expected 'error' to be set in statebag after failure")
				}
			}
		})
	}
}

type fakeDiskAttacher struct {
	attachError        error
	waitForDeviceError error
	ui                 packersdk.Ui
}

var _ DiskAttacher = &fakeDiskAttacher{}

func (da *fakeDiskAttacher) AttachDisk(ctx context.Context, disk string) (lun int32, err error) {
	if da.attachError != nil {
		return 0, da.attachError
	}
	return 3, nil
}

func (da *fakeDiskAttacher) DiskPathForLun(lun int32) string {
	panic("not implemented")
}

func (da *fakeDiskAttacher) WaitForDevice(ctx context.Context, lun int32) (device string, err error) {
	if da.waitForDeviceError != nil {
		return "", da.waitForDeviceError
	}
	if lun == 3 {
		return "/dev/sdq", nil
	}
	panic("expected lun==3")
}

func (da *fakeDiskAttacher) DetachDisk(ctx context.Context, disk string) (err error) {
	panic("not implemented")
}

func (da *fakeDiskAttacher) WaitForDetach(ctx context.Context, diskID string) error {
	panic("not implemented")
}
