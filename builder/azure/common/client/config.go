// Copyright (c) HashiCorp, Inc.
// SPDX-License-Identifier: MPL-2.0

//go:generate packer-sdc struct-markdown

package client

import (
	"context"
	"errors"
	"fmt"
	"net/http"
	"os"
	"regexp"
	"strings"

	"github.com/hashicorp/packer-plugin-azure/builder/azure/common/log"

	"github.com/Azure/go-autorest/autorest/azure/cli"
	jwt "github.com/golang-jwt/jwt"
	"github.com/hashicorp/go-azure-helpers/resourcemanager/commonids"
	"github.com/hashicorp/go-azure-sdk/resource-manager/resources/2022-12-01/subscriptions"
	"github.com/hashicorp/go-azure-sdk/sdk/environments"
	packersdk "github.com/hashicorp/packer-plugin-sdk/packer"
)

// This error is thrown whenever the Azure SDK returns a null model with no error
// We do not expect this error to happen ever, but also don't want to throw a null pointer exception here.
var NullModelSDKErr = fmt.Errorf("Unexpected SDK response, please open an issue on the Azure plugin issue tracker")

// Config allows for various ways to authenticate Azure clients.  When
// `client_id` and `subscription_id` are specified in addition to one and only
// one of the following: `client_secret`, `client_jwt`, `client_cert_path` --
// Packer will use the specified Azure Active Directory (AAD) Service Principal
// (SP).
// If none ofthese options are specified, Packer will attempt to use the Managed Identity
// and subscription of the VM that Packer is running on.  This will only work if
// Packer is running on an Azure VM with either a System Assigned Managed
// Identity or User Assigned Managed Identity.
type Config struct {
	// One of Public, China, or
	// USGovernment. Defaults to Public. Long forms such as
	// USGovernmentCloud and AzureUSGovernmentCloud are also supported.
	CloudEnvironmentName string `mapstructure:"cloud_environment_name" required:"false"`
	cloudEnvironment     *environments.Environment
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
	// The path to a PKCS#12 bundle (.pfx file) to be used as the client certificate
	// that will be used to authenticate as the specified AAD SP.
	ClientCertPath string `mapstructure:"client_cert_path"`
	// The password for decrypting the client certificate bundle.
	ClientCertPassword string `mapstructure:"client_cert_password"`
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
}

// allow override for unit tests
var findTenantID = FindTenantID

const (
	AuthTypeMSI             = "ManagedIdentity"
	AuthTypeClientSecret    = "ClientSecret"
	AuthTypeClientCert      = "ClientCertificate"
	AuthTypeClientBearerJWT = "ClientBearerJWT"
	AuthTypeAzureCLI        = "AzureCLI"
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

func (c *Config) CloudEnvironment() *environments.Environment {
	return c.cloudEnvironment
}
func (c *Config) AuthType() string {
	return c.authType
}

func (c *Config) setCloudEnvironment() error {
	if c.MetadataHost == "" {
		if v := os.Getenv("ARM_METADATA_URL"); v != "" {
			c.MetadataHost = v
		}
	}
	env, err := environments.FromEndpoint(context.TODO(), c.MetadataHost, c.CloudEnvironmentName)
	c.cloudEnvironment = env
	if err != nil {
		// fall back to old method of normalizing and looking up cloud names.
		log.Printf("Error looking up environment using metadata host: %s. \n"+
			"Falling back to hardcoded mechanism...", err.Error())
		lookup := map[string]string{
			"CHINA":           "china",
			"CHINACLOUD":      "china",
			"AZURECHINACLOUD": "china",

			"PUBLIC":           "public",
			"PUBLICCLOUD":      "public",
			"AZUREPUBLICCLOUD": "public",

			"USGOVERNMENT":           "usgovernment",
			"USGOVERNMENTCLOUD":      "usgovernment",
			"AZUREUSGOVERNMENTCLOUD": "usgovernment",
		}

		name := strings.ToUpper(c.CloudEnvironmentName)
		envName, ok := lookup[name]
		if !ok {
			return fmt.Errorf("There is no cloud environment matching the name '%s'!", c.CloudEnvironmentName)
		}

		env, err := environments.FromName(envName)
		if err != nil {
			return err
		}
		c.cloudEnvironment = env
	}
	return nil
}

//nolint:ineffassign //this triggers a false positive because errs is passed by reference
func (c Config) Validate(errs *packersdk.MultiError) {
	/////////////////////////////////////////////
	// Authentication via OAUTH

	if c.UseCLI() {
		return
	}

	if c.UseMSI() {
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
		return
	}

	if c.SubscriptionID != "" && c.ClientID != "" &&
		c.ClientSecret == "" &&
		c.ClientCertPath == "" &&
		c.ClientJWT != "" {
		p := jwt.Parser{}
		claims := jwt.StandardClaims{}
		token, _, err := p.ParseUnverified(c.ClientJWT, &claims)
		if err != nil {
			errs = packersdk.MultiErrorAppend(errs, fmt.Errorf("client_jwt is not a JWT: %v", err))
		} else {
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
		"  - use_azure_cli_auth\n"+
		"  to use an Azure Active Directory service principal, specify either:\n"+
		"  - subscription_id, client_id and client_secret\n"+
		"  - subscription_id, client_id and client_cert_path\n"+
		"  - subscription_id, client_id and client_jwt."))
}

func (c Config) UseCLI() bool {
	return c.UseAzureCLIAuth
}

func (c Config) UseMSI() bool {
	return !c.UseAzureCLIAuth &&
		c.ClientSecret == "" &&
		c.ClientJWT == "" &&
		c.ClientCertPath == "" &&
		c.TenantID == ""
}

// FillParameters capture the user intent from the supplied parameter set in AuthType, retrieves the TenantID and CloudEnvironment if not specified.
// The SubscriptionID is also retrieved in case MSI auth is requested.
func (c *Config) FillParameters() error {
	if c.authType == "" {
		if c.UseCLI() {
			c.authType = AuthTypeAzureCLI
		} else if c.UseMSI() {
			c.authType = AuthTypeMSI
		} else if c.ClientSecret != "" {
			c.authType = AuthTypeClientSecret
		} else if c.ClientCertPath != "" {
			c.authType = AuthTypeClientCert
		} else {
			c.authType = AuthTypeClientBearerJWT
		}
	}

	if c.authType == AuthTypeMSI && c.SubscriptionID == "" {
		subscriptionID, err := getSubscriptionFromIMDS()
		if err != nil {
			return fmt.Errorf("error fetching subscriptionID from VM metadata service for Managed Identity authentication: %v", err)
		}
		c.SubscriptionID = subscriptionID
	}
	if c.cloudEnvironment == nil {
		newCloudErr := c.setCloudEnvironment()
		if newCloudErr != nil {
			return newCloudErr
		}
	}

	if c.authType == AuthTypeAzureCLI {
		tenantID, subscriptionID, err := getIDsFromAzureCLI()
		if err != nil {
			return fmt.Errorf("error fetching tenantID and subscriptionID from Azure CLI (are you logged on using `az login`?): %v", err)
		}

		c.TenantID = tenantID
		c.SubscriptionID = subscriptionID
	}

	// CLI Auth does not require tenant, SDK parses that for us
	if c.TenantID == "" && !c.UseAzureCLIAuth {
		tenantID, err := findTenantID(*c.cloudEnvironment, c.SubscriptionID)
		if err != nil {
			return err
		}
		c.TenantID = tenantID
	}

	return nil
}

// getIDsFromAzureCLI returns the TenantID and SubscriptionID from an active Azure CLI login session
func getIDsFromAzureCLI() (string, string, error) {
	profilePath, err := cli.ProfilePath()
	if err != nil {
		return "", "", err
	}

	profile, err := cli.LoadProfile(profilePath)
	if err != nil {
		return "", "", err
	}

	for _, p := range profile.Subscriptions {
		if p.IsDefault {
			return p.TenantID, p.ID, nil
		}
	}

	return "", "", errors.New("Unable to find default subscription")
}

func FindTenantID(env environments.Environment, subscriptionID string) (string, error) {
	const hdrKey = "WWW-Authenticate"
	resourceManagerEndpoint, _ := env.ResourceManager.Endpoint()
	c := subscriptions.NewSubscriptionsClientWithBaseURI(*resourceManagerEndpoint)

	// we expect this request to fail (err != nil), but we are only interested
	// in headers, so surface the error if the Response is not present (i.e.
	// network error etc)
	subs, err := c.Get(context.TODO(), commonids.NewSubscriptionID(subscriptionID))
	if subs.HttpResponse == nil {
		return "", fmt.Errorf("Request failed: %v", err)
	}

	// Expecting 401 StatusUnauthorized here, just read the header
	if subs.HttpResponse.StatusCode != http.StatusUnauthorized {
		return "", fmt.Errorf("Unexpected response from Get Subscription: %v", err)
	}
	hdr := subs.HttpResponse.Header.Get(hdrKey)
	if hdr == "" {
		return "", fmt.Errorf("Header %v not found in Get Subscription response", hdrKey)
	}

	// Example value for hdr:
	//   Bearer authorization_uri="https://login.windows.net/996fe9d1-6171-40aa-945b-4c64b63bf655", error="invalid_token", error_description="The authentication failed because of missing 'Authorization' header."
	r := regexp.MustCompile(`authorization_uri=".*/([0-9a-f\-]+)"`)
	m := r.FindStringSubmatch(hdr)
	if m == nil {
		return "", fmt.Errorf("Could not find the tenant ID in header: %s %q", hdrKey, hdr)
	}
	return m[1], nil
}
