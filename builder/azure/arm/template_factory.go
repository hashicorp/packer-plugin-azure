package arm

import (
	"crypto/rsa"
	"encoding/json"
	"errors"
	"fmt"

	"github.com/Azure/azure-sdk-for-go/services/compute/mgmt/2021-11-01/compute"
	"github.com/Azure/azure-sdk-for-go/services/resources/mgmt/2018-02-01/resources"

	"github.com/hashicorp/packer-plugin-azure/builder/azure/common/constants"
	"github.com/hashicorp/packer-plugin-azure/builder/azure/common/template"
	"golang.org/x/crypto/ssh"
)

type templateFactoryFunc func(*Config) (*resources.Deployment, error)

func GetCommunicatorSpecificKeyVaultDeployment(config *Config) (*resources.Deployment, error) {
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
		return GetKeyVaultDeployment(config, secret)
	} else {
		return GetKeyVaultDeployment(config, config.winrmCertificate)
	}
}

func GetKeyVaultDeployment(config *Config, secretValue string) (*resources.Deployment, error) {
	params := &template.TemplateParameters{
		KeyVaultName:        &template.TemplateParameter{Value: config.tmpKeyVaultName},
		KeyVaultSKU:         &template.TemplateParameter{Value: config.BuildKeyVaultSKU},
		KeyVaultSecretValue: &template.TemplateParameter{Value: secretValue},
		ObjectId:            &template.TemplateParameter{Value: config.ClientConfig.ObjectID},
		TenantId:            &template.TemplateParameter{Value: config.ClientConfig.TenantID},
	}

	builder, _ := template.NewTemplateBuilder(template.KeyVault)
	_ = builder.SetTags(&config.AzureTags)

	doc, _ := builder.ToJSON()
	return createDeploymentParameters(*doc, params)
}

func GetVirtualMachineDeployment(config *Config) (*resources.Deployment, error) {
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

	builder, err := template.NewTemplateBuilder(template.BasicTemplate)
	if err != nil {
		return nil, err
	}
	osType := compute.OperatingSystemTypesLinux

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
		osType = compute.OperatingSystemTypesWindows
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
		imageID := fmt.Sprintf("/subscriptions/%s/resourceGroups/%s/providers/Microsoft.Compute/galleries/%s/images/%s",
			config.SharedGallery.Subscription,
			config.SharedGallery.ResourceGroup,
			config.SharedGallery.GalleryName,
			config.SharedGallery.ImageName)
		if config.SharedGallery.ImageVersion != "" {
			imageID += fmt.Sprintf("/versions/%s",
				config.SharedGallery.ImageVersion)
		}

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

	if config.OSDiskSizeGB > 0 {
		err = builder.SetOSDiskSizeGB(config.OSDiskSizeGB)
		if err != nil {
			return nil, err
		}
	}

	if len(config.AdditionalDiskSize) > 0 {
		isManaged := config.CustomManagedImageName != "" || (config.ManagedImageName != "" && config.ImagePublisher != "") || config.SharedGallery.Subscription != ""
		err = builder.SetAdditionalDisks(config.AdditionalDiskSize, config.tmpDataDiskName, isManaged, config.diskCachingType)
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

	doc, _ := builder.ToJSON()
	return createDeploymentParameters(*doc, params)
}

func createDeploymentParameters(doc string, parameters *template.TemplateParameters) (*resources.Deployment, error) {
	var template map[string]interface{}
	err := json.Unmarshal(([]byte)(doc), &template)
	if err != nil {
		return nil, err
	}

	bs, err := json.Marshal(*parameters)
	if err != nil {
		return nil, err
	}

	var templateParameters map[string]interface{}
	err = json.Unmarshal(bs, &templateParameters)
	if err != nil {
		return nil, err
	}

	return &resources.Deployment{
		Properties: &resources.DeploymentProperties{
			Mode:       resources.Incremental,
			Template:   &template,
			Parameters: &templateParameters,
		},
	}, nil
}
