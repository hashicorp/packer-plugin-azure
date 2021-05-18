package keyvaultsecret

import (
	"context"
	_ "embed"
	"fmt"
	"io/ioutil"
	"os"
	"os/exec"
	"regexp"
	"testing"

	"github.com/Azure/azure-sdk-for-go/services/keyvault/auth"
	"github.com/Azure/azure-sdk-for-go/services/keyvault/v7.1/keyvault"
	"github.com/Azure/go-autorest/autorest"
	"github.com/hashicorp/packer-plugin-sdk/acctest"
)

//go:embed test-fixtures/template.pkr.hcl
var testDatasourceBasic string

func TestAccAzureKeyvaultSecret(t *testing.T) {
	testEnv := "dev"
	secret := &KeyvaultSecret{
		Name:        "packer-datasource-keyvault-test-secret",
		Value:       "this_is_the_packer_test_secret_value",
		ContentType: "text/html",
		Tags:        map[string]*string{"environment": &testEnv},
	}

	testCase := &acctest.PluginTestCase{
		Name: "azure_keyvault_secret_datasource_basic_test",
		Setup: func() error {
			return secret.Create()
		},
		Teardown: func() error {
			return secret.Delete()
		},
		Template: testDatasourceBasic,
		Check: func(buildCommand *exec.Cmd, logfile string) error {
			if buildCommand.ProcessState != nil {
				if buildCommand.ProcessState.ExitCode() != 0 {
					return fmt.Errorf("Bad exit code. Logfile: %s", logfile)
				}
			}

			logs, err := os.Open(logfile)
			if err != nil {
				return fmt.Errorf("Unable find %s", logfile)
			}
			defer logs.Close()

			logsBytes, err := ioutil.ReadAll(logs)
			if err != nil {
				return fmt.Errorf("Unable to read %s", logfile)
			}
			logsString := string(logsBytes)

			valueLog := fmt.Sprintf("null.basic-example: secret value: %s", secret.Value)
			contentTypeLog := fmt.Sprintf("null.basic-example: secret content_type: %s", secret.ContentType)
			environmentLog := fmt.Sprintf("null.basic-example: secret environment: %s", *secret.Tags["environment"])

			if matched, _ := regexp.MatchString(valueLog+".*", logsString); !matched {
				t.Fatalf("logs doesn't contain expected secret value %q", logsString)
			}
			if matched, _ := regexp.MatchString(contentTypeLog+".*", logsString); !matched {
				t.Fatalf("logs doesn't contain expected secret ContentType %q", logsString)
			}
			if matched, _ := regexp.MatchString(environmentLog+".*", logsString); !matched {
				t.Fatalf("logs doesn't contain expected secret tag %q", logsString)
			}

			return nil
		},
	}
	acctest.TestPlugin(t, testCase)
}

type KeyvaultSecret struct {
	Name        string
	Value       string
	Id          string
	ContentType string
	Tags        map[string]*string

	client keyvault.BaseClient
}

func getAuthorizer() (autorest.Authorizer, error) {
	os.Setenv("AZURE_AD_RESOURCE", "https://vault.azure.net")
	os.Setenv("AZURE_TENANT_ID", os.Getenv("PKR_VAR_tenant_id"))
	os.Setenv("AZURE_CLIENT_ID", os.Getenv("PKR_VAR_client_id"))
	os.Setenv("AZURE_CLIENT_SECRET", os.Getenv("PKR_VAR_client_secret"))

	credAuthorizer, err := auth.NewAuthorizerFromEnvironment()
	return credAuthorizer, err
}

func (as *KeyvaultSecret) Create() error {
	as.client = keyvault.New()
	authorizer, err := getAuthorizer()
	if err != nil {
		return err
	}

	var parameters keyvault.SecretSetParameters
	parameters.ContentType = &as.ContentType
	parameters.Value = &as.Value
	parameters.Tags = as.Tags

	as.client.Authorizer = authorizer

	_, err = as.client.SetSecret(context.TODO(), "https://"+os.Getenv("PKR_VAR_keyvault_id")+".vault.azure.net", as.Name, parameters)
	if err != nil {
		return err
	}
	return err
}

func (as *KeyvaultSecret) Delete() error {
	var err error
	_, err = as.client.DeleteSecret(context.TODO(), "https://"+os.Getenv("PKR_VAR_keyvault_id")+".vault.azure.net", as.Name)
	return err
}
