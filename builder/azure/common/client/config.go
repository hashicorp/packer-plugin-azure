// Copyright (c) HashiCorp, Inc.
// SPDX-License-Identifier: MPL-2.0

//go:generate packer-sdc struct-markdown

package client

import (
	"context"
	"fmt"
	"log"
	"os"
	"strings"
	"time"

	"github.com/Azure/go-autorest/autorest/adal"
	"github.com/Azure/go-autorest/autorest/azure"
	jwt "github.com/golang-jwt/jwt"
	"github.com/hashicorp/go-azure-helpers/authentication"
	packersdk "github.com/hashicorp/packer-plugin-sdk/packer"
)

// Config allows for various ways to authenticate Azure clients.  When
// `client_id` and `subscription_id` are specified in addition to one and only
// one of the following: `client_secret`, `client_jwt`, `client_cert_path` --
// Packer will use the specified Azure Active Directory (AAD) Service Principal
// (SP).  If only `use_interactive_auth` is specified, Packer will try to
// interactively log on the current user (tokens will be cached).  If none of
// these options are specified, Packer will attempt to use the Managed Identity
// and subscription of the VM that Packer is running on.  This will only work if
// Packer is running on an Azure VM with either a System Assigned Managed
// Identity or User Assigned Managed Identity.
type Config struct {
	// One of Public, China, Germany, or
	// USGovernment. Defaults to Public. Long forms such as
	// USGovernmentCloud and AzureUSGovernmentCloud are also supported.
	CloudEnvironmentName string `mapstructure:"cloud_environment_name" required:"false"`
	cloudEnvironment     *azure.Environment
	// The Hostname of the Azure Metadata Service
	// (for example management.azure.com), used to obtain the Cloud Environment
	// when using a Custom Azure Environment. This can also be sourced from the
	// ARM_METADATA_HOST Environment Variable.
	// Note: CloudEnvironmentName must be set to the requested environment
	// name in the list of available environments held in the metadata_host.
	MetadataHost string `mapstructure:"metadata_host" required:"false"`

	// Authentication fields

	// The application ID of the AAD Service Principal.
	// Requires either `client_secret`, `client_cert_path` or `client_jwt` to be set as well.
	ClientID string `mapstructure:"client_id"`
	// A password/secret registered for the AAD SP.
	ClientSecret string `mapstructure:"client_secret"`
	// The path to a pem-encoded certificate that will be used to authenticate
	// as the specified AAD SP.
	ClientCertPath string `mapstructure:"client_cert_path"`
	// The timeout for the JWT Token when using a [client certificate](#client_cert_path). Defaults to 1 hour.
	ClientCertExpireTimeout time.Duration `mapstructure:"client_cert_token_timeout" required:"false"`
	// A JWT bearer token for client auth (RFC 7523, Sec. 2.2) that will be used
	// to authenticate the AAD SP. Provides more control over token the expiration
	// when using certificate authentication than when using `client_cert_path`.
	ClientJWT string `mapstructure:"client_jwt"`
	// The object ID for the AAD SP. Optional, will be derived from the oAuth token if left empty.
	ObjectID string `mapstructure:"object_id"`

	// The Active Directory tenant identifier with which your `client_id` and
	// `subscription_id` are associated. If not specified, `tenant_id` will be
	// looked up using `subscription_id`.
	TenantID string `mapstructure:"tenant_id" required:"false"`
	// The subscription to use.
	SubscriptionID string `mapstructure:"subscription_id"`

	authType string

	// Flag to use Azure CLI authentication. Defaults to false.
	// CLI auth will use the information from an active `az login` session to connect to Azure and set the subscription id and tenant id associated to the signed in account.
	// If enabled, it will use the authentication provided by the `az` CLI.
	// Azure CLI authentication will use the credential marked as `isDefault` and can be verified using `az account show`.
	// Works with normal authentication (`az login`) and service principals (`az login --service-principal --username APP_ID --password PASSWORD --tenant TENANT_ID`).
	// Ignores all other configurations if enabled.
	UseAzureCLIAuth bool `mapstructure:"use_azure_cli_auth" required:"false"`
	// Flag to use interactive login (use device code) authentication. Defaults to false.
	// If enabled, it will use interactive authentication.
	UseInteractiveAuth bool `mapstructure:"use_interactive_auth" required:"false"`
}

const (
	authTypeDeviceLogin     = "DeviceLogin"
	authTypeMSI             = "ManagedIdentity"
	authTypeClientSecret    = "ClientSecret"
	authTypeClientCert      = "ClientCertificate"
	authTypeClientBearerJWT = "ClientBearerJWT"
	authTypeAzureCLI        = "AzureCLI"
)

const DefaultCloudEnvironmentName = "Public"

// CloudEnvironmentName is deprecated in favor of MetadataHost. This is retained
// for now to preserve backward compatability, but should eventually be removed.
func (c *Config) SetDefaultValues() error {
	if c.CloudEnvironmentName == "" {
		c.CloudEnvironmentName = DefaultCloudEnvironmentName
	}

	return c.setCloudEnvironment()
}

func (c *Config) CloudEnvironment() *azure.Environment {
	return c.cloudEnvironment
}

func (c *Config) setCloudEnvironment() error {
	// First, try using the metadata host to look up the cloud.
	if c.MetadataHost == "" {
		if v := os.Getenv("ARM_METADATA_URL"); v != "" {
			c.MetadataHost = v
		}
	}

	env, err := authentication.AzureEnvironmentByNameFromEndpoint(context.TODO(), c.MetadataHost, c.CloudEnvironmentName)
	c.cloudEnvironment = env

	if err != nil {
		// fall back to old method of normalizing and looking up cloud names.
		log.Printf(fmt.Sprintf("Error looking up environment using metadata host: %s. \n"+
			"Falling back to hardcoded mechanism...", err.Error()))
		lookup := map[string]string{
			"CHINA":           "AzureChinaCloud",
			"CHINACLOUD":      "AzureChinaCloud",
			"AZURECHINACLOUD": "AzureChinaCloud",

			"GERMAN":           "AzureGermanCloud",
			"GERMANCLOUD":      "AzureGermanCloud",
			"AZUREGERMANCLOUD": "AzureGermanCloud",

			"GERMANY":           "AzureGermanCloud",
			"GERMANYCLOUD":      "AzureGermanCloud",
			"AZUREGERMANYCLOUD": "AzureGermanCloud",

			"PUBLIC":           "AzurePublicCloud",
			"PUBLICCLOUD":      "AzurePublicCloud",
			"AZUREPUBLICCLOUD": "AzurePublicCloud",

			"USGOVERNMENT":           "AzureUSGovernmentCloud",
			"USGOVERNMENTCLOUD":      "AzureUSGovernmentCloud",
			"AZUREUSGOVERNMENTCLOUD": "AzureUSGovernmentCloud",
		}

		name := strings.ToUpper(c.CloudEnvironmentName)
		envName, ok := lookup[name]
		if !ok {
			return fmt.Errorf("There is no cloud environment matching the name '%s'!", c.CloudEnvironmentName)
		}

		env, err := azure.EnvironmentFromName(envName)
		if err != nil {
			return err
		}
		c.cloudEnvironment = &env
	}

	return nil
}

//nolint:ineffassign //this triggers a false positive because errs is passed by reference
func (c Config) Validate(errs *packersdk.MultiError) {
	/////////////////////////////////////////////
	// Authentication via OAUTH

	// Check if device login is being asked for, and is allowed.
	//
	// Device login is enabled if the user only defines SubscriptionID and not
	// ClientID, ClientSecret, and TenantID.
	//
	// Device login is not enabled for Windows because the WinRM certificate is
	// readable by the ObjectID of the App.  There may be another way to handle
	// this case, but I am not currently aware of it - send feedback.

	if c.UseCLI() {
		return
	}

	if c.UseMSI() {
		return
	}

	if c.useDeviceLogin() {
		return
	}

	if c.SubscriptionID != "" && c.ClientID != "" &&
		c.ClientSecret != "" &&
		c.ClientCertPath == "" &&
		c.ClientJWT == "" {
		// Service principal using secret
		return
	}

	if c.SubscriptionID != "" && c.ClientID != "" &&
		c.ClientSecret == "" &&
		c.ClientCertPath != "" &&
		c.ClientJWT == "" {
		// Service principal using certificate

		if _, err := os.Stat(c.ClientCertPath); err != nil {
			errs = packersdk.MultiErrorAppend(errs, fmt.Errorf("client_cert_path is not an accessible file: %v", err))
		}
		if c.ClientCertExpireTimeout != 0 && c.ClientCertExpireTimeout < 5*time.Minute {
			errs = packersdk.MultiErrorAppend(errs, fmt.Errorf("client_cert_token_timeout will expire within 5 minutes, please set a value greater than 5 minutes"))
		}
		return
	}

	if c.SubscriptionID != "" && c.ClientID != "" &&
		c.ClientSecret == "" &&
		c.ClientCertPath == "" &&
		c.ClientJWT != "" {
		// Service principal using JWT
		// Check that JWT is valid for at least 5 more minutes

		p := jwt.Parser{}
		claims := jwt.StandardClaims{}
		token, _, err := p.ParseUnverified(c.ClientJWT, &claims)
		if err != nil {
			errs = packersdk.MultiErrorAppend(errs, fmt.Errorf("client_jwt is not a JWT: %v", err))
		} else {
			if claims.ExpiresAt < time.Now().Add(5*time.Minute).Unix() {
				errs = packersdk.MultiErrorAppend(errs, fmt.Errorf("client_jwt will expire within 5 minutes, please use a JWT that is valid for at least 5 minutes"))
			}
			if t, ok := token.Header["x5t"]; !ok || t == "" {
				errs = packersdk.MultiErrorAppend(errs, fmt.Errorf("client_jwt is missing the x5t header value, which is required for bearer JWT client authentication to Azure"))
			}
		}

		return
	}

	errs = packersdk.MultiErrorAppend(errs, fmt.Errorf("No valid set of authentication values specified:\n"+
		"  to use the Managed Identity of the current machine, do not specify any of the fields below:\n"+
		"  - client_secret\n"+
		"  - client_jwt\n"+
		"  - client_cert_path\n"+
		"  - use_interactive_auth\n"+
		"  - use_azure_cli_auth\n"+
		"  to use interactive user authentication, specify only the following fields:\n"+
		"  - subscription_id\n"+
		"  - use_interactive_auth\n"+
		"  to use an Azure Active Directory service principal, specify either:\n"+
		"  - subscription_id, client_id and client_secret\n"+
		"  - subscription_id, client_id and client_cert_path\n"+
		"  - subscription_id, client_id and client_jwt."))
}

func (c Config) useDeviceLogin() bool {
	return c.UseInteractiveAuth
}

func (c Config) UseCLI() bool {
	return c.UseAzureCLIAuth
}

func (c Config) UseMSI() bool {
	return !c.UseInteractiveAuth &&
		!c.UseAzureCLIAuth &&
		c.ClientSecret == "" &&
		c.ClientJWT == "" &&
		c.ClientCertPath == "" &&
		c.TenantID == ""
}

func (c Config) GetServicePrincipalTokens(say func(string)) (
	servicePrincipalToken *adal.ServicePrincipalToken,
	servicePrincipalTokenVault *adal.ServicePrincipalToken,
	err error) {

	servicePrincipalToken, err = c.GetServicePrincipalToken(say,
		c.CloudEnvironment().ResourceManagerEndpoint)
	if err != nil {
		return nil, nil, err
	}
	servicePrincipalTokenVault, err = c.GetServicePrincipalToken(say,
		strings.TrimRight(c.CloudEnvironment().KeyVaultEndpoint, "/"))
	if err != nil {
		return nil, nil, err
	}
	return servicePrincipalToken, servicePrincipalTokenVault, nil
}

func (c Config) GetServicePrincipalToken(
	say func(string), forResource string) (
	servicePrincipalToken *adal.ServicePrincipalToken,
	err error) {

	var auth oAuthTokenProvider
	switch c.authType {
	case authTypeDeviceLogin:
		say("Getting tokens using device flow")
		auth = NewDeviceFlowOAuthTokenProvider(*c.cloudEnvironment, say, c.TenantID)
	case authTypeAzureCLI:
		say("Getting tokens using Azure CLI")
		auth = NewCliOAuthTokenProvider(*c.cloudEnvironment, say, c.TenantID)
	case authTypeMSI:
		say("Getting tokens using Managed Identity for Azure")
		auth = NewMSIOAuthTokenProvider(*c.cloudEnvironment, c.ClientID)
	case authTypeClientSecret:
		say("Getting tokens using client secret")
		auth = NewSecretOAuthTokenProvider(*c.cloudEnvironment, c.ClientID, c.ClientSecret, c.TenantID)
	case authTypeClientCert:
		say("Getting tokens using client certificate")
		auth, err = NewCertOAuthTokenProvider(*c.cloudEnvironment, c.ClientID, c.ClientCertPath, c.TenantID, c.ClientCertExpireTimeout)
		if err != nil {
			return nil, err
		}
	case authTypeClientBearerJWT:
		say("Getting tokens using client bearer JWT")
		auth = NewJWTOAuthTokenProvider(*c.cloudEnvironment, c.ClientID, c.ClientJWT, c.TenantID)
	default:
		panic("authType not set, call FillParameters, or set explicitly")
	}

	servicePrincipalToken, err = auth.getServicePrincipalTokenWithResource(forResource)
	if err != nil {
		return nil, err
	}

	err = servicePrincipalToken.EnsureFresh()
	if err != nil {
		return nil, err
	}

	return servicePrincipalToken, nil
}

// FillParameters capture the user intent from the supplied parameter set in authType, retrieves the TenantID and CloudEnvironment if not specified.
// The SubscriptionID is also retrieved in case MSI auth is requested.
func (c *Config) FillParameters() error {
	if c.authType == "" {
		if c.useDeviceLogin() {
			c.authType = authTypeDeviceLogin
		} else if c.UseCLI() {
			c.authType = authTypeAzureCLI
		} else if c.UseMSI() {
			c.authType = authTypeMSI
		} else if c.ClientSecret != "" {
			c.authType = authTypeClientSecret
		} else if c.ClientCertPath != "" {
			c.authType = authTypeClientCert
		} else {
			c.authType = authTypeClientBearerJWT
		}
	}

	if c.authType == authTypeMSI && c.SubscriptionID == "" {

		subscriptionID, err := getSubscriptionFromIMDS()
		if err != nil {
			return fmt.Errorf("error fetching subscriptionID from VM metadata service for Managed Identity authentication: %v", err)
		}
		c.SubscriptionID = subscriptionID
	}

	if c.cloudEnvironment == nil {
		err := c.setCloudEnvironment()
		if err != nil {
			return err
		}
	}

	if c.authType == authTypeAzureCLI {
		tenantID, subscriptionID, err := getIDsFromAzureCLI()
		if err != nil {
			return fmt.Errorf("error fetching tenantID and subscriptionID from Azure CLI (are you logged on using `az login`?): %v", err)
		}

		c.TenantID = tenantID
		c.SubscriptionID = subscriptionID
	}

	if c.TenantID == "" {
		tenantID, err := findTenantID(*c.cloudEnvironment, c.SubscriptionID)
		if err != nil {
			return err
		}
		c.TenantID = tenantID
	}

	if c.ClientCertExpireTimeout == 0 {
		c.ClientCertExpireTimeout = time.Hour
	}

	return nil
}

// allow override for unit tests
var findTenantID = FindTenantID
