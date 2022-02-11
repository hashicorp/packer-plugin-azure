package client

import (
	"github.com/Azure/go-autorest/autorest/adal"
	"github.com/Azure/go-autorest/autorest/azure"
)

// for managed identity auth
type msiOAuthTokenProvider struct {
	env      azure.Environment
	ClientID string
}

func NewMSIOAuthTokenProvider(env azure.Environment, ClientID string) oAuthTokenProvider {
	return &msiOAuthTokenProvider{env, ClientID}
}

func (tp *msiOAuthTokenProvider) getServicePrincipalToken() (*adal.ServicePrincipalToken, error) {
	return tp.getServicePrincipalTokenWithResource(tp.env.ResourceManagerEndpoint)
}

func (tp *msiOAuthTokenProvider) getServicePrincipalTokenWithResource(resource string) (*adal.ServicePrincipalToken, error) {
	return adal.NewServicePrincipalTokenFromManagedIdentity(resource, &adal.ManagedIdentityOptions{
		ClientID: tp.ClientID,
	})
}
