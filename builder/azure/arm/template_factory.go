// Copyright (c) HashiCorp, Inc.
// SPDX-License-Identifier: MPL-2.0

package arm

import (
	"crypto/rsa"
	"encoding/json"
	"errors"
	"fmt"
	"time"

	hashiVMSDK "github.com/hashicorp/go-azure-sdk/resource-manager/compute/2022-03-01/virtualmachines"
	"github.com/hashicorp/go-azure-sdk/resource-manager/resources/2022-09-01/deployments"
	"github.com/hashicorp/packer-plugin-azure/builder/azure/common/constants"
	"github.com/hashicorp/packer-plugin-azure/builder/azure/common/template"
	"github.com/hashicorp/packer-plugin-sdk/packer"
	"golang.org/x/crypto/ssh"
)

type templateFactoryFunc func(*Config) (*deployments.Deployment, error)

func GetCommunicatorSpecificKeyVaultDeployment(config *Config) (*deployments.Deployment, error) {
	if config.Comm.Type == "ssh" {
		privateKey, err := ssh.ParseRawPrivateKey(config.Comm.SSHPrivateKey)
		if err != nil {
			return nil, err
		}
		pk, ok := privateKey.(*rsa.PrivateKey)
		if !ok {
			//https://learn.microsoft.com/en-us/azure/virtual-machines/windows/connect-ssh?tabs=azurecli#supported-ssh-key-formats
			return nil, errors.New("Provided private key must be in RSA format to use for SSH on Windows on Azure")
		}
		secret, err := config.formatCertificateForKeyVault(pk)
		if err != nil {
			return nil, err
		}

		// Hide the secret key pair blob from logs
		packer.LogSecretFilter.Set(secret)

		return GetKeyVaultDeployment(config, secret, nil)
	} else {
		var exp *int64
		if config.WinrmExpirationTime != 0 {
			unixSeconds := time.Now().Add(config.WinrmExpirationTime).Unix()
			exp = &unixSeconds
		}
		return GetKeyVaultDeployment(config, config.winrmCertificate, exp)
	}
}

func GetKeyVaultDeployment(config *Config, secretValue string, exp *int64) (*deployments.Deployment, error) {
	params := &template.TemplateParameters{
		KeyVaultName:        &template.TemplateParameter{Value: config.tmpKeyVaultName},
		KeyVaultSKU:         &template.TemplateParameter{Value: config.BuildKeyVaultSKU},
		KeyVaultSecretName:  &template.TemplateParameter{Value: config.BuildKeyVaultSecretName},
		KeyVaultSecretValue: &template.TemplateParameter{Value: secretValue},
		ObjectId:            &template.TemplateParameter{Value: config.ClientConfig.ObjectID},
		TenantId:            &template.TemplateParameter{Value: config.ClientConfig.TenantID},
	}

	builder, _ := template.NewTemplateBuilder(template.KeyVault)
	_ = builder.SetTags(&config.AzureTags)

	if exp != nil {
		err := builder.SetSecretExpiry(*exp)
		if err != nil {
			return nil, err
		}
	}
	doc, _ := builder.ToJSON()
	return createDeploymentParameters(*doc, params)
}

func GetSpecializedVirtualMachineDeployment(config *Config) (*deployments.Deployment, error) {
	builder, err := GetVirtualMachineTemplateBuilder(config)
	if err != nil {
		return nil, err
	}
	params := &template.TemplateParameters{
		AdminUsername:              &template.TemplateParameter{Value: config.UserName},
		AdminPassword:              &template.TemplateParameter{Value: config.Password},
		DnsNameForPublicIP:         &template.TemplateParameter{Value: config.tmpComputeName},
		NicName:                    &template.TemplateParameter{Value: config.tmpNicName},
		OSDiskName:                 &template.TemplateParameter{Value: config.tmpOSDiskName},
		DataDiskName:               &template.TemplateParameter{Value: config.tmpDataDiskName},
		PublicIPAddressName:        &template.TemplateParameter{Value: config.tmpPublicIPAddressName},
		SubnetName:                 &template.TemplateParameter{Value: config.tmpSubnetName},
		StorageAccountBlobEndpoint: &template.TemplateParameter{Value: config.storageAccountBlobEndpoint},
		VirtualNetworkName:         &template.TemplateParameter{Value: config.tmpVirtualNetworkName},
		NsgName:                    &template.TemplateParameter{Value: config.tmpNsgName},
		VMSize:                     &template.TemplateParameter{Value: config.VMSize},
		VMName:                     &template.TemplateParameter{Value: config.tmpComputeName},
		CommandToExecute:           &template.TemplateParameter{Value: config.CustomScript},
	}

	err = builder.ClearOsProfile()
	if err != nil {
		return nil, err
	}
	doc, _ := builder.ToJSON()
	return createDeploymentParameters(*doc, params)
}

func GetVirtualMachineDeployment(config *Config) (*deployments.Deployment, error) {
	builder, err := GetVirtualMachineTemplateBuilder(config)
	if err != nil {
		return nil, err
	}
	params := &template.TemplateParameters{
		AdminUsername:              &template.TemplateParameter{Value: config.UserName},
		AdminPassword:              &template.TemplateParameter{Value: config.Password},
		DnsNameForPublicIP:         &template.TemplateParameter{Value: config.tmpComputeName},
		NicName:                    &template.TemplateParameter{Value: config.tmpNicName},
		OSDiskName:                 &template.TemplateParameter{Value: config.tmpOSDiskName},
		DataDiskName:               &template.TemplateParameter{Value: config.tmpDataDiskName},
		PublicIPAddressName:        &template.TemplateParameter{Value: config.tmpPublicIPAddressName},
		SubnetName:                 &template.TemplateParameter{Value: config.tmpSubnetName},
		StorageAccountBlobEndpoint: &template.TemplateParameter{Value: config.storageAccountBlobEndpoint},
		VirtualNetworkName:         &template.TemplateParameter{Value: config.tmpVirtualNetworkName},
		NsgName:                    &template.TemplateParameter{Value: config.tmpNsgName},
		VMSize:                     &template.TemplateParameter{Value: config.VMSize},
		VMName:                     &template.TemplateParameter{Value: config.tmpComputeName},
		CommandToExecute:           &template.TemplateParameter{Value: config.CustomScript},
	}

	doc, _ := builder.ToJSON()
	return createDeploymentParameters(*doc, params)
}

func GetVirtualMachineTemplateBuilder(config *Config) (*template.TemplateBuilder, error) {
	builder, err := template.NewTemplateBuilder(template.BasicTemplate)
	if err != nil {
		return nil, err
	}
	osType := hashiVMSDK.OperatingSystemTypesLinux

	switch config.OSType {
	case constants.Target_Linux:
		if config.CustomScript != "" {
			return nil, fmt.Errorf("CustomScript was set, but this build targets Linux; CustomScript is only supported for Windows")
		}
		err = builder.BuildLinux(config.sshAuthorizedKey, config.Comm.SSHPassword == "") // if ssh password is not explicitly specified, disable password auth
		if err != nil {
			return nil, err
		}
	case constants.Target_Windows:
		osType = hashiVMSDK.OperatingSystemTypesWindows
		err = builder.BuildWindows(config.Comm.Type, config.tmpKeyVaultName, config.tmpWinRMCertificateUrl)
		if err != nil {
			return nil, err
		}
	}

	if len(config.UserAssignedManagedIdentities) != 0 {
		if err := builder.SetIdentity(config.UserAssignedManagedIdentities); err != nil {
			return nil, err
		}
	}

	if config.ImageUrl != "" {
		err = builder.SetImageUrl(config.ImageUrl, osType, config.diskCachingType)
		if err != nil {
			return nil, err
		}
	} else if config.CustomManagedImageName != "" {
		err = builder.SetManagedDiskUrl(config.customManagedImageID, config.managedImageStorageAccountType, config.diskCachingType)
		if err != nil {
			return nil, err
		}
	} else if (config.isManagedImage() || config.isPublishToSIG()) && config.ImagePublisher != "" {
		// TODO : Handle this error
		_ = builder.SetManagedMarketplaceImage(config.ImagePublisher, config.ImageOffer, config.ImageSku, config.ImageVersion, config.managedImageStorageAccountType, config.diskCachingType)
	} else if config.SharedGallery.Subscription != "" {
		imageID := config.getSourceSharedImageGalleryID()

		err = builder.SetSharedGalleryImage(config.Location, imageID, config.diskCachingType)
		if err != nil {
			return nil, err
		}
	} else if config.SharedGallery.CommunityGalleryImageId != "" {
		imageID := config.SharedGallery.CommunityGalleryImageId

		err = builder.SetCommunityGalleryImage(config.Location, imageID, config.diskCachingType)
		if err != nil {
			return nil, err
		}
	} else if config.SharedGallery.DirectSharedGalleryImageID != "" {
		imageID := config.SharedGallery.DirectSharedGalleryImageID

		err = builder.SetDirectSharedGalleryImage(config.Location, imageID, config.diskCachingType)
		if err != nil {
			return nil, err
		}
	} else {
		err = builder.SetMarketPlaceImage(config.ImagePublisher, config.ImageOffer, config.ImageSku, config.ImageVersion, config.diskCachingType)
		if err != nil {
			return nil, err
		}
	}

	securityProfile := struct {
		SecureBootEnabled bool
		VTpmEnabled       bool
		EncryptionAtHost  *bool
		securityType      *hashiVMSDK.SecurityTypes
	}{
		SecureBootEnabled: config.SecureBootEnabled,
		VTpmEnabled:       config.VTpmEnabled,
		EncryptionAtHost:  nil,
		securityType:      nil,
	}

	if config.EncryptionAtHost != nil {
		securityProfile.EncryptionAtHost = config.EncryptionAtHost
	}

	if config.SecurityType != "" {
		switch config.SecurityType {
		case constants.TrustedLaunch:
			tl := hashiVMSDK.SecurityTypesTrustedLaunch
			securityProfile.securityType = &tl
		case constants.ConfidentialVM:
			cvm := hashiVMSDK.SecurityTypesConfidentialVM
			securityProfile.securityType = &cvm
		}
	}

	err = builder.SetSecurityProfile(securityProfile.SecureBootEnabled, securityProfile.VTpmEnabled, securityProfile.EncryptionAtHost, securityProfile.securityType)
	if err != nil {
		return nil, err
	}

	if config.OSDiskSizeGB > 0 {
		err = builder.SetOSDiskSizeGB(config.OSDiskSizeGB)
		if err != nil {
			return nil, err
		}
	}

	if config.DiskEncryptionSetId != "" {
		err = builder.SetDiskEncryptionSetID(config.DiskEncryptionSetId, securityProfile.securityType)
		if err != nil {
			return nil, err
		}
	}

	if config.DiskEncryptionSetId == "" {
		err = builder.SetDiskEncryptionWithPaaSKey(securityProfile.securityType)
		if err != nil {
			return nil, err
		}
	}

	if len(config.AdditionalDiskSize) > 0 {
		isLegacyVHD := config.CustomManagedImageName == "" && config.ManagedImageName == "" && config.SharedGalleryDestination.SigDestinationGalleryName == ""
		err = builder.SetAdditionalDisks(config.AdditionalDiskSize, config.tmpDataDiskName, isLegacyVHD, config.diskCachingType)
		if err != nil {
			return nil, err
		}
	}

	if config.Spot.EvictionPolicy != "" {
		err = builder.SetSpot(config.Spot.EvictionPolicy, config.Spot.MaxPrice)
		if err != nil {
			return nil, err
		}
	}

	if config.customData != "" {
		err = builder.SetCustomData(config.customData)
		if err != nil {
			return nil, err
		}
	}

	if config.userData != "" {
		err = builder.SetUserData(config.userData)
		if err != nil {
			return nil, err
		}
	}

	if config.PlanInfo.PlanName != "" {
		err = builder.SetPlanInfo(config.PlanInfo.PlanName, config.PlanInfo.PlanProduct, config.PlanInfo.PlanPublisher, config.PlanInfo.PlanPromotionCode)
		if err != nil {
			return nil, err
		}
	}

	if config.VirtualNetworkName != "" && DefaultPrivateVirtualNetworkWithPublicIp != config.PrivateVirtualNetworkWithPublicIp {
		err = builder.SetPrivateVirtualNetworkWithPublicIp(
			config.VirtualNetworkResourceGroupName,
			config.VirtualNetworkName,
			config.VirtualNetworkSubnetName)
		if err != nil {
			return nil, err
		}
	} else if config.VirtualNetworkName != "" {
		err = builder.SetVirtualNetwork(
			config.VirtualNetworkResourceGroupName,
			config.VirtualNetworkName,
			config.VirtualNetworkSubnetName)
		if err != nil {
			return nil, err
		}
	}

	if config.AllowedInboundIpAddresses != nil && len(config.AllowedInboundIpAddresses) >= 1 && config.Comm.Port() != 0 {
		err = builder.SetNetworkSecurityGroup(config.AllowedInboundIpAddresses, config.Comm.Port())
		if err != nil {
			return nil, err
		}
	}

	if config.BootDiagSTGAccount != "" {
		err = builder.SetBootDiagnostics(config.BootDiagSTGAccount)
		if err != nil {
			return nil, err
		}
	}

	if config.LicenseType != "" {
		err = builder.SetLicenseType(config.LicenseType)
		if err != nil {
			return nil, err
		}
	}

	err = builder.SetTags(&config.AzureTags)
	if err != nil {
		return nil, err
	}
	return builder, nil
}

func createDeploymentParameters(doc string, parameters *template.TemplateParameters) (*deployments.Deployment, error) {
	var template interface{}
	err := json.Unmarshal(([]byte)(doc), &template)
	if err != nil {
		return nil, err
	}

	bs, err := json.Marshal(*parameters)
	if err != nil {
		return nil, err
	}

	var templateParameters interface{}
	err = json.Unmarshal(bs, &templateParameters)
	if err != nil {
		return nil, err
	}

	return &deployments.Deployment{
		Properties: deployments.DeploymentProperties{
			Mode:       deployments.DeploymentModeIncremental,
			Template:   &template,
			Parameters: &templateParameters,
		},
	}, nil
}
