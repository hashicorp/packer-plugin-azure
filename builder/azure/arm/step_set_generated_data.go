// Copyright (c) HashiCorp, Inc.
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
	return multistep.ActionContinue
}

func (s *StepSetGeneratedData) Cleanup(state multistep.StateBag) {
	// No cleanup...
}
