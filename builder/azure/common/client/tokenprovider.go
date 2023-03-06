// Copyright (c) HashiCorp, Inc.
// SPDX-License-Identifier: MPL-2.0

package client

import (
	"github.com/Azure/go-autorest/autorest/adal"
)

type oAuthTokenProvider interface {
	getServicePrincipalToken() (*adal.ServicePrincipalToken, error)
	getServicePrincipalTokenWithResource(resource string) (*adal.ServicePrincipalToken, error)
}
