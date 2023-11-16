package common

import (
	"os"
	"os/exec"
	"testing"
)

type CheckAcceptanceTestEnvVarsParams struct {
	CheckAzureCLI          bool
	CheckSSHPrivateKeyFile bool
}

func CheckAcceptanceTestEnvVars(t *testing.T, params CheckAcceptanceTestEnvVarsParams) {
	if os.Getenv("PACKER_ACC") == "" {
		t.Skipf("Skipping acceptance test %s has environment variable `PACKER_ACC` is not set", t.Name())
		return
	}

	if os.Getenv("ARM_RESOURCE_GROUP_NAME") == "" {
		t.Fatalf("Test %s requires environment variable ARM_RESOURCE_GROUP_NAME is set", t.Name())
		return
	}
	if os.Getenv("ARM_CLIENT_ID") == "" {
		t.Fatalf("Test %s requires environment variable ARM_CLIENT_ID is set", t.Name())
		return
	}
	if os.Getenv("ARM_CLIENT_SECRET") == "" {
		t.Fatalf("Test %s requires environment variable ARM_CLIENT_SECRET is set", t.Name())
		return
	}
	if os.Getenv("ARM_SUBSCRIPTION_ID") == "" {
		t.Fatalf("Test %s requires environment variable ARM_SUBSCRIPTION_ID is set", t.Name())
		return
	}

	if params.CheckAzureCLI && !loggedIntoAzureCLI(t) {
		t.Fatalf("Test %s requires CLI authentication, install the Azure CLI and log in", t.Name())
		return
	}
	if params.CheckSSHPrivateKeyFile && os.Getenv("ARM_SSH_PRIVATE_KEY_FILE") == "" {
		t.Fatalf("Test %s requires environment variable ARM_SSH_PRIVATE_KEY_FILE is set", t.Name())
		return
	}
}

func loggedIntoAzureCLI(t *testing.T) bool {
	command := exec.Command("az", "account", "show")
	commandStdout, err := command.CombinedOutput()
	if err != nil {
		t.Logf("`az account show` failed\n"+
			"error: %v\n"+
			"output: \n%s", err, string(commandStdout))
	}

	return err == nil
}
