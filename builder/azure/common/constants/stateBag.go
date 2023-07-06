// Copyright (c) HashiCorp, Inc.
// SPDX-License-Identifier: MPL-2.0

package constants

// complete flags
const (
	AuthorizedKey string = "authorizedKey"
	Certificate   string = "certificate"
	Error         string = "error"
	SSHHost       string = "sshHost"
	Thumbprint    string = "thumbprint"
	Ui            string = "ui"
)

// Default replica count for image versions in shared image gallery
const (
	SharedImageGalleryImageVersionDefaultMinReplicaCount int64 = 1
	SharedImageGalleryImageVersionDefaultMaxReplicaCount int64 = 100
)

const (
	ArmCaptureTemplate            string = "arm.CaptureTemplate"
	ArmComputeName                string = "arm.ComputeName"
	ArmImageParameters            string = "arm.ImageParameters"
	ArmCertificateUrl             string = "arm.CertificateUrl"
	ArmKeyVaultDeploymentName     string = "arm.KeyVaultDeploymentName"
	ArmDeploymentName             string = "arm.DeploymentName"
	ArmNicName                    string = "arm.NicName"
	ArmKeyVaultName               string = "arm.KeyVaultName"
	ArmLocation                   string = "arm.Location"
	ArmOSDiskVhd                  string = "arm.OSDiskVhd"
	ArmAdditionalDiskVhds         string = "arm.AdditionalDiskVhds"
	ArmPublicIPAddressName        string = "arm.PublicIPAddressName"
	ArmResourceGroupName          string = "arm.ResourceGroupName"
	ArmIsResourceGroupCreated     string = "arm.IsResourceGroupCreated"
	ArmDoubleResourceGroupNameSet string = "arm.DoubleResourceGroupNameSet"
	// TODO Replace ArmTags with ArmNewSDKTags
	// Temporary object, new SDK expects *map[string]string instead of map [string]*string
	ArmNewSDKTags         string = "arm.NewSDKTags"
	ArmStorageAccountName string = "arm.StorageAccountName"
	ArmTags               string = "arm.Tags"
	// TODO Replace ArmVirtualMachineCaptureParameters with ArmNewVirtualMachineCaptureParameters
	// Temporary object, this code is shared by all three builders so we need a new object for the diff type of capture parameters
	ArmNewVirtualMachineCaptureParameters                      string = "arm.NewVirtualMachineCaptureParameters"
	ArmVirtualMachineCaptureParameters                         string = "arm.VirtualMachineCaptureParameters"
	ArmIsExistingResourceGroup                                 string = "arm.IsExistingResourceGroup"
	ArmIsExistingKeyVault                                      string = "arm.IsExistingKeyVault"
	ArmIsManagedImage                                          string = "arm.IsManagedImage"
	ArmIsSIGImage                                              string = "arm.IsSIGImage"
	ArmManagedImageResourceGroupName                           string = "arm.ManagedImageResourceGroupName"
	ArmManagedImageName                                        string = "arm.ManagedImageName"
	ArmManagedImageSigPublishResourceGroup                     string = "arm.ManagedImageSigPublishResourceGroup"
	ArmManagedImageSharedGalleryName                           string = "arm.ManagedImageSharedGalleryName"
	ArmManagedImageSharedGalleryImageName                      string = "arm.ManagedImageSharedGalleryImageName"
	ArmManagedImageSharedGalleryImageVersion                   string = "arm.ManagedImageSharedGalleryImageVersion"
	ArmManagedImageSharedGalleryReplicationRegions             string = "arm.ManagedImageSharedGalleryReplicationRegions"
	ArmManagedImageSharedGalleryId                             string = "arm.ArmManagedImageSharedGalleryId"
	ArmManagedImageSharedGalleryImageVersionEndOfLifeDate      string = "arm.ArmManagedImageSharedGalleryImageVersionEndOfLifeDate"
	ArmManagedImageSharedGalleryImageVersionReplicaCount       string = "arm.ArmManagedImageSharedGalleryImageVersionReplicaCount"
	ArmManagedImageSharedGalleryImageVersionExcludeFromLatest  string = "arm.ArmManagedImageSharedGalleryImageVersionExcludeFromLatest"
	ArmManagedImageSharedGalleryImageVersionStorageAccountType string = "arm.ArmManagedImageSharedGalleryImageVersionStorageAccountType"
	ArmSharedImageGalleryDestinationSubscription               string = "arm.ArmSharedImageGalleryDestinationSubscription"
	ArmSharedImageGalleryDestinationSpecialized                string = "arm.ArmSharedImageGalleryDestinationSpecialized"
	ArmManagedImageSubscription                                string = "arm.ArmManagedImageSubscription"
	ArmAsyncResourceGroupDelete                                string = "arm.AsyncResourceGroupDelete"
	ArmManagedImageOSDiskSnapshotName                          string = "arm.ManagedImageOSDiskSnapshotName"
	ArmManagedImageDataDiskSnapshotPrefix                      string = "arm.ManagedImageDataDiskSnapshotPrefix"
	ArmKeepOSDisk                                              string = "arm.KeepOSDisk"
	ArmBuildDiskEncryptionSetId                                string = "arm.ArmBuildDiskEncryptionSetId"
	ArmSubscription                                            string = "arm.Subscription"

	DtlLabName                         string = "dtl.LabName"
)
