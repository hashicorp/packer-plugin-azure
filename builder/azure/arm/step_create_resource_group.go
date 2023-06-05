// Copyright (c) HashiCorp, Inc.
// SPDX-License-Identifier: MPL-2.0

package arm

import (
	"context"
	"errors"
	"fmt"
	"time"

	"github.com/hashicorp/go-azure-helpers/resourcemanager/commonids"
	hashiGroupsSDK "github.com/hashicorp/go-azure-sdk/resource-manager/resources/2022-09-01/resourcegroups"
	"github.com/hashicorp/packer-plugin-azure/builder/azure/common/constants"
	"github.com/hashicorp/packer-plugin-sdk/multistep"
	packersdk "github.com/hashicorp/packer-plugin-sdk/packer"
)

type StepCreateResourceGroup struct {
	client *AzureClient
	create func(ctx context.Context, subscriptionId string, resourceGroupName string, location string, tags map[string]string) error
	say    func(message string)
	error  func(e error)
	exists func(ctx context.Context, subscriptionId string, resourceGroupName string) (bool, error)
}

func NewStepCreateResourceGroup(client *AzureClient, ui packersdk.Ui) *StepCreateResourceGroup {
	var step = &StepCreateResourceGroup{
		client: client,
		say:    func(message string) { ui.Say(message) },
		error:  func(e error) { ui.Error(e.Error()) },
	}

	step.create = step.createResourceGroup
	step.exists = step.doesResourceGroupExist
	return step
}

func (s *StepCreateResourceGroup) createResourceGroup(ctx context.Context, subscriptionId string, resourceGroupName string, location string, tags map[string]string) error {
	id := commonids.NewResourceGroupID(subscriptionId, resourceGroupName)
	_, err := s.client.ResourceGroupsClient.CreateOrUpdate(ctx, id, hashiGroupsSDK.ResourceGroup{
		Location: location,
		Tags:     &tags,
	})

	if err != nil {
		s.say(s.client.LastError.Error())
	}
	return err
}

func (s *StepCreateResourceGroup) doesResourceGroupExist(ctx context.Context, subscriptionId string, resourceGroupName string) (bool, error) {
	id := commonids.NewResourceGroupID(subscriptionId, resourceGroupName)
	exists, err := s.client.ResourceGroupsClient.Get(ctx, id)
	if err != nil {
		if exists.HttpResponse.StatusCode == 404 {
			return false, nil
		}
		s.say(s.client.LastError.Error())
	}

	return exists.HttpResponse.StatusCode != 404, nil
}

func (s *StepCreateResourceGroup) Run(ctx context.Context, state multistep.StateBag) multistep.StepAction {
	var doubleResource, ok = state.GetOk(constants.ArmDoubleResourceGroupNameSet)
	if ok && doubleResource.(bool) {
		err := errors.New("You have filled in both temp_resource_group_name and build_resource_group_name. Please choose one.")
		return processStepResult(err, s.error, state)
	}

	var resourceGroupName = state.Get(constants.ArmResourceGroupName).(string)
	var location = state.Get(constants.ArmLocation).(string)
	tags, ok := state.Get(constants.ArmNewSDKTags).(map[string]string)
	if !ok {
		err := fmt.Errorf("failed to extract tags from state bag")
		state.Put(constants.Error, err)
		s.error(err)
		return multistep.ActionHalt
	}

	subscriptionId := state.Get(constants.ArmSubscription).(string)
	exists, err := s.exists(ctx, subscriptionId, resourceGroupName)
	if err != nil {
		return processStepResult(err, s.error, state)
	}
	configThinksExists := state.Get(constants.ArmIsExistingResourceGroup).(bool)
	if exists != configThinksExists {
		if configThinksExists {
			err = errors.New("The resource group you want to use does not exist yet. Please use temp_resource_group_name to create a temporary resource group.")
		} else {
			err = errors.New("A resource group with that name already exists. Please use build_resource_group_name to use an existing resource group.")
		}
		return processStepResult(err, s.error, state)
	}

	// If the resource group exists, we may not have permissions to update it so we don't.
	if !exists {
		s.say("Creating resource group ...")

		s.say(fmt.Sprintf(" -> ResourceGroupName : '%s'", resourceGroupName))
		s.say(fmt.Sprintf(" -> Location          : '%s'", location))
		s.say(" -> Tags              :")
		for k, v := range tags {
			s.say(fmt.Sprintf(" ->> %s : %s", k, v))
		}
		err = s.create(ctx, subscriptionId, resourceGroupName, location, tags)
		if err == nil {
			state.Put(constants.ArmIsResourceGroupCreated, true)
		}
	} else {
		s.say("Using existing resource group ...")
		s.say(fmt.Sprintf(" -> ResourceGroupName : '%s'", resourceGroupName))
		s.say(fmt.Sprintf(" -> Location          : '%s'", location))
		state.Put(constants.ArmIsResourceGroupCreated, true)
	}

	return processStepResult(err, s.error, state)
}

func (s *StepCreateResourceGroup) Cleanup(state multistep.StateBag) {
	isCreated, ok := state.GetOk(constants.ArmIsResourceGroupCreated)
	if !ok || !isCreated.(bool) {
		return
	}

	ui := state.Get("ui").(packersdk.Ui)
	if state.Get(constants.ArmIsExistingResourceGroup).(bool) {
		ui.Say("\nThe resource group was not created by Packer, not deleting ...")
		return
	}

	ctx, cancelFunc := context.WithTimeout(context.Background(), time.Minute*10)
	defer cancelFunc()

	resourceGroupName := state.Get(constants.ArmResourceGroupName).(string)
	subscriptionId := state.Get(constants.ArmSubscription).(string)
	if exists, err := s.exists(ctx, subscriptionId, resourceGroupName); !exists || err != nil {
		return
	}

	ui.Say("\nCleanup requested, deleting resource group ...")
	id := commonids.NewResourceGroupID(subscriptionId, resourceGroupName)
	if state.Get(constants.ArmAsyncResourceGroupDelete).(bool) {
		_, deleteErr := s.client.ResourceGroupsClient.Delete(ctx, id, hashiGroupsSDK.DefaultDeleteOperationOptions())
		if deleteErr != nil {
			ui.Error(fmt.Sprintf("Error deleting resource group.  Please delete it manually.\n\n"+
				"Name: %s\n"+
				"Error: %s", resourceGroupName, deleteErr))
			return
		}
		s.say(fmt.Sprintf("\n Not waiting for Resource Group delete as requested by user. Resource Group Name is %s", resourceGroupName))
	} else {
		err := s.client.ResourceGroupsClient.DeleteThenPoll(ctx, id, hashiGroupsSDK.DefaultDeleteOperationOptions())
		if err != nil {
			ui.Error(fmt.Sprintf("Error deleting resource group.  Please delete it manually.\n\n"+
				"Name: %s\n"+
				"Error: %s", resourceGroupName, err))
			return
		}
		ui.Say("Resource group has been deleted.")
	}
}
