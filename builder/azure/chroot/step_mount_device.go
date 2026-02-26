// Copyright IBM Corp. 2013, 2025
// SPDX-License-Identifier: MPL-2.0

package chroot

// mostly borrowed from ./builder/amazon/chroot/step_mount_device.go

import (
	"bytes"
	"context"
	"fmt"
	"os"
	"os/exec"
	"path/filepath"
	"runtime"
	"strings"

	"github.com/hashicorp/packer-plugin-azure/builder/azure/common/log"

	"github.com/hashicorp/packer-plugin-sdk/common"
	"github.com/hashicorp/packer-plugin-sdk/multistep"
	packersdk "github.com/hashicorp/packer-plugin-sdk/packer"
	"github.com/hashicorp/packer-plugin-sdk/template/interpolate"
)

var _ multistep.Step = &StepMountDevice{}

type StepMountDevice struct {
	Command        string
	MountOptions   []string
	MountPartition string
	MountPath      string

	mountPath     string
	isManualMount bool
}

func (s *StepMountDevice) Run(ctx context.Context, state multistep.StateBag) multistep.StepAction {
	ui := state.Get("ui").(packersdk.Ui)
	device := state.Get("device").(string)
	config := state.Get("config").(*Config)
	isManualMount := s.Command != ""
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
	if !isManualMount {
		if err := os.MkdirAll(mountPath, 0755); err != nil {
			err := fmt.Errorf("error creating mount directory: %s", err)
			state.Put("error", err)
			ui.Error(err.Error())
			return multistep.ActionHalt
		}
	}

	// When LVM is active, the device is already a full LV path (e.g. /dev/mapper/rhel-root)
	// and no partition suffix should be appended.
	mountPartition := s.MountPartition
	if _, ok := state.GetOk("lvm_active"); ok {
		mountPartition = ""
	}

	var deviceMount string
	if mountPartition == "" {
		deviceMount = device
	} else {
		switch runtime.GOOS {
		case "freebsd":
			deviceMount = fmt.Sprintf("%sp%s", device, mountPartition)
		default:
			deviceMount = fmt.Sprintf("%s%s", device, mountPartition)
		}
	}

	state.Put("deviceMount", deviceMount)

	ui.Say("Mounting the root device...")
	stderr := new(bytes.Buffer)
	var cmd *exec.Cmd
	if !isManualMount {
		// build mount options from mount_options config, useful for nouuid options
		// or other specific device type settings for mount
		opts := ""
		if len(s.MountOptions) > 0 {
			opts = "-o " + strings.Join(s.MountOptions, " -o ")
		}
		wrappedCommand := state.Get("wrappedCommand").(common.CommandWrapper)
		mountCommand, err := wrappedCommand(
			fmt.Sprintf("mount %s %s %s", opts, deviceMount, mountPath))
		if err != nil {
			err := fmt.Errorf("error creating mount command: %s", err)
			state.Put("error", err)
			ui.Error(err.Error())
			return multistep.ActionHalt
		}
		log.Printf("[DEBUG] (step mount) mount command is %s", mountCommand)
		cmd = common.ShellCommand(mountCommand)

	} else {
		log.Printf("[DEBUG] (step mount) mount command is %s", s.Command)
		cmd = common.ShellCommand(fmt.Sprintf("%s %s", s.Command, mountPath))
	}

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
	s.isManualMount = isManualMount
	state.Put("mount_path", s.mountPath)
	state.Put("mount_device_cleanup", s)

	return multistep.ActionContinue
}

func (s *StepMountDevice) Cleanup(state multistep.StateBag) {
	ui := state.Get("ui").(packersdk.Ui)
	if err := s.CleanupFunc(state); err != nil {
		ui.Error(err.Error())
	}
}

func (s *StepMountDevice) CleanupFunc(state multistep.StateBag) error {
	if s.mountPath == "" {
		return nil
	}

	ui := state.Get("ui").(packersdk.Ui)
	if !s.isManualMount {
		wrappedCommand := state.Get("wrappedCommand").(common.CommandWrapper)

		ui.Say("Unmounting the root device...")
		unmountCommand, err := wrappedCommand(fmt.Sprintf("umount -R %s", s.mountPath))
		if err != nil {
			return fmt.Errorf("error creating unmount command: %s", err)
		}

		cmd := common.ShellCommand(unmountCommand)
		if err := cmd.Run(); err != nil {
			return fmt.Errorf("error unmounting root device: %s", err)
		}
	} else {
		ui.Say("Skipping Unmounting the root device, it is manually unmounted via manual mount command script...")
	}
	s.mountPath = ""
	return nil
}
