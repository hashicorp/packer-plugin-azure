// Copyright (c) HashiCorp, Inc.
// SPDX-License-Identifier: MPL-2.0

package common

import (
	"context"

	"github.com/hashicorp/packer-plugin-sdk/multistep"
)

type StepNotify struct {
	message string
	say     func(string)
}

func NewStepNotify(message string, say func(string)) *StepNotify {
	return &StepNotify{
		message: message,
		say:     say,
	}
}

func (step *StepNotify) Run(
	ctx context.Context,
	state multistep.StateBag,
) multistep.StepAction {
	step.say(step.message)
	return multistep.ActionContinue
}

func (step *StepNotify) Cleanup(state multistep.StateBag) {}

var _ multistep.Step = (*StepNotify)(nil)
