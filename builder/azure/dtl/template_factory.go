// Copyright (c) HashiCorp, Inc.
// SPDX-License-Identifier: MPL-2.0

package dtl

import (
	"fmt"

	hashiDTLLabsSDK "github.com/hashicorp/go-azure-sdk/resource-manager/devtestlab/2018-09-15/labs"
)

type templateFactoryFuncDtl func(*Config) (*hashiDTLLabsSDK.LabVirtualMachineCreationParameter, error)

func newBool(val bool) *bool {
	b := true
	if val == b {
		return &b
	} else {
		b = false
		return &b
	}
}

func getCustomImageId(config *Config) *string {
	if config.CustomManagedImageName != "" && config.CustomManagedImageResourceGroupName != "" {
		customManagedImageID := fmt.Sprintf("/subscriptions/%s/resourceGroups/%s/providers/Microsoft.Compute/images/%s",
			config.ClientConfig.SubscriptionID,
			config.CustomManagedImageResourceGroupName,
			config.CustomManagedImageName)
		return &customManagedImageID
	}
	return nil
}

func GetVirtualMachineDeployment(config *Config) (*hashiDTLLabsSDK.LabVirtualMachineCreationParameter, error) {

	galleryImageRef := hashiDTLLabsSDK.GalleryImageReference{
		Offer:     &config.ImageOffer,
		Publisher: &config.ImagePublisher,
		Sku:       &config.ImageSku,
		OsType:    &config.OSType,
		Version:   &config.ImageVersion,
	}

	labVirtualNetworkID := fmt.Sprintf("/subscriptions/%s/resourceGroups/%s/providers/Microsoft.DevTestLab/labs/%s/virtualnetworks/%s",
		config.ClientConfig.SubscriptionID,
		config.tmpResourceGroupName,
		config.LabName,
		config.LabVirtualNetworkName)

	dtlArtifacts := []hashiDTLLabsSDK.ArtifactInstallProperties{}

	if config.DtlArtifacts != nil {
		for i := range config.DtlArtifacts {
			if config.DtlArtifacts[i].RepositoryName == "" {
				config.DtlArtifacts[i].RepositoryName = "public repo"
			}
			config.DtlArtifacts[i].ArtifactId = fmt.Sprintf("/subscriptions/%s/resourceGroups/%s/providers/Microsoft.DevTestLab/labs/%s/artifactSources/%s/artifacts/%s",
				config.ClientConfig.SubscriptionID,
				config.tmpResourceGroupName,
				config.LabName,
				config.DtlArtifacts[i].RepositoryName,
				config.DtlArtifacts[i].ArtifactName)

			dparams := []hashiDTLLabsSDK.ArtifactParameterProperties{}
			for j := range config.DtlArtifacts[i].Parameters {

				dp := &hashiDTLLabsSDK.ArtifactParameterProperties{}
				dp.Name = &config.DtlArtifacts[i].Parameters[j].Name
				dp.Value = &config.DtlArtifacts[i].Parameters[j].Value

				dparams = append(dparams, *dp)
			}
			dtlArtifact := &hashiDTLLabsSDK.ArtifactInstallProperties{
				ArtifactTitle: &config.DtlArtifacts[i].ArtifactName,
				ArtifactId:    &config.DtlArtifacts[i].ArtifactId,
				Parameters:    &dparams,
			}
			dtlArtifacts = append(dtlArtifacts, *dtlArtifact)
		}
	}

	labMachineProps := &hashiDTLLabsSDK.LabVirtualMachineCreationParameterProperties{
		OwnerUserPrincipalName:            &config.ClientConfig.ClientID,
		OwnerObjectId:              &config.ClientConfig.ObjectID,
		Size:                       &config.VMSize,
		UserName:                   &config.UserName,
		Password:                   &config.Password,
		SshKey:                     &config.sshAuthorizedKey,
		IsAuthenticationWithSshKey: newBool(true),
		LabSubnetName:              &config.LabSubnetName,
		LabVirtualNetworkId:        &labVirtualNetworkID,
		DisallowPublicIPAddress:    &config.DisallowPublicIP,
		GalleryImageReference:      &galleryImageRef,
		CustomImageId:              getCustomImageId(config),
		PlanId:                     &config.PlanID,

		AllowClaim:                   newBool(false),
		StorageType:                  &config.StorageType,
		Artifacts:                    &dtlArtifacts,
	}

	labMachine := &hashiDTLLabsSDK.LabVirtualMachineCreationParameter{
		Name:     &config.tmpComputeName,
		Location: &config.Location,
		// TODO 
		//Tags:     config.AzureTags,
		Properties: labMachineProps,
	}

	return labMachine, nil
}
