// Copyright (c) HashiCorp, Inc.
// SPDX-License-Identifier: MPL-2.0

package common

import "github.com/hashicorp/packer-plugin-sdk/multistep"

func IsStateCancelled(stateBag multistep.StateBag) bool {
	_, ok := stateBag.GetOk(multistep.StateCancelled)
	return ok
}
