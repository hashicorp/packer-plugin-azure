// Copyright IBM Corp. 2013, 2025
// SPDX-License-Identifier: MPL-2.0

package chroot

import (
	"context"
	"testing"
	"time"

	"github.com/hashicorp/packer-plugin-azure/builder/azure/common/client"
	"github.com/hashicorp/packer-plugin-sdk/multistep"

	"github.com/hashicorp/go-azure-sdk/resource-manager/compute/2022-03-01/virtualmachineimages"
)

func TestStepResolvePlatformImageVersion_Run(t *testing.T) {

	var expectedSkuId, actualSkuId virtualmachineimages.SkuId
	expectedSku := "Linux"
	expectedOffer := "Offer"
	expectedPublisher := "Arch"
	subscriptionID := "1234"
	expectedLocation := "linuxland"
	expectedSkuId = virtualmachineimages.NewSkuID(subscriptionID, expectedLocation, expectedPublisher, expectedOffer, expectedSku)
	var actualListOperations virtualmachineimages.ListOperationOptions
	returnedVMImages := []virtualmachineimages.VirtualMachineImageResource{
		{
			Name: "1.2.3",
		},
		{
			Name: "0.2.1",
		},
	}
	pi := &StepResolvePlatformImageVersion{
		PlatformImage: &client.PlatformImage{
			Version:   "latest",
			Sku:       expectedSku,
			Offer:     expectedOffer,
			Publisher: expectedPublisher,
		},
		Location: expectedLocation,
		list: func(ctx context.Context, azcli client.AzureClientSet, skuID virtualmachineimages.SkuId, operations virtualmachineimages.ListOperationOptions) (*[]virtualmachineimages.VirtualMachineImageResource, error) {

			actualSkuId = skuID
			actualListOperations = operations
			return &returnedVMImages, nil
		},
	}

	state := new(multistep.BasicStateBag)

	ui, _ := testUI()
	state.Put("azureclient", &client.AzureClientSetMock{
		SubscriptionIDMock: subscriptionID,
	})
	state.Put("ui", ui)

	ctx, cancel := context.WithTimeout(context.Background(), time.Second)
	defer cancel()

	got := pi.Run(ctx, state)
	if got != multistep.ActionContinue {
		t.Errorf("Expected 'continue', but got %q", got)
	}

	if pi.PlatformImage.Version != "1.2.3" {
		t.Errorf("Expected version '1.2.3', but got %q", pi.PlatformImage.Version)
	}
	if actualSkuId != expectedSkuId {
		t.Fatalf("Expected sku ID %+v got sku ID %+v", expectedSkuId, actualSkuId)
	}
	if *actualListOperations.Orderby != "name desc" {
		t.Fatalf("Expected name desc order by list operation, got %s", *actualListOperations.Orderby)
	}
}
