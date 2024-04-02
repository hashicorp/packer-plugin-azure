// Copyright (c) HashiCorp, Inc.
// SPDX-License-Identifier: MPL-2.0

package arm

import (
	"context"
	"errors"
	"fmt"

	"github.com/hashicorp/go-azure-sdk/resource-manager/compute/2022-03-01/virtualmachines"
	"github.com/hashicorp/packer-plugin-azure/builder/azure/common/constants"

	"github.com/hashicorp/packer-plugin-sdk/multistep"
	packersdk "github.com/hashicorp/packer-plugin-sdk/packer"
)

type StepGetDataDisk struct {
	client *AzureClient
	query  func(ctx context.Context, subscriptionId string, resourceGroupName string, computeName string) (*virtualmachines.VirtualMachine, error)
	say    func(message string)
	error  func(e error)
}

func NewStepGetAdditionalDisks(client *AzureClient, ui packersdk.Ui) *StepGetDataDisk {
	var step = &StepGetDataDisk{
		client: client,
		say:    func(message string) { ui.Say(message) },
		error:  func(e error) { ui.Error(e.Error()) },
	}

	step.query = step.queryCompute
	return step
}

func (s *StepGetDataDisk) queryCompute(ctx context.Context, subscriptionId string, resourceGroupName string, computeName string) (*virtualmachines.VirtualMachine, error) {
	vmID := virtualmachines.NewVirtualMachineID(subscriptionId, resourceGroupName, computeName)
	pollingContext, cancel := context.WithTimeout(ctx, s.client.PollingDuration)
	defer cancel()
	vm, err := s.client.VirtualMachinesClient.Get(pollingContext, vmID, virtualmachines.DefaultGetOperationOptions())
	if err != nil {
		s.say(s.client.LastError.Error())
	}
	if model := vm.Model; model == nil {
		return nil, errors.New("TODO")
	}
	return vm.Model, err
}

func (s *StepGetDataDisk) Run(ctx context.Context, state multistep.StateBag) multistep.StepAction {
	s.say("Querying the machine's additional disks properties ...")

	var resourceGroupName = state.Get(constants.ArmResourceGroupName).(string)
	var computeName = state.Get(constants.ArmComputeName).(string)
	var subscriptionId = state.Get(constants.ArmSubscription).(string)
	s.say(fmt.Sprintf(" -> ResourceGroupName : '%s'", resourceGroupName))
	s.say(fmt.Sprintf(" -> ComputeName       : '%s'", computeName))

	vm, err := s.query(ctx, subscriptionId, resourceGroupName, computeName)
	if err != nil {
		state.Put(constants.Error, err)
		s.error(err)

		return multistep.ActionHalt
	}

	if vm.Properties.StorageProfile.DataDisks != nil {
		var vhdUri string
		additional_disks := make([]string, len(*vm.Properties.StorageProfile.DataDisks))
		for i, additionaldisk := range *vm.Properties.StorageProfile.DataDisks {
			if additionaldisk.Vhd != nil {
				vhdUri = *additionaldisk.Vhd.Uri
				s.say(fmt.Sprintf(" -> Additional Disk %d          : '%s'", i+1, vhdUri))
			} else {
				vhdUri = *additionaldisk.ManagedDisk.Id
				s.say(fmt.Sprintf(" -> Managed Additional Disk %d  : '%s'", i+1, vhdUri))
			}
			additional_disks[i] = vhdUri
		}
		state.Put(constants.ArmAdditionalDiskVhds, additional_disks)
	}

	return multistep.ActionContinue
}

func (*StepGetDataDisk) Cleanup(multistep.StateBag) {
}
