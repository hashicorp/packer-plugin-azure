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
	ArmComputeName                                                    string = "arm.ComputeName"
	ArmImageParameters                                                string = "arm.ImageParameters"
	ArmCertificateUrl                                                 string = "arm.CertificateUrl"
	ArmKeyVaultDeploymentName                                         string = "arm.KeyVaultDeploymentName"
	ArmDeploymentName                                                 string = "arm.DeploymentName"
	ArmNicName                                                        string = "arm.NicName"
	ArmVnetName                                                       string = "arm.VnetName"
	ArmSubnetName                                                     string = "arm.SubnetName"
	ArmNsgName                                                        string = "arm.NsgName"
	ArmKeyVaultName                                                   string = "arm.KeyVaultName"
	ArmKeyVaultSecretName                                             string = "arm.KeyVaultSecretName"
	ArmLocation                                                       string = "arm.Location"
	ArmOSDiskUri                                                      string = "arm.OSDiskUri"
	ArmAdditionalDiskVhds                                             string = "arm.AdditionalDiskVhds"
	ArmPublicIPAddressName                                            string = "arm.PublicIPAddressName"
	ArmResourceGroupName                                              string = "arm.ResourceGroupName"
	ArmVnetResourceGroupName                                          string = "arm.VnetResourceGroupName"
	ArmIsResourceGroupCreated                                         string = "arm.IsResourceGroupCreated"
	ArmDoubleResourceGroupNameSet                                     string = "arm.DoubleResourceGroupNameSet"
	ArmStorageAccountName                                             string = "arm.StorageAccountName"
	ArmTags                                                           string = "arm.Tags"
	ArmVirtualMachineCaptureParameters                                string = "arm.VirtualMachineCaptureParameters"
	ArmIsExistingResourceGroup                                        string = "arm.IsExistingResourceGroup"
	ArmIsExistingKeyVault                                             string = "arm.IsExistingKeyVault"
	ArmIsVHDSaveToStorage                                             string = "arm.IsVHDSaveToStorage"
	ArmIsManagedImage                                                 string = "arm.IsManagedImage"
	ArmIsSIGImage                                                     string = "arm.IsSIGImage"
	ArmManagedImageResourceGroupName                                  string = "arm.ManagedImageResourceGroupName"
	ArmManagedImageName                                               string = "arm.ManagedImageName"
	ArmManagedImageSigPublishResourceGroup                            string = "arm.ManagedImageSigPublishResourceGroup"
	ArmManagedImageSharedGalleryName                                  string = "arm.ManagedImageSharedGalleryName"
	ArmManagedImageSharedGalleryImageName                             string = "arm.ManagedImageSharedGalleryImageName"
	ArmManagedImageSharedGalleryImageVersion                          string = "arm.ManagedImageSharedGalleryImageVersion"
	ArmManagedImageSharedGalleryReplicationRegions                    string = "arm.ManagedImageSharedGalleryReplicationRegions"
	ArmManagedImageSharedGalleryId                                    string = "arm.ArmManagedImageSharedGalleryId"
	ArmManagedImageSharedGalleryImageVersionEndOfLifeDate             string = "arm.ArmManagedImageSharedGalleryImageVersionEndOfLifeDate"
	ArmManagedImageSharedGalleryImageVersionReplicaCount              string = "arm.ArmManagedImageSharedGalleryImageVersionReplicaCount"
	ArmManagedImageSharedGalleryImageVersionExcludeFromLatest         string = "arm.ArmManagedImageSharedGalleryImageVersionExcludeFromLatest"
	ArmManagedImageSharedGalleryImageVersionStorageAccountType        string = "arm.ArmManagedImageSharedGalleryImageVersionStorageAccountType"
	ArmSharedImageGalleryDestinationSubscription                      string = "arm.ArmSharedImageGalleryDestinationSubscription"
	ArmSharedImageGalleryDestinationSpecialized                       string = "arm.ArmSharedImageGalleryDestinationSpecialized"
	ArmSharedImageGalleryDestinationShallowReplication                string = "arm.ArmSharedImageGalleryDestinationShallowReplication"
	ArmSharedImageGalleryDestinationTargetRegions                     string = "arm.SharedImageGalleryTargetRegions"
	ArmSharedImageGalleryDestinationConfidentialVMImageEncryptionType string = "arm.ArmSharedImageGalleryDestinationConfidentialVMImageEncryptionType"
	ArmManagedImageSubscription                                       string = "arm.ArmManagedImageSubscription"
	ArmAsyncResourceGroupDelete                                       string = "arm.AsyncResourceGroupDelete"
	ArmManagedImageOSDiskSnapshotName                                 string = "arm.ManagedImageOSDiskSnapshotName"
	ArmManagedImageDataDiskSnapshotPrefix                             string = "arm.ManagedImageDataDiskSnapshotPrefix"
	ArmKeepOSDisk                                                     string = "arm.KeepOSDisk"
	ArmBuildDiskEncryptionSetId                                       string = "arm.ArmBuildDiskEncryptionSetId"
	ArmSubscription                                                   string = "arm.Subscription"
	ArmBuildVMInternalId                                              string = "arm.BuildVMInternalId"
	DtlLabName                                                        string = "dtl.LabName"
)
