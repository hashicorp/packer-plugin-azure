// Copyright IBM Corp. 2013, 2025
// SPDX-License-Identifier: MPL-2.0

package arm

import (
	"context"
	"testing"

	"github.com/google/go-cmp/cmp"
	"github.com/google/go-cmp/cmp/cmpopts"
	"github.com/hashicorp/go-azure-sdk/resource-manager/compute/2022-03-01/images"
	"github.com/hashicorp/go-azure-sdk/resource-manager/compute/2023-07-03/galleryimageversions"

	"github.com/hashicorp/packer-plugin-azure/builder/azure/common"
	"github.com/hashicorp/packer-plugin-azure/builder/azure/common/constants"
	"github.com/hashicorp/packer-plugin-sdk/multistep"
)

func TestStepPublishToSharedImageGalleryShouldNotPublishForVhd(t *testing.T) {
	var testSubject = &StepPublishToSharedImageGallery{
		publish: func(context.Context, PublishArgs) (string, error) {
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
	var actualPublishArgs PublishArgs
	expectedSource := "/subscriptions/00000000-0000-0000-0000-000000000000/resourceGroups/my-group/providers/Microsoft.Compute/images/packer-test"
	expectedPublishArgs := PublishArgs{
		SubscriptionID:  "Unit Test: ManagedImageSubscription",
		ReplicaCount:    1,
		ReplicationMode: "Full",
		Location:        "Unit Test: Location",
		GallerySource: galleryimageversions.GalleryArtifactVersionFullSource{
			Id: &expectedSource,
		},
		SharedImageGallery: SharedImageGalleryDestination{
			SigDestinationGalleryName:   "Unit Test: ManagedImageSharedGalleryName",
			SigDestinationImageName:     "Unit Test: ManagedImageSharedGalleryImageName",
			SigDestinationSubscription:  "00000000-0000-0000-0000-000000000000",
			SigDestinationImageVersion:  "Unit Test: ManagedImageSharedGalleryImageVersion",
			SigDestinationResourceGroup: "Unit Test: ManagedImageSigPublishResourceGroup",
			SigDestinationReplicationRegions: []string{
				"ManagedImageSharedGalleryReplicationRegionA",
				"ManagedImageSharedGalleryReplicationRegionB",
			},
			SigDestinationTargetRegions: []TargetRegion{
				{Name: "ManagedImageSharedGalleryReplicationRegionA"},
				{Name: "ManagedImageSharedGalleryReplicationRegionB"},
			},
			SigDestinationStorageAccountType: "Standard_LRS",
		},
		Tags: map[string]string{"tag01": "Unit Test: Tags"},
	}
	var testSubject = &StepPublishToSharedImageGallery{
		publish: func(ctx context.Context, args PublishArgs) (string, error) {
			actualPublishArgs = args
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
	if diff := cmp.Diff(actualPublishArgs, expectedPublishArgs, []cmp.Option{
		cmpopts.IgnoreUnexported(PublishArgs{}),
	}...); diff != "" {
		t.Fatalf("Unexpected diff %s", diff)
	}
}

func TestStepPublishToSharedImageGalleryShouldPublishForNonManagedImageWithSig(t *testing.T) {
	var actualPublishArgs PublishArgs
	expectedSource := "/subscriptions/00000000-0000-0000-0000-000000000000/resourceGroups/my-group/providers/Microsoft.Compute/virtualMachines/packer-test"
	expectedPublishArgs := PublishArgs{
		SubscriptionID:  "Unit Test: ManagedImageSubscription",
		ReplicaCount:    1,
		ReplicationMode: "Full",
		Location:        "Unit Test: Location",
		GallerySource: galleryimageversions.GalleryArtifactVersionFullSource{
			VirtualMachineId: &expectedSource,
		},
		SharedImageGallery: SharedImageGalleryDestination{
			SigDestinationGalleryName:   "Unit Test: ManagedImageSharedGalleryName",
			SigDestinationImageName:     "Unit Test: ManagedImageSharedGalleryImageName",
			SigDestinationSubscription:  "00000000-0000-0000-0000-000000000000",
			SigDestinationImageVersion:  "Unit Test: ManagedImageSharedGalleryImageVersion",
			SigDestinationResourceGroup: "Unit Test: ManagedImageSigPublishResourceGroup",
			SigDestinationReplicationRegions: []string{
				"ManagedImageSharedGalleryReplicationRegionA",
				"ManagedImageSharedGalleryReplicationRegionB",
			},
			SigDestinationTargetRegions: []TargetRegion{
				{Name: "ManagedImageSharedGalleryReplicationRegionA"},
				{Name: "ManagedImageSharedGalleryReplicationRegionB"},
			},
			SigDestinationStorageAccountType: "Standard_LRS",
		},
		Tags: map[string]string{"tag01": "Unit Test: Tags"},
	}
	var testSubject = &StepPublishToSharedImageGallery{
		publish: func(ctx context.Context, args PublishArgs) (string, error) {
			actualPublishArgs = args
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

	if diff := cmp.Diff(actualPublishArgs, expectedPublishArgs, []cmp.Option{
		cmpopts.IgnoreUnexported(PublishArgs{}),
	}...); diff != "" {
		t.Fatalf("Unexpected diff %s", diff)
	}
}

func TestStepPublishToSharedImageGalleryShouldPublishWithShallowReplication(t *testing.T) {
	expectedSource := "/subscriptions/00000000-0000-0000-0000-000000000000/resourceGroups/my-group/providers/Microsoft.Compute/virtualMachines/packer-test"
	var actualPublishArgs PublishArgs
	expectedPublishArgs := PublishArgs{
		SubscriptionID:  "Unit Test: ManagedImageSubscription",
		ReplicaCount:    1,
		ReplicationMode: "Shallow",
		Location:        "Unit Test: Location",
		GallerySource: galleryimageversions.GalleryArtifactVersionFullSource{
			VirtualMachineId: &expectedSource,
		},
		SharedImageGallery: SharedImageGalleryDestination{
			SigDestinationGalleryName:   "Unit Test: ManagedImageSharedGalleryName",
			SigDestinationImageName:     "Unit Test: ManagedImageSharedGalleryImageName",
			SigDestinationSubscription:  "00000000-0000-0000-0000-000000000000",
			SigDestinationImageVersion:  "Unit Test: ManagedImageSharedGalleryImageVersion",
			SigDestinationResourceGroup: "Unit Test: ManagedImageSigPublishResourceGroup",
			SigDestinationReplicationRegions: []string{
				"ManagedImageSharedGalleryReplicationRegionA",
				"ManagedImageSharedGalleryReplicationRegionB",
			},
			SigDestinationTargetRegions: []TargetRegion{
				{Name: "ManagedImageSharedGalleryReplicationRegionA"},
				{Name: "ManagedImageSharedGalleryReplicationRegionB"},
			},
			SigDestinationStorageAccountType: "Standard_LRS",
		},
		Tags: map[string]string{"tag01": "Unit Test: Tags"},
	}
	var testSubject = &StepPublishToSharedImageGallery{
		publish: func(ctx context.Context, args PublishArgs) (string, error) {
			actualPublishArgs = args
			return "", nil
		},
		say:   func(message string) {},
		error: func(e error) {},
		toSIG: func() bool { return true },
	}

	stateBag := createTestStateBagStepPublishToSharedImageGallery(false)
	stateBag.Put(constants.ArmSharedImageGalleryDestinationShallowReplication, true)
	var result = testSubject.Run(context.Background(), stateBag)
	if result != multistep.ActionContinue {
		t.Fatalf("Expected the step to return 'ActionContinue', but got '%d'.", result)
	}

	if _, ok := stateBag.GetOk(constants.Error); ok == true {
		t.Fatalf("Expected the step to not set stateBag['%s'], but it was.", constants.Error)
	}

	if diff := cmp.Diff(actualPublishArgs, expectedPublishArgs, []cmp.Option{
		cmpopts.IgnoreUnexported(PublishArgs{}),
	}...); diff != "" {
		t.Fatalf("Unexpected diff %s", diff)
	}
}

func TestStepPublishToSharedImageGalleryShouldPublishWithReplicationCount(t *testing.T) {
	expectedSource := "/subscriptions/00000000-0000-0000-0000-000000000000/resourceGroups/my-group/providers/Microsoft.Compute/virtualMachines/packer-test"
	var actualPublishArgs PublishArgs
	expectedPublishArgs := PublishArgs{
		SubscriptionID:  "Unit Test: ManagedImageSubscription",
		ReplicaCount:    5,
		ReplicationMode: "Full",
		Location:        "Unit Test: Location",
		GallerySource: galleryimageversions.GalleryArtifactVersionFullSource{
			VirtualMachineId: &expectedSource,
		},
		SharedImageGallery: SharedImageGalleryDestination{
			SigDestinationGalleryName:   "Unit Test: ManagedImageSharedGalleryName",
			SigDestinationImageName:     "Unit Test: ManagedImageSharedGalleryImageName",
			SigDestinationSubscription:  "00000000-0000-0000-0000-000000000000",
			SigDestinationImageVersion:  "Unit Test: ManagedImageSharedGalleryImageVersion",
			SigDestinationResourceGroup: "Unit Test: ManagedImageSigPublishResourceGroup",
			SigDestinationReplicationRegions: []string{
				"ManagedImageSharedGalleryReplicationRegionA",
				"ManagedImageSharedGalleryReplicationRegionB",
			},
			SigDestinationTargetRegions: []TargetRegion{
				{Name: "ManagedImageSharedGalleryReplicationRegionA"},
				{Name: "ManagedImageSharedGalleryReplicationRegionB"},
			},
			SigDestinationStorageAccountType: "Standard_LRS",
		},
		Tags: map[string]string{"tag01": "Unit Test: Tags"},
	}
	var testSubject = &StepPublishToSharedImageGallery{
		publish: func(ctx context.Context, args PublishArgs) (string, error) {
			actualPublishArgs = args
			return "", nil
		},
		say:   func(message string) {},
		error: func(e error) {},
		toSIG: func() bool { return true },
	}

	stateBag := createTestStateBagStepPublishToSharedImageGallery(false)
	stateBag.Put(constants.ArmManagedImageSharedGalleryImageVersionReplicaCount, int64(5))
	var result = testSubject.Run(context.Background(), stateBag)
	if result != multistep.ActionContinue {
		t.Fatalf("Expected the step to return 'ActionContinue', but got '%d'.", result)
	}

	if _, ok := stateBag.GetOk(constants.Error); ok == true {
		t.Fatalf("Expected the step to not set stateBag['%s'], but it was.", constants.Error)
	}

	if diff := cmp.Diff(actualPublishArgs, expectedPublishArgs, []cmp.Option{
		cmpopts.IgnoreUnexported(PublishArgs{}),
	}...); diff != "" {
		t.Fatalf("Unexpected diff %s", diff)
	}
}

func TestStepPublishToSharedImageGalleryShouldPublishTargetRegions(t *testing.T) {
	expectedSource := "/subscriptions/00000000-0000-0000-0000-000000000000/resourceGroups/my-group/providers/Microsoft.Compute/virtualMachines/packer-test"
	var actualPublishArgs PublishArgs
	expectedPublishArgs := PublishArgs{
		SubscriptionID:  "Unit Test: ManagedImageSubscription",
		ReplicaCount:    1,
		ReplicationMode: "Full",
		Location:        "Unit Test: Location",
		GallerySource: galleryimageversions.GalleryArtifactVersionFullSource{
			VirtualMachineId: &expectedSource,
		},
		SharedImageGallery: SharedImageGalleryDestination{
			SigDestinationGalleryName:   "Unit Test: ManagedImageSharedGalleryName",
			SigDestinationImageName:     "Unit Test: ManagedImageSharedGalleryImageName",
			SigDestinationSubscription:  "00000000-0000-0000-0000-000000000000",
			SigDestinationImageVersion:  "Unit Test: ManagedImageSharedGalleryImageVersion",
			SigDestinationResourceGroup: "Unit Test: ManagedImageSigPublishResourceGroup",
			SigDestinationReplicationRegions: []string{
				"ManagedImageSharedGalleryReplicationRegionA",
				"ManagedImageSharedGalleryReplicationRegionB",
			},
			SigDestinationTargetRegions: []TargetRegion{
				{Name: "ManagedImageSharedGalleryReplicationRegionA"},
				{Name: "ManagedImageSharedGalleryReplicationRegionB"},
			},
			SigDestinationStorageAccountType: "Standard_LRS",
		},
		Tags: map[string]string{"tag01": "Unit Test: Tags"},
	}
	var testSubject = &StepPublishToSharedImageGallery{
		publish: func(ctx context.Context, args PublishArgs) (string, error) {
			actualPublishArgs = args
			return "", nil
		},
		say:   func(message string) {},
		error: func(e error) {},
		toSIG: func() bool { return true },
	}

	stateBag := createTestStateBagStepPublishToSharedImageGallery(false)
	stateBag.Put(constants.ArmSharedImageGalleryDestinationTargetRegions, []TargetRegion{
		{Name: "ManagedImageSharedGalleryReplicationRegionA"},
		{Name: "ManagedImageSharedGalleryReplicationRegionB"},
	})
	var result = testSubject.Run(context.Background(), stateBag)
	if result != multistep.ActionContinue {
		t.Fatalf("Expected the step to return 'ActionContinue', but got '%d'.", result)
	}

	if _, ok := stateBag.GetOk(constants.Error); ok == true {
		t.Fatalf("Expected the step to not set stateBag['%s'], but it was.", constants.Error)
	}

	if diff := cmp.Diff(actualPublishArgs, expectedPublishArgs, []cmp.Option{
		cmpopts.IgnoreUnexported(PublishArgs{}),
	}...); diff != "" {
		t.Fatalf("Unexpected diff %s", diff)
	}
}

func TestPublishToSharedImageGalleryBuildAzureImageTargetRegions(t *testing.T) {
	type SIG = SharedImageGalleryDestination
	tt := []struct {
		name            string
		in              SIG
		expectedRegions int
	}{
		{name: "empty regions", in: SIG{SigDestinationTargetRegions: nil}, expectedRegions: 0},
		{name: "empty regions non nil", in: SIG{SigDestinationTargetRegions: make([]TargetRegion, 0)}, expectedRegions: 0},
		{name: "one named region", in: SIG{SigDestinationTargetRegions: []TargetRegion{{Name: "unit-test-location"}}}, expectedRegions: 1},
		{name: "two named region", in: SIG{SigDestinationTargetRegions: []TargetRegion{{Name: "unit-test-location"}, {Name: "unit-test-location-2"}}}, expectedRegions: 2},
		{name: "two named regions with replica counts", in: SIG{SigDestinationTargetRegions: []TargetRegion{{Name: "unit-test-location", ReplicaCount: 1}, {Name: "unit-test-location-2", ReplicaCount: 2}}}, expectedRegions: 2},
		{
			name:            "named region with encryption",
			in:              SIG{SigDestinationTargetRegions: []TargetRegion{{Name: "unit-test-location", DiskEncryptionSetId: "boguskey"}}},
			expectedRegions: 1,
		},
		{
			name:            "two named region with encryption",
			in:              SIG{SigDestinationTargetRegions: []TargetRegion{{Name: "unit-test-location", DiskEncryptionSetId: "boguskey"}, {Name: "unit-test-location-west", DiskEncryptionSetId: "boguskeywest"}}},
			expectedRegions: 2,
		},
		{
			name: "one named region with cvm paas key encryption",
			in: SIG{
				SigDestinationTargetRegions:                     []TargetRegion{{Name: "unit-test-location"}},
				SigDestinationConfidentialVMImageEncryptionType: "EncryptedVMGuestStateOnlyWithPmk",
			},
			expectedRegions: 1,
		},
		{
			name: "two named region with cvm paas key encryption",
			in: SIG{
				SigDestinationTargetRegions:                     []TargetRegion{{Name: "unit-test-location"}, {Name: "unit-test-location-2"}},
				SigDestinationConfidentialVMImageEncryptionType: "EncryptedVMGuestStateOnlyWithPmk",
			},
			expectedRegions: 2,
		},
		{
			name: "one named region with cvm des key encryption",
			in: SIG{
				SigDestinationTargetRegions: []TargetRegion{
					{
						Name:                "unit-test-location",
						DiskEncryptionSetId: "boguskey",
					},
				},
				SigDestinationConfidentialVMImageEncryptionType: "EncryptedWithCmk",
			},
			expectedRegions: 1,
		},
		{
			name: "two named region with cvm des key encryption",
			in: SIG{
				SigDestinationTargetRegions: []TargetRegion{
					{
						Name:                "unit-test-location",
						DiskEncryptionSetId: "boguskey",
					},
					{
						Name:                "unit-test-location-west",
						DiskEncryptionSetId: "boguskeywest",
					},
				},
				SigDestinationConfidentialVMImageEncryptionType: "EncryptedWithCmk",
			},
			expectedRegions: 2,
		},
	}

	for _, tc := range tt {
		got := buildAzureImageTargetRegions(tc.in)
		if len(got) != tc.expectedRegions {
			t.Errorf("expected configureTargetRegion() to have same region count: got %d expected %d", len(tc.in.SigDestinationTargetRegions), tc.expectedRegions)
		}

		for i, tr := range got {
			inputRegion := tc.in.SigDestinationTargetRegions[i]
			if tr.Name != inputRegion.Name {
				t.Errorf("expected configured region to contain same name as input %q but got %q", inputRegion.Name, tr.Name)
			}

			if (inputRegion.DiskEncryptionSetId == "") && (tr.Encryption != nil) && (tc.in.SigDestinationConfidentialVMImageEncryptionType == "") {
				t.Errorf("[%q]: expected configured region with no DES id to not contain encryption %q but got %v", tc.name, inputRegion.DiskEncryptionSetId, *tr.Encryption)
			}

			if tc.in.SigDestinationConfidentialVMImageEncryptionType != "" {
				if (inputRegion.DiskEncryptionSetId != "") && (*tr.Encryption.OsDiskImage.SecurityProfile.SecureVMDiskEncryptionSetId != inputRegion.DiskEncryptionSetId) {
					t.Errorf("[%q]: expected configured region to contain set DES Id %q but got %q", tc.name, inputRegion.DiskEncryptionSetId, *tr.Encryption.OsDiskImage.SecurityProfile.SecureVMDiskEncryptionSetId)
				}
			} else {
				if (inputRegion.DiskEncryptionSetId != "") && (*tr.Encryption.OsDiskImage.DiskEncryptionSetId != inputRegion.DiskEncryptionSetId) {
					t.Errorf("[%q]: expected configured region to contain set DES Id %q but got %q", tc.name, inputRegion.DiskEncryptionSetId, *tr.Encryption.OsDiskImage.DiskEncryptionSetId)
				}
			}

			if (inputRegion.ReplicaCount != 0) && (*tr.RegionalReplicaCount != inputRegion.ReplicaCount) {
				t.Errorf("[%q]: expected configured region to contain replica count of %d but got %d", tc.name, inputRegion.ReplicaCount, *tr.RegionalReplicaCount)
			}
			// default replica count
			if (inputRegion.ReplicaCount == 0) && (*tr.RegionalReplicaCount != 1) {
				t.Errorf("[%q]: expected configured region to with no replica count to default to 1 but got %d", tc.name, *tr.RegionalReplicaCount)
			}

		}
	}

}

func TestStepPublishToSharedImageGalleryShouldPublishForConfidentialVMImageWithSig(t *testing.T) {
	var actualPublishArgs PublishArgs
	expectedSource := "/subscriptions/00000000-0000-0000-0000-000000000000/resourceGroups/my-group/providers/Microsoft.Compute/virtualMachines/packer-test"
	expectedPublishArgs := PublishArgs{
		SubscriptionID:  "Unit Test: ManagedImageSubscription",
		ReplicaCount:    1,
		ReplicationMode: "Full",
		Location:        "Unit Test: Location",
		GallerySource: galleryimageversions.GalleryArtifactVersionFullSource{
			VirtualMachineId: &expectedSource,
		},
		SharedImageGallery: SharedImageGalleryDestination{
			SigDestinationGalleryName:   "Unit Test: ManagedImageSharedGalleryName",
			SigDestinationImageName:     "Unit Test: ManagedImageSharedGalleryImageName",
			SigDestinationSubscription:  "00000000-0000-0000-0000-000000000000",
			SigDestinationImageVersion:  "Unit Test: ManagedImageSharedGalleryImageVersion",
			SigDestinationResourceGroup: "Unit Test: ManagedImageSigPublishResourceGroup",
			SigDestinationReplicationRegions: []string{
				"ManagedImageSharedGalleryReplicationRegionA",
				"ManagedImageSharedGalleryReplicationRegionB",
			},
			SigDestinationTargetRegions: []TargetRegion{
				{Name: "ManagedImageSharedGalleryReplicationRegionA"},
				{Name: "ManagedImageSharedGalleryReplicationRegionB"},
			},
			SigDestinationStorageAccountType:                "Standard_LRS",
			SigDestinationConfidentialVMImageEncryptionType: "EncryptedVMGuestStateOnlyWithPmk",
		},
		Tags: map[string]string{"tag01": "Unit Test: Tags"},
	}
	var testSubject = &StepPublishToSharedImageGallery{
		publish: func(ctx context.Context, args PublishArgs) (string, error) {
			actualPublishArgs = args
			return "", nil
		},
		say:   func(message string) {},
		error: func(e error) {},
		toSIG: func() bool { return true },
	}

	stateBag := createTestStateBagStepPublishToSharedImageGallery(false)
	stateBag.Put(constants.ArmSharedImageGalleryDestinationConfidentialVMImageEncryptionType, "EncryptedVMGuestStateOnlyWithPmk")
	var result = testSubject.Run(context.Background(), stateBag)
	if result != multistep.ActionContinue {
		t.Fatalf("Expected the step to return 'ActionContinue', but got '%d'.", result)
	}

	if _, ok := stateBag.GetOk(constants.Error); ok == true {
		t.Fatalf("Expected the step to not set stateBag['%s'], but it was.", constants.Error)
	}

	if diff := cmp.Diff(actualPublishArgs, expectedPublishArgs, []cmp.Option{
		cmpopts.IgnoreUnexported(PublishArgs{}),
	}...); diff != "" {
		t.Fatalf("Unexpected diff %s", diff)
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
	stateBag.Put(constants.ArmTags, tags)
	stateBag.Put(constants.ArmSharedImageGalleryDestinationTargetRegions, []TargetRegion{
		{Name: "ManagedImageSharedGalleryReplicationRegionA"},
		{Name: "ManagedImageSharedGalleryReplicationRegionB"},
	})
	stateBag.Put(constants.ArmManagedImageSharedGalleryImageVersionStorageAccountType, "Standard_LRS")
	if managed {
		stateBag.Put(constants.ArmManagedImageResourceGroupName, "my-group")
		stateBag.Put(constants.ArmManagedImageName, "packer-test")
	} else {
		stateBag.Put(constants.ArmImageParameters, &images.Image{Properties: &images.ImageProperties{
			SourceVirtualMachine: &images.SubResource{Id: common.StringPtr("/subscriptions/00000000-0000-0000-0000-000000000000/resourceGroups/my-group/providers/Microsoft.Compute/virtualMachines/packer-test")},
		}})
	}
	stateBag.Put(constants.ArmManagedImageSubscription, "00000000-0000-0000-0000-000000000000")
	stateBag.Put(constants.ArmSharedImageGalleryDestinationSubscription, "Unit Test: ManagedImageSubscription")
	stateBag.Put(constants.ArmIsManagedImage, managed)
	stateBag.Put(constants.ArmIsSIGImage, true)
	stateBag.Put(constants.ArmSharedImageGalleryDestinationShallowReplication, false)

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
