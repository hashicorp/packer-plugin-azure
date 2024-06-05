// Copyright (c) HashiCorp, Inc.
// SPDX-License-Identifier: MPL-2.0

package arm

import (
	"context"
	"crypto/rsa"
	"errors"
	"fmt"
	"os"
	"runtime"
	"strings"

	"github.com/hashicorp/packer-plugin-azure/builder/azure/common/log"

	"github.com/hashicorp/go-azure-sdk/resource-manager/compute/2022-03-01/images"
	"github.com/hashicorp/go-azure-sdk/resource-manager/compute/2023-07-03/galleryimages"
	"github.com/hashicorp/go-azure-sdk/resource-manager/compute/2023-07-03/galleryimageversions"
	"github.com/hashicorp/go-azure-sdk/resource-manager/network/2023-09-01/publicipaddresses"
	"github.com/hashicorp/go-azure-sdk/resource-manager/storage/2023-01-01/storageaccounts"

	"github.com/hashicorp/go-azure-helpers/resourcemanager/commonids"
	"github.com/hashicorp/hcl/v2/hcldec"
	packerAzureCommon "github.com/hashicorp/packer-plugin-azure/builder/azure/common"
	commonclient "github.com/hashicorp/packer-plugin-azure/builder/azure/common/client"
	"github.com/hashicorp/packer-plugin-azure/builder/azure/common/constants"
	"github.com/hashicorp/packer-plugin-sdk/communicator"
	"github.com/hashicorp/packer-plugin-sdk/multistep"
	"github.com/hashicorp/packer-plugin-sdk/multistep/commonsteps"
	"github.com/hashicorp/packer-plugin-sdk/packer"
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
	// All requests on the new (non auto rest) base layer of the azure SDK require a context with a timeout for polling purposes
	ui.Say("Running builder ...")

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
	authOptions := commonclient.AzureAuthOptions{
		AuthType:           b.config.ClientConfig.AuthType(),
		ClientID:           b.config.ClientConfig.ClientID,
		ClientSecret:       b.config.ClientConfig.ClientSecret,
		ClientJWT:          b.config.ClientConfig.ClientJWT,
		ClientCertPath:     b.config.ClientConfig.ClientCertPath,
		ClientCertPassword: b.config.ClientConfig.ClientCertPassword,
		TenantID:           b.config.ClientConfig.TenantID,
		SubscriptionID:     b.config.ClientConfig.SubscriptionID,
		OidcRequestUrl:     b.config.ClientConfig.OidcRequestURL,
		OidcRequestToken:   b.config.ClientConfig.OidcRequestToken,
	}

	ui.Message("Creating Azure Resource Manager (ARM) client ...")
	azureClient, err := NewAzureClient(
		ctx,
		b.config.StorageAccount,
		b.config.ClientConfig.CloudEnvironment(),
		b.config.SharedGalleryTimeout,
		b.config.PollingDurationTimeout,
		authOptions,
	)

	ui.Message("ARM Client successfully created")
	if err != nil {
		return nil, err
	}

	resolver := newResourceResolver(azureClient)
	if err := resolver.Resolve(&b.config); err != nil {
		return nil, err
	}
	// All requests against go-azure-sdk require a polling duration
	builderPollingContext, builderCancel := context.WithTimeout(ctx, azureClient.PollingDuration)
	defer builderCancel()
	objectID := azureClient.ObjectID
	if b.config.ClientConfig.ObjectID == "" {
		b.config.ClientConfig.ObjectID = objectID
	} else {
		ui.Message("You have provided Object_ID which is no longer needed, Azure Packer ARM builder determines this automatically using the Azure Access Token")
	}

	if b.config.ClientConfig.ObjectID == "" && b.config.OSType != constants.Target_Linux {
		return nil, fmt.Errorf("could not determine the ObjectID for the user, which is required for Windows builds")
	}
	publicIPWarning := "On March 31, 2025, Azure will no longer allow the creation of public IP addresses with a Basic SKU, which is the default SKU type for this builder. You are seeing this warning because this build will create a new temporary public IP address using the `Basic` sku.  You can remove this warning by setting the standard public IP SKU (`public_ip_sku`) field to `Standard`. This builder will update its default value from `Basic` to `Standard` in a future release closer to Azure’s removal date. You can learn more about this change in the official Azure announcement https://azure.microsoft.com/en-us/updates/upgrade-to-standard-sku-public-ip-addresses-in-azure-by-30-september-2025-basic-sku-will-be-retired/."
	// If the user is bringing their own vnet, don't warn, if its an invalid SKU it will get disabled at a different date and Microsoft will send out warings
	if b.config.VirtualNetworkName == "" {
		if b.config.PublicIpSKU == "" || b.config.PublicIpSKU == string(publicipaddresses.PublicIPAddressSkuNameBasic) {
			ui.Message(publicIPWarning)
		}
	}

	if b.config.isManagedImage() {
		groupId := commonids.NewResourceGroupID(b.config.ClientConfig.SubscriptionID, b.config.ManagedImageResourceGroupName)
		_, err := azureClient.ResourceGroupsClient.Get(builderPollingContext, groupId)
		if err != nil {
			return nil, fmt.Errorf("Cannot locate the managed image resource group %s.", b.config.ManagedImageResourceGroupName)
		}

		// If a managed image already exists it cannot be overwritten.
		imageId := images.NewImageID(b.config.ClientConfig.SubscriptionID, b.config.ManagedImageResourceGroupName, b.config.ManagedImageName)
		_, err = azureClient.ImagesClient.Get(builderPollingContext, imageId, images.DefaultGetOperationOptions())
		if err == nil {
			if b.config.PackerForce {
				ui.Say(fmt.Sprintf("the managed image named %s already exists, but deleting it due to -force flag", b.config.ManagedImageName))
				err := azureClient.ImagesClient.DeleteThenPoll(builderPollingContext, imageId)
				if err != nil {
					return nil, fmt.Errorf("failed to delete the managed image named %s : %s", b.config.ManagedImageName, azureClient.LastError.Error())
				}
			} else {
				return nil, fmt.Errorf("the managed image named %s already exists in the resource group %s, use a different manage image name or use the -force option to automatically delete it.", b.config.ManagedImageName, b.config.ManagedImageResourceGroupName)
			}
		}
	}

	if b.config.BuildResourceGroupName != "" {
		buildGroupId := commonids.NewResourceGroupID(b.config.ClientConfig.SubscriptionID, b.config.BuildResourceGroupName)
		group, err := azureClient.ResourceGroupsClient.Get(builderPollingContext, buildGroupId)
		if err != nil {
			return nil, fmt.Errorf("Cannot locate the existing build resource resource group %s.", b.config.BuildResourceGroupName)
		}

		b.config.Location = group.Model.Location
	}

	b.config.validateLocationZoneResiliency(ui.Say)

	if b.config.StorageAccount != "" {
		account, err := b.getBlobAccount(builderPollingContext, azureClient, b.config.ClientConfig.SubscriptionID, b.config.ResourceGroupName, b.config.StorageAccount)
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
	if b.config.isPublishToSIG() {
		// Validate that Shared Gallery Image exists before publishing to SIG
		sigSubscriptionID := b.config.SharedGalleryDestination.SigDestinationSubscription
		if sigSubscriptionID == "" {
			sigSubscriptionID = b.stateBag.Get(constants.ArmSubscription).(string)
		}
		b.stateBag.Put(constants.ArmSharedImageGalleryDestinationSubscription, sigSubscriptionID)
		galleryId := galleryimages.NewGalleryImageID(sigSubscriptionID, b.config.SharedGalleryDestination.SigDestinationResourceGroup, b.config.SharedGalleryDestination.SigDestinationGalleryName, b.config.SharedGalleryDestination.SigDestinationImageName)
		_, err = azureClient.GalleryImagesClient.Get(builderPollingContext, galleryId)
		if err != nil {
			return nil, fmt.Errorf("the Shared Gallery Image '%s' to which to publish the managed image version to does not exist in the resource group '%s' or does not contain managed image '%s'", b.config.SharedGalleryDestination.SigDestinationGalleryName, b.config.SharedGalleryDestination.SigDestinationResourceGroup, b.config.SharedGalleryDestination.SigDestinationImageName)
		}
		// Check if a Image Version already exists for our target destination
		galleryImageVersionId := galleryimageversions.NewImageVersionID(sigSubscriptionID, b.config.SharedGalleryDestination.SigDestinationResourceGroup, b.config.SharedGalleryDestination.SigDestinationGalleryName, b.config.SharedGalleryDestination.SigDestinationImageName, b.config.SharedGalleryDestination.SigDestinationImageVersion)
		_, err := azureClient.GalleryImageVersionsClient.Get(builderPollingContext, galleryImageVersionId, galleryimageversions.DefaultGetOperationOptions())
		if err == nil {
			if b.config.PackerForce {
				ui.Say(fmt.Sprintf("a gallery image version for image name:version %s:%s already exists in gallery %s, but deleting it due to -force flag", b.config.SharedGalleryDestination.SigDestinationGalleryName, b.config.SharedGalleryDestination.SigDestinationImageVersion, b.config.SharedGalleryDestination.SigDestinationImageName))
				deleteImageContext, cancel := context.WithTimeout(ctx, azureClient.PollingDuration)
				defer cancel()
				err := azureClient.GalleryImageVersionsClient.DeleteThenPoll(deleteImageContext, galleryImageVersionId)
				if err != nil {
					return nil, fmt.Errorf("failed to delete gallery image version for image name:version %s:%s in gallery %s", b.config.SharedGalleryDestination.SigDestinationImageName, b.config.SharedGalleryDestination.SigDestinationImageVersion, b.config.SharedGalleryDestination.SigDestinationGalleryName)
				}

			} else {
				return nil, fmt.Errorf("a gallery image version for image name:version %s:%s already exists in gallery %s, use a different gallery image version or use the -force option to automatically delete it.", b.config.SharedGalleryDestination.SigDestinationImageName, b.config.SharedGalleryDestination.SigDestinationImageVersion, b.config.SharedGalleryDestination.SigDestinationGalleryName)
			}
		}

		buildLocation := normalizeAzureRegion(b.stateBag.Get(constants.ArmLocation).(string))
		if len(b.config.SharedGalleryDestination.SigDestinationTargetRegions) > 0 {
			normalizedRegions := make([]TargetRegion, 0, len(b.config.SharedGalleryDestination.SigDestinationTargetRegions))
			for _, tr := range b.config.SharedGalleryDestination.SigDestinationTargetRegions {
				tr.Name = normalizeAzureRegion(tr.Name)
				normalizedRegions = append(normalizedRegions, tr)
				if strings.EqualFold(tr.Name, buildLocation) && tr.ReplicaCount != 0 {
					// By default the global replica count takes precedence so lets update it to use
					// the define replica count from the target_region config for the build target_region.
					b.config.SharedGalleryImageVersionReplicaCount = tr.ReplicaCount
					b.stateBag.Put(constants.ArmManagedImageSharedGalleryImageVersionReplicaCount, tr.ReplicaCount)
				}
			}
			b.config.SharedGalleryDestination.SigDestinationTargetRegions = normalizedRegions
		}

		// Convert deprecated replication_regions to []TargetRegion
		if len(b.config.SharedGalleryDestination.SigDestinationReplicationRegions) > 0 {
			var foundMandatoryReplicationRegion bool
			normalizedRegions := make([]TargetRegion, 0, len(b.config.SharedGalleryDestination.SigDestinationReplicationRegions))
			for _, region := range b.config.SharedGalleryDestination.SigDestinationReplicationRegions {
				region := normalizeAzureRegion(region)
				if strings.EqualFold(region, buildLocation) {
					// backwards compatibility DiskEncryptionSetId was set on the global config not on the target region.
					// Users using target_region blocks are responsible for setting the DES within the block
					normalizedRegions = append(normalizedRegions, TargetRegion{Name: region, DiskEncryptionSetId: b.config.DiskEncryptionSetId})
					foundMandatoryReplicationRegion = true
					continue
				}
				normalizedRegions = append(normalizedRegions, TargetRegion{Name: region, ReplicaCount: b.config.SharedGalleryImageVersionReplicaCount})
			}
			// SIG requires that replication regions include the region in which the created image version resides
			if foundMandatoryReplicationRegion == false {
				normalizedRegions = append(normalizedRegions, TargetRegion{
					Name:                buildLocation,
					DiskEncryptionSetId: b.config.DiskEncryptionSetId,
					ReplicaCount:        b.config.SharedGalleryImageVersionReplicaCount,
				})
			}
			b.config.SharedGalleryDestination.SigDestinationTargetRegions = normalizedRegions
		}

		if len(b.config.SharedGalleryDestination.SigDestinationTargetRegions) == 0 {
			buildLocation := normalizeAzureRegion(b.stateBag.Get(constants.ArmLocation).(string))
			b.config.SharedGalleryDestination.SigDestinationTargetRegions = []TargetRegion{
				{
					Name:                buildLocation,
					DiskEncryptionSetId: b.config.DiskEncryptionSetId,
					//Default region replica count is set at the Gallery Level
					ReplicaCount: b.config.SharedGalleryImageVersionReplicaCount,
				},
			}
		}

		b.stateBag.Put(constants.ArmManagedImageSharedGalleryImageVersionReplicaCount, b.config.SharedGalleryImageVersionReplicaCount)
		b.stateBag.Put(constants.ArmSharedImageGalleryDestinationTargetRegions, b.config.SharedGalleryDestination.SigDestinationTargetRegions)
	}

	sourceImageSpecialized := false
	if b.config.SharedGallery.GalleryName != "" {
		client := azureClient.GalleryImagesClient
		id := galleryimages.NewGalleryImageID(b.config.SharedGallery.Subscription, b.config.SharedGallery.ResourceGroup, b.config.SharedGallery.GalleryName, b.config.SharedGallery.ImageName)
		galleryImage, err := client.Get(builderPollingContext, id)
		if err != nil {
			return nil, fmt.Errorf("the parent Shared Gallery Image '%s' from which to source the managed image version to does not exist in the resource group '%s' or does not contain managed image '%s'", b.config.SharedGallery.GalleryName, b.config.SharedGallery.ResourceGroup, b.config.SharedGallery.ImageName)
		}
		if galleryImage.Model == nil {
			return nil, commonclient.NullModelSDKErr
		}
		if galleryImage.Model.Properties.OsState == galleryimages.OperatingSystemStateTypesSpecialized {
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
				Host:      communicator.CommHost(b.config.Comm.SSHHost, constants.SSHHost),
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

		if b.config.SkipCreateBuildKeyVault {
			ui.Message("Skipping build keyvault creation...")
		} else if b.config.BuildKeyVaultName == "" {
			keyVaultDeploymentName := b.stateBag.Get(constants.ArmKeyVaultDeploymentName).(string)
			steps = append(steps,
				NewStepValidateTemplate(azureClient, ui, &b.config, keyVaultDeploymentName, GetCommunicatorSpecificKeyVaultDeployment),
				NewStepDeployTemplate(azureClient, ui, &b.config, keyVaultDeploymentName, GetCommunicatorSpecificKeyVaultDeployment, KeyVaultTemplate),
			)
		} else if b.config.Comm.Type == "winrm" {
			steps = append(steps, NewStepCertificateInKeyVault(azureClient, ui, &b.config, b.config.winrmCertificate, b.config.WinrmExpirationTime))
		} else {
			privateKey, err := ssh.ParseRawPrivateKey(b.config.Comm.SSHPrivateKey)
			if err != nil {
				return nil, err
			}
			pk, ok := privateKey.(*rsa.PrivateKey)
			if !ok {
				// https://learn.microsoft.com/en-us/azure/virtual-machines/windows/connect-ssh?tabs=azurecli#supported-ssh-key-formats
				return nil, errors.New("Provided private key must be in RSA format to use for SSH on Windows on Azure")
			}
			secret, err := b.config.formatCertificateForKeyVault(pk)
			if err != nil {
				return nil, err
			}

			packer.LogSecretFilter.Set(secret)

			steps = append(steps, NewStepCertificateInKeyVault(azureClient, ui, &b.config, secret, b.config.WinrmExpirationTime))
		}

		if !b.config.SkipCreateBuildKeyVault {
			steps = append(steps,
				NewStepGetCertificate(azureClient, ui),
				NewStepSetCertificate(&b.config, ui),
			)
		}
		steps = append(steps,
			NewStepValidateTemplate(azureClient, ui, &b.config, deploymentName, getVirtualMachineDeploymentFunction),
			NewStepDeployTemplate(azureClient, ui, &b.config, deploymentName, getVirtualMachineDeploymentFunction, VirtualMachineTemplate),
			NewStepGetIPAddress(azureClient, ui, endpointConnectType),
		)

		if b.config.Comm.Type == "ssh" {
			steps = append(steps,
				&communicator.StepConnectSSH{
					Config:    &b.config.Comm,
					Host:      communicator.CommHost(b.config.Comm.SSHHost, constants.SSHHost),
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
	return NewArtifact(
		b.stateBag.Get(constants.ArmBuildVMInternalId).(string),
		b.config.CaptureNamePrefix,
		b.config.CaptureContainerName,
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
	return b.config.PrivateVirtualNetworkWithPublicIp
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

func (b *Builder) getBlobAccount(ctx context.Context, client *AzureClient, subscriptionId string, resourceGroupName string, storageAccountName string) (*storageaccounts.StorageAccount, error) {
	id := commonids.NewStorageAccountID(subscriptionId, resourceGroupName, storageAccountName)
	account, err := client.StorageAccountsClient.GetProperties(ctx, id, storageaccounts.DefaultGetPropertiesOperationOptions())
	if err != nil {
		return nil, err
	}

	return account.Model, err
}

func (b *Builder) configureStateBag(stateBag multistep.StateBag) {
	stateBag.Put(constants.AuthorizedKey, b.config.sshAuthorizedKey)

	stateBag.Put(constants.ArmTags, b.config.AzureTags)
	stateBag.Put(constants.ArmComputeName, b.config.tmpComputeName)
	stateBag.Put(constants.ArmDeploymentName, b.config.tmpDeploymentName)

	if b.config.OSType == constants.Target_Windows && b.config.BuildKeyVaultName == "" {
		stateBag.Put(constants.ArmKeyVaultDeploymentName, fmt.Sprintf("kv%s", b.config.tmpDeploymentName))
	}

	stateBag.Put(constants.ArmKeyVaultName, b.config.tmpKeyVaultName)
	stateBag.Put(constants.ArmKeyVaultSecretName, DefaultSecretName)
	stateBag.Put(constants.ArmIsExistingKeyVault, false)
	if b.config.BuildKeyVaultName != "" {
		stateBag.Put(constants.ArmKeyVaultName, b.config.BuildKeyVaultName)
		b.config.tmpKeyVaultName = b.config.BuildKeyVaultName
		stateBag.Put(constants.ArmIsExistingKeyVault, true)
	}
	if b.config.BuildKeyVaultSecretName != "" {
		stateBag.Put(constants.ArmKeyVaultSecretName, b.config.BuildKeyVaultSecretName)
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
		stateBag.Put(constants.ArmSharedImageGalleryDestinationShallowReplication, b.config.SharedGalleryDestination.SigDestinationUseShallowReplicationMode)
		stateBag.Put(constants.ArmManagedImageSubscription, b.config.ClientConfig.SubscriptionID)
		stateBag.Put(constants.ArmManagedImageSharedGalleryImageVersionEndOfLifeDate, b.config.SharedGalleryImageVersionEndOfLifeDate)
		stateBag.Put(constants.ArmManagedImageSharedGalleryImageVersionReplicaCount, b.config.SharedGalleryImageVersionReplicaCount)
		stateBag.Put(constants.ArmManagedImageSharedGalleryImageVersionExcludeFromLatest, b.config.SharedGalleryImageVersionExcludeFromLatest)
		stateBag.Put(constants.ArmSharedImageGalleryDestinationConfidentialVMImageEncryptionType, b.config.SharedGalleryDestination.SigDestinationConfidentialVMImageEncryptionType)
	}
}

// Parameters that are only known at runtime after querying Azure.
func (b *Builder) setRuntimeParameters(stateBag multistep.StateBag) {
	stateBag.Put(constants.ArmLocation, b.config.Location)
}

func (b *Builder) setTemplateParameters(stateBag multistep.StateBag) {
	stateBag.Put(constants.ArmVirtualMachineCaptureParameters, b.config.toVirtualMachineCaptureParameters())
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
