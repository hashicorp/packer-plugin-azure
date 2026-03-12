// Copyright IBM Corp. 2013, 2025
// SPDX-License-Identifier: MPL-2.0

package chroot

import (
	"testing"

	"github.com/hashicorp/packer-plugin-azure/builder/azure/common/client"
	"github.com/hashicorp/packer-plugin-sdk/multistep"
	"github.com/hashicorp/packer-plugin-sdk/packerbuilderdata"
)

func Test_buildsteps_LVM(t *testing.T) {
	info := &client.ComputeInfo{
		Location:          "northpole",
		Name:              "unittestVM",
		ResourceGroupName: "unittestResourceGroup",
		SubscriptionID:    "96854241-60c7-426d-9a27-3fdeec8957f4",
	}

	tests := []struct {
		name   string
		config Config
		verify func([]multistep.Step, *testing.T)
	}{
		{
			name:   "StepSetupLVM always present",
			config: Config{Source: "diskresourceid", sourceType: sourceDisk},
			verify: func(steps []multistep.Step, t *testing.T) {
				for _, s := range steps {
					if _, ok := s.(*StepSetupLVM); ok {
						return
					}
				}
				t.Error("did not find a StepSetupLVM in step chain")
			},
		},
		{
			name: "LVMRootDevice passed to StepSetupLVM",
			config: Config{
				Source:        "diskresourceid",
				sourceType:    sourceDisk,
				LVMRootDevice: "/dev/mapper/rhel-root",
			},
			verify: func(steps []multistep.Step, t *testing.T) {
				for _, s := range steps {
					if lvm, ok := s.(*StepSetupLVM); ok {
						if lvm.LVMRootDevice != "/dev/mapper/rhel-root" {
							t.Errorf("expected LVMRootDevice to be %q, got %q",
								"/dev/mapper/rhel-root", lvm.LVMRootDevice)
						}
						return
					}
				}
				t.Error("did not find a StepSetupLVM in step chain")
			},
		},
		{
			name: "MountPartition preserved in StepMountDevice",
			config: Config{
				Source:         "diskresourceid",
				sourceType:     sourceDisk,
				MountPartition: "1",
			},
			verify: func(steps []multistep.Step, t *testing.T) {
				for _, s := range steps {
					if mount, ok := s.(*StepMountDevice); ok {
						if mount.MountPartition != "1" {
							t.Errorf("expected MountPartition to be %q, got %q",
								"1", mount.MountPartition)
						}
						return
					}
				}
				t.Error("did not find a StepMountDevice in step chain")
			},
		},
		{
			name:   "LVM step comes after StepAttachDisk and before StepMountDevice",
			config: Config{Source: "diskresourceid", sourceType: sourceDisk},
			verify: func(steps []multistep.Step, t *testing.T) {
				attachIdx, lvmIdx, mountIdx := -1, -1, -1
				for i, s := range steps {
					switch s.(type) {
					case *StepAttachDisk:
						attachIdx = i
					case *StepSetupLVM:
						lvmIdx = i
					case *StepMountDevice:
						mountIdx = i
					}
				}
				if attachIdx < 0 {
					t.Fatal("StepAttachDisk not found in step chain")
				}
				if lvmIdx < 0 {
					t.Fatal("StepSetupLVM not found in step chain")
				}
				if mountIdx < 0 {
					t.Fatal("StepMountDevice not found in step chain")
				}
				if !(attachIdx < lvmIdx && lvmIdx < mountIdx) {
					t.Errorf("incorrect step order: attach=%d, lvm=%d, mount=%d",
						attachIdx, lvmIdx, mountIdx)
				}
			},
		},
		{
			name:   "Custom StepEarlyCleanup used (not SDK version)",
			config: Config{Source: "diskresourceid", sourceType: sourceDisk},
			verify: func(steps []multistep.Step, t *testing.T) {
				for _, s := range steps {
					if _, ok := s.(*StepEarlyCleanup); ok {
						return
					}
				}
				t.Error("did not find local StepEarlyCleanup in step chain")
			},
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			withMetadataStub(func() {
				got := buildsteps(tt.config, info, &packerbuilderdata.GeneratedData{}, func(string) {})
				tt.verify(got, t)
			})
		})
	}
}
