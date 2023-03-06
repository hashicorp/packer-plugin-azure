// Copyright (c) HashiCorp, Inc.
// SPDX-License-Identifier: MPL-2.0

package client

import (
	"github.com/Azure/go-autorest/autorest/adal"
	"github.com/Azure/go-autorest/autorest/azure"
)

// for managed identity auth
type msiOAuthTokenProvider struct {
	env      azure.Environment
	clientID string
}

func NewMSIOAuthTokenProvider(env azure.Environment, clientID string) oAuthTokenProvider {
	return &msiOAuthTokenProvider{env: env, clientID: clientID}
}

func (tp *msiOAuthTokenProvider) getServicePrincipalToken() (*adal.ServicePrincipalToken, error) {
	return tp.getServicePrincipalTokenWithResource(tp.env.ResourceManagerEndpoint)
}

func (tp *msiOAuthTokenProvider) getServicePrincipalTokenWithResource(resource string) (*adal.ServicePrincipalToken, error) {
	return adal.NewServicePrincipalTokenFromManagedIdentity(resource, &adal.ManagedIdentityOptions{
		ClientID: tp.clientID,
	})
}
