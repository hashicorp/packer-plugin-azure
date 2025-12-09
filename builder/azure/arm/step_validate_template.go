// Copyright IBM Corp. 2013, 2025
// SPDX-License-Identifier: MPL-2.0

package arm

import (
	"context"
	"fmt"

	"github.com/hashicorp/go-azure-sdk/resource-manager/resources/2022-09-01/deployments"
	"github.com/hashicorp/packer-plugin-azure/builder/azure/common/constants"
	"github.com/hashicorp/packer-plugin-sdk/multistep"
	packersdk "github.com/hashicorp/packer-plugin-sdk/packer"
)

type StepValidateTemplate struct {
	client   *AzureClient
	validate func(ctx context.Context, subscriptionId string, resourceGroupName string, deploymentName string) error
	say      func(message string)
	error    func(e error)
	config   *Config
	factory  templateFactoryFunc
	name     string
}

func NewStepValidateTemplate(client *AzureClient, ui packersdk.Ui, config *Config, deploymentName string, factory templateFactoryFunc) *StepValidateTemplate {
	var step = &StepValidateTemplate{
		client:  client,
		say:     func(message string) { ui.Say(message) },
		error:   func(e error) { ui.Error(e.Error()) },
		config:  config,
		factory: factory,
		name:    deploymentName,
	}

	step.validate = step.validateTemplate
	return step
}

func (s *StepValidateTemplate) validateTemplate(ctx context.Context, subscriptionId string, resourceGroupName string, deploymentName string) error {
	deployment, err := s.factory(s.config)
	if err != nil {
		return err
	}
	pollingContext, cancel := context.WithTimeout(ctx, s.client.PollingDuration)
	defer cancel()

	id := deployments.NewResourceGroupProviderDeploymentID(subscriptionId, resourceGroupName, deploymentName)
	_, err = s.client.DeploymentsClient.Validate(pollingContext, id, *deployment)
	if err != nil {
		s.say(s.client.LastError.Error())
	}
	return err
}

func (s *StepValidateTemplate) Run(ctx context.Context, state multistep.StateBag) multistep.StepAction {
	s.say("Validating deployment template ...")

	var resourceGroupName = state.Get(constants.ArmResourceGroupName).(string)
	var subscriptionId = state.Get(constants.ArmSubscription).(string)

	s.say(fmt.Sprintf(" -> ResourceGroupName : '%s'", resourceGroupName))
	s.say(fmt.Sprintf(" -> DeploymentName    : '%s'", s.name))

	err := s.validate(ctx, subscriptionId, resourceGroupName, s.name)
	return processStepResult(err, s.error, state)
}

func (*StepValidateTemplate) Cleanup(multistep.StateBag) {
}
