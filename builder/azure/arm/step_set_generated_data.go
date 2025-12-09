// Copyright IBM Corp. 2013, 2025
// SPDX-License-Identifier: MPL-2.0

package arm

import (
	"context"

	"github.com/hashicorp/packer-plugin-sdk/multistep"
	"github.com/hashicorp/packer-plugin-sdk/packerbuilderdata"
)

type StepSetGeneratedData struct {
	GeneratedData *packerbuilderdata.GeneratedData
	Config        *Config
}

func (s *StepSetGeneratedData) Run(ctx context.Context, state multistep.StateBag) multistep.StepAction {

	s.GeneratedData.Put("TenantID", s.Config.ClientConfig.TenantID)
	s.GeneratedData.Put("SubscriptionID", s.Config.ClientConfig.SubscriptionID)
	s.GeneratedData.Put("TempComputeName", s.Config.tmpComputeName)
	s.GeneratedData.Put("TempNicName", s.Config.tmpNicName)
	s.GeneratedData.Put("TempOSDiskName", s.Config.tmpOSDiskName)
	s.GeneratedData.Put("TempDataDiskName", s.Config.tmpDataDiskName)
	s.GeneratedData.Put("TempDeploymentName", s.Config.tmpDeploymentName)
	s.GeneratedData.Put("TempVirtualNetworkName", s.Config.tmpVirtualNetworkName)
	s.GeneratedData.Put("TempKeyVaultName", s.Config.tmpKeyVaultName)
	s.GeneratedData.Put("TempResourceGroupName", s.Config.tmpResourceGroupName)
	s.GeneratedData.Put("TempNsgName", s.Config.tmpNsgName)
	s.GeneratedData.Put("TempSubnetName", s.Config.tmpSubnetName)
	s.GeneratedData.Put("TempPublicIPAddressName", s.Config.tmpPublicIPAddressName)
	return multistep.ActionContinue
}

func (s *StepSetGeneratedData) Cleanup(state multistep.StateBag) {
	// No cleanup...
}
