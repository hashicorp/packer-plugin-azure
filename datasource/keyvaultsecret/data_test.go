package keyvaultsecret

import (
	"testing"
)

func TestDatasourceConfigure_EmptySecretName(t *testing.T) {
	datasource := Datasource{
		config: Config{},
	}
	if err := datasource.Configure(nil); err == nil {
		t.Fatalf("Should error if secret name is not specified")
	}
}

func TestDatasourceConfigure_KeyvaultID(t *testing.T) {
	datasource := Datasource{
		config: Config{
			Name: "my-secret",
		},
	}
	if err := datasource.Configure(nil); err == nil {
		t.Fatalf("Should error if keyvault id is not specified")
	}
}

func TestDatasourceConfigure_OkConfig(t *testing.T) {
	datasource := Datasource{
		config: Config{
			Name:       "my-secret",
			KeyvaultId: "my-keyvault",
		},
	}
	if err := datasource.Configure(nil); err != nil {
		t.Fatalf("Should not issue error if configuration is okay")
	}
}
