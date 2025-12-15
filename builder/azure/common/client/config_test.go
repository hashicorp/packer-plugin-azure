// Copyright IBM Corp. 2013, 2025
// SPDX-License-Identifier: MPL-2.0

package client

import (
	crand "crypto/rand"
	"crypto/rsa"
	"encoding/base64"
	"encoding/binary"
	"io"
	mrand "math/rand"
	"os"
	"testing"
	"time"

	jwt "github.com/golang-jwt/jwt/v5"
	"github.com/hashicorp/go-azure-sdk/sdk/environments"
	packersdk "github.com/hashicorp/packer-plugin-sdk/packer"
)

func Test_ClientConfig_RequiredParametersSet(t *testing.T) {

	tests := []struct {
		name    string
		config  Config
		wantErr bool
	}{
		{
			name:    "no client_id, client_secret or subscription_id should enable MSI auth",
			config:  Config{},
			wantErr: false,
		},
		{
			name: "client_id, with no client_secret or subscription_id should enable MSI auth",
			config: Config{
				ClientID: "fake-id",
			},
			wantErr: false,
		},
		{
			name: "use_azure_cli_auth will trigger Azure CLI auth",
			config: Config{
				UseAzureCLIAuth: true,
			},
			wantErr: false,
		},
		{
			name: "client_id without client_secret, client_cert_path or client_jwt should not error",
			config: Config{
				SubscriptionID: "error",
			},
			wantErr: false,
		},
		{
			name: "oidc request url, oidc request token, client id, and tenant sh",
			config: Config{
				TenantID: "ok",
			},
			wantErr: true,
		},
		{
			name: "client_secret without client_id should error",
			config: Config{
				ClientSecret: "error",
			},
			wantErr: true,
		},
		{
			name: "client_cert_path without client_id should error",
			config: Config{
				ClientCertPath: "/dev/null",
			},
			wantErr: true,
		},
		{
			name: "client_jwt without client_id should error",
			config: Config{
				ClientJWT: "error",
			},
			wantErr: true,
		},
		{
			name: "missing subscription_id when using secret",
			config: Config{
				ClientID:     "ok",
				ClientSecret: "ok",
			},
			wantErr: true,
		},
		{
			name: "missing subscription_id when using certificate",
			config: Config{
				ClientID:       "ok",
				ClientCertPath: "ok",
			},
			wantErr: true,
		},
		{
			name: "missing subscription_id when using JWT",
			config: Config{
				ClientID:  "ok",
				ClientJWT: "ok",
			},
			wantErr: true,
		},
		{
			name: "too many client_* values",
			config: Config{
				SubscriptionID: "ok",
				ClientID:       "ok",
				ClientSecret:   "ok",
				ClientCertPath: "error",
			},
			wantErr: true,
		},
		{
			name: "too many client_* values (2)",
			config: Config{
				SubscriptionID: "ok",
				ClientID:       "ok",
				ClientSecret:   "ok",
				ClientJWT:      "error",
			},
			wantErr: true,
		},
		{
			name: "tenant_id alone should fail",
			config: Config{
				TenantID: "ok",
			},
			wantErr: true,
		},
	}
	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {

			errs := &packersdk.MultiError{}
			tt.config.Validate(errs)
			if (len(errs.Errors) != 0) != tt.wantErr {
				t.Errorf("newConfig() error = %v, wantErr %v", errs, tt.wantErr)
				return
			}
		})
	}
}

func Test_ClientConfig_AzureCli(t *testing.T) {
	// Azure CLI tests skipped unless env 'AZURE_CLI_AUTH' is set, and an active `az login` session has been established
	getEnvOrSkip(t, "AZURE_CLI_AUTH")

	cfg := Config{
		UseAzureCLIAuth:  true,
		cloudEnvironment: environments.AzurePublic(),
	}
	assertValid(t, cfg)

	err := cfg.FillParameters()
	if err != nil {
		t.Fatalf("Expected nil err, but got: %v", err)
	}

	if cfg.AuthType() != AuthTypeAzureCLI {
		t.Fatalf("Expected authType to be %q, but got: %q", AuthTypeAzureCLI, cfg.AuthType())
	}

	if cfg.SubscriptionID == "" {
		t.Fatalf("Expected SubscriptionId to not be empty, but got %s", cfg.SubscriptionID)
	}
}

func Test_ClientConfig_AzureCli_with_subscription_id_set(t *testing.T) {
	// Azure CLI tests skipped unless env 'AZURE_CLI_AUTH' is set, and an active `az login` session has been established
	getEnvOrSkip(t, "AZURE_CLI_AUTH")
	subId := "non-default-subscription_id"
	cfg := Config{
		UseAzureCLIAuth:  true,
		cloudEnvironment: environments.AzurePublic(),
		SubscriptionID:   subId,
	}
	assertValid(t, cfg)

	err := cfg.FillParameters()
	if err != nil {
		t.Fatalf("Expected nil err, but got: %v", err)
	}

	if cfg.AuthType() != AuthTypeAzureCLI {
		t.Fatalf("Expected authType to be %q, but got: %q", AuthTypeAzureCLI, cfg.AuthType())
	}

	if cfg.SubscriptionID != subId {
		t.Fatalf("Expected SubscriptionId to be %s, but got: %s", subId, cfg.SubscriptionID)
	}
}

func Test_ClientConfig_GitHubOIDC(t *testing.T) {
	retrievedTid := "my-tenant-id"
	findTenantID = func(environments.Environment, string) (string, error) { return retrievedTid, nil }
	cfg := Config{
		cloudEnvironment: environments.AzurePublic(),
		OidcRequestToken: "whatever",
		OidcRequestURL:   "whatever",
		ClientID:         "whatever",
		SubscriptionID:   "whatever",
	}
	assertValid(t, cfg)

	err := cfg.FillParameters()
	if err != nil {
		t.Fatalf("Expected nil err, but got: %v", err)
	}

	if cfg.AuthType() != AuthTypeOidcURL {
		t.Fatalf("Expected authType to be %q, but got: %q", AuthTypeAzureCLI, cfg.AuthType())
	}
}

func Test_ClientConfig_GitHubOIDC_Rejections(t *testing.T) {
	// No Subscription
	cfg := Config{
		cloudEnvironment: environments.AzurePublic(),
		OidcRequestToken: "whatever",
		OidcRequestURL:   "whatever",
		ClientID:         "whatever",
	}
	assertInvalid(t, cfg)

	// No Request Token
	cfg = Config{
		cloudEnvironment: environments.AzurePublic(),
		SubscriptionID:   "whatever",
		OidcRequestURL:   "whatever",
		ClientID:         "whatever",
	}
	assertInvalid(t, cfg)

	// No Request URL
	cfg = Config{
		cloudEnvironment: environments.AzurePublic(),
		OidcRequestToken: "whatever",
		SubscriptionID:   "whatever",
		ClientID:         "whatever",
	}
	assertInvalid(t, cfg)

	// No Client ID
	cfg = Config{
		cloudEnvironment: environments.AzurePublic(),
		OidcRequestToken: "whatever",
		SubscriptionID:   "whatever",
		OidcRequestURL:   "whatever",
	}
	assertInvalid(t, cfg)
}

func getEnvOrSkip(t *testing.T, envVar string) string {
	v := os.Getenv(envVar)
	if v == "" {
		t.Skipf("%s is empty, skipping", envVar)
	}
	return v
}

// tests for assertRequiredParametersSet

func assertValid(t *testing.T, cfg Config) {
	errs := &packersdk.MultiError{}
	cfg.Validate(errs)
	if len(errs.Errors) != 0 {
		t.Fatal("Expected errs to be empty: ", errs)
	}
}

func assertInvalid(t *testing.T, cfg Config) {
	errs := &packersdk.MultiError{}
	cfg.Validate(errs)
	if len(errs.Errors) == 0 {
		t.Fatal("Expected errs to be non-empty")
	}
}

func Test_ClientConfig_CanUseClientSecret(t *testing.T) {
	cfg := Config{
		SubscriptionID: "12345",
		ClientID:       "12345",
		ClientSecret:   "12345",
	}

	assertValid(t, cfg)
}

func Test_ClientConfig_CanUseClientSecretWithTenantID(t *testing.T) {
	cfg := Config{
		SubscriptionID: "12345",
		ClientID:       "12345",
		ClientSecret:   "12345",
		TenantID:       "12345",
	}

	assertValid(t, cfg)
}

func Test_ClientConfig_CanUseClientJWT(t *testing.T) {
	cfg := Config{
		SubscriptionID: "12345",
		ClientID:       "12345",
		ClientJWT:      getJWT(10*time.Minute, true),
	}

	assertValid(t, cfg)
}

func Test_ClientConfig_CanUseClientJWTWithTenantID(t *testing.T) {
	cfg := Config{
		SubscriptionID: "12345",
		ClientID:       "12345",
		ClientJWT:      getJWT(10*time.Minute, true),
		TenantID:       "12345",
	}

	assertValid(t, cfg)
}

func Test_ClientConfig_CannotUseBothClientJWTAndSecret(t *testing.T) {
	cfg := Config{
		SubscriptionID: "12345",
		ClientID:       "12345",
		ClientSecret:   "12345",
		ClientJWT:      getJWT(10*time.Minute, true),
	}

	assertInvalid(t, cfg)
}

func Test_getJWT(t *testing.T) {
	if getJWT(time.Minute, true) == "" {
		t.Fatalf("getJWT is broken")
	}
}

func newRandReader() io.Reader {
	var seed int64
	_ = binary.Read(crand.Reader, binary.LittleEndian, &seed)

	return mrand.New(mrand.NewSource(seed))
}

func getJWT(validFor time.Duration, withX5tHeader bool) string {
	token := jwt.New(jwt.SigningMethodRS256)
	key, _ := rsa.GenerateKey(newRandReader(), 2048)

	token.Claims = jwt.MapClaims{
		"aud": "https://login.microsoftonline.com/tenant.onmicrosoft.com/oauth2/token?api-version=1.0",
		"iss": "355dff10-cd78-11e8-89fe-000d3afd16e3",
		"sub": "355dff10-cd78-11e8-89fe-000d3afd16e3",
		"jti": base64.URLEncoding.EncodeToString([]byte{0}),
		"nbf": time.Now().Unix(),
		"exp": time.Now().Add(validFor).Unix(),
	}
	if withX5tHeader {
		token.Header["x5t"] = base64.URLEncoding.EncodeToString([]byte("thumbprint"))
	}

	jwt, _ := token.SignedString(key)
	return jwt
}
