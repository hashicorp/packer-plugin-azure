// Copyright (c) HashiCorp, Inc.
// SPDX-License-Identifier: MPL-2.0

package arm

import (
	"context"
	"testing"

	"github.com/Azure/go-autorest/autorest/to"
	hashiImagesSDK "github.com/hashicorp/go-azure-sdk/resource-manager/compute/2022-03-01/images"

	"github.com/hashicorp/packer-plugin-azure/builder/azure/common/constants"
	"github.com/hashicorp/packer-plugin-sdk/multistep"
)

func TestStepPublishToSharedImageGalleryShouldNotPublishForVhd(t *testing.T) {
	var testSubject = &StepPublishToSharedImageGallery{
		publish: func(context.Context, string, SharedImageGalleryDestination, string, bool, int64, string, string, map[string]string) (string, error) {
			return "test", nil
		},
		say:   func(message string) {},
		error: func(e error) {},
		toSIG: func() bool { return false },
	}

	stateBag := createTestStateBagStepPublishToSharedImageGalleryForVhd()
	var result = testSubject.Run(context.Background(), stateBag)
	if result != multistep.ActionContinue {
		t.Fatalf("Expected the step to return 'ActionContinue', but got '%d'.", result)
	}

	if _, ok := stateBag.GetOk(constants.Error); ok == true {
		t.Fatalf("Expected the step to not set stateBag['%s'], but it was.", constants.Error)
	}
}

func TestStepPublishToSharedImageGalleryShouldPublishForManagedImageWithSig(t *testing.T) {
	var testSubject = &StepPublishToSharedImageGallery{
		publish: func(context.Context, string, SharedImageGalleryDestination, string, bool, int64, string, string, map[string]string) (string, error) {
			return "", nil
		},
		say:   func(message string) {},
		error: func(e error) {},
		toSIG: func() bool { return true },
	}

	stateBag := createTestStateBagStepPublishToSharedImageGallery(true)
	var result = testSubject.Run(context.Background(), stateBag)
	if result != multistep.ActionContinue {
		t.Fatalf("Expected the step to return 'ActionContinue', but got '%d'.", result)
	}

	if _, ok := stateBag.GetOk(constants.Error); ok == true {
		t.Fatalf("Expected the step to not set stateBag['%s'], but it was.", constants.Error)
	}
}

func TestStepPublishToSharedImageGalleryShouldPublishForNonManagedImageWithSig(t *testing.T) {
	var testSubject = &StepPublishToSharedImageGallery{
		publish: func(context.Context, string, SharedImageGalleryDestination, string, bool, int64, string, string, map[string]string) (string, error) {
			return "", nil
		},
		say:   func(message string) {},
		error: func(e error) {},
		toSIG: func() bool { return true },
	}

	stateBag := createTestStateBagStepPublishToSharedImageGallery(false)
	var result = testSubject.Run(context.Background(), stateBag)
	if result != multistep.ActionContinue {
		t.Fatalf("Expected the step to return 'ActionContinue', but got '%d'.", result)
	}

	if _, ok := stateBag.GetOk(constants.Error); ok == true {
		t.Fatalf("Expected the step to not set stateBag['%s'], but it was.", constants.Error)
	}
}

func createTestStateBagStepPublishToSharedImageGallery(managed bool) multistep.StateBag {
	stateBag := new(multistep.BasicStateBag)

	stateBag.Put(constants.ArmManagedImageSigPublishResourceGroup, "Unit Test: ManagedImageSigPublishResourceGroup")
	stateBag.Put(constants.ArmManagedImageSharedGalleryName, "Unit Test: ManagedImageSharedGalleryName")
	stateBag.Put(constants.ArmManagedImageSharedGalleryImageName, "Unit Test: ManagedImageSharedGalleryImageName")
	stateBag.Put(constants.ArmManagedImageSharedGalleryImageVersion, "Unit Test: ManagedImageSharedGalleryImageVersion")
	stateBag.Put(constants.ArmLocation, "Unit Test: Location")
	value := "Unit Test: Tags"
	tags := map[string]string{
		"tag01": value,
	}
	stateBag.Put(constants.ArmNewSDKTags, tags)
	stateBag.Put(constants.ArmManagedImageSharedGalleryReplicationRegions, []string{"ManagedImageSharedGalleryReplicationRegionA", "ManagedImageSharedGalleryReplicationRegionB"})
	stateBag.Put(constants.ArmManagedImageSharedGalleryImageVersionStorageAccountType, "Standard_LRS")
	if managed {
		stateBag.Put(constants.ArmManagedImageResourceGroupName, "Unit Test: ManagedImageResourceGroupName")
		stateBag.Put(constants.ArmManagedImageName, "Unit Test: ManagedImageName")
	} else {
		stateBag.Put(constants.ArmImageParameters, &hashiImagesSDK.Image{Properties: &hashiImagesSDK.ImageProperties{
			SourceVirtualMachine: &hashiImagesSDK.SubResource{Id: to.StringPtr("Unit Test: VM ID")},
		}})
	}
	stateBag.Put(constants.ArmManagedImageSubscription, "Unit Test: ManagedImageSubscription")
	stateBag.Put(constants.ArmIsManagedImage, managed)
	stateBag.Put(constants.ArmIsSIGImage, true)

	return stateBag
}

func createTestStateBagStepPublishToSharedImageGalleryForVhd() multistep.StateBag {
	stateBag := new(multistep.BasicStateBag)

	stateBag.Put(constants.ArmLocation, "Unit Test: Location")
	value := "Unit Test: Tags"
	tags := map[string]*string{
		"tag01": &value,
	}
	stateBag.Put(constants.ArmTags, tags)

	return stateBag
}
