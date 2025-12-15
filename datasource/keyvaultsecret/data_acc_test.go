// // Copyright IBM Corp. 2013, 2025
// // SPDX-License-Identifier: MPL-2.0

package keyvaultsecret

import (
	"context"
	_ "embed"
	"errors"
	"fmt"
	"log"
	"os"
	"os/exec"
	"regexp"
	"testing"
	"time"

	"github.com/Azure/azure-sdk-for-go/sdk/azcore"
	"github.com/Azure/azure-sdk-for-go/sdk/azidentity"
	"github.com/Azure/azure-sdk-for-go/sdk/keyvault/azsecrets"
	"github.com/hashicorp/packer-plugin-sdk/acctest"
)

//go:embed test-fixtures/template.pkr.hcl
var testTemplate string

func TestAccAzureKeyVaultSecret(t *testing.T) {
	packerResourcePrefix := os.Getenv("ARM_RESOURCE_PREFIX")
	if packerResourcePrefix == "" {
		packerResourcePrefix = "packer"
	}
	testVaultName := fmt.Sprintf("%s-pkr-test-vault", packerResourcePrefix)

	cases := []struct {
		name       string
		secret     *AzureKeyVault
		expect     string
		wantErr    bool
		skipCreate bool
	}{
		{
			name: "valid json value",
			secret: &AzureKeyVault{
				SecretName: "packer-secret-valid",
				Value:      "secret-valid-value",
				VaultName:  testVaultName,
			},
			expect: "secret value: secret-valid-value",
		},
		{
			name: "non-json value",
			secret: &AzureKeyVault{
				SecretName: "packer-secret-nonjson",
				Value:      "some random string",
				VaultName:  testVaultName,
			},
			expect: "secret value: some random string",
		},
		{
			name: "empty value",
			secret: &AzureKeyVault{
				SecretName: "packer-secret-empty",
				Value:      `{"foo":""}`,
				VaultName:  testVaultName,
			},
			expect: "secret value:",
		},
		{
			name: "missing secret",
			secret: &AzureKeyVault{
				SecretName: "packer-secret-does-not-exist",
				VaultName:  testVaultName,
			},
			wantErr:    true,
			skipCreate: true,
		},
	}

	for _, tc := range cases {

		extraArgs := []string{
			"-var", fmt.Sprintf("secret_name=%s", tc.secret.SecretName),
			"-var", fmt.Sprintf("vault_name=%s", testVaultName),
			"-var", fmt.Sprintf("tenant_id=%s", os.Getenv("ARM_TENANT_ID")),
			"-var", fmt.Sprintf("client_id=%s", os.Getenv("ARM_CLIENT_ID")),
			"-var", fmt.Sprintf("client_secret=%s", os.Getenv("ARM_CLIENT_SECRET")),
			"-var", fmt.Sprintf("subscription_id=%s", os.Getenv("ARM_SUBSCRIPTION_ID")),
		}

		t.Run(tc.name, func(t *testing.T) {
			testCase := &acctest.PluginTestCase{
				Name: "azure_keyvaultsecret_" + tc.name,
				Setup: func() error {
					if tc.skipCreate {
						return nil
					}
					return tc.secret.Create()
				},
				Teardown: func() error {
					if tc.skipCreate {
						return nil
					}
					return tc.secret.Delete()
				},
				Template:       testTemplate,
				BuildExtraArgs: extraArgs,
				Check: func(cmd *exec.Cmd, logFile string) error {
					logs, err := os.ReadFile(logFile)
					if err != nil {
						return fmt.Errorf("failed to read log file: %w", err)
					}
					logsString := string(logs)

					log.Print("Checking logs for expected output...")
					if tc.wantErr {
						if matched := regexp.MustCompile("failed to get secret:").MatchString(logsString); !matched {
							t.Errorf("Expected failure not found in logs")
						}
						return nil
					}

					if !regexp.MustCompile(regexp.QuoteMeta(tc.expect)).MatchString(logsString) {
						t.Errorf("Expected log not found: %s\nLogs: %s", tc.expect, logsString)
					}
					return nil
				},
			}

			acctest.TestPlugin(t, testCase)
		})
	}
}

// AzureKeyVault represents a secret in Azure Key Vault.
type AzureKeyVault struct {
	VaultName  string `mapstructure:"vault_name" required:"true"`
	SecretName string `mapstructure:"secret_name" required:"true"`
	Value      string `mapstructure:"value" required:"true"`
}

func (s *AzureKeyVault) getSecretsClient() (*azsecrets.Client, error) {
	cred, err := azidentity.NewDefaultAzureCredential(nil)
	if err != nil {
		log.Printf("failed to obtain a credential: %v", err)
		return nil, fmt.Errorf("failed to obtain a credential: %v", err)
	}

	vaultURI := fmt.Sprintf("https://%s.vault.azure.net", s.VaultName)
	// Establish a connection to the Key Vault client
	client, err := azsecrets.NewClient(vaultURI, cred, nil)
	if err != nil {
		log.Printf("failed to create a Key Vault client: %v", err)
		return nil, fmt.Errorf("failed to create Key Vault client: %v", err)
	}

	return client, nil
}

func (s *AzureKeyVault) Create() error {

	client, err := s.getSecretsClient()
	if err != nil {
		return err
	}

	_, err = client.SetSecret(context.TODO(), s.SecretName, azsecrets.SetSecretParameters{
		Value: &s.Value,
	}, nil)

	if err != nil && !isAlreadyExists(err) {
		return fmt.Errorf("failed to create secret: %w", err)
	}

	log.Printf("Secret %q created successfully", s.SecretName)

	return nil
}

func (s *AzureKeyVault) Delete() error {

	client, err := s.getSecretsClient()
	if err != nil {
		return err
	}

	_, err = client.DeleteSecret(context.TODO(), s.SecretName, nil)
	if err != nil && !isNotFound(err) {
		return fmt.Errorf("failed to delete secret: %w", err)
	}
	time.Sleep(1 * time.Second) // Wait for the secret to be deleted

	_, err = client.PurgeDeletedSecret(context.TODO(), s.SecretName, nil)
	if err != nil {
		return fmt.Errorf("failed to purge deleted secret: %w", err)
	}

	log.Printf("Secret %q deleted successfully", s.SecretName)

	return nil
}

// Helpers
func isAlreadyExists(err error) bool {
	var respErr *azcore.ResponseError
	// errors.As is the idiomatic way to check if an error is of a specific type.
	// It checks the entire error chain.
	if errors.As(err, &respErr) {
		// Azure APIs commonly return the "Conflict" error code when
		// attempting to create a resource that already exists.
		log.Printf("Azure error: %v", respErr.RawResponse)
		return respErr.ErrorCode == "Conflict" || respErr.StatusCode == 409
	}
	return false
}

func isNotFound(err error) bool {
	var respErr *azcore.ResponseError
	if errors.As(err, &respErr) {
		log.Printf("Azure error: %v", respErr.RawResponse)
		// Azure APIs commonly return the "NotFound" error code when
		// attempting to access a resource that does not exist.
		return respErr.ErrorCode == "NotFound" || respErr.StatusCode == 404
	}
	return false
}
