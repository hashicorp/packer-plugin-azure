// Copyright (c) HashiCorp, Inc.
// SPDX-License-Identifier: MPL-2.0

package dtl

import (
	"context"
	"fmt"
	"strings"
	"time"

	"github.com/hashicorp/go-azure-helpers/resourcemanager/commonids"

	"github.com/hashicorp/go-azure-sdk/resource-manager/devtestlab/2018-09-15/labs"
	"github.com/hashicorp/go-azure-sdk/resource-manager/devtestlab/2018-09-15/virtualmachines"
	"github.com/hashicorp/go-azure-sdk/resource-manager/network/2022-09-01/networkinterfaces"
	"github.com/hashicorp/packer-plugin-azure/builder/azure/common/constants"
	"github.com/hashicorp/packer-plugin-sdk/multistep"
	packersdk "github.com/hashicorp/packer-plugin-sdk/packer"
	"github.com/hashicorp/packer-plugin-sdk/retry"
)

type StepDeployTemplate struct {
	client  *AzureClient
	deploy  func(ctx context.Context, resourceGroupName string, deploymentName string, state multistep.StateBag) error
	say     func(message string)
	error   func(e error)
	config  *Config
	factory templateFactoryFuncDtl
	name    string
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
	return step
}

func (s *StepDeployTemplate) deployTemplate(ctx context.Context, resourceGroupName string, deploymentName string, state multistep.StateBag) error {

	subscriptionId := s.config.ClientConfig.SubscriptionID
	labName := s.config.LabName

	// TODO Talk to Tom(s) about this, we have to have two different Labs IDs in different calls, so we can probably move this into the commonids package
	labResourceId := virtualmachines.NewLabID(subscriptionId, resourceGroupName, labName)
	labId := labs.NewLabID(subscriptionId, s.config.tmpResourceGroupName, labName)
	vmlistPage, err := s.client.DtlMetaClient.VirtualMachines.List(ctx, labResourceId, virtualmachines.DefaultListOperationOptions())
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

	err = s.client.DtlMetaClient.Labs.CreateEnvironmentThenPoll(ctx, labId, *labMachine)
	if err != nil {
		s.say(s.client.LastError.Error())
		return err
	}

	expand := "Properties($expand=ComputeVm,Artifacts,NetworkInterface)"
	vmResourceId := virtualmachines.NewVirtualMachineID(subscriptionId, s.config.tmpResourceGroupName, labName, s.config.tmpComputeName)
	vm, err := s.client.DtlMetaClient.VirtualMachines.Get(ctx, vmResourceId, virtualmachines.GetOperationOptions{Expand: &expand})
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
		dp := &virtualmachines.ArtifactParameterProperties{}
		dp.Name = &hostname
		dp.Value = &s.config.tmpFQDN
		dparams := []virtualmachines.ArtifactParameterProperties{*dp}

		winrmArtifact := &virtualmachines.ArtifactInstallProperties{
			ArtifactTitle: &winrma,
			ArtifactId:    &artifactid,
			Parameters:    &dparams,
		}

		dtlArtifacts := []virtualmachines.ArtifactInstallProperties{*winrmArtifact}
		dtlArtifactsRequest := virtualmachines.ApplyArtifactsRequest{Artifacts: &dtlArtifacts}

		// TODO this was an infinite loop, I have seen apply artifacts fail
		// But this needs a bit further validation into why it fails and
		// How we can avoid the need for a retry backoff
		// But a retry backoff is much more preferable to an infinite loop

		retryConfig := retry.Config{
			Tries:      5,
			RetryDelay: (&retry.Backoff{InitialBackoff: 5 * time.Second, MaxBackoff: 60 * time.Second, Multiplier: 1.5}).Linear,
		}
		err = retryConfig.Run(ctx, func(ctx context.Context) error {
			err := s.client.DtlMetaClient.VirtualMachines.ApplyArtifactsThenPoll(ctx, vmResourceId, dtlArtifactsRequest)
			if err != nil {
				s.say("WinRM artifact deployment failed, retrying")
			}
			return nil
		})
		if err != nil {
			return err
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

func (s *StepDeployTemplate) Cleanup(state multistep.StateBag) {
	// TODO are there any resources created in DTL builds we should tear down?
	// There was teardown code from the ARM builder copy pasted in but it was never called
}
