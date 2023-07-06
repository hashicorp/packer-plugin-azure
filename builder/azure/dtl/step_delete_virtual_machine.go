// Copyright (c) HashiCorp, Inc.
// SPDX-License-Identifier: MPL-2.0

package dtl

import (
	"context"
	"fmt"

	hashiDTLVMSDK "github.com/hashicorp/go-azure-sdk/resource-manager/devtestlab/2018-09-15/virtualmachines"
	"github.com/hashicorp/packer-plugin-azure/builder/azure/common/constants"
	"github.com/hashicorp/packer-plugin-sdk/multistep"
	packersdk "github.com/hashicorp/packer-plugin-sdk/packer"
)

type StepDeleteVirtualMachine struct {
	client *AzureClient
	config *Config
	delete func(ctx context.Context, resourceGroupName string, subscriptionId string, labName string, computeName string) error
	say    func(message string)
	error  func(e error)
}

func NewStepDeleteVirtualMachine(client *AzureClient, ui packersdk.Ui, config *Config) *StepDeleteVirtualMachine {
	var step = &StepDeleteVirtualMachine{
		client: client,
		config: config,
		say:    func(message string) { ui.Say(message) },
		error:  func(e error) { ui.Error(e.Error()) },
	}

	step.delete = step.deleteVirtualMachine
	return step
}

func (s *StepDeleteVirtualMachine) deleteVirtualMachine(ctx context.Context, subscriptionId string, labName string, resourceGroupName string, vmName string) error {
	vmId := hashiDTLVMSDK.NewVirtualMachineID(subscriptionId, resourceGroupName, labName, vmName)
	err := s.client.DtlMetaClient.VirtualMachines.DeleteThenPoll(ctx, vmId)
	if err != nil {
		s.say("Error from delete VM")
		s.say(s.client.LastError.Error())
	}

	return err
}

func (s *StepDeleteVirtualMachine) Run(ctx context.Context, state multistep.StateBag) multistep.StepAction {
	s.say("Deleting the virtual machine ...")

	var resourceGroupName = state.Get(constants.ArmResourceGroupName).(string)
	var computeName = state.Get(constants.ArmComputeName).(string)
	var dtlLabName = state.Get(constants.DtlLabName).(string)
	var subscriptionId = state.Get(constants.ArmSubscription).(string)

	s.say(fmt.Sprintf(" -> ResourceGroupName : '%s'", resourceGroupName))

	s.say(fmt.Sprintf(" -> ComputeName       : '%s'", computeName))

	err := s.deleteVirtualMachine(ctx, subscriptionId, dtlLabName, resourceGroupName, computeName)

	s.say("Deleting virtual machine ...Complete")
	return processStepResult(err, s.error, state)
}

func (*StepDeleteVirtualMachine) Cleanup(multistep.StateBag) {
}
