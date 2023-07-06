// Copyright (c) HashiCorp, Inc.
// SPDX-License-Identifier: MPL-2.0

package dtl

import (
	"context"
	"fmt"

	"github.com/hashicorp/packer-plugin-azure/builder/azure/common/constants"
	"github.com/hashicorp/packer-plugin-sdk/multistep"
	packersdk "github.com/hashicorp/packer-plugin-sdk/packer"
	hashiDTLVMSDK "github.com/hashicorp/go-azure-sdk/resource-manager/devtestlab/2018-09-15/virtualmachines"

)

type StepPowerOffCompute struct {
	client   *AzureClient
	config   *Config
	powerOff func(ctx context.Context, resourceGroupName string, labName, computeName string) error
	say      func(message string)
	error    func(e error)
}

func NewStepPowerOffCompute(client *AzureClient, ui packersdk.Ui, config *Config) *StepPowerOffCompute {

	var step = &StepPowerOffCompute{
		client: client,
		config: config,
		say:    func(message string) { ui.Say(message) },
		error:  func(e error) { ui.Error(e.Error()) },
	}

	step.powerOff = step.powerOffCompute
	return step
}

func (s *StepPowerOffCompute) powerOffCompute(ctx context.Context, resourceGroupName string, labName, computeName string) error {
	//f, err := s.client.VirtualMachinesClient.Deallocate(ctx, resourceGroupName, computeName)
	vmResourceId := hashiDTLVMSDK.NewVirtualMachineID(s.config.ClientConfig.SubscriptionID, s.config.tmpResourceGroupName, labName, computeName)
	err := s.client.DtlMetaClient.VirtualMachines.StopThenPoll(ctx, vmResourceId)
	if err != nil {
		s.say(s.client.LastError.Error())
	}
	return err
}

func (s *StepPowerOffCompute) Run(ctx context.Context, state multistep.StateBag) multistep.StepAction {
	s.say("Powering off machine ...")

	var resourceGroupName = state.Get(constants.ArmResourceGroupName).(string)
	var computeName = state.Get(constants.ArmComputeName).(string)

	s.say(fmt.Sprintf(" -> ResourceGroupName : '%s'", resourceGroupName))
	s.say(fmt.Sprintf(" -> ComputeName       : '%s'", computeName))

	err := s.powerOff(ctx, s.config.LabResourceGroupName, s.config.LabName, computeName)

	s.say("Powering off machine ...Complete")
	return processStepResult(err, s.error, state)
}

func (*StepPowerOffCompute) Cleanup(multistep.StateBag) {
}
