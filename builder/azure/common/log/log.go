// Copyright IBM Corp. 2013, 2025
// SPDX-License-Identifier: MPL-2.0

// log allows users to replicate Packer's behaviour with logging, i.e. mask
// potential secrets by replacing them if they occur with `<sensitive>`.
//
// This is intended as a drop-in replacement for the standard `log` package,
// and relies on it for final printing.
package log

import (
	"fmt"
	"log"

	"github.com/hashicorp/packer-plugin-sdk/packer"
)

func Print(v ...any) {
	raw := string(fmt.Append(nil, v...))
	log.Print(packer.LogSecretFilter.FilterString(raw))
}

func Printf(format string, v ...any) {
	raw := string(fmt.Appendf(nil, format, v...))
	log.Print(packer.LogSecretFilter.FilterString(raw))
}

func Println(v ...any) {
	raw := string(fmt.Appendln(nil, v...))
	log.Print(packer.LogSecretFilter.FilterString(raw))
}
