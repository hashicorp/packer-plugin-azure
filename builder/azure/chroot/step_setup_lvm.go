// Copyright IBM Corp. 2013, 2025
// SPDX-License-Identifier: MPL-2.0

//go:build linux || freebsd

package chroot

import (
	"bytes"
	"context"
	"fmt"
	"os/exec"
	"strings"
	"time"

	"github.com/hashicorp/packer-plugin-azure/builder/azure/common/log"

	"github.com/hashicorp/packer-plugin-sdk/multistep"
	packersdk "github.com/hashicorp/packer-plugin-sdk/packer"
)

var _ multistep.Step = &StepSetupLVM{}

// StepSetupLVM detects and activates LVM volume groups on the attached disk,
// replacing the "device" state bag entry with the root logical volume path.
// If no LVM is detected, this step is a no-op.
type StepSetupLVM struct {
	// device is populated from the state bag ("device") at runtime.
	device string

	// volumeGroups that were activated by this step (for cleanup).
	volumeGroups []string

	// LVMRootDevice override — if set by the user, skip auto-detection
	// and use this path directly (e.g. "/dev/mapper/rhel-root").
	LVMRootDevice string

	// activated tracks whether LVM was detected and activated (for cleanup).
	activated bool
}

func (s *StepSetupLVM) Run(ctx context.Context, state multistep.StateBag) multistep.StepAction {
	ui := state.Get("ui").(packersdk.Ui)
	device := state.Get("device").(string)
	s.device = device

	if s.LVMRootDevice != "" {
		// Manual override path
		ui.Say(fmt.Sprintf("LVM: using user-specified root device: %s", s.LVMRootDevice))

		vgs, err := s.detectVolumeGroups(device)
		if err != nil {
			log.Printf("LVM: warning: could not detect volume groups for cleanup scoping: %v", err)
			// Without VG names we cannot safely scope vgchange; try to
			// extract the VG name from the user-specified device path.
			if vg := vgFromDevicePath(s.LVMRootDevice); vg != "" {
				s.volumeGroups = []string{vg}
				log.Printf("LVM: inferred VG %q from lvm_root_device", vg)
			}
		} else {
			s.volumeGroups = vgs
		}

		if len(s.volumeGroups) == 0 {
			return s.halt(state, ui, fmt.Errorf(
				"LVM: cannot activate volume groups: unable to detect VGs on %s and unable to infer VG from lvm_root_device %q; "+
					"ensure the source disk contains LVM physical volumes",
				device, s.LVMRootDevice))
		}

		if err := s.activateVolumeGroups(ui); err != nil {
			return s.halt(state, ui, fmt.Errorf("LVM: error activating volume groups: %v", err))
		}

		state.Put("device", s.LVMRootDevice)
		state.Put("lvm_active", true)
		state.Put("lvm_cleanup", s)
		return multistep.ActionContinue
	}

	// Auto-detection path
	ui.Say("LVM: scanning attached disk for LVM physical volumes...")

	vgs, err := s.detectVolumeGroups(device)
	if err != nil {
		log.Printf("LVM: warning: error detecting volume groups: %v", err)
		ui.Say("LVM: could not detect volume groups, continuing without LVM support")
		state.Put("lvm_cleanup", s)
		return multistep.ActionContinue
	}

	if len(vgs) == 0 {
		ui.Say("LVM: no volume groups found on attached disk, continuing without LVM")
		state.Put("lvm_cleanup", s)
		return multistep.ActionContinue
	}

	s.volumeGroups = vgs
	ui.Say(fmt.Sprintf("LVM: found volume group(s): %s", strings.Join(vgs, ", ")))

	if err := s.activateVolumeGroups(ui); err != nil {
		return s.halt(state, ui, fmt.Errorf("LVM: error activating volume groups: %v", err))
	}

	rootLV, err := s.findRootLV(vgs, ui)
	if err != nil {
		return s.halt(state, ui, fmt.Errorf("LVM: error finding root logical volume: %v", err))
	}

	s.verifyDevice(ui, rootLV)

	ui.Say(fmt.Sprintf("LVM: using root logical volume: %s", rootLV))
	state.Put("device", rootLV)
	state.Put("lvm_active", true)
	state.Put("lvm_cleanup", s)
	return multistep.ActionContinue
}

// detectVolumeGroups ensures partition device nodes are visible, then scans for
// LVM physical volumes on the attached disk.
func (s *StepSetupLVM) detectVolumeGroups(device string) ([]string, error) {
	// Run partprobe to ensure partition device nodes exist
	if out, err := exec.Command("partprobe", device).CombinedOutput(); err != nil {
		log.Printf("LVM: partprobe %s: %v (output: %s)", device, err, strings.TrimSpace(string(out)))
	}

	// Wait for udev to settle
	if out, err := exec.Command("udevadm", "settle", "--timeout=10").CombinedOutput(); err != nil {
		log.Printf("LVM: udevadm settle: %v (output: %s)", err, strings.TrimSpace(string(out)))
	}

	// Refresh PV cache
	if out, err := exec.Command("pvscan", "--cache").CombinedOutput(); err != nil {
		log.Printf("LVM: pvscan --cache: %v (output: %s)", err, strings.TrimSpace(string(out)))
	}

	// Retry loop: Azure disk attachment is asynchronous — partition device nodes
	// may not exist immediately after the disk appears.
	for attempt := 0; attempt < 3; attempt++ {
		if attempt > 0 {
			log.Printf("LVM: retry attempt %d/3, waiting for device nodes...", attempt+1)
			time.Sleep(2 * time.Second)

			if out, err := exec.Command("partprobe", device).CombinedOutput(); err != nil {
				log.Printf("LVM: partprobe %s (retry): %v (output: %s)", device, err, strings.TrimSpace(string(out)))
			}
			if out, err := exec.Command("udevadm", "settle", "--timeout=5").CombinedOutput(); err != nil {
				log.Printf("LVM: udevadm settle (retry): %v (output: %s)", err, strings.TrimSpace(string(out)))
			}
			if out, err := exec.Command("pvscan", "--cache").CombinedOutput(); err != nil {
				log.Printf("LVM: pvscan --cache (retry): %v (output: %s)", err, strings.TrimSpace(string(out)))
			}
		}

		vgs, err := s.scanPVS(device)
		if err != nil {
			log.Printf("LVM: scanPVS attempt %d: %v", attempt+1, err)
			continue
		}
		if len(vgs) > 0 {
			return vgs, nil
		}
	}

	return nil, nil
}

// scanPVS runs `pvs` and returns VG names for PVs that belong to the specified device.
func (s *StepSetupLVM) scanPVS(device string) ([]string, error) {
	cmd := exec.Command("pvs", "--noheadings", "--nosuffix", "-o", "pv_name,vg_name", "--separator", ",")
	var stdout, stderr bytes.Buffer
	cmd.Stdout = &stdout
	cmd.Stderr = &stderr

	if err := cmd.Run(); err != nil {
		return nil, fmt.Errorf("pvs: %v (stderr: %s)", err, strings.TrimSpace(stderr.String()))
	}

	seen := make(map[string]bool)
	var vgs []string

	for _, line := range strings.Split(stdout.String(), "\n") {
		line = strings.TrimSpace(line)
		if line == "" {
			continue
		}
		parts := strings.SplitN(line, ",", 2)
		if len(parts) != 2 {
			continue
		}
		pvName := strings.TrimSpace(parts[0])
		vgName := strings.TrimSpace(parts[1])

		// Scope to our disk only: check that the PV is on the attached device
		if !strings.HasPrefix(pvName, device) {
			continue
		}

		if vgName != "" && !seen[vgName] {
			seen[vgName] = true
			vgs = append(vgs, vgName)
		}
	}

	return vgs, nil
}

// activateVolumeGroups activates the discovered volume groups.
func (s *StepSetupLVM) activateVolumeGroups(ui packersdk.Ui) error {
	// Scan for VGs first
	if out, err := exec.Command("vgscan").CombinedOutput(); err != nil {
		log.Printf("LVM: vgscan: %v (output: %s)", err, strings.TrimSpace(string(out)))
	}

	// Build vgchange args, scoping to our VGs if known
	args := []string{"-ay"}
	if len(s.volumeGroups) > 0 {
		args = append(args, s.volumeGroups...)
	}

	cmd := exec.Command("vgchange", args...)
	var stdout, stderr bytes.Buffer
	cmd.Stdout = &stdout
	cmd.Stderr = &stderr

	if err := cmd.Run(); err != nil {
		return fmt.Errorf("vgchange %s: %v (stdout: %s, stderr: %s)",
			strings.Join(args, " "), err,
			strings.TrimSpace(stdout.String()),
			strings.TrimSpace(stderr.String()))
	}
	s.activated = true

	// Ensure /dev/mapper/ nodes exist
	if out, err := exec.Command("vgmknodes").CombinedOutput(); err != nil {
		log.Printf("LVM: vgmknodes: %v (output: %s)", err, strings.TrimSpace(string(out)))
	}

	// Wait for udev to settle
	if out, err := exec.Command("udevadm", "settle", "--timeout=10").CombinedOutput(); err != nil {
		log.Printf("LVM: udevadm settle: %v (output: %s)", err, strings.TrimSpace(string(out)))
	}

	ui.Say("LVM: volume groups activated successfully")
	return nil
}

// lvInfo holds parsed information about a logical volume.
type lvInfo struct {
	name string
	vg   string
	path string
	attr string
}

// findRootLV identifies the root logical volume from the activated volume groups.
func (s *StepSetupLVM) findRootLV(vgs []string, ui packersdk.Ui) (string, error) {
	cmd := exec.Command("lvs", "--noheadings", "--nosuffix", "-o", "lv_name,vg_name,lv_path,lv_attr", "--separator", ",")
	var stdout, stderr bytes.Buffer
	cmd.Stdout = &stdout
	cmd.Stderr = &stderr

	if err := cmd.Run(); err != nil {
		return "", fmt.Errorf("lvs: %v (stderr: %s)", err, strings.TrimSpace(stderr.String()))
	}

	vgSet := make(map[string]bool)
	for _, vg := range vgs {
		vgSet[vg] = true
	}

	var allLVs []lvInfo
	for _, line := range strings.Split(stdout.String(), "\n") {
		line = strings.TrimSpace(line)
		if line == "" {
			continue
		}
		parts := strings.SplitN(line, ",", 4)
		if len(parts) != 4 {
			continue
		}
		lv := lvInfo{
			name: strings.TrimSpace(parts[0]),
			vg:   strings.TrimSpace(parts[1]),
			path: strings.TrimSpace(parts[2]),
			attr: strings.TrimSpace(parts[3]),
		}
		if vgSet[lv.vg] {
			allLVs = append(allLVs, lv)
		}
	}

	// Log all discovered LVs
	for _, lv := range allLVs {
		log.Printf("LVM: discovered LV: name=%s vg=%s path=%s attr=%s", lv.name, lv.vg, lv.path, lv.attr)
	}

	// Filter by LV type: exclude non-mountable volumes
	var mountable []lvInfo
	for _, lv := range allLVs {
		if isMountableLV(lv.attr) {
			mountable = append(mountable, lv)
		} else {
			log.Printf("LVM: skipping LV %s (%s): %s", lv.name, lv.path, lvTypeDescription(lv.attr))
		}
	}

	if len(mountable) == 0 {
		return "", fmt.Errorf("no mountable logical volumes found in volume groups %v", vgs)
	}

	// Filter out swap volumes using blkid
	var nonSwap []lvInfo
	for _, lv := range mountable {
		fsType := blkidType(lv.path)
		if strings.EqualFold(fsType, "swap") {
			log.Printf("LVM: skipping LV %s (%s): swap filesystem", lv.name, lv.path)
			continue
		}
		nonSwap = append(nonSwap, lv)
	}

	// If blkid filtered everything, fall back to attribute-filtered list
	if len(nonSwap) == 0 {
		log.Printf("LVM: blkid filtered all candidates, falling back to attribute-filtered list")
		nonSwap = mountable
	}

	// If exactly one candidate, return it
	if len(nonSwap) == 1 {
		return nonSwap[0].path, nil
	}

	// Multiple candidates: try exact name-based matching first
	for _, lv := range nonSwap {
		nameLower := strings.ToLower(lv.name)
		if nameLower == "root" || nameLower == "lv_root" || nameLower == "rootlv" || nameLower == "lvroot" {
			ui.Say(fmt.Sprintf("LVM: selected root LV by exact name match: %s (%s)", lv.name, lv.path))
			return lv.path, nil
		}
	}

	// Fallback: partial name match containing "root"
	for _, lv := range nonSwap {
		if strings.Contains(strings.ToLower(lv.name), "root") {
			ui.Say(fmt.Sprintf("LVM: selected root LV by partial name match: %s (%s)", lv.name, lv.path))
			return lv.path, nil
		}
	}

	// No name match: warn user and return the first candidate
	ui.Say("WARNING: LVM: multiple logical volumes found, unable to determine root by name:")
	for _, lv := range nonSwap {
		fsType := blkidType(lv.path)
		if fsType == "" {
			fsType = "unknown"
		}
		ui.Say(fmt.Sprintf("  - %s (%s) [fs: %s]", lv.name, lv.path, fsType))
	}
	ui.Say(fmt.Sprintf("LVM: selecting first candidate: %s", nonSwap[0].path))
	ui.Say("LVM: if this is incorrect, set 'lvm_root_device' in your Packer template")
	return nonSwap[0].path, nil
}

// isMountableLV returns true unless the first character of lv_attr indicates
// a non-mountable LV type.
func isMountableLV(attr string) bool {
	if attr == "" {
		return true
	}
	switch attr[0] {
	case 's', 'S': // snapshot
		return false
	case 'v', 'V': // virtual (e.g. thin snapshot device)
		return false
	case 't', 'T': // thin pool
		return false
	case 'e', 'E': // RAID/pool metadata
		return false
	case 'i', 'I': // internal
		return false
	case 'l', 'L': // mirror log
		return false
	case 'd', 'D': // mirror/RAID image
		return false
	case 'p': // pvmove
		return false
	default:
		return true
	}
}

// lvTypeDescription returns a human-readable description of the LV type.
func lvTypeDescription(attr string) string {
	if attr == "" {
		return "unknown type"
	}
	switch attr[0] {
	case 's', 'S':
		return "snapshot"
	case 'v', 'V':
		return "virtual volume"
	case 't', 'T':
		return "thin pool"
	case 'e', 'E':
		return "RAID/pool metadata"
	case 'i', 'I':
		return "internal volume"
	case 'l', 'L':
		return "mirror log"
	case 'd', 'D':
		return "mirror/RAID image"
	case 'p':
		return "pvmove volume"
	default:
		return "unknown type"
	}
}

// blkidType runs `blkid -o value -s TYPE` on the device and returns the filesystem type.
func blkidType(device string) string {
	out, err := exec.Command("blkid", "-o", "value", "-s", "TYPE", device).Output()
	if err != nil {
		return ""
	}
	return strings.TrimSpace(string(out))
}

// verifyDevice checks that the selected LV is readable by blkid, attempting
// a refresh if not.
func (s *StepSetupLVM) verifyDevice(ui packersdk.Ui, device string) {
	fsType := blkidType(device)
	if fsType != "" {
		log.Printf("LVM: device %s has filesystem type: %s", device, fsType)
		return
	}

	log.Printf("LVM: device %s not immediately readable by blkid, attempting refresh...", device)

	// Try to determine VG/LV for lvchange --refresh
	vgLV := resolveVGLV(device)
	if vgLV != "" {
		log.Printf("LVM: running lvchange --refresh %s", vgLV)
		if out, err := exec.Command("lvchange", "--refresh", vgLV).CombinedOutput(); err != nil {
			log.Printf("LVM: lvchange --refresh %s: %v (output: %s)", vgLV, err, strings.TrimSpace(string(out)))
		}
	}

	// Wait for udev and retry
	if out, err := exec.Command("udevadm", "settle", "--timeout=10").CombinedOutput(); err != nil {
		log.Printf("LVM: udevadm settle: %v (output: %s)", err, strings.TrimSpace(string(out)))
	}
	time.Sleep(1 * time.Second)

	fsType = blkidType(device)
	if fsType != "" {
		log.Printf("LVM: device %s now has filesystem type: %s (after refresh)", device, fsType)
		return
	}

	// Still unreadable — dump diagnostics
	ui.Say(fmt.Sprintf("WARNING: LVM: device %s is not readable by blkid after refresh", device))

	if out, err := exec.Command("lvs", "-a").CombinedOutput(); err != nil {
		log.Printf("LVM: lvs -a: %v (output: %s)", err, strings.TrimSpace(string(out)))
	} else {
		log.Printf("LVM: lvs -a output:\n%s", string(out))
	}

	// For dmsetup table, extract the dm name from the device path
	dmName := device
	if strings.HasPrefix(device, "/dev/mapper/") {
		dmName = strings.TrimPrefix(device, "/dev/mapper/")
	}
	if out, err := exec.Command("dmsetup", "table", dmName).CombinedOutput(); err != nil {
		log.Printf("LVM: dmsetup table %s: %v (output: %s)", dmName, err, strings.TrimSpace(string(out)))
	} else {
		log.Printf("LVM: dmsetup table %s output:\n%s", dmName, string(out))
	}
}

// resolveVGLV determines the VG/LV identifier from a device path for use with
// lvchange --refresh. It handles both /dev/mapper/ and /dev/<vg>/<lv> paths.
func resolveVGLV(device string) string {
	if strings.HasPrefix(device, "/dev/mapper/") {
		basename := device[len("/dev/mapper/"):]
		// Try dmsetup splitname to correctly handle double-dash escaping
		out, err := exec.Command("dmsetup", "splitname", "--noheadings", "--separator", "/", basename, "LVM").Output()
		if err == nil {
			// dmsetup splitname output format: "  vg/lv/layer" or "  vg/lv/"
			result := strings.TrimSpace(string(out))
			parts := strings.SplitN(result, "/", 3)
			if len(parts) >= 2 && parts[0] != "" && parts[1] != "" {
				return parts[0] + "/" + parts[1]
			}
		}
		log.Printf("LVM: dmsetup splitname failed or unavailable, falling back to heuristic for %s", basename)

		// Fallback heuristic: find the last single-dash boundary
		// LVM uses double-dashes to escape dashes in VG/LV names, so
		// "my--vg-root" means VG="my-vg", LV="root"
		// We look for a dash that is NOT preceded or followed by another dash.
		lastIdx := -1
		for i := 1; i < len(basename)-1; i++ {
			if basename[i] == '-' && basename[i-1] != '-' && basename[i+1] != '-' {
				lastIdx = i
			}
		}
		if lastIdx > 0 {
			vg := strings.ReplaceAll(basename[:lastIdx], "--", "-")
			lv := strings.ReplaceAll(basename[lastIdx+1:], "--", "-")
			return vg + "/" + lv
		}
		return ""
	}

	// /dev/<vg>/<lv> style path
	parts := strings.Split(device, "/")
	if len(parts) >= 4 {
		// parts: ["", "dev", "<vg>", "<lv>"]
		return parts[len(parts)-2] + "/" + parts[len(parts)-1]
	}

	return ""
}

// vgFromDevicePath extracts just the VG name from a device path.
// For /dev/mapper/rhel-root → "rhel", for /dev/rhel/root → "rhel".
func vgFromDevicePath(device string) string {
	vgLV := resolveVGLV(device)
	if vgLV == "" {
		return ""
	}
	parts := strings.SplitN(vgLV, "/", 2)
	if len(parts) >= 1 && parts[0] != "" {
		return parts[0]
	}
	return ""
}

func (s *StepSetupLVM) halt(state multistep.StateBag, ui packersdk.Ui, err error) multistep.StepAction {
	log.Printf("LVM: error: %v", err)
	state.Put("error", err)
	ui.Error(err.Error())
	return multistep.ActionHalt
}

func (s *StepSetupLVM) Cleanup(state multistep.StateBag) {
	if err := s.CleanupFunc(state); err != nil {
		ui := state.Get("ui").(packersdk.Ui)
		ui.Error(err.Error())
	}
}

// CleanupFunc deactivates LVM volume groups that were activated by this step.
func (s *StepSetupLVM) CleanupFunc(state multistep.StateBag) error {
	if !s.activated {
		return nil
	}

	ui := state.Get("ui").(packersdk.Ui)
	ui.Say(fmt.Sprintf("LVM: deactivating volume groups: %s", strings.Join(s.volumeGroups, ", ")))

	args := []string{"-an"}
	if len(s.volumeGroups) > 0 {
		args = append(args, s.volumeGroups...)
	}

	cmd := exec.Command("vgchange", args...)
	var stdout, stderr bytes.Buffer
	cmd.Stdout = &stdout
	cmd.Stderr = &stderr

	if err := cmd.Run(); err != nil {
		return fmt.Errorf("LVM: vgchange %s: %v (stdout: %s, stderr: %s)",
			strings.Join(args, " "), err,
			strings.TrimSpace(stdout.String()),
			strings.TrimSpace(stderr.String()))
	}

	s.activated = false
	ui.Say("LVM: volume groups deactivated")
	return nil
}
