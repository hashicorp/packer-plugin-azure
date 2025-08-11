// Copyright (c) HashiCorp, Inc.
// SPDX-License-Identifier: MPL-2.0

package keyvaultsecret

import (
	"fmt"
	"net/http"

	"github.com/hashicorp/go-azure-sdk/resource-manager/keyvault/2023-07-01/secrets"
	"github.com/hashicorp/go-azure-sdk/sdk/client/resourcemanager"
	"github.com/hashicorp/go-azure-sdk/sdk/odata"

	sdkEnv "github.com/hashicorp/go-azure-sdk/sdk/environments"
)

const KeyVaultAPIVersion = "7.5"

type Secret struct {
	secrets.Secret `mapstructure:",squash"`
	Value          string `mapstructure:"value"`
}

type GetOperationResponse struct {
	HttpResponse *http.Response
	OData        *odata.OData
	Model        *Secret
}

func NewSecretsClientWithBaseURI(sdkApi sdkEnv.Api) (*secrets.SecretsClient, error) {
	client, err := resourcemanager.NewClient(sdkApi, "secrets", KeyVaultAPIVersion)
	if err != nil {
		return nil, fmt.Errorf("instantiating SecretsClient: %+v", err)
	}

	return &secrets.SecretsClient{
		Client: client,
	}, nil
}
