// Copyright IBM Corp. 2013, 2025
// SPDX-License-Identifier: MPL-2.0

package common

import (
	"io"
	"net"
	"net/http"
	"os"
	"os/exec"
	"strings"
	"testing"
	"time"
)

type CheckAcceptanceTestEnvVarsParams struct {
	CheckAzureCLI          bool
	CheckSSHPrivateKeyFile bool
}

func CheckAcceptanceTestEnvVars(t *testing.T, params CheckAcceptanceTestEnvVarsParams) {
	if os.Getenv("PACKER_ACC") == "" {
		t.Skipf("Skipping acceptance test %s has environment variable `PACKER_ACC` is not set", t.Name())
	}
	if os.Getenv("ARM_RESOURCE_GROUP_NAME") == "" {
		t.Fatalf("Test %s requires environment variable ARM_RESOURCE_GROUP_NAME is set", t.Name())
	}
	if os.Getenv("ARM_RESOURCE_PREFIX") == "" {
		t.Fatalf("Test %s requires environment variable ARM_RESOURCE_PREFIX is set", t.Name())
	}
	if os.Getenv("ARM_TENANT_ID") == "" {
		t.Fatalf("Test %s requires environment variable ARM_TENANT_ID is set", t.Name())
	}
	if os.Getenv("ARM_CLIENT_ID") == "" {
		t.Fatalf("Test %s requires environment variable ARM_CLIENT_ID is set", t.Name())
	}
	if os.Getenv("ARM_CLIENT_SECRET") == "" {
		t.Fatalf("Test %s requires environment variable ARM_CLIENT_SECRET is set", t.Name())
	}
	if os.Getenv("ARM_SUBSCRIPTION_ID") == "" {
		t.Fatalf("Test %s requires environment variable ARM_SUBSCRIPTION_ID is set", t.Name())
	}
	if os.Getenv("ARM_VIRTUAL_NETWORK_NAME") == "" {
		t.Fatalf("Test %s requires environment variable ARM_VIRTUAL_NETWORK_NAME is set", t.Name())
	}

	if params.CheckAzureCLI && !loggedIntoAzureCLI(t) {
		t.Fatalf("Test %s requires CLI authentication, install the Azure CLI and log in", t.Name())
	}
	if params.CheckSSHPrivateKeyFile && os.Getenv("ARM_SSH_PRIVATE_KEY_FILE") == "" {
		t.Fatalf("Test %s requires environment variable ARM_SSH_PRIVATE_KEY_FILE is set", t.Name())
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

func DetectPackerPublicIP(t *testing.T) string {
	t.Helper()

	client := &http.Client{Timeout: 10 * time.Second}
	resp, err := client.Get("https://api.ipify.org")
	if err != nil {
		t.Fatalf("Failed to detect runner public IP from api.ipify.org: %v. "+
			"NSG allowlist tests require the runner's public IP to permit SSH. "+
			"Ensure outbound HTTP access to api.ipify.org is available.",
			err)
	}
	defer func() { _ = resp.Body.Close() }()

	if resp.StatusCode != http.StatusOK {
		t.Fatalf("api.ipify.org returned status %d. "+
			"Cannot determine runner public IP for NSG allowlist tests.",
			resp.StatusCode)
	}

	body, err := io.ReadAll(io.LimitReader(resp.Body, 64))
	if err != nil {
		t.Fatalf("Failed to read response from api.ipify.org: %v", err)
	}

	ipStr := strings.TrimSpace(string(body))
	if ipStr == "" {
		t.Fatal("api.ipify.org returned empty body. Cannot determine runner public IP.")
	}

	if net.ParseIP(ipStr) == nil {
		t.Fatalf("api.ipify.org returned %q which is not a valid IP address.", ipStr)
	}

	if strings.Contains(ipStr, ":") {
		return ipStr + "/128"
	}
	return ipStr + "/32"
}
