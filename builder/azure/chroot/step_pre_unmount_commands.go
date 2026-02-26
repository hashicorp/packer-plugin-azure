// Copyright IBM Corp. 2013, 2025
// SPDX-License-Identifier: MPL-2.0

package chroot

import (
	"context"

	"github.com/hashicorp/packer-plugin-sdk/chroot"
	"github.com/hashicorp/packer-plugin-sdk/common"
	"github.com/hashicorp/packer-plugin-sdk/multistep"
	packersdk "github.com/hashicorp/packer-plugin-sdk/packer"
	"github.com/hashicorp/packer-plugin-sdk/template/interpolate"
)

// contextProvider is a local interface to access the config's interpolation context.
type contextProvider interface {
	GetContext() interpolate.Context
}

// preUnmountCommandsData provides template data for pre-unmount commands.
type preUnmountCommandsData struct {
	Device    string
	MountPath string
}

// StepPreUnmountCommands runs user-specified commands after provisioning but
// before the chroot is unmounted and LVM volumes are deactivated.
type StepPreUnmountCommands struct {
	Commands []string
}

func (s *StepPreUnmountCommands) Run(ctx context.Context, state multistep.StateBag) multistep.StepAction {
	if len(s.Commands) == 0 {
		return multistep.ActionContinue
	}

	ui := state.Get("ui").(packersdk.Ui)
	config := state.Get("config").(contextProvider)
	device := state.Get("device").(string)
	mountPath := state.Get("mount_path").(string)
	wrappedCommand := state.Get("wrappedCommand").(common.CommandWrapper)

	ictx := config.GetContext()
	ictx.Data = &preUnmountCommandsData{
		Device:    device,
		MountPath: mountPath,
	}

	ui.Say("Running pre-unmount commands...")
	if err := chroot.RunLocalCommands(s.Commands, wrappedCommand, ictx, ui); err != nil {
		state.Put("error", err)
		ui.Error(err.Error())
		return multistep.ActionHalt
	}

	return multistep.ActionContinue
}

func (s *StepPreUnmountCommands) Cleanup(state multistep.StateBag) {}
