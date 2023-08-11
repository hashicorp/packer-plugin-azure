// Copyright (c) HashiCorp, Inc.
// SPDX-License-Identifier: MPL-2.0

//go:generate packer-sdc struct-markdown

package common

// StringPtr returns a pointer to the passed string.
func StringPtr(s string) *string {
	return &s
}

// BoolPtr returns a pointer to the passed bool.
func BoolPtr(b bool) *bool {
	return &b
}

// IntPtr returns a pointer to the passed int.
func IntPtr(i int) *int {
	return &i
}

// Int32Ptr returns a pointer to the passed int32.
func Int32Ptr(i int32) *int32 {
	return &i
}

// Int64Ptr returns a pointer to the passed int64.
func Int64Ptr(i int64) *int64 {
	return &i
}

// Float64Ptr returns a pointer to the passed float64.
func Float64Ptr(i float64) *float64 {
	return &i
}
