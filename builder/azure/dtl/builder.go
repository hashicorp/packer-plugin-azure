// Copyright (c) HashiCorp, Inc.
// SPDX-License-Identifier: MPL-2.0

package dtl

import (
	"context"
	"errors"
	"fmt"
	"log"
	"os"
	"runtime"
	"strings"

	"github.com/Azure/go-autorest/autorest/adal"
	hashiGalleryImagesSDK "github.com/hashicorp/go-azure-sdk/resource-manager/compute/2022-03-03/galleryimages"
	hashiDTLCustomImagesSDK "github.com/hashicorp/go-azure-sdk/resource-manager/devtestlab/2018-09-15/customimages"
	hashiDTLLabsSDK "github.com/hashicorp/go-azure-sdk/resource-manager/devtestlab/2018-09-15/labs"
	hashiDTLVNETSDK "github.com/hashicorp/go-azure-sdk/resource-manager/devtestlab/2018-09-15/virtualnetworks"
	"github.com/hashicorp/hcl/v2/hcldec"
	packerAzureCommon "github.com/hashicorp/packer-plugin-azure/builder/azure/common"
	"github.com/hashicorp/packer-plugin-azure/builder/azure/common/constants"
	"github.com/hashicorp/packer-plugin-azure/builder/azure/common/lin"
	"github.com/hashicorp/packer-plugin-sdk/communicator"
	"github.com/hashicorp/packer-plugin-sdk/multistep"
	"github.com/hashicorp/packer-plugin-sdk/multistep/commonsteps"
	packersdk "github.com/hashicorp/packer-plugin-sdk/packer"
)

type Builder struct {
	config   Config
	stateBag multistep.StateBag
	runner   multistep.Runner
}

const (
	DefaultSasBlobContainer = "system/Microsoft.Compute"
	DefaultSecretName       = "packerKeyVaultSecret"
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

	return nil, warnings, errs
}

func (b *Builder) Run(ctx context.Context, ui packersdk.Ui, hook packersdk.Hook) (packersdk.Artifact, error) {

	ui.Say("Running builder ...")

	ctx, cancel := context.WithCancel(ctx)
	defer cancel()

	// FillParameters function captures authType and sets defaults.
	err := b.config.ClientConfig.FillParameters()
	if err != nil {
		return nil, err
	}

	log.Print(":: Configuration")
	packerAzureCommon.DumpConfig(b.config, func(s string) { log.Print(s) })

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
	ui.Message("Creating Azure DevTestLab (DTL) client ...")
	azureClient, objectId, err := NewAzureClient(
		ctx,
		b.config.ClientConfig.SubscriptionID,
		b.config.ClientConfig.NewCloudEnvironment(),
		b.config.SharedGalleryTimeout,
		b.config.CustomImageCaptureTimeout,
		b.config.PollingDurationTimeout,
		authOptions)

	if err != nil {
		return nil, err
	}

	resolver := newResourceResolver(azureClient)
	if err := resolver.Resolve(&b.config); err != nil {
		return nil, err
	}
	if b.config.ClientConfig.ObjectID == "" {
		b.config.ClientConfig.ObjectID = *objectId
	} else {
		ui.Message("You have provided Object_ID which is no longer needed, azure packer builder determines this dynamically from the authentication token")
	}

	if b.config.ClientConfig.ObjectID == "" && b.config.OSType != constants.Target_Linux {
		return nil, fmt.Errorf("could not determine the ObjectID for the user, which is required for Windows builds")
	}

	if b.config.isManagedImage() {
		// If a managed image already exists it cannot be overwritten. We need to delete it if the user has provided  -force flag
		customImageResourceId := hashiDTLCustomImagesSDK.NewCustomImageID(b.config.ClientConfig.SubscriptionID, b.config.ManagedImageResourceGroupName, b.config.LabName, b.config.ManagedImageName)
		_, err = azureClient.DtlMetaClient.CustomImages.Get(ctx, customImageResourceId, hashiDTLCustomImagesSDK.DefaultGetOperationOptions())

		if err == nil {
			if b.config.PackerForce {
				ui.Say(fmt.Sprintf("the managed image named %s already exists, but deleting it due to -force flag", b.config.ManagedImageName))
				err := azureClient.DtlMetaClient.CustomImages.DeleteThenPoll(ctx, customImageResourceId)
				if err != nil {
					return nil, fmt.Errorf("failed to delete the managed image named %s : %s", b.config.ManagedImageName, azureClient.LastError.Error())
				}
			} else {
				return nil, fmt.Errorf("the managed image named %s already exists in the resource group %s, use the -force option to automatically delete it.", b.config.ManagedImageName, b.config.ManagedImageResourceGroupName)
			}
		}

	} else {
		// User is not using Managed Images to build, warning message here that this path is being deprecated
		ui.Error("Warning: You are using Azure Packer Builder to create VHDs which is being deprecated, consider using Managed Images. Learn more https://www.packer.io/docs/builders/azure/arm#azure-arm-builder-specific-options")
	}

	b.config.validateLocationZoneResiliency(ui.Say)

	b.setRuntimeParameters(b.stateBag)
	b.setTemplateParameters(b.stateBag)
	var steps []multistep.Step

	deploymentName := b.stateBag.Get(constants.ArmDeploymentName).(string)

	b.stateBag.Put(constants.DtlLabName, b.config.LabName)
	// For Managed Images, validate that Shared Gallery Image exists before publishing to SIG
	if b.config.isManagedImage() && b.config.SharedGalleryDestination.SigDestinationGalleryName != "" {
		sigSubscriptionID := b.stateBag.Get(constants.ArmSubscription).(string)
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

	// Find the lab location
	labResourceId := hashiDTLLabsSDK.NewLabID(b.config.ClientConfig.SubscriptionID, b.config.LabResourceGroupName, b.config.LabName)
	lab, err := azureClient.DtlMetaClient.Labs.Get(ctx, labResourceId, hashiDTLLabsSDK.DefaultGetOperationOptions())
	if err != nil {
		return nil, fmt.Errorf("Unable to fetch the Lab %s information in %s resource group", b.config.LabName, b.config.LabResourceGroupName)
	}
	if lab.Model == nil {

	}
	b.config.Location = *lab.Model.Location

	if b.config.LabVirtualNetworkName == "" || b.config.LabSubnetName == "" {
		virtualNetwork, subnet, err := b.getSubnetInformation(ctx, ui, *azureClient)

		if err != nil {
			return nil, err
		}
		b.config.LabVirtualNetworkName = *virtualNetwork
		b.config.LabSubnetName = *subnet

		ui.Message(fmt.Sprintf("No lab network information provided. Using %s Virtual network and %s subnet for Virtual Machine creation", b.config.LabVirtualNetworkName, b.config.LabSubnetName))
	}

	if b.config.OSType == constants.Target_Linux {
		steps = []multistep.Step{
			NewStepDeployTemplate(azureClient, ui, &b.config, deploymentName, GetVirtualMachineDeployment),
			&communicator.StepConnectSSH{
				Config:    &b.config.Comm,
				Host:      lin.SSHHost,
				SSHConfig: b.config.Comm.SSHConfigFunc(),
			},
			&commonsteps.StepProvision{},
			&commonsteps.StepCleanupTempKeys{
				Comm: &b.config.Comm,
			},
			NewStepPowerOffCompute(azureClient, ui, &b.config),
		}
	} else if b.config.OSType == constants.Target_Windows {
		steps = []multistep.Step{
			NewStepDeployTemplate(azureClient, ui, &b.config, deploymentName, GetVirtualMachineDeployment),
			&StepSaveWinRMPassword{
				Password:  b.config.tmpAdminPassword,
				BuildName: b.config.PackerBuildName,
			},
			&communicator.StepConnectWinRM{
				Config: &b.config.Comm,
				Host: func(stateBag multistep.StateBag) (string, error) {
					return stateBag.Get(constants.SSHHost).(string), nil
				},
				WinRMConfig: func(multistep.StateBag) (*communicator.WinRMConfig, error) {
					return &communicator.WinRMConfig{
						Username: b.config.UserName,
						Password: b.config.tmpAdminPassword,
					}, nil
				},
			},
			&commonsteps.StepProvision{},
			NewStepPowerOffCompute(azureClient, ui, &b.config),
		}
	} else {
		return nil, fmt.Errorf("Builder does not support the os_type '%s'", b.config.OSType)
	}

	captureSteps := b.config.CaptureSteps(
		ui.Say,
		NewStepCaptureImage(azureClient, ui, &b.config),
		NewStepPublishToSharedImageGallery(azureClient, ui, &b.config),
	)

	steps = append(steps, captureSteps...)
	steps = append(steps, NewStepDeleteVirtualMachine(azureClient, ui, &b.config))

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

	if b.config.isManagedImage() {
		managedImageID := fmt.Sprintf("/subscriptions/%s/resourceGroups/%s/providers/Microsoft.Compute/images/%s", b.config.ClientConfig.SubscriptionID, b.config.ManagedImageResourceGroupName, b.config.ManagedImageName)
		return NewManagedImageArtifact(b.config.OSType, b.config.ManagedImageResourceGroupName, b.config.ManagedImageName, b.config.Location, managedImageID)
	}
	return &Artifact{}, nil
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

func (b *Builder) configureStateBag(stateBag multistep.StateBag) {
	stateBag.Put(constants.AuthorizedKey, b.config.sshAuthorizedKey)

	stateBag.Put(constants.ArmTags, packerAzureCommon.MapToAzureTags(b.config.AzureTags))
	stateBag.Put(constants.ArmNewSDKTags, b.config.AzureTags)
	stateBag.Put(constants.ArmTags, b.config.AzureTags)
	stateBag.Put(constants.ArmComputeName, b.config.tmpComputeName)
	stateBag.Put(constants.ArmDeploymentName, b.config.tmpDeploymentName)
	stateBag.Put(constants.ArmKeyVaultName, b.config.tmpKeyVaultName)
	stateBag.Put(constants.ArmNicName, b.config.tmpNicName)
	stateBag.Put(constants.ArmPublicIPAddressName, b.config.tmpPublicIPAddressName)
	if b.config.tmpResourceGroupName != "" {
		stateBag.Put(constants.ArmResourceGroupName, b.config.tmpResourceGroupName)
		stateBag.Put(constants.ArmIsExistingResourceGroup, false)
	} else {
		stateBag.Put(constants.ArmIsExistingResourceGroup, true)
	}

	stateBag.Put(constants.ArmIsManagedImage, b.config.isManagedImage())
	stateBag.Put(constants.ArmManagedImageResourceGroupName, b.config.ManagedImageResourceGroupName)
	stateBag.Put(constants.ArmManagedImageName, b.config.ManagedImageName)
	if b.config.isManagedImage() && b.config.SharedGalleryDestination.SigDestinationGalleryName != "" {
		stateBag.Put(constants.ArmManagedImageSigPublishResourceGroup, b.config.SharedGalleryDestination.SigDestinationResourceGroup)
		stateBag.Put(constants.ArmManagedImageSharedGalleryName, b.config.SharedGalleryDestination.SigDestinationGalleryName)
		stateBag.Put(constants.ArmManagedImageSharedGalleryImageName, b.config.SharedGalleryDestination.SigDestinationImageName)
		stateBag.Put(constants.ArmManagedImageSharedGalleryImageVersion, b.config.SharedGalleryDestination.SigDestinationImageVersion)
		stateBag.Put(constants.ArmManagedImageSubscription, b.config.ClientConfig.SubscriptionID)
	}
	stateBag.Put(constants.ArmSubscription, b.config.ClientConfig.SubscriptionID)
}

// Parameters that are only known at runtime after querying Azure.
func (b *Builder) setRuntimeParameters(stateBag multistep.StateBag) {
	stateBag.Put(constants.ArmLocation, b.config.Location)
}

func (b *Builder) setTemplateParameters(stateBag multistep.StateBag) {
	stateBag.Put(constants.ArmVirtualMachineCaptureParameters, b.config.toVirtualMachineCaptureParameters())
}

func (b *Builder) getServicePrincipalToken(say func(string)) (*adal.ServicePrincipalToken, error) {
	return b.config.ClientConfig.GetServicePrincipalToken(say, b.config.ClientConfig.CloudEnvironment().ResourceManagerEndpoint)
}

func (b *Builder) getSubnetInformation(ctx context.Context, ui packersdk.Ui, azClient AzureClient) (*string, *string, error) {
	num := int64(10)
	labResourceId := hashiDTLVNETSDK.NewLabID(b.config.ClientConfig.SubscriptionID, b.config.LabResourceGroupName, b.config.LabName)
	virtualNetworkPage, err := azClient.DtlMetaClient.VirtualNetworks.List(ctx, labResourceId, hashiDTLVNETSDK.ListOperationOptions{Top: &num})

	if err != nil {
		return nil, nil, fmt.Errorf("Error retrieving Virtual networks in Resourcegroup %s", b.config.LabResourceGroupName)
	}

	virtualNetworks := virtualNetworkPage.Model
	for _, virtualNetwork := range *virtualNetworks {
		for _, subnetOverride := range *virtualNetwork.Properties.SubnetOverrides {

			// Check if the Subnet is allowed to create VMs having Public IP
			if *subnetOverride.UseInVMCreationPermission == hashiDTLVNETSDK.UsagePermissionTypeAllow && *subnetOverride.UsePublicIPAddressPermission == hashiDTLVNETSDK.UsagePermissionTypeAllow {
				// Return Virtual Network Name and Subnet Name
				// Since we cannot query the Usage information from DTL network we cannot know the current remaining capacity.
				// TODO (vaangadi) : Fix this to query the subnets that actually have space to create VM.
				return virtualNetwork.Name, subnetOverride.LabSubnetName, nil
			}
		}
	}
	return nil, nil, fmt.Errorf("No available Subnet with available space in resource group %s", b.config.LabResourceGroupName)
}

func normalizeAzureRegion(name string) string {
	return strings.ToLower(strings.Replace(name, " ", "", -1))
}
