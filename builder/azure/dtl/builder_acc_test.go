// Copyright (c) HashiCorp, Inc.
// SPDX-License-Identifier: MPL-2.0

package dtl

// Required environment variables
// * ARM_CLIENT_ID
// * ARM_CLIENT_SECRET
// * ARM_SUBSCRIPTION_ID
// * ARM_RESOURCE_GROUP_NAME
// ** ARM_RESOURCE_PREFIX - String prefix for resources unique name constraints, such as Dev Test Lab
// * As well as the following misc env variables
// Your resource group should be the South Central US region
// This test also requires a DTL (Dev Test Lab) named `(ARM_RESOURCE_PREFIX)-packer_acceptance_test`
// This can be created using the terraform config in the `terraform` folder at the root of this repo
// It is recommended to set the required environment variables and then run the acceptance_test_setup.sh script in the terraform directory
// In addition, the PACKER_ACC variable should also be set to
// a non-empty value to enable Packer acceptance tests and the
// options "-v -timeout 90m" should be provided to the test
// command, e.g.:
//   go test -v -timeout 90m -run TestBuilderAcc_.*

import (
	_ "embed"
	"fmt"
	"os/exec"
	"testing"

	"github.com/hashicorp/packer-plugin-azure/builder/azure/common"
	"github.com/hashicorp/packer-plugin-sdk/acctest"
)

func TestDTLBuilderAcc_ManagedDisk_Windows(t *testing.T) {
	t.Parallel()
	common.CheckAcceptanceTestEnvVars(t, common.CheckAcceptanceTestEnvVarsParams{})
	acctest.TestPlugin(t, &acctest.PluginTestCase{
		Name:     "test-azure-managedisk-windows",
		Type:     "azure-dtl",
		Template: string(armWindowsDTLTemplate),
		Check: func(buildCommand *exec.Cmd, logfile string) error {
			if buildCommand.ProcessState != nil {
				if buildCommand.ProcessState.ExitCode() != 0 {
					return fmt.Errorf("Bad exit code. Logfile: %s", logfile)
				}
			}
			return nil
		},
	})
}
func TestDTLBuilderAcc_ManagedDisk_Linux_Artifacts(t *testing.T) {
	t.Parallel()
	common.CheckAcceptanceTestEnvVars(t, common.CheckAcceptanceTestEnvVarsParams{})
	acctest.TestPlugin(t, &acctest.PluginTestCase{
		Name:     "test-azure-managedisk-linux",
		Type:     "azure-dtl",
		Template: string(armLinuxDTLTemplate),
		Check: func(buildCommand *exec.Cmd, logfile string) error {
			if buildCommand.ProcessState != nil {
				if buildCommand.ProcessState.ExitCode() != 0 {
					return fmt.Errorf("Bad exit code. Logfile: %s", logfile)
				}
			}
			return nil
		},
	})
}

//go:embed testdata/windows.pkr.hcl
var armWindowsDTLTemplate []byte

//go:embed testdata/linux.pkr.hcl
var armLinuxDTLTemplate []byte
