// Copyright (c) HashiCorp, Inc.
// SPDX-License-Identifier: MPL-2.0

package template

import (
	"encoding/json"
	"fmt"
	"strconv"
	"strings"

	hashiVMSDK "github.com/hashicorp/go-azure-sdk/resource-manager/compute/2022-03-01/virtualmachines"
	hashiSecurityRulesSDK "github.com/hashicorp/go-azure-sdk/resource-manager/network/2022-09-01/securityrules"
	hashiSubnetsSDK "github.com/hashicorp/go-azure-sdk/resource-manager/network/2022-09-01/subnets"
	"github.com/hashicorp/packer-plugin-azure/builder/azure/common"
)

const (
	jsonPrefix = ""
	jsonIndent = "  "

	resourceKeyVaults             = "Microsoft.KeyVault/vaults"
	resourceKeyVaultSecret        = "Microsoft.KeyVault/vaults/secrets"
	resourceNetworkInterfaces     = "Microsoft.Network/networkInterfaces"
	resourcePublicIPAddresses     = "Microsoft.Network/publicIPAddresses"
	resourceVirtualMachine        = "Microsoft.Compute/virtualMachines"
	resourceVirtualNetworks       = "Microsoft.Network/virtualNetworks"
	resourceNetworkSecurityGroups = "Microsoft.Network/networkSecurityGroups"

	variableSshKeyPath = "sshKeyPath"
)

type TemplateBuilder struct {
	template *Template
	osType   hashiVMSDK.OperatingSystemTypes
}

func NewTemplateBuilder(template string) (*TemplateBuilder, error) {
	var t Template

	err := json.Unmarshal([]byte(template), &t)
	if err != nil {
		return nil, err
	}

	return &TemplateBuilder{
		template: &t,
	}, nil
}

func (s *TemplateBuilder) BuildLinux(sshAuthorizedKey string, disablePasswordAuthentication bool) error {
	resource, err := s.getResourceByType(resourceVirtualMachine)
	if err != nil {
		return err
	}

	variableSshKeyPath := s.toVariable(variableSshKeyPath)
	profile := resource.Properties.OsProfile
	profile.LinuxConfiguration = &hashiVMSDK.LinuxConfiguration{
		Ssh: &hashiVMSDK.SshConfiguration{
			PublicKeys: &[]hashiVMSDK.SshPublicKey{
				{
					Path:    &variableSshKeyPath,
					KeyData: &sshAuthorizedKey,
				},
			},
		},
	}

	if disablePasswordAuthentication {
		profile.LinuxConfiguration.DisablePasswordAuthentication = common.BoolPtr(true)
		profile.AdminPassword = nil
	}

	s.osType = hashiVMSDK.OperatingSystemTypesLinux
	return nil
}

func (s *TemplateBuilder) BuildWindows(communicatorType string, keyVaultName string, certificateUrl string, skipCreateKV bool) error {
	resource, err := s.getResourceByType(resourceVirtualMachine)
	if err != nil {
		return err
	}

	profile := resource.Properties.OsProfile
	s.osType = hashiVMSDK.OperatingSystemTypesWindows

	certifacteStore := "My"
	resourceID := s.toResourceID(resourceKeyVaults, keyVaultName)
	profile.Secrets = &[]hashiVMSDK.VaultSecretGroup{
		{
			SourceVault: &hashiVMSDK.SubResource{
				Id: &resourceID,
			},
			VaultCertificates: &[]hashiVMSDK.VaultCertificate{
				{
					CertificateStore: &certifacteStore,
					CertificateUrl:   &certificateUrl,
				},
			},
		},
	}

	provisionVMAgent := true
	basicWindowConfiguration := hashiVMSDK.WindowsConfiguration{
		ProvisionVMAgent: &provisionVMAgent,
	}

	if communicatorType == "ssh" {
		profile.WindowsConfiguration = &basicWindowConfiguration
		return nil
	}

	// when communicator type is winrm
	if !skipCreateKV {
		// when skip kv create is not set, add secrets and listener
		protocol := hashiVMSDK.ProtocolTypesHTTPS
		profile.WindowsConfiguration = &hashiVMSDK.WindowsConfiguration{
			ProvisionVMAgent: common.BoolPtr(true),
			WinRM: &hashiVMSDK.WinRMConfiguration{
				Listeners: &[]hashiVMSDK.WinRMListener{
					{
						Protocol:       &protocol,
						CertificateUrl: common.StringPtr(certificateUrl),
					},
				},
			},
		}
	} else {
		// when skip kv create is set, no need to add secrets and listener in template
		profile.Secrets = nil
		profile.WindowsConfiguration = &basicWindowConfiguration
	}

	return nil
}

func (s *TemplateBuilder) SetSecretExpiry(exp int64) error {
	resource, err := s.getResourceByType(resourceKeyVaultSecret)
	if err != nil {
		return err
	}

	resource.Properties.Attributes = &Attributes{
		Exp: exp,
	}
	return nil
}

func (s *TemplateBuilder) SetIdentity(userAssignedManagedIdentities []string) error {
	resource, err := s.getResourceByType(resourceVirtualMachine)
	if err != nil {
		return err
	}

	var id *Identity

	if len(userAssignedManagedIdentities) != 0 {
		id = &Identity{
			Type:                   common.StringPtr("UserAssigned"),
			UserAssignedIdentities: make(map[string]struct{}),
		}
		for _, uid := range userAssignedManagedIdentities {
			id.UserAssignedIdentities[uid] = struct{}{}
		}
	}

	resource.Identity = id
	return nil
}

func (s *TemplateBuilder) SetManagedDiskUrl(managedImageId string, storageAccountType hashiVMSDK.StorageAccountTypes, cachingType hashiVMSDK.CachingTypes) error {
	resource, err := s.getResourceByType(resourceVirtualMachine)
	if err != nil {
		return err
	}

	profile := resource.Properties.StorageProfile
	profile.ImageReference = &hashiVMSDK.ImageReference{
		Id: &managedImageId,
	}
	profile.OsDisk.OsType = s.osType
	profile.OsDisk.CreateOption = hashiVMSDK.DiskCreateOptionTypesFromImage
	profile.OsDisk.Vhd = nil
	profile.OsDisk.Caching = cachingType
	profile.OsDisk.ManagedDisk = &ManagedDisk{
		StorageAccountType: storageAccountType,
	}

	return nil
}

func (s *TemplateBuilder) SetManagedMarketplaceImage(publisher, offer, sku, version string, storageAccountType hashiVMSDK.StorageAccountTypes, cachingType hashiVMSDK.CachingTypes) error {
	resource, err := s.getResourceByType(resourceVirtualMachine)
	if err != nil {
		return err
	}

	profile := resource.Properties.StorageProfile
	profile.ImageReference = &hashiVMSDK.ImageReference{
		Publisher: &publisher,
		Offer:     &offer,
		Sku:       &sku,
		Version:   &version,
	}
	profile.OsDisk.OsType = s.osType
	profile.OsDisk.CreateOption = hashiVMSDK.DiskCreateOptionTypesFromImage
	profile.OsDisk.Vhd = nil
	profile.OsDisk.Caching = cachingType
	profile.OsDisk.ManagedDisk = &ManagedDisk{
		StorageAccountType: storageAccountType,
	}

	return nil
}

func (s *TemplateBuilder) SetSharedGalleryImage(location, imageID string, cachingType hashiVMSDK.CachingTypes) error {
	resource, err := s.getResourceByType(resourceVirtualMachine)
	if err != nil {
		return err
	}

	profile := resource.Properties.StorageProfile
	profile.ImageReference = &hashiVMSDK.ImageReference{Id: &imageID}
	profile.OsDisk.OsType = s.osType
	profile.OsDisk.Vhd = nil
	profile.OsDisk.Caching = cachingType

	return nil
}

func (s *TemplateBuilder) SetCommunityGalleryImage(location, imageID string, cachingType hashiVMSDK.CachingTypes) error {
	resource, err := s.getResourceByType(resourceVirtualMachine)
	if err != nil {
		return err
	}

	profile := resource.Properties.StorageProfile
	profile.ImageReference = &hashiVMSDK.ImageReference{CommunityGalleryImageId: &imageID}
	profile.OsDisk.OsType = s.osType
	profile.OsDisk.Vhd = nil
	profile.OsDisk.Caching = cachingType

	return nil
}

func (s *TemplateBuilder) SetDirectSharedGalleryImage(location, imageID string, cachingType hashiVMSDK.CachingTypes) error {
	resource, err := s.getResourceByType(resourceVirtualMachine)
	if err != nil {
		return err
	}

	profile := resource.Properties.StorageProfile
	profile.ImageReference = &hashiVMSDK.ImageReference{SharedGalleryImageId: &imageID}
	profile.OsDisk.OsType = s.osType
	profile.OsDisk.Vhd = nil
	profile.OsDisk.Caching = cachingType

	return nil
}

func (s *TemplateBuilder) SetMarketPlaceImage(publisher, offer, sku, version string, cachingType hashiVMSDK.CachingTypes) error {
	resource, err := s.getResourceByType(resourceVirtualMachine)
	if err != nil {
		return err
	}

	profile := resource.Properties.StorageProfile
	profile.OsDisk.Caching = cachingType
	profile.ImageReference = &hashiVMSDK.ImageReference{
		Publisher: common.StringPtr(publisher),
		Offer:     common.StringPtr(offer),
		Sku:       common.StringPtr(sku),
		Version:   common.StringPtr(version),
	}

	return nil
}

func (s *TemplateBuilder) SetImageUrl(imageUrl string, osType hashiVMSDK.OperatingSystemTypes, cachingType hashiVMSDK.CachingTypes) error {
	resource, err := s.getResourceByType(resourceVirtualMachine)
	if err != nil {
		return err
	}

	profile := resource.Properties.StorageProfile
	profile.OsDisk.OsType = osType
	profile.OsDisk.Caching = cachingType

	profile.OsDisk.Image = &hashiVMSDK.VirtualHardDisk{
		Uri: &imageUrl,
	}

	return nil
}

func (s *TemplateBuilder) SetPlanInfo(name, product, publisher, promotionCode string) error {
	var promotionCodeVal *string = nil
	if promotionCode != "" {
		promotionCodeVal = common.StringPtr(promotionCode)
	}

	for i, x := range s.template.Resources {
		if strings.EqualFold(*x.Type, resourceVirtualMachine) {
			s.template.Resources[i].Plan = &Plan{
				Name:          common.StringPtr(name),
				Product:       common.StringPtr(product),
				Publisher:     common.StringPtr(publisher),
				PromotionCode: promotionCodeVal,
			}
		}
	}

	return nil
}

func (s *TemplateBuilder) SetOSDiskSizeGB(diskSizeGB int32) error {
	resource, err := s.getResourceByType(resourceVirtualMachine)
	if err != nil {
		return err
	}

	profile := resource.Properties.StorageProfile
	profile.OsDisk.DiskSizeGB = common.Int32Ptr(diskSizeGB)

	return nil
}

func (s *TemplateBuilder) SetDiskEncryptionSetID(diskEncryptionSetID string) error {
	resource, err := s.getResourceByType(resourceVirtualMachine)
	if err != nil {
		return err
	}

	profile := resource.Properties.StorageProfile
	profile.OsDisk.ManagedDisk.DiskEncryptionSet = &DiskEncryptionSetParameters{
		ID: &diskEncryptionSetID,
	}

	return nil
}

func (s *TemplateBuilder) SetAdditionalDisks(diskSizeGB []int32, dataDiskname string, isLegacyVHD bool, cachingType hashiVMSDK.CachingTypes) error {
	resource, err := s.getResourceByType(resourceVirtualMachine)
	if err != nil {
		return err
	}

	profile := resource.Properties.StorageProfile
	dataDisks := make([]DataDiskUnion, len(diskSizeGB))

	for i, additionalSize := range diskSizeGB {
		dataDisks[i].DiskSizeGB = common.Int32Ptr(additionalSize)
		dataDisks[i].Lun = common.IntPtr(i)
		// dataDisks[i].Name = to.StringPtr(fmt.Sprintf("%s-%d", dataDiskname, i+1))
		dataDisks[i].Name = common.StringPtr(fmt.Sprintf("[concat(parameters('dataDiskName'),'-%d')]", i+1))
		dataDisks[i].CreateOption = "Empty"
		dataDisks[i].Caching = cachingType
		if !isLegacyVHD {
			dataDisks[i].Vhd = nil
			dataDisks[i].ManagedDisk = profile.OsDisk.ManagedDisk
		} else {
			dataDisks[i].Vhd = &hashiVMSDK.VirtualHardDisk{
				Uri: common.StringPtr(fmt.Sprintf("[concat(parameters('storageAccountBlobEndpoint'),variables('vmStorageAccountContainerName'),'/',parameters('dataDiskName'),'-%d','.vhd')]", i+1)),
			}
			dataDisks[i].ManagedDisk = nil
		}
	}
	profile.DataDisks = &dataDisks
	return nil
}

func (s *TemplateBuilder) SetSpot(policy hashiVMSDK.VirtualMachineEvictionPolicyTypes, price float32) error {
	resource, err := s.getResourceByType(resourceVirtualMachine)
	if err != nil {
		return err
	}

	resource.Properties.Priority = common.StringPtr("Spot")
	resource.Properties.EvictionPolicy = &policy
	if price == 0 {
		price = -1
	}
	resource.Properties.BillingProfile = &BillingProfile{MaxPrice: price}
	return nil
}

func (s *TemplateBuilder) SetCustomData(customData string) error {
	resource, err := s.getResourceByType(resourceVirtualMachine)
	if err != nil {
		return err
	}

	profile := resource.Properties.OsProfile
	profile.CustomData = common.StringPtr(customData)

	return nil
}

func (s *TemplateBuilder) SetUserData(userData string) error {
	resource, err := s.getResourceByType(resourceVirtualMachine)
	if err != nil {
		return err
	}

	resource.Properties.UserData = common.StringPtr(userData)

	return nil
}

func (s *TemplateBuilder) SetVirtualNetwork(virtualNetworkResourceGroup, virtualNetworkName, subnetName string) error {
	s.setVariable("virtualNetworkResourceGroup", virtualNetworkResourceGroup)
	s.setVariable("virtualNetworkName", virtualNetworkName)
	s.setVariable("subnetName", subnetName)

	s.deleteResourceByType(resourceVirtualNetworks)
	s.deleteResourceByType(resourcePublicIPAddresses)
	resource, err := s.getResourceByType(resourceNetworkInterfaces)
	if err != nil {
		return err
	}

	s.deleteResourceDependency(resource, func(s string) bool {
		return strings.Contains(s, "Microsoft.Network/virtualNetworks") ||
			strings.Contains(s, "Microsoft.Network/publicIPAddresses")
	})

	(*resource.Properties.IPConfigurations)[0].Properties.PublicIPAddress = nil

	return nil
}

func (s *TemplateBuilder) SetPrivateVirtualNetworkWithPublicIp(virtualNetworkResourceGroup, virtualNetworkName, subnetName string) error {
	s.setVariable("virtualNetworkResourceGroup", virtualNetworkResourceGroup)
	s.setVariable("virtualNetworkName", virtualNetworkName)
	s.setVariable("subnetName", subnetName)

	s.deleteResourceByType(resourceVirtualNetworks)
	resource, err := s.getResourceByType(resourceNetworkInterfaces)
	if err != nil {
		return err
	}

	s.deleteResourceDependency(resource, func(s string) bool {
		return strings.Contains(s, "Microsoft.Network/virtualNetworks")
	})

	return nil
}

func (s *TemplateBuilder) SetNetworkSecurityGroup(ipAddresses []string, port int) error {
	nsgResource, dependency, resourceId := s.createNsgResource(ipAddresses, port)
	if err := s.addResource(nsgResource); err != nil {
		return err
	}

	vnetResource, err := s.getResourceByType(resourceVirtualNetworks)
	if err != nil {
		return err
	}
	s.deleteResourceByType(resourceVirtualNetworks)

	s.addResourceDependency(vnetResource, dependency)

	if vnetResource.Properties == nil || vnetResource.Properties.Subnets == nil || len(*vnetResource.Properties.Subnets) != 1 {
		return fmt.Errorf("template: could not find virtual network/subnet to add default network security group to")
	}
	subnet := ((*vnetResource.Properties.Subnets)[0])
	if subnet.Properties == nil {
		subnet.Properties = &hashiSubnetsSDK.SubnetPropertiesFormat{}
	}
	if subnet.Properties.NetworkSecurityGroup != nil {
		return fmt.Errorf("template: subnet already has an associated network security group")
	}
	subnet.Properties.NetworkSecurityGroup = &hashiSubnetsSDK.NetworkSecurityGroup{
		Id: common.StringPtr(resourceId),
	}

	err = s.addResource(vnetResource)
	if err != nil {
		return err
	}

	return nil
}

func (s *TemplateBuilder) SetTags(tags *map[string]string) error {
	if tags == nil || len(*tags) == 0 {
		return nil
	}

	for i := range s.template.Resources {
		s.template.Resources[i].Tags = tags
	}
	return nil
}

func (s *TemplateBuilder) SetBootDiagnostics(diagSTG string) error {

	resource, err := s.getResourceByType(resourceVirtualMachine)
	if err != nil {
		return err
	}

	t := true
	stg := fmt.Sprintf("https://%s.blob.core.windows.net", diagSTG)

	resource.Properties.DiagnosticsProfile.BootDiagnostics.Enabled = &t
	resource.Properties.DiagnosticsProfile.BootDiagnostics.StorageUri = &stg

	return nil
}

func (s *TemplateBuilder) SetLicenseType(licenseType string) error {
	resource, err := s.getResourceByType(resourceVirtualMachine)
	if err != nil {
		return err
	}

	resource.Properties.LicenseType = common.StringPtr(licenseType)

	return nil
}

func (s *TemplateBuilder) SetSecurityProfile(secureBootEnabled bool, vtpmEnabled bool, encryptionAtHost *bool) error {
	resource, err := s.getResourceByType(resourceVirtualMachine)
	if err != nil {
		return err
	}

	resource.Properties.SecurityProfile = &hashiVMSDK.SecurityProfile{}
	securityTrustedLaunch := hashiVMSDK.SecurityTypesTrustedLaunch
	if secureBootEnabled || vtpmEnabled {
		resource.Properties.SecurityProfile.UefiSettings = &hashiVMSDK.UefiSettings{}
		resource.Properties.SecurityProfile.SecurityType = &securityTrustedLaunch
		resource.Properties.SecurityProfile.UefiSettings.SecureBootEnabled = common.BoolPtr(secureBootEnabled)
		resource.Properties.SecurityProfile.UefiSettings.VTpmEnabled = common.BoolPtr(vtpmEnabled)
	}
	if encryptionAtHost != nil && *encryptionAtHost {
		resource.Properties.SecurityProfile.EncryptionAtHost = encryptionAtHost
	}

	return nil
}

func (s *TemplateBuilder) ClearOsProfile() error {
	resource, err := s.getResourceByType(resourceVirtualMachine)
	if err != nil {
		return err
	}
	resource.Properties.OsProfile = nil
	return nil
}

func (s *TemplateBuilder) ToJSON() (*string, error) {
	bs, err := json.MarshalIndent(s.template, jsonPrefix, jsonIndent)

	if err != nil {
		return nil, err
	}
	return common.StringPtr(string(bs)), err
}

func (s *TemplateBuilder) getResourceByType(t string) (*Resource, error) {
	for _, x := range s.template.Resources {
		if strings.EqualFold(*x.Type, t) {
			return x, nil
		}
	}

	return nil, fmt.Errorf("template: could not find a resource of type %s", t)
}

func (s *TemplateBuilder) setVariable(name string, value string) {
	(*s.template.Variables)[name] = value
}

func (s *TemplateBuilder) toResourceID(id, name string) string {
	return fmt.Sprintf("[resourceId(resourceGroup().name, '%s', '%s')]", id, name)
}

func (s *TemplateBuilder) toVariable(name string) string {
	return fmt.Sprintf("[variables('%s')]", name)
}

func (s *TemplateBuilder) addResource(newResource *Resource) error {
	for _, resource := range s.template.Resources {
		if *resource.Type == *newResource.Type {
			return fmt.Errorf("template: found an existing resource of type %s", *resource.Type)
		}
	}

	resources := append(s.template.Resources, newResource)
	s.template.Resources = resources
	return nil
}

func (s *TemplateBuilder) deleteResourceByType(resourceType string) {
	resources := make([]*Resource, 0)

	for _, resource := range s.template.Resources {
		if *resource.Type == resourceType {
			continue
		}
		resources = append(resources, resource)
	}

	s.template.Resources = resources
}

func (s *TemplateBuilder) addResourceDependency(resource *Resource, dep string) {
	if resource.DependsOn != nil {
		deps := append(*resource.DependsOn, dep)
		resource.DependsOn = &deps
	} else {
		resource.DependsOn = &[]string{dep}
	}
}

func (s *TemplateBuilder) deleteResourceDependency(resource *Resource, predicate func(string) bool) {
	deps := make([]string, 0)

	for _, dep := range *resource.DependsOn {
		if !predicate(dep) {
			deps = append(deps, dep)
		}
	}

	*resource.DependsOn = deps
}

func (s *TemplateBuilder) createNsgResource(srcIpAddresses []string, port int) (*Resource, string, string) {
	resource := &Resource{
		ApiVersion: common.StringPtr("[variables('networkApiVersion')]"),
		Name:       common.StringPtr("[parameters('nsgName')]"),
		Type:       common.StringPtr(resourceNetworkSecurityGroups),
		Location:   common.StringPtr("[variables('location')]"),
		Properties: &Properties{
			SecurityRules: &[]hashiSecurityRulesSDK.SecurityRule{
				{
					Name: common.StringPtr("AllowIPsToSshWinRMInbound"),
					Properties: &hashiSecurityRulesSDK.SecurityRulePropertiesFormat{
						Description:              common.StringPtr("Allow inbound traffic from specified IP addresses"),
						Protocol:                 hashiSecurityRulesSDK.SecurityRuleProtocolTcp,
						Priority:                 100,
						Access:                   hashiSecurityRulesSDK.SecurityRuleAccessAllow,
						Direction:                hashiSecurityRulesSDK.SecurityRuleDirectionInbound,
						SourceAddressPrefixes:    &srcIpAddresses,
						SourcePortRange:          common.StringPtr("*"),
						DestinationAddressPrefix: common.StringPtr("VirtualNetwork"),
						DestinationPortRange:     common.StringPtr(strconv.Itoa(port)),
					},
				},
			},
		},
	}

	dependency := fmt.Sprintf("[concat('%s/', parameters('nsgName'))]", resourceNetworkSecurityGroups)
	resourceId := fmt.Sprintf("[resourceId('%s', parameters('nsgName'))]", resourceNetworkSecurityGroups)

	return resource, dependency, resourceId
}

// See https://github.com/Azure/azure-quickstart-templates for a extensive list of templates.

// Template to deploy a KeyVault.
//
// This template is still hard-coded unlike the ARM templates used for VMs for
// a couple of reasons.
//
//  1. The SDK defines no types for a Key Vault
//  2. The Key Vault template is relatively simple, and is static.
const KeyVault = `{
  "$schema": "https://schema.management.azure.com/schemas/2019-04-01/deploymentTemplate.json#",
  "contentVersion": "1.0.0.0",
  "parameters": {
    "keyVaultName": {
      "type": "string"
    },
    "keyVaultSKU": {
      "type": "string"
    },
    "keyVaultSecretName": {
      "type": "string"
    },
    "keyVaultSecretValue": {
      "type": "securestring"
    },
    "objectId": {
      "type": "string"
    },
    "tenantId": {
      "type": "string"
    }
  },
  "variables": {
    "apiVersion": "2022-07-01",
    "location": "[resourceGroup().location]"
  },
  "resources": [
    {
      "type": "Microsoft.KeyVault/vaults",
      "apiVersion": "[variables('apiVersion')]",
      "name": "[parameters('keyVaultName')]",
      "location": "[variables('location')]",
      "properties": {
        "enabledForDeployment": "true",
        "enabledForTemplateDeployment": "true",
        "enableSoftDelete": "true",
        "tenantId": "[parameters('tenantId')]",
        "accessPolicies": [
          {
            "tenantId": "[parameters('tenantId')]",
            "objectId": "[parameters('objectId')]",
            "permissions": {
              "keys": ["all"],
              "secrets": ["all"]
            }
          }
        ],
        "sku": {
          "name": "[parameters('keyVaultSKU')]",
          "family": "A"
        }
      }
    },
    {
      "type": "Microsoft.KeyVault/vaults/secrets",
      "apiVersion": "[variables('apiVersion')]",
      "name": "[format('{0}/{1}', parameters('keyVaultName'), parameters('keyVaultSecretName'))]",
      "properties": {
        "value": "[parameters('keyVaultSecretValue')]"
      },
      "dependsOn": [
        "[resourceId('Microsoft.KeyVault/vaults/', parameters('keyVaultName'))]"
      ]
    }
  ]
}`

const BasicTemplate = `{
  "$schema": "https://schema.management.azure.com/schemas/2019-04-01/deploymentTemplate.json#",
  "contentVersion": "1.0.0.0",
  "parameters": {
    "adminUsername": {
      "type": "string"
    },
    "adminPassword": {
      "type": "securestring"
    },
    "dnsNameForPublicIP": {
      "type": "string"
    },
    "nicName": {
      "type": "string"
    },
    "osDiskName": {
      "type": "string"
    },
    "publicIPAddressName": {
      "type": "string"
    },
    "subnetName": {
      "type": "string"
    },
    "storageAccountBlobEndpoint": {
      "type": "string"
    },
    "virtualNetworkName": {
      "type": "string"
    },
    "nsgName": {
      "type": "string"
    },
    "vmSize": {
      "type": "string"
    },
    "vmName": {
      "type": "string"
    },
    "dataDiskName": {
      "type": "string"
    },
    "commandToExecute": {
      "type": "string"
    }
  },
  "variables": {
    "addressPrefix": "10.0.0.0/16",
    "computeApiVersion": "2023-03-01",
    "location": "[resourceGroup().location]",
    "networkApiVersion": "2023-04-01",
    "publicIPAddressType": "Dynamic",
    "sshKeyPath": "[concat('/home/',parameters('adminUsername'),'/.ssh/authorized_keys')]",
    "subnetName": "[parameters('subnetName')]",
    "subnetAddressPrefix": "10.0.0.0/24",
    "subnetRef": "[concat(variables('vnetID'),'/subnets/',variables('subnetName'))]",
    "virtualNetworkName": "[parameters('virtualNetworkName')]",
    "virtualNetworkResourceGroup": "[resourceGroup().name]",
    "vmStorageAccountContainerName": "images",
    "vnetID": "[resourceId(variables('virtualNetworkResourceGroup'), 'Microsoft.Network/virtualNetworks', variables('virtualNetworkName'))]"
  },
  "resources": [
    {
      "type": "Microsoft.Network/publicIPAddresses",
      "apiVersion": "[variables('networkApiVersion')]",
      "name": "[parameters('publicIPAddressName')]",
      "location": "[variables('location')]",
      "properties": {
        "publicIPAllocationMethod": "[variables('publicIPAddressType')]",
        "dnsSettings": {
          "domainNameLabel": "[parameters('dnsNameForPublicIP')]"
        }
      }
    },
    {
      "type": "Microsoft.Network/virtualNetworks",
      "apiVersion": "[variables('networkApiVersion')]",
      "name": "[variables('virtualNetworkName')]",
      "location": "[variables('location')]",
      "properties": {
        "addressSpace": {
          "addressPrefixes": [
            "[variables('addressPrefix')]"
          ]
        },
        "subnets": [
          {
            "name": "[variables('subnetName')]",
            "properties": {
              "addressPrefix": "[variables('subnetAddressPrefix')]"
            }
          }
        ]
      }
    },
    {
      "type": "Microsoft.Network/networkInterfaces",
      "apiVersion": "[variables('networkApiVersion')]",
      "name": "[parameters('nicName')]",
      "location": "[variables('location')]",
      "dependsOn": [
        "[concat('Microsoft.Network/publicIPAddresses/', parameters('publicIPAddressName'))]",
        "[concat('Microsoft.Network/virtualNetworks/', variables('virtualNetworkName'))]"
      ],
      "properties": {
        "ipConfigurations": [
          {
            "name": "ipconfig",
            "properties": {
              "privateIPAllocationMethod": "Dynamic",
              "publicIPAddress": {
                "id": "[resourceId('Microsoft.Network/publicIPAddresses', parameters('publicIPAddressName'))]"
              },
              "subnet": {
                "id": "[variables('subnetRef')]"
              }
            }
          }
        ]
      }
    },
    {
      "type": "Microsoft.Compute/virtualMachines",
      "apiVersion": "[variables('computeApiVersion')]",
      "name": "[parameters('vmName')]",
      "location": "[variables('location')]",
      "dependsOn": [
        "[concat('Microsoft.Network/networkInterfaces/', parameters('nicName'))]"
      ],
      "properties": {
        "hardwareProfile": {
          "vmSize": "[parameters('vmSize')]"
        },
        "osProfile": {
          "computerName": "[parameters('vmName')]",
          "adminUsername": "[parameters('adminUsername')]",
          "adminPassword": "[parameters('adminPassword')]"
        },
        "storageProfile": {
          "osDisk": {
            "name": "[parameters('osDiskName')]",
            "vhd": {
              "uri": "[concat(parameters('storageAccountBlobEndpoint'),variables('vmStorageAccountContainerName'),'/', parameters('osDiskName'),'.vhd')]"
            },
            "caching": "ReadWrite",
            "createOption": "FromImage"
          }
        },
        "networkProfile": {
          "networkInterfaces": [
            {
              "id": "[resourceId('Microsoft.Network/networkInterfaces', parameters('nicName'))]"
            }
          ]
        },
        "diagnosticsProfile": {
          "bootDiagnostics": {
            "enabled": false
          }
        }
      }
    },
    {
      "condition": "[not(empty(parameters('commandToExecute')))]",
      "type": "Microsoft.Compute/virtualMachines/extensions",
      "apiVersion": "[variables('computeApiVersion')]",
      "name": "[concat(parameters('vmName'), '/extension-customscript')]",
      "location": "[variables('location')]",
      "properties": {
        "publisher": "Microsoft.Compute",
        "type": "CustomScriptExtension",
        "typeHandlerVersion": "1.10",
        "autoUpgradeMinorVersion": true,
        "settings": {
          "commandToExecute": "[parameters('commandToExecute')]"
        }
      },
      "dependsOn": [
        "[resourceId('Microsoft.Compute/virtualMachines/', parameters('vmName'))]"
      ]
    }
  ]
}`
