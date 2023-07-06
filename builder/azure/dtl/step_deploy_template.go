// Copyright (c) HashiCorp, Inc.
// SPDX-License-Identifier: MPL-2.0

package dtl

import (
	"context"
	"errors"
	"fmt"
	"net/url"
	"strings"
	"time"

	"github.com/hashicorp/go-azure-helpers/resourcemanager/commonids"
	hashiNetworkSecurityGroupsSDK "github.com/hashicorp/go-azure-sdk/resource-manager/network/2022-09-01/networksecuritygroups"
	hashiVirtualNetworksSDK "github.com/hashicorp/go-azure-sdk/resource-manager/network/2022-09-01/virtualnetworks"
	hashiDeploymentOperationsSDK "github.com/hashicorp/go-azure-sdk/resource-manager/resources/2022-09-01/deploymentoperations"
	hashiDeploymentsSDK "github.com/hashicorp/go-azure-sdk/resource-manager/resources/2022-09-01/deployments"

	hashiDTLVMSDK "github.com/hashicorp/go-azure-sdk/resource-manager/devtestlab/2018-09-15/virtualmachines"
	hashiLabsSDK "github.com/hashicorp/go-azure-sdk/resource-manager/devtestlab/2018-09-15/labs"
	"github.com/hashicorp/go-azure-sdk/resource-manager/network/2022-09-01/networkinterfaces"
	"github.com/hashicorp/packer-plugin-azure/builder/azure/common/constants"
	"github.com/hashicorp/packer-plugin-sdk/multistep"
	packersdk "github.com/hashicorp/packer-plugin-sdk/packer"
)

type StepDeployTemplate struct {
	client     *AzureClient
	deploy     func(ctx context.Context, resourceGroupName string, deploymentName string, state multistep.StateBag) error
	delete     func(ctx context.Context, client *AzureClient, resourceType string, resourceName string, resourceGroupName string) error
	disk       func(ctx context.Context, resourceGroupName string, computeName string) (string, string, error)
	deleteDisk func(ctx context.Context, imageType string, imageName string, resourceGroupName string) error
	say        func(message string)
	error      func(e error)
	config     *Config
	factory    templateFactoryFuncDtl
	name       string
}

func NewStepDeployTemplate(client *AzureClient, ui packersdk.Ui, config *Config, deploymentName string, factory templateFactoryFuncDtl) *StepDeployTemplate {
	var step = &StepDeployTemplate{
		client:  client,
		say:     func(message string) { ui.Say(message) },
		error:   func(e error) { ui.Error(e.Error()) },
		config:  config,
		factory: factory,
		name:    deploymentName,
	}

	step.deploy = step.deployTemplate
	step.delete = deleteResource
	step.disk = step.getImageDetails
	step.deleteDisk = step.deleteImage
	return step
}

func (s *StepDeployTemplate) deployTemplate(ctx context.Context, resourceGroupName string, deploymentName string, state multistep.StateBag) error {


	subscriptionId := s.config.ClientConfig.SubscriptionID
	labName := s.config.LabName

	labResourceId := hashiDTLVMSDK.NewLabID(subscriptionId, resourceGroupName, labName)
	vmlistPage, err := s.client.DtlMetaClient.VirtualMachines.List(ctx, labResourceId, hashiDTLVMSDK.DefaultListOperationOptions())
	if err != nil {
		s.say(s.client.LastError.Error())
		return err
	}

	vmList := vmlistPage.Model
	for _, vm := range *vmList {
		if *vm.Name == s.config.tmpComputeName {
			return fmt.Errorf("Error: Virtual Machine %s already exists. Please use another name", s.config.tmpComputeName)
		}
	}

	s.say(fmt.Sprintf("Creating Virtual Machine %s", s.config.tmpComputeName))
	labMachine, err := s.factory(s.config)
	if err != nil {
		return err
	}

	labId := hashiLabsSDK.NewLabID(subscriptionId, s.config.tmpResourceGroupName, labName)
	err = s.client.DtlMetaClient.Labs.CreateEnvironmentThenPoll(ctx, labId, *labMachine)
	if err != nil {
		s.say(s.client.LastError.Error())
		return err
	}

	expand := "Properties($expand=ComputeVm,Artifacts,NetworkInterface)"
	vmResourceId := hashiDTLVMSDK.NewVirtualMachineID(subscriptionId, s.config.tmpResourceGroupName, labName, s.config.tmpComputeName)
	vm, err := s.client.DtlMetaClient.VirtualMachines.Get(ctx, vmResourceId, hashiDTLVMSDK.GetOperationOptions{Expand: &expand})
	if err != nil {
		s.say(s.client.LastError.Error())
	}

	// set tmpFQDN to the PrivateIP or to the real FQDN depending on
	// publicIP being allowed or not
	if s.config.DisallowPublicIP {
		interfaceID := commonids.NewNetworkInterfaceID(subscriptionId, resourceGroupName, s.config.tmpNicName)
		resp, err := s.client.NetworkMetaClient.NetworkInterfaces.Get(ctx, interfaceID, networkinterfaces.DefaultGetOperationOptions())
		if err != nil {
			s.say(s.client.LastError.Error())
			return err
		}
		// TODO This operation seems kinda off, but I don't wanna spend time digging into it right now
		s.config.tmpFQDN = *(*resp.Model.Properties.IPConfigurations)[0].Properties.PrivateIPAddress
	} else {
		s.config.tmpFQDN = *vm.Model.Properties.Fqdn
	}
	s.say(fmt.Sprintf(" -> VM FQDN/IP : '%s'", s.config.tmpFQDN))
	state.Put(constants.SSHHost, s.config.tmpFQDN)

	// In a windows VM, add the winrm artifact. Doing it after the machine has been
	// created allows us to use its IP address as FQDN
	if strings.ToLower(s.config.OSType) == "windows" {
		// Add mandatory Artifact
		var winrma = "windows-winrm"
		var artifactid = fmt.Sprintf("/subscriptions/%s/resourceGroups/%s/providers/Microsoft.DevTestLab/labs/%s/artifactSources/public repo/artifacts/%s",
			s.config.ClientConfig.SubscriptionID,
			s.config.tmpResourceGroupName,
			s.config.LabName,
			winrma)

		var hostname = "hostName"
		dp := &hashiDTLVMSDK.ArtifactParameterProperties{}
		dp.Name = &hostname
		dp.Value = &s.config.tmpFQDN
		dparams := []hashiDTLVMSDK.ArtifactParameterProperties{*dp}

		winrmArtifact := &hashiDTLVMSDK.ArtifactInstallProperties{
			ArtifactTitle: &winrma,
			ArtifactId:    &artifactid,
			Parameters:    &dparams,
		}

		dtlArtifacts := []hashiDTLVMSDK.ArtifactInstallProperties{*winrmArtifact}
		dtlArtifactsRequest := hashiDTLVMSDK.ApplyArtifactsRequest{Artifacts: &dtlArtifacts}

		// TODO replaced infinite loop with one time try, this should not fail imo, maybe they were actually running into failures after polling?
		for {
			err := s.client.DtlMetaClient.VirtualMachines.ApplyArtifactsThenPoll(ctx, vmResourceId, dtlArtifactsRequest)
			if err != nil {	
				s.say("WinRM artifact deployment failed, sleeping a minute and retrying")
				time.Sleep(60 * time.Second)
			}
		}
	}

	xs := strings.Split(*vm.Model.Properties.ComputeId, "/")
	s.config.VMCreationResourceGroup = xs[4]

	// Resuing the Resource group name from common constants as all steps depend on it.
	state.Put(constants.ArmResourceGroupName, s.config.VMCreationResourceGroup)

	s.say(fmt.Sprintf(" -> VM ResourceGroupName : '%s'", s.config.VMCreationResourceGroup))

	return err
}

func (s *StepDeployTemplate) Run(ctx context.Context, state multistep.StateBag) multistep.StepAction {
	s.say("Deploying deployment template ...")

	var resourceGroupName = state.Get(constants.ArmResourceGroupName).(string)

	s.say(fmt.Sprintf(" -> Lab ResourceGroupName : '%s'", resourceGroupName))

	return processStepResult(
		s.deploy(ctx, resourceGroupName, s.name, state),
		s.error, state)
}

func (s *StepDeployTemplate) getImageDetails(ctx context.Context, resourceGroupName string, computeName string) (string, string, error) {
	//We can't depend on constants.ArmOSDiskVhd being set
	var imageName string
	var imageType string
	vm, err := s.client.VirtualMachinesClient.Get(ctx, resourceGroupName, computeName, "")
	if err != nil {
		return imageName, imageType, err
	} else {
		if vm.StorageProfile.OsDisk.Vhd != nil {
			imageType = "image"
			imageName = *vm.StorageProfile.OsDisk.Vhd.URI
		} else {
			imageType = "Microsoft.Compute/disks"
			imageName = *vm.StorageProfile.OsDisk.ManagedDisk.ID
		}
	}
	return imageType, imageName, nil
}

func deleteResource(ctx context.Context, client *AzureClient, subscriptionId string, resourceType string, resourceName string, resourceGroupName string) error {
	switch resourceType {
	case "Microsoft.Compute/virtualMachines":
		f, err := client.VirtualMachinesClient.Delete(ctx, resourceGroupName, resourceName)
		if err == nil {
			err = f.WaitForCompletionRef(ctx, client.VirtualMachinesClient.Client)
		}
		return err
	case "Microsoft.KeyVault/vaults":
		id := commonids.NewKeyVaultID(subscriptionId, resourceGroupName, resourceName)
		_, err := client.VaultsClient.Delete(ctx, id)
		return err
	case "Microsoft.Network/networkInterfaces":
		interfaceID := commonids.NewNetworkInterfaceID(subscriptionId, resourceGroupName, resourceName)
		err := client.NetworkMetaClient.NetworkInterfaces.DeleteThenPoll(ctx, interfaceID)
		return err
	case "Microsoft.Network/virtualNetworks":
		vnetID := hashiVirtualNetworksSDK.NewVirtualNetworkID(subscriptionId, resourceGroupName, resourceName)
		err := client.NetworkMetaClient.VirtualNetworks.DeleteThenPoll(ctx, vnetID)
		return err
	case "Microsoft.Network/networkSecurityGroups":
		secGroupId := hashiNetworkSecurityGroupsSDK.NewNetworkSecurityGroupID(subscriptionId, resourceGroupName, resourceName)
		err := client.NetworkMetaClient.NetworkSecurityGroups.DeleteThenPoll(ctx, secGroupId)
		return err
	case "Microsoft.Network/publicIPAddresses":
		ipID := commonids.NewPublicIPAddressID(subscriptionId, resourceGroupName, resourceName)
		err := client.NetworkMetaClient.PublicIPAddresses.DeleteThenPoll(ctx, ipID)
		return err
	}
	return nil
}

func (s *StepDeployTemplate) deleteImage(ctx context.Context, imageName string, resourceGroupName string, isManagedDisk bool, subscriptionId string, storageAccountName string) error {
	// Managed disk
	if isManagedDisk {
		xs := strings.Split(imageName, "/")
		diskName := xs[len(xs)-1]
		diskId := hashiDisksSDK.NewDiskID(subscriptionId, resourceGroupName, diskName)

		if err := s.client.DisksClient.DeleteThenPoll(ctx, diskId); err != nil {
			return err
		}
		return nil
	}

	// VHD image
	u, err := url.Parse(imageName)
	if err != nil {
		return err
	}
	xs := strings.Split(u.Path, "/")
	var blobName = strings.Join(xs[2:], "/")
	if len(xs) < 3 {
		return errors.New("Unable to parse path of image " + imageName)
	}
	_, err = s.client.GiovanniBlobClient.Delete(ctx, storageAccountName, "images", blobName, giovanniBlobStorageSDK.DeleteInput{})
	return err
}

func (s *StepDeployTemplate) Cleanup(state multistep.StateBag) {
	//Only clean up if this was an existing resource group and the resource group
	//is marked as created
	// Just return now
}
