// Copyright IBM Corp. 2013, 2025
// SPDX-License-Identifier: MPL-2.0

package chroot

import (
	"context"
	"fmt"
	"reflect"
	"testing"

	"github.com/hashicorp/go-azure-sdk/resource-manager/compute/2022-03-01/images"
	"github.com/hashicorp/packer-plugin-azure/builder/azure/common/client"
	"github.com/hashicorp/packer-plugin-sdk/multistep"
	packersdk "github.com/hashicorp/packer-plugin-sdk/packer"
)

func TestStepCreateImage_Run(t *testing.T) {
	subscriptionID := "12345"
	resourceGroup := "group1"
	imageName := "myImage"
	type fields struct {
		ImageResourceID            string
		ImageOSState               string
		OSDiskStorageAccountType   string
		OSDiskCacheType            string
		DataDiskStorageAccountType string
		DataDiskCacheType          string
		Location                   string
	}
	tests := []struct {
		name    string
		fields  fields
		diskset Diskset
		want    multistep.StepAction
	}{
		{
			name: "happy path",
			fields: fields{
				ImageResourceID:            fmt.Sprintf("/subscriptions/%s/resourceGroups/%s/providers/Microsoft.Compute/images/%s", subscriptionID, resourceGroup, imageName),
				Location:                   "location1",
				OSDiskStorageAccountType:   "Standard_LRS",
				OSDiskCacheType:            "ReadWrite",
				DataDiskStorageAccountType: "Premium_LRS",
				DataDiskCacheType:          "ReadOnly",
			},
			diskset: diskset(
				"/subscriptions/12345/resourceGroups/group1/providers/Microsoft.Compute/disks/osdisk",
				"/subscriptions/12345/resourceGroups/group1/providers/Microsoft.Compute/disks/datadisk0",
				"/subscriptions/12345/resourceGroups/group1/providers/Microsoft.Compute/disks/datadisk1",
				"/subscriptions/12345/resourceGroups/group1/providers/Microsoft.Compute/disks/datadisk2"),
			want: multistep.ActionContinue,
		},
	}
	for _, tt := range tests {
		state := new(multistep.BasicStateBag)
		state.Put("azureclient", &client.AzureClientSetMock{SubscriptionIDMock: subscriptionID})
		state.Put("ui", packersdk.TestUi(t))
		state.Put(stateBagKey_Diskset, tt.diskset)
		expectedImageID := images.NewImageID(subscriptionID, resourceGroup, imageName)
		t.Run(tt.name, func(t *testing.T) {
			var actualImageID images.ImageId
			var actualImage images.Image
			s := &StepCreateImage{
				ImageResourceID:            tt.fields.ImageResourceID,
				ImageOSState:               tt.fields.ImageOSState,
				OSDiskStorageAccountType:   tt.fields.OSDiskStorageAccountType,
				OSDiskCacheType:            tt.fields.OSDiskCacheType,
				DataDiskStorageAccountType: tt.fields.DataDiskStorageAccountType,
				DataDiskCacheType:          tt.fields.DataDiskCacheType,
				Location:                   tt.fields.Location,
				create: func(ctx context.Context, client client.AzureClientSet, id images.ImageId, image images.Image) error {
					actualImageID = id
					actualImage = image
					return nil
				},
			}
			if got := s.Run(context.TODO(), state); !reflect.DeepEqual(got, tt.want) {
				t.Errorf("StepCreateImage.Run() = %v, want %v", got, tt.want)
			}
			if expectedImageID != actualImageID {
				t.Fatalf("Expected StepCreateImage.create called with image ID %s got image ID %s", expectedImageID, actualImageID)
			}

			if len(*actualImage.Properties.StorageProfile.DataDisks) != 3 {
				t.Fatalf("Expected 3 data disks attached to created image got %d", len(*actualImage.Properties.StorageProfile.DataDisks))
			}
			if actualImage.Properties.StorageProfile.OsDisk.OsType != "Linux" {
				t.Fatalf("Expected actual image to be Linux, got %s", actualImage.Properties.StorageProfile.OsDisk.OsType)
			}
			if actualImage.Location != tt.fields.Location {
				t.Fatalf("Expected %s location got %s location", tt.fields.Location, actualImage.Location)
			}
		})
	}
}
