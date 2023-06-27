// Copyright (c) HashiCorp, Inc.
// SPDX-License-Identifier: MPL-2.0

package arm

import (
	"context"
	"crypto/rsa"
	"errors"
	"fmt"
	"log"
	"os"
	"runtime"
	"strings"
	"time"

	hashiImagesSDK "github.com/hashicorp/go-azure-sdk/resource-manager/compute/2022-03-01/images"
	hashiGalleryImagesSDK "github.com/hashicorp/go-azure-sdk/resource-manager/compute/2022-03-03/galleryimages"
	hashiStorageAccountsSDK "github.com/hashicorp/go-azure-sdk/resource-manager/storage/2022-09-01/storageaccounts"

	"github.com/hashicorp/go-azure-helpers/resourcemanager/commonids"
	"github.com/hashicorp/hcl/v2/hcldec"
	packerAzureCommon "github.com/hashicorp/packer-plugin-azure/builder/azure/common"
	"github.com/hashicorp/packer-plugin-azure/builder/azure/common/constants"
	"github.com/hashicorp/packer-plugin-azure/builder/azure/common/lin"
	"github.com/hashicorp/packer-plugin-sdk/communicator"
	"github.com/hashicorp/packer-plugin-sdk/multistep"
	"github.com/hashicorp/packer-plugin-sdk/multistep/commonsteps"
	packersdk "github.com/hashicorp/packer-plugin-sdk/packer"
	"github.com/hashicorp/packer-plugin-sdk/packerbuilderdata"
	"golang.org/x/crypto/ssh"
)

var ErrNoImage = errors.New("failed to find shared image gallery id in state")

type Builder struct {
	config   Config
	stateBag multistep.StateBag
	runner   multistep.Runner
}

const (
	DefaultSecretName = "packerKeyVaultSecret"
)

func (b *Builder) ConfigSpec() hcldec.ObjectSpec { return b.config.FlatMapstructure().HCL2Spec() }

func (b *Builder) Prepare(raws ...interface{}) ([]string, []string, error) {
	warnings, errs := b.config.Prepare(raws...)
	if errs != nil {
		return nil, warnings, errs
	}

	b.stateBag = new(multistep.BasicStateBag)
	b.configureStateBag(b.stateBag)
	b.setTemplateParameters(b.stateBag)
	b.setImageParameters(b.stateBag)

	generatedDataKeys := []string{"SourceImageName"}

	return generatedDataKeys, warnings, nil
}

func (b *Builder) Run(ctx context.Context, ui packersdk.Ui, hook packersdk.Hook) (packersdk.Artifact, error) {

	ui.Say("Running builder ...")

	ctx, cancel := context.WithTimeout(ctx, time.Minute*60)
	defer cancel()

	// FillParameters function captures authType and sets defaults.
	err := b.config.ClientConfig.FillParameters()
	if err != nil {
		return nil, err
	}

	//When running Packer on an Azure instance using Managed Identity, FillParameters will update SubscriptionID from the instance
	// so lets make sure to update our state bag with the valid subscriptionID.
	if b.config.isPublishToSIG() {
		b.stateBag.Put(constants.ArmManagedImageSubscription, b.config.ClientConfig.SubscriptionID)
	}

	b.stateBag.Put(constants.ArmSubscription, b.config.ClientConfig.SubscriptionID)

	log.Print(":: Configuration")
	packerAzureCommon.DumpConfig(&b.config, func(s string) { log.Print(s) })

	b.stateBag.Put("hook", hook)
	b.stateBag.Put(constants.Ui, ui)

	// Pass in relevant auth information for hashicorp/go-azure-sdk
	authOptions := NewSDKAuthOptions{
		AuthType:       b.config.ClientConfig.AuthType(),
		ClientID:       b.config.ClientConfig.ClientID,
		ClientSecret:   b.config.ClientConfig.ClientSecret,
		ClientJWT:      b.config.ClientConfig.ClientJWT,
		ClientCertPath: b.config.ClientConfig.ClientCertPath,
		TenantID:       b.config.ClientConfig.TenantID,
		SubscriptionID: b.config.ClientConfig.SubscriptionID,
	}

	ui.Message("Creating Azure Resource Manager (ARM) client ...")
	azureClient, objectID, err := NewAzureClient(
		ctx,
		(b.config.ResourceGroupName != "" || b.config.StorageAccount != ""),
		b.config.ClientConfig.NewCloudEnvironment(),
		b.config.SharedGalleryTimeout,
		b.config.PollingDurationTimeout,
		authOptions,
	)

	if err != nil {
		return nil, err
	}

	resolver := newResourceResolver(azureClient)
	if err := resolver.Resolve(&b.config); err != nil {
		return nil, err
	}

	if b.config.ClientConfig.ObjectID == "" {
		b.config.ClientConfig.ObjectID = *objectID
	} else {
		ui.Message("You have provided Object_ID which is no longer needed, Azure Packer ARM builder determines this automatically using the Azure Access Token")
	}

	if b.config.ClientConfig.ObjectID == "" && b.config.OSType != constants.Target_Linux {
		return nil, fmt.Errorf("could not determine the ObjectID for the user, which is required for Windows builds")
	}

	if b.config.isManagedImage() {
		groupId := commonids.NewResourceGroupID(b.config.ClientConfig.SubscriptionID, b.config.ManagedImageResourceGroupName)
		_, err := azureClient.ResourceGroupsClient.Get(ctx, groupId)
		if err != nil {
			return nil, fmt.Errorf("Cannot locate the managed image resource group %s.", b.config.ManagedImageResourceGroupName)
		}

		// If a managed image already exists it cannot be overwritten.
		imageId := hashiImagesSDK.NewImageID(b.config.ClientConfig.SubscriptionID, b.config.ManagedImageResourceGroupName, b.config.ManagedImageName)
		_, err = azureClient.ImagesClient.Get(ctx, imageId, hashiImagesSDK.DefaultGetOperationOptions())
		if err == nil {
			if b.config.PackerForce {
				ui.Say(fmt.Sprintf("the managed image named %s already exists, but deleting it due to -force flag", b.config.ManagedImageName))
				err := azureClient.ImagesClient.DeleteThenPoll(ctx, imageId)
				if err != nil {
					return nil, fmt.Errorf("failed to delete the managed image named %s : %s", b.config.ManagedImageName, azureClient.LastError.Error())
				}
			} else {
				return nil, fmt.Errorf("the managed image named %s already exists in the resource group %s, use the -force option to automatically delete it.", b.config.ManagedImageName, b.config.ManagedImageResourceGroupName)
			}
		}
	}

	if b.config.BuildResourceGroupName != "" {
		buildGroupId := commonids.NewResourceGroupID(b.config.ClientConfig.SubscriptionID, b.config.BuildResourceGroupName)
		group, err := azureClient.ResourceGroupsClient.Get(ctx, buildGroupId)
		if err != nil {
			return nil, fmt.Errorf("Cannot locate the existing build resource resource group %s.", b.config.BuildResourceGroupName)
		}

		b.config.Location = group.Model.Location
	}

	b.config.validateLocationZoneResiliency(ui.Say)

	if b.config.StorageAccount != "" {
		account, err := b.getBlobAccount(ctx, azureClient, b.config.ClientConfig.SubscriptionID, b.config.ResourceGroupName, b.config.StorageAccount)
		if err != nil {
			return nil, err
		}
		b.config.storageAccountBlobEndpoint = *account.Properties.PrimaryEndpoints.Blob
		if !equalLocation(account.Location, b.config.Location) {
			return nil, fmt.Errorf("The storage account is located in %s, but the build will take place in %s. The locations must be identical", account.Location, b.config.Location)
		}
	}

	endpointConnectType := PublicEndpoint
	if b.isPublicPrivateNetworkCommunication() && b.isPrivateNetworkCommunication() {
		endpointConnectType = PublicEndpointInPrivateNetwork
	} else if b.isPrivateNetworkCommunication() {
		endpointConnectType = PrivateEndpoint
	}

	b.setRuntimeParameters(b.stateBag)
	b.setTemplateParameters(b.stateBag)
	b.setImageParameters(b.stateBag)

	deploymentName := b.stateBag.Get(constants.ArmDeploymentName).(string)

	if b.config.DiskEncryptionSetId != "" {
		b.stateBag.Put(constants.ArmBuildDiskEncryptionSetId, b.config.DiskEncryptionSetId)
	}
	// Validate that Shared Gallery Image exists before publishing to SIG
	if b.config.isPublishToSIG() {
		sigSubscriptionID := b.config.SharedGalleryDestination.SigDestinationSubscription
		if sigSubscriptionID == "" {
			sigSubscriptionID = b.stateBag.Get(constants.ArmSubscription).(string)
		}
		b.stateBag.Put(constants.ArmSharedImageGalleryDestinationSubscription, sigSubscriptionID)
		galleryId := hashiGalleryImagesSDK.NewGalleryImageID(sigSubscriptionID, b.config.SharedGalleryDestination.SigDestinationResourceGroup, b.config.SharedGalleryDestination.SigDestinationGalleryName, b.config.SharedGalleryDestination.SigDestinationImageName)
		_, err = azureClient.GalleryImagesClient.Get(ctx, galleryId)
		if err != nil {
			return nil, fmt.Errorf("the Shared Gallery Image '%s' to which to publish the managed image version to does not exist in the resource group '%s' or does not contain managed image '%s'", b.config.SharedGalleryDestination.SigDestinationGalleryName, b.config.SharedGalleryDestination.SigDestinationResourceGroup, b.config.SharedGalleryDestination.SigDestinationImageName)
		}
		// SIG requires that replication regions include the region in which the Managed Image resides
		managedImageLocation := normalizeAzureRegion(b.stateBag.Get(constants.ArmLocation).(string))
		foundMandatoryReplicationRegion := false
		var normalizedReplicationRegions []string
		for _, region := range b.config.SharedGalleryDestination.SigDestinationReplicationRegions {
			// change region to lower-case and strip spaces
			normalizedRegion := normalizeAzureRegion(region)
			normalizedReplicationRegions = append(normalizedReplicationRegions, normalizedRegion)
			if strings.EqualFold(normalizedRegion, managedImageLocation) {
				foundMandatoryReplicationRegion = true
				continue
			}
		}
		if foundMandatoryReplicationRegion == false {
			b.config.SharedGalleryDestination.SigDestinationReplicationRegions = append(normalizedReplicationRegions, managedImageLocation)
		}
		b.stateBag.Put(constants.ArmManagedImageSharedGalleryReplicationRegions, b.config.SharedGalleryDestination.SigDestinationReplicationRegions)
	}
	sourceImageSpecialized := false
	if b.config.SharedGallery.GalleryName != "" {
		client := azureClient.GalleryImagesClient
		id := hashiGalleryImagesSDK.NewGalleryImageID(b.config.SharedGallery.Subscription, b.config.SharedGallery.ResourceGroup, b.config.SharedGallery.GalleryName, b.config.SharedGallery.ImageName)
		galleryImage, err := client.Get(ctx, id)
		if err != nil {
			return nil, fmt.Errorf("the parent Shared Gallery Image '%s' from which to source the managed image version to does not exist in the resource group '%s' or does not contain managed image '%s'", b.config.SharedGallery.GalleryName, b.config.SharedGallery.ResourceGroup, b.config.SharedGallery.ImageName)
		}
		if galleryImage.Model == nil {
			return nil, fmt.Errorf("SDK returned empty model for gallery image")
		}
		if galleryImage.Model.Properties.OsState == hashiGalleryImagesSDK.OperatingSystemStateTypesSpecialized {
			sourceImageSpecialized = true
		}
	}

	getVirtualMachineDeploymentFunction := GetVirtualMachineDeployment
	if sourceImageSpecialized {
		getVirtualMachineDeploymentFunction = GetSpecializedVirtualMachineDeployment
	}
	generatedData := &packerbuilderdata.GeneratedData{State: b.stateBag}
	var steps []multistep.Step
	if b.config.OSType == constants.Target_Linux {
		steps = []multistep.Step{
			NewStepGetSourceImageName(azureClient, ui, &b.config, generatedData),
			NewStepCreateResourceGroup(azureClient, ui),
			NewStepValidateTemplate(azureClient, ui, &b.config, deploymentName, getVirtualMachineDeploymentFunction),
			NewStepDeployTemplate(azureClient, ui, &b.config, deploymentName, getVirtualMachineDeploymentFunction, VirtualMachineTemplate),
			NewStepGetIPAddress(azureClient, ui, endpointConnectType),
			&communicator.StepConnectSSH{
				Config:    &b.config.Comm,
				Host:      lin.SSHHost,
				SSHConfig: b.config.Comm.SSHConfigFunc(),
			},
			&commonsteps.StepProvision{},
			&commonsteps.StepCleanupTempKeys{
				Comm: &b.config.Comm,
			},
			NewStepGetOSDisk(azureClient, ui),
			NewStepGetAdditionalDisks(azureClient, ui),
			NewStepPowerOffCompute(azureClient, ui),
			NewStepSnapshotOSDisk(azureClient, ui, &b.config),
			NewStepSnapshotDataDisks(azureClient, ui, &b.config),
		}
	} else if b.config.OSType == constants.Target_Windows {
		steps = []multistep.Step{
			NewStepGetSourceImageName(azureClient, ui, &b.config, generatedData),
			NewStepCreateResourceGroup(azureClient, ui),
		}
		if b.config.BuildKeyVaultName == "" {
			keyVaultDeploymentName := b.stateBag.Get(constants.ArmKeyVaultDeploymentName).(string)
			steps = append(steps,
				NewStepValidateTemplate(azureClient, ui, &b.config, keyVaultDeploymentName, GetCommunicatorSpecificKeyVaultDeployment),
				NewStepDeployTemplate(azureClient, ui, &b.config, keyVaultDeploymentName, GetCommunicatorSpecificKeyVaultDeployment, KeyVaultTemplate),
			)
		} else if b.config.Comm.Type == "winrm" {
			steps = append(steps, NewStepCertificateInKeyVault(azureClient, ui, &b.config, b.config.winrmCertificate))
		} else {
			privateKey, err := ssh.ParseRawPrivateKey(b.config.Comm.SSHPrivateKey)
			if err != nil {
				return nil, err
			}
			pk, ok := privateKey.(*rsa.PrivateKey)
			if !ok {
				//https://learn.microsoft.com/en-us/azure/virtual-machines/windows/connect-ssh?tabs=azurecli#supported-ssh-key-formats
				return nil, errors.New("Provided private key must be in RSA format to use for SSH on Windows on Azure")
			}
			secret, err := b.config.formatCertificateForKeyVault(pk)
			if err != nil {
				return nil, err
			}
			steps = append(steps, NewStepCertificateInKeyVault(azureClient, ui, &b.config, secret))
		}
		steps = append(steps,
			NewStepGetCertificate(azureClient, ui),
			NewStepSetCertificate(&b.config, ui),
			NewStepValidateTemplate(azureClient, ui, &b.config, deploymentName, getVirtualMachineDeploymentFunction),
			NewStepDeployTemplate(azureClient, ui, &b.config, deploymentName, getVirtualMachineDeploymentFunction, VirtualMachineTemplate),
			NewStepGetIPAddress(azureClient, ui, endpointConnectType),
		)

		if b.config.Comm.Type == "ssh" {
			steps = append(steps,
				&communicator.StepConnectSSH{
					Config:    &b.config.Comm,
					Host:      lin.SSHHost,
					SSHConfig: b.config.Comm.SSHConfigFunc(),
				},
			)
		} else {
			steps = append(steps,
				&communicator.StepConnectWinRM{
					Config: &b.config.Comm,
					Host: func(stateBag multistep.StateBag) (string, error) {
						return stateBag.Get(constants.SSHHost).(string), nil
					},
					WinRMConfig: func(multistep.StateBag) (*communicator.WinRMConfig, error) {
						return &communicator.WinRMConfig{
							Username: b.config.UserName,
							Password: b.config.Password,
						}, nil
					},
				},
			)
		}
		steps = append(steps,
			&commonsteps.StepProvision{},
			NewStepGetOSDisk(azureClient, ui),
			NewStepGetAdditionalDisks(azureClient, ui),
			NewStepPowerOffCompute(azureClient, ui),
			NewStepSnapshotOSDisk(azureClient, ui, &b.config),
			NewStepSnapshotDataDisks(azureClient, ui, &b.config),
		)
	} else {
		return nil, fmt.Errorf("Builder does not support the os_type '%s'", b.config.OSType)
	}

	captureSteps := b.config.CaptureSteps(
		ui.Say,
		NewStepCaptureImage(azureClient, ui),
		NewStepPublishToSharedImageGallery(azureClient, ui, &b.config),
	)

	steps = append(steps, captureSteps...)

	if b.config.PackerDebug {
		ui.Message(fmt.Sprintf("temp admin user: '%s'", b.config.UserName))
		ui.Message(fmt.Sprintf("temp admin password: '%s'", b.config.Password))

		if len(b.config.Comm.SSHPrivateKey) != 0 {
			debugKeyPath := fmt.Sprintf("%s-%s.pem", b.config.PackerBuildName, b.config.tmpComputeName)
			ui.Message(fmt.Sprintf("temp ssh key: %s", debugKeyPath))

			b.writeSSHPrivateKey(ui, debugKeyPath)
		}
	}

	b.runner = commonsteps.NewRunner(steps, b.config.PackerConfig, ui)
	b.runner.Run(ctx, b.stateBag)

	// Report any errors.
	if rawErr, ok := b.stateBag.GetOk(constants.Error); ok {
		return nil, rawErr.(error)
	}

	// If we were interrupted or cancelled, then just exit.
	if _, ok := b.stateBag.GetOk(multistep.StateCancelled); ok {
		return nil, errors.New("Build was cancelled.")
	}

	if _, ok := b.stateBag.GetOk(multistep.StateHalted); ok {
		return nil, errors.New("Build was halted.")
	}

	if b.config.SkipCreateImage {
		// NOTE(jkoelker) if the capture was skipped, then just return
		return nil, nil
	}

	stateData := map[string]interface{}{"generated_data": b.stateBag.Get("generated_data")}
	if b.config.isManagedImage() {
		managedImageID := fmt.Sprintf("/subscriptions/%s/resourceGroups/%s/providers/Microsoft.Compute/images/%s",
			b.config.ClientConfig.SubscriptionID, b.config.ManagedImageResourceGroupName, b.config.ManagedImageName)

		if b.config.isPublishToSIG() {
			return b.managedImageArtifactWithSIGAsDestination(managedImageID, stateData)
		}

		var osDiskUri string
		if b.stateBag.Get(constants.ArmKeepOSDisk).(bool) {
			osDiskUri = b.stateBag.Get(constants.ArmOSDiskUri).(string)
		}
		return NewManagedImageArtifact(b.config.OSType,
			b.config.ManagedImageResourceGroupName,
			b.config.ManagedImageName,
			b.config.Location,
			managedImageID,
			b.config.ManagedImageOSDiskSnapshotName,
			b.config.ManagedImageDataDiskSnapshotPrefix,
			stateData,
			osDiskUri,
		)
	}

	if b.config.isPublishToSIG() {
		return b.sharedImageArtifact(stateData)
	}
	ui.Say(b.config.storageAccountBlobEndpoint)
	ui.Say(fmt.Sprintf("%d", len(b.config.AdditionalDiskSize)))
	return NewArtifact(
		b.stateBag.Get(constants.ArmBuildVMInternalId).(string),
		b.config.storageAccountBlobEndpoint,
		b.config.StorageAccount,
		b.config.OSType,
		len(b.config.AdditionalDiskSize),
		stateData)
}

func (b *Builder) writeSSHPrivateKey(ui packersdk.Ui, debugKeyPath string) {
	f, err := os.Create(debugKeyPath)
	if err != nil {
		ui.Say(fmt.Sprintf("Error saving debug key: %s", err))
	}
	defer f.Close()

	// Write the key out
	if _, err := f.Write(b.config.Comm.SSHPrivateKey); err != nil {
		ui.Say(fmt.Sprintf("Error saving debug key: %s", err))
		return
	}

	// Chmod it so that it is SSH ready
	if runtime.GOOS != "windows" {
		if err := f.Chmod(0600); err != nil {
			ui.Say(fmt.Sprintf("Error setting permissions of debug key: %s", err))
		}
	}
}

func (b *Builder) isPublicPrivateNetworkCommunication() bool {
	return DefaultPrivateVirtualNetworkWithPublicIp != b.config.PrivateVirtualNetworkWithPublicIp
}

func (b *Builder) isPrivateNetworkCommunication() bool {
	return b.config.VirtualNetworkName != ""
}

func equalLocation(location1, location2 string) bool {
	return strings.EqualFold(canonicalizeLocation(location1), canonicalizeLocation(location2))
}

func canonicalizeLocation(location string) string {
	return strings.Replace(location, " ", "", -1)
}

func (b *Builder) getBlobAccount(ctx context.Context, client *AzureClient, subscriptionId string, resourceGroupName string, storageAccountName string) (*hashiStorageAccountsSDK.StorageAccount, error) {
	id := hashiStorageAccountsSDK.NewStorageAccountID(subscriptionId, resourceGroupName, storageAccountName)
	account, err := client.StorageAccountsClient.GetProperties(ctx, id, hashiStorageAccountsSDK.DefaultGetPropertiesOperationOptions())
	if err != nil {
		return nil, err
	}

	return account.Model, err
}

func (b *Builder) configureStateBag(stateBag multistep.StateBag) {
	stateBag.Put(constants.AuthorizedKey, b.config.sshAuthorizedKey)

	stateBag.Put(constants.ArmTags, packerAzureCommon.MapToAzureTags(b.config.AzureTags))
	stateBag.Put(constants.ArmNewSDKTags, b.config.AzureTags)
	stateBag.Put(constants.ArmComputeName, b.config.tmpComputeName)
	stateBag.Put(constants.ArmDeploymentName, b.config.tmpDeploymentName)

	if b.config.OSType == constants.Target_Windows && b.config.BuildKeyVaultName == "" {
		stateBag.Put(constants.ArmKeyVaultDeploymentName, fmt.Sprintf("kv%s", b.config.tmpDeploymentName))
	}

	stateBag.Put(constants.ArmKeyVaultName, b.config.tmpKeyVaultName)
	stateBag.Put(constants.ArmIsExistingKeyVault, false)
	if b.config.BuildKeyVaultName != "" {
		stateBag.Put(constants.ArmKeyVaultName, b.config.BuildKeyVaultName)
		b.config.tmpKeyVaultName = b.config.BuildKeyVaultName
		stateBag.Put(constants.ArmIsExistingKeyVault, true)
	}

	stateBag.Put(constants.ArmNicName, b.config.tmpNicName)
	stateBag.Put(constants.ArmPublicIPAddressName, b.config.tmpPublicIPAddressName)
	stateBag.Put(constants.ArmResourceGroupName, b.config.BuildResourceGroupName)
	stateBag.Put(constants.ArmIsExistingResourceGroup, true)

	if b.config.tmpResourceGroupName != "" {
		stateBag.Put(constants.ArmResourceGroupName, b.config.tmpResourceGroupName)
		stateBag.Put(constants.ArmIsExistingResourceGroup, false)

		if b.config.BuildResourceGroupName != "" {
			stateBag.Put(constants.ArmDoubleResourceGroupNameSet, true)
		}
	}

	stateBag.Put(constants.ArmStorageAccountName, b.config.StorageAccount)
	stateBag.Put(constants.ArmIsManagedImage, b.config.isManagedImage())
	stateBag.Put(constants.ArmManagedImageResourceGroupName, b.config.ManagedImageResourceGroupName)
	stateBag.Put(constants.ArmManagedImageName, b.config.ManagedImageName)
	stateBag.Put(constants.ArmManagedImageOSDiskSnapshotName, b.config.ManagedImageOSDiskSnapshotName)
	stateBag.Put(constants.ArmManagedImageDataDiskSnapshotPrefix, b.config.ManagedImageDataDiskSnapshotPrefix)
	stateBag.Put(constants.ArmAsyncResourceGroupDelete, b.config.AsyncResourceGroupDelete)
	stateBag.Put(constants.ArmKeepOSDisk, b.config.KeepOSDisk)

	stateBag.Put(constants.ArmIsSIGImage, b.config.isPublishToSIG())
	// Set Specialized as false so that we can pull it from the state later even if we're not publishing to SIG
	stateBag.Put(constants.ArmSharedImageGalleryDestinationSpecialized, false)
	if b.config.isPublishToSIG() {
		stateBag.Put(constants.ArmManagedImageSigPublishResourceGroup, b.config.SharedGalleryDestination.SigDestinationResourceGroup)
		stateBag.Put(constants.ArmManagedImageSharedGalleryName, b.config.SharedGalleryDestination.SigDestinationGalleryName)
		stateBag.Put(constants.ArmManagedImageSharedGalleryImageName, b.config.SharedGalleryDestination.SigDestinationImageName)
		stateBag.Put(constants.ArmManagedImageSharedGalleryImageVersion, b.config.SharedGalleryDestination.SigDestinationImageVersion)
		stateBag.Put(constants.ArmManagedImageSharedGalleryImageVersionStorageAccountType, b.config.SharedGalleryDestination.SigDestinationStorageAccountType)
		stateBag.Put(constants.ArmSharedImageGalleryDestinationSpecialized, b.config.SharedGalleryDestination.SigDestinationSpecialized)
		stateBag.Put(constants.ArmManagedImageSubscription, b.config.ClientConfig.SubscriptionID)
		stateBag.Put(constants.ArmManagedImageSharedGalleryImageVersionEndOfLifeDate, b.config.SharedGalleryImageVersionEndOfLifeDate)
		stateBag.Put(constants.ArmManagedImageSharedGalleryImageVersionReplicaCount, b.config.SharedGalleryImageVersionReplicaCount)
		stateBag.Put(constants.ArmManagedImageSharedGalleryImageVersionExcludeFromLatest, b.config.SharedGalleryImageVersionExcludeFromLatest)
	}
}

// Parameters that are only known at runtime after querying Azure.
func (b *Builder) setRuntimeParameters(stateBag multistep.StateBag) {
	stateBag.Put(constants.ArmLocation, b.config.Location)
}

func (b *Builder) setTemplateParameters(stateBag multistep.StateBag) {
	stateBag.Put(constants.ArmNewVirtualMachineCaptureParameters, b.config.toVirtualMachineCaptureParameters())
}

func (b *Builder) setImageParameters(stateBag multistep.StateBag) {
	stateBag.Put(constants.ArmImageParameters, b.config.toImageParameters())
}

func normalizeAzureRegion(name string) string {
	return strings.ToLower(strings.Replace(name, " ", "", -1))
}

func (b *Builder) managedImageArtifactWithSIGAsDestination(managedImageID string, stateData map[string]interface{}) (*Artifact, error) {

	sigDestinationStateKeys := []string{
		constants.ArmManagedImageSigPublishResourceGroup,
		constants.ArmManagedImageSharedGalleryName,
		constants.ArmManagedImageSharedGalleryImageName,
		constants.ArmManagedImageSharedGalleryImageVersion,
		constants.ArmManagedImageSharedGalleryReplicationRegions,
	}

	for _, key := range sigDestinationStateKeys {
		v, ok := b.stateBag.GetOk(key)
		if !ok {
			continue
		}
		stateData[key] = v
	}

	destinationSharedImageGalleryId := ""
	if galleryID, ok := b.stateBag.GetOk(constants.ArmManagedImageSharedGalleryId); ok {
		destinationSharedImageGalleryId = galleryID.(string)
	} else {
		return nil, ErrNoImage
	}

	return NewManagedImageArtifactWithSIGAsDestination(b.config.OSType,
		b.config.ManagedImageResourceGroupName,
		b.config.ManagedImageName,
		b.config.Location,
		managedImageID,
		b.config.ManagedImageOSDiskSnapshotName,
		b.config.ManagedImageDataDiskSnapshotPrefix,
		destinationSharedImageGalleryId,
		stateData)
}

func (b *Builder) sharedImageArtifact(stateData map[string]interface{}) (*Artifact, error) {

	sigDestinationStateKeys := []string{
		constants.ArmManagedImageSigPublishResourceGroup,
		constants.ArmManagedImageSharedGalleryName,
		constants.ArmManagedImageSharedGalleryImageName,
		constants.ArmManagedImageSharedGalleryImageVersion,
		constants.ArmManagedImageSharedGalleryReplicationRegions,
	}

	for _, key := range sigDestinationStateKeys {
		v, ok := b.stateBag.GetOk(key)
		if !ok {
			continue
		}
		stateData[key] = v
	}

	destinationSharedImageGalleryId := ""
	if galleryID, ok := b.stateBag.GetOk(constants.ArmManagedImageSharedGalleryId); ok {
		destinationSharedImageGalleryId = galleryID.(string)
	} else {
		return nil, ErrNoImage
	}

	return NewSharedImageArtifact(b.config.OSType, destinationSharedImageGalleryId, b.config.Location, stateData)
}
