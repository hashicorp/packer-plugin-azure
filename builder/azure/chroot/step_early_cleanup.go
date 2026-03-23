// Copyright IBM Corp. 2013, 2025
// SPDX-License-Identifier: MPL-2.0

package chroot

import (
	"context"
	"fmt"

	"github.com/hashicorp/packer-plugin-azure/builder/azure/common/log"

	"github.com/hashicorp/packer-plugin-sdk/multistep"
	packersdk "github.com/hashicorp/packer-plugin-sdk/packer"
)

// earlyCleanup is a local interface to avoid import-path confusion with the
// SDK's chroot package (our package is also named "chroot").
type earlyCleanup interface {
	CleanupFunc(multistep.StateBag) error
}

// StepEarlyCleanup is a custom replacement for the SDK's chroot.StepEarlyCleanup
// that includes LVM deactivation between unmount and disk detach.
type StepEarlyCleanup struct{}

func (s *StepEarlyCleanup) Run(ctx context.Context, state multistep.StateBag) multistep.StepAction {
	ui := state.Get("ui").(packersdk.Ui)

	// Cleanup keys in order: unmount filesystem, deactivate LVM, detach disk.
	cleanupKeys := []string{
		"copy_files_cleanup",
		"mount_extra_cleanup",
		"mount_device_cleanup",
		"lvm_cleanup",
		"attach_cleanup",
	}

	for _, key := range cleanupKeys {
		c := state.Get(key)
		if c == nil {
			log.Printf("Skipping cleanup func: %s (not set)", key)
			continue
		}

		cleanup, ok := c.(earlyCleanup)
		if !ok {
			log.Printf("Skipping cleanup func: %s (does not implement CleanupFunc)", key)
			continue
		}

		log.Printf("Running cleanup func: %s", key)
		if err := cleanup.CleanupFunc(state); err != nil {
			err = fmt.Errorf("error during cleanup %s: %v", key, err)
			state.Put("error", err)
			ui.Error(err.Error())
			return multistep.ActionHalt
		}
	}

	return multistep.ActionContinue
}

func (s *StepEarlyCleanup) Cleanup(state multistep.StateBag) {}
