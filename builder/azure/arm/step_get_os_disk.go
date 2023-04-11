// Copyright (c) HashiCorp, Inc.
// SPDX-License-Identifier: MPL-2.0

package arm

import (
	"context"
	"errors"
	"fmt"

	hashiVMSDK "github.com/hashicorp/go-azure-sdk/resource-manager/compute/2022-03-01/virtualmachines"
	"github.com/hashicorp/packer-plugin-azure/builder/azure/common/constants"

	"github.com/hashicorp/packer-plugin-sdk/multistep"
	packersdk "github.com/hashicorp/packer-plugin-sdk/packer"
)

type StepGetOSDisk struct {
	client *AzureClient
	query  func(ctx context.Context, resourceGroupName string, computeName string, subscriptionId string) (*hashiVMSDK.VirtualMachine, error)
	say    func(message string)
	error  func(e error)
}

func NewStepGetOSDisk(client *AzureClient, ui packersdk.Ui) *StepGetOSDisk {
	var step = &StepGetOSDisk{
		client: client,
		say:    func(message string) { ui.Say(message) },
		error:  func(e error) { ui.Error(e.Error()) },
	}

	step.query = step.queryCompute
	return step
}

func (s *StepGetOSDisk) queryCompute(ctx context.Context, resourceGroupName string, computeName string, subscriptionId string) (*hashiVMSDK.VirtualMachine, error) {
	vmID := hashiVMSDK.NewVirtualMachineID(subscriptionId, resourceGroupName, computeName)
	vm, err := s.client.VirtualMachinesClient.Get(ctx, vmID, hashiVMSDK.DefaultGetOperationOptions())
	if err != nil {
		s.say(s.client.LastError.Error())
		return nil, err
	}
	if model := vm.Model; model != nil {
		return model, nil
	}
	return nil, errors.New("TODO")
}

func (s *StepGetOSDisk) Run(ctx context.Context, state multistep.StateBag) multistep.StepAction {
	s.say("Querying the machine's properties ...")

	var resourceGroupName = state.Get(constants.ArmResourceGroupName).(string)
	var computeName = state.Get(constants.ArmComputeName).(string)
	var subscriptionId = state.Get(constants.ArmSubscription).(string)

	s.say(fmt.Sprintf(" -> ResourceGroupName : '%s'", resourceGroupName))
	s.say(fmt.Sprintf(" -> ComputeName       : '%s'", computeName))

	vm, err := s.query(ctx, resourceGroupName, computeName, subscriptionId)
	if err != nil {
		state.Put(constants.Error, err)
		s.error(err)

		return multistep.ActionHalt
	}

	var vhdUri string
	if vm.Properties.StorageProfile.OsDisk.Vhd != nil {
		vhdUri = *vm.Properties.StorageProfile.OsDisk.Vhd.Uri
		s.say(fmt.Sprintf(" -> OS Disk           : '%s'", vhdUri))
	} else {
		vhdUri = *vm.Properties.StorageProfile.OsDisk.ManagedDisk.Id
		s.say(fmt.Sprintf(" -> Managed OS Disk   : '%s'", vhdUri))
	}

	state.Put(constants.ArmOSDiskVhd, vhdUri)
	return multistep.ActionContinue
}

func (*StepGetOSDisk) Cleanup(multistep.StateBag) {
}
