// Copyright IBM Corp. 2013, 2025
// SPDX-License-Identifier: MPL-2.0

package common

func MapToAzureTags(in map[string]string) map[string]*string {
	res := map[string]*string{}
	for k := range in {
		v := in[k]
		res[k] = &v
	}
	return res
}
