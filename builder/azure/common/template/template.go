// Copyright (c) HashiCorp, Inc.
// SPDX-License-Identifier: MPL-2.0

package template

import (
	"github.com/hashicorp/go-azure-sdk/resource-manager/compute/2022-03-01/images"
	"github.com/hashicorp/go-azure-sdk/resource-manager/compute/2024-03-01/virtualmachines"

	"github.com/hashicorp/go-azure-sdk/resource-manager/network/2023-09-01/securityrules"

	"github.com/hashicorp/go-azure-sdk/resource-manager/network/2023-09-01/publicipaddresses"
	"github.com/hashicorp/go-azure-sdk/resource-manager/network/2023-09-01/subnets"
	"github.com/hashicorp/go-azure-sdk/resource-manager/network/2023-09-01/virtualnetworks"
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
	StorageAccountType virtualmachines.StorageAccountTypes    `json:"storageAccountType,omitempty"`
	DiskEncryptionSet  *DiskEncryptionSetParameters           `json:"diskEncryptionSet,omitempty"`
	SecurityProfile    *virtualmachines.VMDiskSecurityProfile `json:"securityProfile,omitempty"`
	// ID - Resource Id
	ID *string `json:"id,omitempty"`
}

type DiskEncryptionSetParameters struct {
	// ID - Resource Id
	ID *string `json:"id,omitempty"`
}

type OSDiskUnion struct {
	OsType       virtualmachines.OperatingSystemTypes  `json:"osType,omitempty"`
	OsState      images.OperatingSystemStateTypes      `json:"osState,omitempty"`
	BlobURI      *string                               `json:"blobUri,omitempty"`
	Name         *string                               `json:"name,omitempty"`
	Vhd          *virtualmachines.VirtualHardDisk      `json:"vhd,omitempty"`
	Image        *virtualmachines.VirtualHardDisk      `json:"image,omitempty"`
	Caching      virtualmachines.CachingTypes          `json:"caching,omitempty"`
	CreateOption virtualmachines.DiskCreateOptionTypes `json:"createOption,omitempty"`
	DiskSizeGB   *int32                                `json:"diskSizeGB,omitempty"`
	ManagedDisk  *ManagedDisk                          `json:"managedDisk,omitempty"`
}

type DataDiskUnion struct {
	Lun          *int                                  `json:"lun,omitempty"`
	BlobURI      *string                               `json:"blobUri,omitempty"`
	Name         *string                               `json:"name,omitempty"`
	Vhd          *virtualmachines.VirtualHardDisk      `json:"vhd,omitempty"`
	Image        *virtualmachines.VirtualHardDisk      `json:"image,omitempty"`
	Caching      virtualmachines.CachingTypes          `json:"caching,omitempty"`
	CreateOption virtualmachines.DiskCreateOptionTypes `json:"createOption,omitempty"`
	DiskSizeGB   *int32                                `json:"diskSizeGB,omitempty"`
	ManagedDisk  *ManagedDisk                          `json:"managedDisk,omitempty"`
}

// Union of the StorageProfile and ImageStorageProfile types.
type StorageProfileUnion struct {
	ImageReference *virtualmachines.ImageReference `json:"imageReference,omitempty"`
	OsDisk         *OSDiskUnion                    `json:"osDisk,omitempty"`
	DataDisks      *[]DataDiskUnion                `json:"dataDisks,omitempty"`
}

type BillingProfile struct {
	MaxPrice float32 `json:"maxPrice,omitempty"`
}

// Template > Resource > Properties
type Properties struct {
	AccessPolicies               *[]AccessPolicies                                  `json:"accessPolicies,omitempty"`
	AddressSpace                 *virtualnetworks.AddressSpace                      `json:"addressSpace,omitempty"`
	DiagnosticsProfile           *virtualmachines.DiagnosticsProfile                `json:"diagnosticsProfile,omitempty"`
	DNSSettings                  *publicipaddresses.PublicIPAddressDnsSettings      `json:"dnsSettings,omitempty"`
	EnabledForDeployment         *string                                            `json:"enabledForDeployment,omitempty"`
	EnabledForTemplateDeployment *string                                            `json:"enabledForTemplateDeployment,omitempty"`
	EnableSoftDelete             *string                                            `json:"enableSoftDelete,omitempty"`
	HardwareProfile              *virtualmachines.HardwareProfile                   `json:"hardwareProfile,omitempty"`
	IPConfigurations             *[]publicipaddresses.IPConfiguration               `json:"ipConfigurations,omitempty"`
	LicenseType                  *string                                            `json:"licenseType,omitempty"`
	NetworkProfile               *virtualmachines.NetworkProfile                    `json:"networkProfile,omitempty"`
	OsProfile                    *virtualmachines.OSProfile                         `json:"osProfile,omitempty"`
	PublicIPAllocatedMethod      *publicipaddresses.IPAllocationMethod              `json:"publicIPAllocationMethod,omitempty"`
	Sku                          *Sku                                               `json:"sku,omitempty"`
	UserData                     *string                                            `json:"userData,omitempty"`
	StorageProfile               *StorageProfileUnion                               `json:"storageProfile,omitempty"`
	SecurityProfile              *virtualmachines.SecurityProfile                   `json:"securityProfile,omitempty"`
	Subnets                      *[]subnets.Subnet                                  `json:"subnets,omitempty"`
	SecurityRules                *[]securityrules.SecurityRule                      `json:"securityRules,omitempty"`
	TenantId                     *string                                            `json:"tenantId,omitempty"`
	Value                        *string                                            `json:"value,omitempty"`
	Priority                     *string                                            `json:"priority,omitempty"`
	EvictionPolicy               *virtualmachines.VirtualMachineEvictionPolicyTypes `json:"evictionPolicy,omitempty"`
	BillingProfile               *BillingProfile                                    `json:"billingProfile,omitempty"`
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
}
