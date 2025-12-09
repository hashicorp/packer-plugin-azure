// Copyright IBM Corp. 2013, 2025
// SPDX-License-Identifier: MPL-2.0

package keyvaultsecret

import (
	"testing"
)

func TestDatasourceConfigure_EmptyVaultName(t *testing.T) {
	d := &Datasource{
		config: Config{
			SecretName: "test-secret",
		},
	}
	err := d.Configure()
	if err == nil {
		t.Fatal("expected error when vault_name is missing")
	}
}

func TestDatasourceConfigure_EmptySecretName(t *testing.T) {
	d := &Datasource{
		config: Config{
			VaultName: "test-vault",
		},
	}
	err := d.Configure()
	if err == nil {
		t.Fatal("expected error when secret_name is missing")
	}
}

func TestDatasourceConfigure_Defaults(t *testing.T) {
	d := &Datasource{
		config: Config{
			SecretName: "test-secret",
			VaultName:  "test-vault",
		},
	}
	err := d.Configure()
	if err != nil {
		t.Fatalf("unexpected error: %s", err)
	}
}
