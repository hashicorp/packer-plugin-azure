// Copyright (c) HashiCorp, Inc.
// SPDX-License-Identifier: MPL-2.0

package template

import (
	hashiImagesSDK "github.com/hashicorp/go-azure-sdk/resource-manager/compute/2022-03-01/images"
	hashiVMSDK "github.com/hashicorp/go-azure-sdk/resource-manager/compute/2022-03-01/virtualmachines"

	hashiSecurityRulesSDK "github.com/hashicorp/go-azure-sdk/resource-manager/network/2023-09-01/securityrules"

	hashiPublicIPSDK "github.com/hashicorp/go-azure-sdk/resource-manager/network/2023-09-01/publicipaddresses"
	hashiSubnetsSDK "github.com/hashicorp/go-azure-sdk/resource-manager/network/2023-09-01/subnets"
	hashiVNETSDK "github.com/hashicorp/go-azure-sdk/resource-manager/network/2023-09-01/virtualnetworks"
	hashiNSGSDK "github.com/hashicorp/go-azure-sdk/resource-manager/network/2023-09-01/networksecuritygroups"
)

// Template
type Template struct {
	Schema         *string                `json:"$schema"`
	ContentVersion *string                `json:"contentVersion"`
	Parameters     *map[string]Parameters `json:"parameters"`
	Variables      *map[string]string     `json:"variables"`
	Resources      []*Resource            `json:"resources"`
}

// Template > Parameters
type Parameters struct {
	Type         *string `json:"type"`
	DefaultValue *string `json:"defaultValue,omitempty"`
}

// Template > Resource
type Resource struct {
	ApiVersion *string            `json:"apiVersion"`
	Name       *string            `json:"name"`
	Type       *string            `json:"type"`
	Location   *string            `json:"location,omitempty"`
	Sku        *Sku               `json:"sku,omitempty"`
	DependsOn  *[]string          `json:"dependsOn,omitempty"`
	Plan       *Plan              `json:"plan,omitempty"`
	Properties *Properties        `json:"properties,omitempty"`
	Tags       *map[string]string `json:"tags,omitempty"`
	Resources  *[]Resource        `json:"resources,omitempty"`
	Identity   *Identity          `json:"identity,omitempty"`
	Condition  *string            `json:"condition,omitempty"`
}

type Plan struct {
	Name          *string `json:"name"`
	Product       *string `json:"product"`
	Publisher     *string `json:"publisher"`
	PromotionCode *string `json:"promotionCode,omitempty"`
}

type ManagedDisk struct {
	StorageAccountType hashiVMSDK.StorageAccountTypes    `json:"storageAccountType,omitempty"`
	DiskEncryptionSet  *DiskEncryptionSetParameters      `json:"diskEncryptionSet,omitempty"`
	SecurityProfile    *hashiVMSDK.VMDiskSecurityProfile `json:"securityProfile,omitempty"`
	// ID - Resource Id
	ID *string `json:"id,omitempty"`
}

type DiskEncryptionSetParameters struct {
	// ID - Resource Id
	ID *string `json:"id,omitempty"`
}

type OSDiskUnion struct {
	OsType       hashiVMSDK.OperatingSystemTypes          `json:"osType,omitempty"`
	OsState      hashiImagesSDK.OperatingSystemStateTypes `json:"osState,omitempty"`
	BlobURI      *string                                  `json:"blobUri,omitempty"`
	Name         *string                                  `json:"name,omitempty"`
	Vhd          *hashiVMSDK.VirtualHardDisk              `json:"vhd,omitempty"`
	Image        *hashiVMSDK.VirtualHardDisk              `json:"image,omitempty"`
	Caching      hashiVMSDK.CachingTypes                  `json:"caching,omitempty"`
	CreateOption hashiVMSDK.DiskCreateOptionTypes         `json:"createOption,omitempty"`
	DiskSizeGB   *int32                                   `json:"diskSizeGB,omitempty"`
	ManagedDisk  *ManagedDisk                             `json:"managedDisk,omitempty"`
}

type DataDiskUnion struct {
	Lun          *int                             `json:"lun,omitempty"`
	BlobURI      *string                          `json:"blobUri,omitempty"`
	Name         *string                          `json:"name,omitempty"`
	Vhd          *hashiVMSDK.VirtualHardDisk      `json:"vhd,omitempty"`
	Image        *hashiVMSDK.VirtualHardDisk      `json:"image,omitempty"`
	Caching      hashiVMSDK.CachingTypes          `json:"caching,omitempty"`
	CreateOption hashiVMSDK.DiskCreateOptionTypes `json:"createOption,omitempty"`
	DiskSizeGB   *int32                           `json:"diskSizeGB,omitempty"`
	ManagedDisk  *ManagedDisk                     `json:"managedDisk,omitempty"`
}

// Union of the StorageProfile and ImageStorageProfile types.
type StorageProfileUnion struct {
	ImageReference *hashiVMSDK.ImageReference `json:"imageReference,omitempty"`
	OsDisk         *OSDiskUnion               `json:"osDisk,omitempty"`
	DataDisks      *[]DataDiskUnion           `json:"dataDisks,omitempty"`
}

type BillingProfile struct {
	MaxPrice float32 `json:"maxPrice,omitempty"`
}

// Template > Resource > Properties
type Properties struct {
	AccessPolicies               *[]AccessPolicies                             `json:"accessPolicies,omitempty"`
	AddressSpace                 *hashiVNETSDK.AddressSpace                    `json:"addressSpace,omitempty"`
	DiagnosticsProfile           *hashiVMSDK.DiagnosticsProfile                `json:"diagnosticsProfile,omitempty"`
	DNSSettings                  *hashiPublicIPSDK.PublicIPAddressDnsSettings  `json:"dnsSettings,omitempty"`
	EnabledForDeployment         *string                                       `json:"enabledForDeployment,omitempty"`
	EnabledForTemplateDeployment *string                                       `json:"enabledForTemplateDeployment,omitempty"`
	EnableSoftDelete             *string                                       `json:"enableSoftDelete,omitempty"`
	HardwareProfile              *hashiVMSDK.HardwareProfile                   `json:"hardwareProfile,omitempty"`
	IPConfigurations             *[]hashiPublicIPSDK.IPConfiguration           `json:"ipConfigurations,omitempty"`
	NetworkSecurityGroup		 *hashiNSGSDK.NetworkSecurityGroup 			   `json:"networkSecurityGroup,omitempty"`
	LicenseType                  *string                                       `json:"licenseType,omitempty"`
	NetworkProfile               *hashiVMSDK.NetworkProfile                    `json:"networkProfile,omitempty"`
	OsProfile                    *hashiVMSDK.OSProfile                         `json:"osProfile,omitempty"`
	PublicIPAllocatedMethod      *hashiPublicIPSDK.IPAllocationMethod          `json:"publicIPAllocationMethod,omitempty"`
	Sku                          *Sku                                          `json:"sku,omitempty"`
	UserData                     *string                                       `json:"userData,omitempty"`
	StorageProfile               *StorageProfileUnion                          `json:"storageProfile,omitempty"`
	SecurityProfile              *hashiVMSDK.SecurityProfile                   `json:"securityProfile,omitempty"`
	Subnets                      *[]hashiSubnetsSDK.Subnet                     `json:"subnets,omitempty"`
	SecurityRules                *[]hashiSecurityRulesSDK.SecurityRule         `json:"securityRules,omitempty"`
	TenantId                     *string                                       `json:"tenantId,omitempty"`
	Value                        *string                                       `json:"value,omitempty"`
	Priority                     *string                                       `json:"priority,omitempty"`
	EvictionPolicy               *hashiVMSDK.VirtualMachineEvictionPolicyTypes `json:"evictionPolicy,omitempty"`
	BillingProfile               *BillingProfile                               `json:"billingProfile,omitempty"`
	//CustomScript extension related properties
	Publisher               *string               `json:"publisher,omitempty"`
	Type                    *string               `json:"type,omitempty"`
	TypeHandlerVersion      *string               `json:"typeHandlerVersion,omitempty"`
	AutoUpgradeMinorVersion *bool                 `json:"autoUpgradeMinorVersion,omitempty"`
	Settings                *CustomScriptSettings `json:"settings,omitempty"`
	Attributes              *Attributes           `json:"attributes,omitempty"`
}

type CustomScriptSettings struct {
	CommandToExecute *string `json:"commandToExecute,omitempty"`
}

// Template > Resource > Identity
// The map values are simplified to struct{} since they are read-only and cannot be set
type Identity struct {
	Type                   *string             `json:"type,omitempty"`
	UserAssignedIdentities map[string]struct{} `json:"userAssignedIdentities,omitempty"`
}

type AccessPolicies struct {
	ObjectId    *string      `json:"objectId,omitempty"`
	TenantId    *string      `json:"tenantId,omitempty"`
	Permissions *Permissions `json:"permissions,omitempty"`
}

type Attributes struct {
	Exp int64 `json:"exp,omitempty"`
}

type Permissions struct {
	Keys    *[]string `json:"keys,omitempty"`
	Secrets *[]string `json:"secrets,omitempty"`
}

type Sku struct {
	Family *string `json:"family,omitempty"`
	Name   *string `json:"name,omitempty"`
	Tier   *string `json:"tier,omitempty"`
}
