// Copyright IBM Corp. 2013, 2025
// SPDX-License-Identifier: MPL-2.0

//go:build !linux && !freebsd

package chroot

import (
	"context"

	"github.com/hashicorp/packer-plugin-sdk/multistep"
)

var _ multistep.Step = &StepSetupLVM{}

type StepSetupLVM struct {
	device        string
	volumeGroups  []string
	LVMRootDevice string
	activated     bool
}

func (s *StepSetupLVM) Run(ctx context.Context, state multistep.StateBag) multistep.StepAction {
	return multistep.ActionContinue
}

func (s *StepSetupLVM) Cleanup(state multistep.StateBag) {}

func (s *StepSetupLVM) CleanupFunc(state multistep.StateBag) error {
	return nil
}
