// Copyright IBM Corp. 2013, 2025
// SPDX-License-Identifier: MPL-2.0

package chroot

import (
	"context"
	"fmt"
	"os"
	"runtime"
	"strings"
	"testing"
	"time"

	"github.com/hashicorp/packer-plugin-sdk/common"
	"github.com/hashicorp/packer-plugin-sdk/multistep"
)

func TestStepMountDevice_Run(t *testing.T) {
	switch runtime.GOOS {
	case "linux", "freebsd":
		break
	default:
		t.Skip("Unsupported operating system")
	}
	mountPath, err := os.MkdirTemp("", "stepmountdevicetest")
	if err != nil {
		t.Fatalf("Unable to create a temporary directory: %q", err)
	}
	defer os.Remove(mountPath)

	step := &StepMountDevice{
		MountOptions:   []string{"foo"},
		MountPartition: "42",
		MountPath:      mountPath,
	}

	var gotCommand string
	var wrapper common.CommandWrapper = func(ran string) (string, error) {
		gotCommand = ran
		return "", nil
	}

	state := new(multistep.BasicStateBag)
	state.Put("wrappedCommand", wrapper)
	state.Put("device", "/dev/quux")

	ui, getErrs := testUI()
	state.Put("ui", ui)

	var config Config
	state.Put("config", &config)

	ctx, cancel := context.WithTimeout(context.Background(), time.Second)
	defer cancel()

	got := step.Run(ctx, state)
	if got != multistep.ActionContinue {
		t.Errorf("Expected 'continue', but got '%v'", got)
	}

	var expectedMountDevice string
	switch runtime.GOOS {
	case "freebsd":
		expectedMountDevice = "/dev/quuxp42"
	default: // currently just Linux
		expectedMountDevice = "/dev/quux42"
	}
	expectedCommand := fmt.Sprintf("mount -o foo %s %s", expectedMountDevice, mountPath)
	if gotCommand != expectedCommand {
		t.Errorf("Expected '%v', but got '%v'", expectedCommand, gotCommand)
	}

	_ = getErrs
}

func TestStepMountDevice_Run_LVMActive(t *testing.T) {
	switch runtime.GOOS {
	case "linux", "freebsd":
		break
	default:
		t.Skip("Unsupported operating system")
	}
	mountPath, err := os.MkdirTemp("", "stepmountdevicetest-lvm")
	if err != nil {
		t.Fatalf("Unable to create a temporary directory: %q", err)
	}
	defer os.Remove(mountPath)

	step := &StepMountDevice{
		MountOptions:   []string{"nouuid"},
		MountPartition: "1", // should be cleared by lvm_active
		MountPath:      mountPath,
	}

	var gotCommand string
	var wrapper common.CommandWrapper = func(ran string) (string, error) {
		gotCommand = ran
		return "", nil
	}

	state := new(multistep.BasicStateBag)
	state.Put("wrappedCommand", wrapper)
	state.Put("device", "/dev/mapper/rhel-root")
	state.Put("lvm_active", true) // LVM active: mount partition should be ignored

	ui, _ := testUI()
	state.Put("ui", ui)

	var config Config
	state.Put("config", &config)

	ctx, cancel := context.WithTimeout(context.Background(), time.Second)
	defer cancel()

	got := step.Run(ctx, state)
	if got != multistep.ActionContinue {
		t.Fatalf("Expected 'continue', but got '%v'", got)
	}

	// When lvm_active is set, the device should be used directly without any partition suffix
	expectedCommand := fmt.Sprintf("mount -o nouuid %s %s", "/dev/mapper/rhel-root", mountPath)
	if gotCommand != expectedCommand {
		t.Errorf("Expected %q, but got %q", expectedCommand, gotCommand)
	}

	// Verify deviceMount in state bag
	deviceMount := state.Get("deviceMount").(string)
	if deviceMount != "/dev/mapper/rhel-root" {
		t.Errorf("Expected deviceMount %q, but got %q", "/dev/mapper/rhel-root", deviceMount)
	}
}

func TestStepMountDevice_CleanupFunc_Unmount(t *testing.T) {
	switch runtime.GOOS {
	case "linux", "freebsd":
		break
	default:
		t.Skip("Unsupported operating system")
	}
	mountPath, err := os.MkdirTemp("", "stepmountdevicetest-cleanup")
	if err != nil {
		t.Fatalf("Unable to create a temporary directory: %q", err)
	}
	defer os.Remove(mountPath)

	step := &StepMountDevice{
		MountPath: mountPath,
	}

	var gotCommand string
	var wrapper common.CommandWrapper = func(ran string) (string, error) {
		gotCommand = ran
		return "", nil
	}

	state := new(multistep.BasicStateBag)
	state.Put("wrappedCommand", wrapper)
	state.Put("device", "/dev/quux")

	ui, _ := testUI()
	state.Put("ui", ui)

	var config Config
	state.Put("config", &config)

	ctx, cancel := context.WithTimeout(context.Background(), time.Second)
	defer cancel()

	got := step.Run(ctx, state)
	if got != multistep.ActionContinue {
		t.Fatalf("Expected 'continue', but got '%v'", got)
	}

	// Reset gotCommand to capture cleanup command
	gotCommand = ""

	err = step.CleanupFunc(state)
	if err != nil {
		t.Errorf("Expected nil error from CleanupFunc, got %v", err)
	}

	expectedCommand := fmt.Sprintf("umount -R %s", mountPath)
	if gotCommand != expectedCommand {
		t.Errorf("Expected %q, but got %q", expectedCommand, gotCommand)
	}

	// Second call should be a no-op
	gotCommand = ""
	err = step.CleanupFunc(state)
	if err != nil {
		t.Errorf("Expected nil error from second CleanupFunc, got %v", err)
	}
	if gotCommand != "" {
		t.Errorf("Expected no command on second CleanupFunc, got %q", gotCommand)
	}
}

func TestStepMountDevice_CleanupFunc_ManualMount_SkipsUnmount(t *testing.T) {
	switch runtime.GOOS {
	case "linux", "freebsd":
		break
	default:
		t.Skip("Unsupported operating system")
	}
	mountPath, err := os.MkdirTemp("", "stepmountdevicetest-manual-cleanup")
	if err != nil {
		t.Fatalf("Unable to create a temporary directory: %q", err)
	}
	defer os.Remove(mountPath)

	step := &StepMountDevice{
		Command:        "custom-mount-script",
		MountPath:      mountPath,
		mountPath:      mountPath,
		isManualMount:  true,
	}

	wrapperCalled := false
	var wrapper common.CommandWrapper = func(ran string) (string, error) {
		wrapperCalled = true
		return "", nil
	}

	state := new(multistep.BasicStateBag)
	state.Put("wrappedCommand", wrapper)

	ui, _ := testUI()
	state.Put("ui", ui)

	err = step.CleanupFunc(state)
	if err != nil {
		t.Errorf("Expected nil error from CleanupFunc, got %v", err)
	}

	if wrapperCalled {
		t.Errorf("Expected wrappedCommand to not be called")
	}

	if step.mountPath != "" {
		t.Errorf("Expected step.mountPath to be reset to empty string, got %q", step.mountPath)
	}
}

func TestStepMountDevice_CleanupFunc_NeverMounted(t *testing.T) {
	switch runtime.GOOS {
	case "linux", "freebsd":
		break
	default:
		t.Skip("Unsupported operating system")
	}
	step := &StepMountDevice{
		mountPath: "",
	}

	state := new(multistep.BasicStateBag)
	// No UI or wrappedCommand in state bag, so if it tries to use them it will panic

	err := step.CleanupFunc(state)
	if err != nil {
		t.Errorf("Expected nil error from CleanupFunc, got %v", err)
	}
}

func TestStepMountDevice_Run_EmptyMountPartition_NoLVM(t *testing.T) {
	switch runtime.GOOS {
	case "linux", "freebsd":
		break
	default:
		t.Skip("Unsupported operating system")
	}
	mountPath, err := os.MkdirTemp("", "stepmountdevicetest-empty-part")
	if err != nil {
		t.Fatalf("Unable to create a temporary directory: %q", err)
	}
	defer os.Remove(mountPath)

	step := &StepMountDevice{
		MountPartition: "",
		MountPath:      mountPath,
	}

	var gotCommand string
	var wrapper common.CommandWrapper = func(ran string) (string, error) {
		gotCommand = ran
		return "", nil
	}

	state := new(multistep.BasicStateBag)
	state.Put("wrappedCommand", wrapper)
	state.Put("device", "/dev/sda")

	ui, _ := testUI()
	state.Put("ui", ui)

	var config Config
	state.Put("config", &config)

	ctx, cancel := context.WithTimeout(context.Background(), time.Second)
	defer cancel()

	got := step.Run(ctx, state)
	if got != multistep.ActionContinue {
		t.Fatalf("Expected 'continue', but got '%v'", got)
	}

	expectedCommand := fmt.Sprintf("mount  %s %s", "/dev/sda", mountPath)
	if gotCommand != expectedCommand {
		t.Errorf("Expected %q, but got %q", expectedCommand, gotCommand)
	}

	deviceMount := state.Get("deviceMount").(string)
	if deviceMount != "/dev/sda" {
		t.Errorf("Expected deviceMount %q, but got %q", "/dev/sda", deviceMount)
	}
}

func TestStepMountDevice_Run_ManualMountCommand(t *testing.T) {
	switch runtime.GOOS {
	case "linux", "freebsd":
		break
	default:
		t.Skip("Unsupported operating system")
	}
	mountPath, err := os.MkdirTemp("", "stepmountdevicetest-manual")
	if err != nil {
		t.Fatalf("Unable to create a temporary directory: %q", err)
	}
	defer os.Remove(mountPath)

	step := &StepMountDevice{
		Command:   "/nonexistent-packer-test-binary-12345",
		MountPath: mountPath,
	}

	wrapperCalled := false
	var wrapper common.CommandWrapper = func(ran string) (string, error) {
		wrapperCalled = true
		return "", nil
	}

	state := new(multistep.BasicStateBag)
	state.Put("wrappedCommand", wrapper)
	state.Put("device", "/dev/quux")

	ui, _ := testUI()
	state.Put("ui", ui)

	var config Config
	state.Put("config", &config)

	ctx, cancel := context.WithTimeout(context.Background(), time.Second)
	defer cancel()

	// Note: This test verifies the manual mount code path is taken.
	// It expects ActionHalt because the command doesn't exist.
	// It does not explicitly verify that os.MkdirAll was skipped,
	// but the fact that it reaches the command execution without
	// erroring on directory creation implicitly tests this path.
	got := step.Run(ctx, state)
	if got != multistep.ActionHalt {
		t.Fatalf("Expected 'halt', but got '%v'", got)
	}

	if wrapperCalled {
		t.Errorf("Expected wrappedCommand to not be called")
	}

	errState, ok := state.GetOk("error")
	if !ok || errState == nil {
		t.Fatalf("Expected error in state bag")
	}

	errMsg := errState.(error).Error()
	if !strings.Contains(errMsg, "error mounting root volume") {
		t.Errorf("Expected error message to contain 'error mounting root volume', got %q", errMsg)
	}
}

func TestStepMountDevice_Run_WrapperError(t *testing.T) {
	switch runtime.GOOS {
	case "linux", "freebsd":
		break
	default:
		t.Skip("Unsupported operating system")
	}
	mountPath, err := os.MkdirTemp("", "stepmountdevicetest-wrapper-err")
	if err != nil {
		t.Fatalf("Unable to create a temporary directory: %q", err)
	}
	defer os.Remove(mountPath)

	step := &StepMountDevice{
		MountPath: mountPath,
	}

	var wrapper common.CommandWrapper = func(ran string) (string, error) {
		return "", fmt.Errorf("wrapper failed")
	}

	state := new(multistep.BasicStateBag)
	state.Put("wrappedCommand", wrapper)
	state.Put("device", "/dev/quux")

	ui, _ := testUI()
	state.Put("ui", ui)

	var config Config
	state.Put("config", &config)

	ctx, cancel := context.WithTimeout(context.Background(), time.Second)
	defer cancel()

	got := step.Run(ctx, state)
	if got != multistep.ActionHalt {
		t.Fatalf("Expected 'halt', but got '%v'", got)
	}

	errState, ok := state.GetOk("error")
	if !ok || errState == nil {
		t.Fatalf("Expected error in state bag")
	}

	errMsg := errState.(error).Error()
	if !strings.Contains(errMsg, "error creating mount command") {
		t.Errorf("Expected error message to contain 'error creating mount command', got %q", errMsg)
	}
}

func TestStepMountDevice_Run_StateBag_MountPath_And_Cleanup(t *testing.T) {
	switch runtime.GOOS {
	case "linux", "freebsd":
		break
	default:
		t.Skip("Unsupported operating system")
	}
	mountPath, err := os.MkdirTemp("", "stepmountdevicetest-statebag")
	if err != nil {
		t.Fatalf("Unable to create a temporary directory: %q", err)
	}
	defer os.Remove(mountPath)

	step := &StepMountDevice{
		MountPath: mountPath,
	}

	var wrapper common.CommandWrapper = func(ran string) (string, error) {
		return "", nil
	}

	state := new(multistep.BasicStateBag)
	state.Put("wrappedCommand", wrapper)
	state.Put("device", "/dev/quux")

	ui, _ := testUI()
	state.Put("ui", ui)

	var config Config
	state.Put("config", &config)

	ctx, cancel := context.WithTimeout(context.Background(), time.Second)
	defer cancel()

	got := step.Run(ctx, state)
	if got != multistep.ActionContinue {
		t.Fatalf("Expected 'continue', but got '%v'", got)
	}

	// Note: mountPath from MkdirTemp is already absolute, and the implementation
	// calls filepath.Abs(). On Linux/FreeBSD these should match. If run on a system
	// with symlinked temp dirs (e.g. macOS /tmp -> /private/tmp), they could diverge,
	// but the OS guard at the top prevents this.
	if state.Get("mount_path") != mountPath {
		t.Errorf("Expected mount_path in state bag to be %q, got %q", mountPath, state.Get("mount_path"))
	}

	if state.Get("mount_device_cleanup") != step {
		t.Errorf("Expected mount_device_cleanup in state bag to be the step instance")
	}

	expectedMountDevice := "/dev/quux"
	if state.Get("deviceMount") != expectedMountDevice {
		t.Errorf("Expected deviceMount in state bag to be %q, got %q", expectedMountDevice, state.Get("deviceMount"))
	}
}

func TestStepMountDevice_Run_MultipleMountOptions(t *testing.T) {
	switch runtime.GOOS {
	case "linux", "freebsd":
		break
	default:
		t.Skip("Unsupported operating system")
	}
	mountPath, err := os.MkdirTemp("", "stepmountdevicetest-multi-opts")
	if err != nil {
		t.Fatalf("Unable to create a temporary directory: %q", err)
	}
	defer os.Remove(mountPath)

	step := &StepMountDevice{
		MountOptions: []string{"nouuid", "ro", "noatime"},
		MountPath:    mountPath,
	}

	var gotCommand string
	var wrapper common.CommandWrapper = func(ran string) (string, error) {
		gotCommand = ran
		return "", nil
	}

	state := new(multistep.BasicStateBag)
	state.Put("wrappedCommand", wrapper)
	state.Put("device", "/dev/quux")

	ui, _ := testUI()
	state.Put("ui", ui)

	var config Config
	state.Put("config", &config)

	ctx, cancel := context.WithTimeout(context.Background(), time.Second)
	defer cancel()

	got := step.Run(ctx, state)
	if got != multistep.ActionContinue {
		t.Fatalf("Expected 'continue', but got '%v'", got)
	}

	expectedMountDevice := "/dev/quux"
	expectedCommand := fmt.Sprintf("mount -o nouuid -o ro -o noatime %s %s", expectedMountDevice, mountPath)
	if gotCommand != expectedCommand {
		t.Errorf("Expected %q, but got %q", expectedCommand, gotCommand)
	}
}

func TestStepMountDevice_Run_NoMountOptions(t *testing.T) {
	switch runtime.GOOS {
	case "linux", "freebsd":
		break
	default:
		t.Skip("Unsupported operating system")
	}
	mountPath, err := os.MkdirTemp("", "stepmountdevicetest-no-opts")
	if err != nil {
		t.Fatalf("Unable to create a temporary directory: %q", err)
	}
	defer os.Remove(mountPath)

	step := &StepMountDevice{
		MountOptions: []string{},
		MountPath:    mountPath,
	}

	var gotCommand string
	var wrapper common.CommandWrapper = func(ran string) (string, error) {
		gotCommand = ran
		return "", nil
	}

	state := new(multistep.BasicStateBag)
	state.Put("wrappedCommand", wrapper)
	state.Put("device", "/dev/quux")

	ui, _ := testUI()
	state.Put("ui", ui)

	var config Config
	state.Put("config", &config)

	ctx, cancel := context.WithTimeout(context.Background(), time.Second)
	defer cancel()

	got := step.Run(ctx, state)
	if got != multistep.ActionContinue {
		t.Fatalf("Expected 'continue', but got '%v'", got)
	}

	expectedMountDevice := "/dev/quux"
	expectedCommand := fmt.Sprintf("mount  %s %s", expectedMountDevice, mountPath)
	if gotCommand != expectedCommand {
		t.Errorf("Expected %q, but got %q", expectedCommand, gotCommand)
	}
}

func TestStepMountDevice_Run_MountPath_Interpolation_Error(t *testing.T) {
	switch runtime.GOOS {
	case "linux", "freebsd":
		break
	default:
		t.Skip("Unsupported operating system")
	}

	step := &StepMountDevice{
		MountPath: "{{.BadField}}",
	}

	state := new(multistep.BasicStateBag)
	state.Put("device", "/dev/quux")

	ui, _ := testUI()
	state.Put("ui", ui)

	var config Config
	state.Put("config", &config)

	ctx, cancel := context.WithTimeout(context.Background(), time.Second)
	defer cancel()

	got := step.Run(ctx, state)
	if got != multistep.ActionHalt {
		t.Fatalf("Expected 'halt', but got '%v'", got)
	}

	errState, ok := state.GetOk("error")
	if !ok || errState == nil {
		t.Fatalf("Expected error in state bag")
	}

	errMsg := errState.(error).Error()
	if !strings.Contains(errMsg, "error preparing mount directory") {
		t.Errorf("Expected error message to contain 'error preparing mount directory', got %q", errMsg)
	}
}

func TestStepMountDevice_Cleanup_Error(t *testing.T) {
	switch runtime.GOOS {
	case "linux", "freebsd":
		break
	default:
		t.Skip("Unsupported operating system")
	}
	mountPath, err := os.MkdirTemp("", "stepmountdevicetest-cleanup-err")
	if err != nil {
		t.Fatalf("Unable to create a temporary directory: %q", err)
	}
	defer os.Remove(mountPath)

	step := &StepMountDevice{
		MountPath: mountPath,
		mountPath: mountPath, // simulate successful mount
	}

	var wrapper common.CommandWrapper = func(ran string) (string, error) {
		return "", fmt.Errorf("wrapper failed")
	}

	state := new(multistep.BasicStateBag)
	state.Put("wrappedCommand", wrapper)

	ui, getErrs := testUI()
	state.Put("ui", ui)

	// Call the public Cleanup method, which wraps CleanupFunc and logs errors to UI
	step.Cleanup(state)

	errs := getErrs()
	if !strings.Contains(errs, "error creating unmount command") {
		t.Errorf("Expected UI error to contain 'error creating unmount command', got %q", errs)
	}
}
