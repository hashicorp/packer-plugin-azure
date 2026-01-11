// Copyright IBM Corp. 2013, 2025
// SPDX-License-Identifier: MPL-2.0

package logutil

import "strings"

import "fmt"

type Fields map[string]interface{}

func (f Fields) String() string {
	var s strings.Builder
	for k, v := range f {
		if sv, ok := v.(string); ok {
			v = fmt.Sprintf("%q", sv)
		}
		s.WriteString(fmt.Sprintf(" %s=%v", k, v))
	}
	return s.String()
}
