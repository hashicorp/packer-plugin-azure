// Copyright IBM Corp. 2013, 2025
// SPDX-License-Identifier: MPL-2.0

package chroot

import (
	"io"
	"strings"

	packersdk "github.com/hashicorp/packer-plugin-sdk/packer"
)

// testUI returns a test ui plus a function to retrieve the errors written to the ui
func testUI() (packersdk.Ui, func() string) {
	errorBuffer := &strings.Builder{}
	ui := &packersdk.BasicUi{
		Reader:      strings.NewReader(""),
		Writer:      io.Discard,
		ErrorWriter: errorBuffer,
	}
	return ui, errorBuffer.String
}
