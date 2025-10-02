// Copyright (c) HashiCorp, Inc.
// SPDX-License-Identifier: MPL-2.0

package chroot

// mostly borrowed from ./builder/amazon/chroot/step_mount_device.go

import (
	"bytes"
	"context"
	"fmt"
	"path/filepath"
	"runtime"

	"github.com/hashicorp/packer-plugin-azure/builder/azure/common/log"

	"github.com/hashicorp/packer-plugin-sdk/common"
	"github.com/hashicorp/packer-plugin-sdk/multistep"
	packersdk "github.com/hashicorp/packer-plugin-sdk/packer"
	"github.com/hashicorp/packer-plugin-sdk/template/interpolate"
)

var _ multistep.Step = &StepManualMountCommand{}

type StepManualMountCommand struct {
	Command        string
	MountPartition string
	MountPath      string

	mountPath string
}

func (s *StepManualMountCommand) Run(ctx context.Context, state multistep.StateBag) multistep.StepAction {
	ui := state.Get("ui").(packersdk.Ui)
	device := state.Get("device").(string)
	config := state.Get("config").(*Config)

	ictx := config.ctx

	ictx.Data = &struct{ Device string }{Device: filepath.Base(device)}
	mountPath, err := interpolate.Render(s.MountPath, &ictx)

	if err != nil {
		err := fmt.Errorf("error preparing mount directory: %s", err)
		state.Put("error", err)
		ui.Error(err.Error())
		return multistep.ActionHalt
	}

	mountPath, err = filepath.Abs(mountPath)
	if err != nil {
		err := fmt.Errorf("error preparing mount directory: %s", err)
		state.Put("error", err)
		ui.Error(err.Error())
		return multistep.ActionHalt
	}

	log.Printf("Mount path: %s", mountPath)

	var deviceMount string
	switch runtime.GOOS {
	case "freebsd":
		deviceMount = fmt.Sprintf("%sp%s", device, s.MountPartition)
	default:
		deviceMount = fmt.Sprintf("%s%s", device, s.MountPartition)
	}

	state.Put("deviceMount", deviceMount)

	ui.Say("Mounting the root device...")
	stderr := new(bytes.Buffer)

	log.Printf("[DEBUG] (step mount) mount command is %s", s.Command)
	cmd := common.ShellCommand(fmt.Sprintf("%s %s", s.Command, mountPath))
	cmd.Stderr = stderr
	if err := cmd.Run(); err != nil {
		err := fmt.Errorf(
			"error mounting root volume: %s\nStderr: %s", err, stderr.String())
		state.Put("error", err)
		ui.Error(err.Error())
		return multistep.ActionHalt
	}

	// Set the mount path so we remember to unmount it later
	s.mountPath = mountPath
	state.Put("mount_path", s.mountPath)
	state.Put("mount_device_cleanup", s)

	return multistep.ActionContinue
}

func (s *StepManualMountCommand) Cleanup(state multistep.StateBag) {
	ui := state.Get("ui").(packersdk.Ui)
	if err := s.CleanupFunc(state); err != nil {
		ui.Error(err.Error())
	}
}

func (s *StepManualMountCommand) CleanupFunc(state multistep.StateBag) error {
	if s.mountPath == "" {
		return nil
	}

	ui := state.Get("ui").(packersdk.Ui)

	ui.Say("Skipping Unmounting the root device, it is manually unmounted via manual mount command script...")

	s.mountPath = ""
	return nil
}
