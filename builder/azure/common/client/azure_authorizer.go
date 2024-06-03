// Copyright (c) HashiCorp, Inc.

// SPDX-License-Identifier: MPL-2.0

package client

import (
	"context"
	"fmt"

	jwt "github.com/golang-jwt/jwt"
	"github.com/hashicorp/go-azure-sdk/sdk/auth"
	"github.com/hashicorp/go-azure-sdk/sdk/environments"
)

type AzureAuthOptions struct {
	AuthType           string
	ClientID           string
	ClientSecret       string
	ClientJWT          string
	ClientCertPath     string
	ClientCertPassword string
	OidcRequestUrl     string
	OidcRequestToken   string
	TenantID           string
	SubscriptionID     string
}

func BuildResourceManagerAuthorizer(ctx context.Context, authOpts AzureAuthOptions, env environments.Environment) (auth.Authorizer, error) {
	authorizer, err := buildAuthorizer(ctx, authOpts, env, env.ResourceManager)
	if err != nil {
		return nil, fmt.Errorf("building Resource Manager authorizer from credentials: %+v", err)
	}
	return authorizer, nil
}

func BuildStorageAuthorizer(ctx context.Context, authOpts AzureAuthOptions, env environments.Environment) (auth.Authorizer, error) {
	authorizer, err := buildAuthorizer(ctx, authOpts, env, env.Storage)
	if err != nil {
		return nil, fmt.Errorf("building Storage authorizer from credentials: %+v", err)
	}
	return authorizer, nil
}

func buildAuthorizer(ctx context.Context, authOpts AzureAuthOptions, env environments.Environment, api environments.Api) (auth.Authorizer, error) {
	var authConfig auth.Credentials
	switch authOpts.AuthType {
	case AuthTypeAzureCLI:
		authConfig = auth.Credentials{
			Environment:                       env,
			EnableAuthenticatingUsingAzureCLI: true,
		}
	case AuthTypeMSI:
		authConfig = auth.Credentials{
			Environment:                              env,
			EnableAuthenticatingUsingManagedIdentity: true,
			ClientID:                                 authOpts.ClientID,
		}
	case AuthTypeClientSecret:
		authConfig = auth.Credentials{
			Environment:                           env,
			EnableAuthenticatingUsingClientSecret: true,
			ClientID:                              authOpts.ClientID,
			ClientSecret:                          authOpts.ClientSecret,
			TenantID:                              authOpts.TenantID,
		}
	case AuthTypeClientCert:
		authConfig = auth.Credentials{
			Environment: env,
			EnableAuthenticatingUsingClientCertificate: true,
			ClientID:                  authOpts.ClientID,
			TenantID:                  authOpts.TenantID,
			ClientCertificatePath:     authOpts.ClientCertPath,
			ClientCertificatePassword: authOpts.ClientCertPassword,
		}
	case AuthTypeClientBearerJWT:
		authConfig = auth.Credentials{
			Environment:                   env,
			EnableAuthenticationUsingOIDC: true,
			ClientID:                      authOpts.ClientID,
			TenantID:                      authOpts.TenantID,
			OIDCAssertionToken:            authOpts.ClientJWT,
		}
	case AuthTypeOidcURL:
		authConfig = auth.Credentials{
			Environment:                         env,
			EnableAuthenticationUsingGitHubOIDC: true,
			ClientID:                            authOpts.ClientID,
			TenantID:                            authOpts.TenantID,
			GitHubOIDCTokenRequestURL:           authOpts.OidcRequestUrl,
			GitHubOIDCTokenRequestToken:         authOpts.OidcRequestToken,
		}
	default:
		return nil, fmt.Errorf("Unexpected AuthType %s set when trying to create Azure Client", authOpts.AuthType)
	}
	authorizer, err := auth.NewAuthorizerFromCredentials(ctx, authConfig, api)
	if err != nil {
		return nil, err
	}
	return authorizer, nil
}

func GetObjectIdFromToken(token string) (string, error) {
	claims := jwt.MapClaims{}
	var p jwt.Parser

	var err error

	_, _, err = p.ParseUnverified(token, claims)

	if err != nil {
		return "", err
	}
	if claims["oid"] == nil {
		return "", fmt.Errorf("unable to parse ObjectID from Azure")
	}
	return claims["oid"].(string), nil
}
