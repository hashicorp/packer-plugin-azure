// Copyright (c) HashiCorp, Inc.
// SPDX-License-Identifier: MPL-2.0

package arm

import (
	"context"
	"fmt"

	"github.com/hashicorp/go-azure-helpers/resourcemanager/commonids"
	"github.com/hashicorp/go-azure-sdk/resource-manager/network/2022-09-01/networkinterfaces"
	"github.com/hashicorp/go-azure-sdk/resource-manager/network/2022-09-01/publicipaddresses"
	"github.com/hashicorp/packer-plugin-azure/builder/azure/common/constants"
	"github.com/hashicorp/packer-plugin-sdk/multistep"
	packersdk "github.com/hashicorp/packer-plugin-sdk/packer"
)

type EndpointType int

const (
	PublicEndpoint EndpointType = iota
	PrivateEndpoint
	PublicEndpointInPrivateNetwork
)

var (
	EndpointCommunicationText = map[EndpointType]string{
		PublicEndpoint:                 "PublicEndpoint",
		PrivateEndpoint:                "PrivateEndpoint",
		PublicEndpointInPrivateNetwork: "PublicEndpointInPrivateNetwork",
	}
)

type StepGetIPAddress struct {
	client   *AzureClient
	endpoint EndpointType
	get      func(ctx context.Context, subscriptionId string, resourceGroupName string, ipAddressName string, interfaceName string) (string, error)
	say      func(message string)
	error    func(e error)
}

func NewStepGetIPAddress(client *AzureClient, ui packersdk.Ui, endpoint EndpointType) *StepGetIPAddress {
	var step = &StepGetIPAddress{
		client:   client,
		endpoint: endpoint,
		say:      func(message string) { ui.Say(message) },
		error:    func(e error) { ui.Error(e.Error()) },
	}

	switch endpoint {
	case PrivateEndpoint:
		step.get = step.getPrivateIP
	case PublicEndpoint:
		step.get = step.getPublicIP
	case PublicEndpointInPrivateNetwork:
		step.get = step.getPublicIPInPrivateNetwork
	}

	return step
}

func (s *StepGetIPAddress) getPrivateIP(ctx context.Context, subscriptionId string, resourceGroupName string, ipAddressName string, interfaceName string) (string, error) {
	getIPContext, cancel := context.WithTimeout(ctx, s.client.PollingDuration)
	defer cancel()
	intID := commonids.NewNetworkInterfaceID(subscriptionId, resourceGroupName, interfaceName)
	resp, err := s.client.NetworkMetaClient.NetworkInterfaces.Get(getIPContext, intID, networkinterfaces.DefaultGetOperationOptions())
	if err != nil {
		s.say(s.client.LastError.Error())
		return "", err
	}

	return *(*resp.Model.Properties.IPConfigurations)[0].Properties.PrivateIPAddress, nil
}

func (s *StepGetIPAddress) getPublicIP(ctx context.Context, subscriptionId string, resourceGroupName string, ipAddressName string, interfaceName string) (string, error) {
	getIPContext, cancel := context.WithTimeout(ctx, s.client.PollingDuration)
	defer cancel()
	ipID := commonids.NewPublicIPAddressID(subscriptionId, resourceGroupName, ipAddressName)
	resp, err := s.client.NetworkMetaClient.PublicIPAddresses.Get(getIPContext, ipID, publicipaddresses.DefaultGetOperationOptions())
	if err != nil {
		return "", err
	}

	return *resp.Model.Properties.IPAddress, nil
}

// TODO The interface name passed into getPublicIP has never done anything
// This code has been around for over 6 years so I'm hesistant to change it without more investigation so we should
// open a seperate GitHub issue for this when looking to merge the SDK branch
func (s *StepGetIPAddress) getPublicIPInPrivateNetwork(ctx context.Context, subscriptionId string, resourceGroupName string, ipAddressName string, interfaceName string) (string, error) {
	return s.getPublicIP(ctx, subscriptionId, resourceGroupName, ipAddressName, interfaceName)
}

func (s *StepGetIPAddress) Run(ctx context.Context, state multistep.StateBag) multistep.StepAction {
	s.say("Getting the VM's IP address ...")

	var resourceGroupName = state.Get(constants.ArmResourceGroupName).(string)
	var ipAddressName = state.Get(constants.ArmPublicIPAddressName).(string)
	var subscriptionId = state.Get(constants.ArmSubscription).(string)
	var nicName = state.Get(constants.ArmNicName).(string)

	s.say(fmt.Sprintf(" -> ResourceGroupName   : '%s'", resourceGroupName))
	s.say(fmt.Sprintf(" -> PublicIPAddressName : '%s'", ipAddressName))
	s.say(fmt.Sprintf(" -> NicName             : '%s'", nicName))
	s.say(fmt.Sprintf(" -> Network Connection  : '%s'", EndpointCommunicationText[s.endpoint]))

	address, err := s.get(ctx, subscriptionId, resourceGroupName, ipAddressName, nicName)
	if err != nil {
		state.Put(constants.Error, err)
		s.error(err)

		return multistep.ActionHalt
	}

	state.Put(constants.SSHHost, address)
	s.say(fmt.Sprintf(" -> IP Address          : '%s'", address))

	return multistep.ActionContinue
}

func (*StepGetIPAddress) Cleanup(multistep.StateBag) {
}
