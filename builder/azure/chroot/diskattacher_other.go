// Copyright (c) HashiCorp, Inc.
// SPDX-License-Identifier: MPL-2.0

//go:build !linux && !freebsd
// +build !linux,!freebsd

package chroot

import (
	"context"
)

func (da diskAttacher) WaitForDevice(ctx context.Context, lun int64) (device string, err error) {
	panic("The azure-chroot builder does not work on this platform.")
}
