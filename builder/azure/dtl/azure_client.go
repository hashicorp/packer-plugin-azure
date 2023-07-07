// Copyright (c) HashiCorp, Inc.
// SPDX-License-Identifier: MPL-2.0

package dtl

import (
	"context"
	"encoding/json"
	"fmt"
	"github.com/golang-jwt/jwt"
	"math"
	"net/http"
	"os"
	"strconv"
	"time"

	"github.com/Azure/go-autorest/autorest"
	hashiImagesSDK "github.com/hashicorp/go-azure-sdk/resource-manager/compute/2022-03-01/images"
	hashiVMSDK "github.com/hashicorp/go-azure-sdk/resource-manager/compute/2022-03-01/virtualmachines"
	hashiDisksSDK "github.com/hashicorp/go-azure-sdk/resource-manager/compute/2022-03-02/disks"
	hashiGalleryImagesSDK "github.com/hashicorp/go-azure-sdk/resource-manager/compute/2022-03-03/galleryimages"
	hashiGalleryImageVersionsSDK "github.com/hashicorp/go-azure-sdk/resource-manager/compute/2022-03-03/galleryimageversions"
	hashiDTLSDK "github.com/hashicorp/go-azure-sdk/resource-manager/devtestlab/2018-09-15"
	hashiVaultsSDK "github.com/hashicorp/go-azure-sdk/resource-manager/keyvault/2023-02-01/vaults"
	hashiNetworkSDK "github.com/hashicorp/go-azure-sdk/resource-manager/network/2022-09-01"
	"github.com/hashicorp/go-azure-sdk/sdk/auth"
	authWrapper "github.com/hashicorp/go-azure-sdk/sdk/auth/autorest"
	"github.com/hashicorp/go-azure-sdk/sdk/client/resourcemanager"
	"github.com/hashicorp/go-azure-sdk/sdk/environments"
	"github.com/hashicorp/packer-plugin-azure/version"
	"github.com/hashicorp/packer-plugin-sdk/useragent"
	giovanniBlobStorageSDK "github.com/tombuildsstuff/giovanni/storage/2020-08-04/blob/blobs"
)

const (
	EnvPackerLogAzureMaxLen = "PACKER_LOG_AZURE_MAXLEN"
)

type AzureClient struct {
	InspectorMaxLength int
	LastError          azureErrorResponse

	GiovanniBlobClient giovanniBlobStorageSDK.Client
	hashiDisksSDK.DisksClient
	hashiVMSDK.VirtualMachinesClient
	hashiImagesSDK.ImagesClient
	hashiVaultsSDK.VaultsClient
	NetworkMetaClient hashiNetworkSDK.Client
	hashiGalleryImageVersionsSDK.GalleryImageVersionsClient
	hashiGalleryImagesSDK.GalleryImagesClient
	DtlMetaClient hashiDTLSDK.Client
}

func getCaptureResponse(body string) *CaptureTemplate {
	var operation CaptureOperation
	err := json.Unmarshal([]byte(body), &operation)
	if err != nil {
		return nil
	}

	if operation.Properties != nil && operation.Properties.Output != nil {
		return operation.Properties.Output
	}

	return nil
}

func errorCapture(client *AzureClient) autorest.RespondDecorator {
	return func(r autorest.Responder) autorest.Responder {
		return autorest.ResponderFunc(func(resp *http.Response) error {
			body, bodyString := handleBody(resp.Body, math.MaxInt64)
			resp.Body = body

			errorResponse := newAzureErrorResponse(bodyString)
			if errorResponse != nil {
				client.LastError = *errorResponse
			}

			return r.Respond(resp)
		})
	}
}

// WAITING(chrboum): I have logged https://github.com/Azure/azure-sdk-for-go/issues/311 to get this
// method included in the SDK.  It has been accepted, and I'll cut over to the official way
// once it ships.
func byConcatDecorators(decorators ...autorest.RespondDecorator) autorest.RespondDecorator {
	return func(r autorest.Responder) autorest.Responder {
		return autorest.DecorateResponder(r, decorators...)
	}
}

type NewSDKAuthOptions struct {
	AuthType       string
	ClientID       string
	ClientSecret   string
	ClientJWT      string
	ClientCertPath string
	TenantID       string
	SubscriptionID string
}

// Returns an Azure Client used for the Azure Resource Manager
// Also returns the Azure object ID for the authentication method used in the build
func NewAzureClient(ctx context.Context, subscriptionID, resourceGroupName string,
	cloud *environments.Environment, SharedGalleryTimeout time.Duration, CustomImageCaptureTimeout time.Duration, PollingDuration time.Duration, newSdkAuthOptions NewSDKAuthOptions) (*AzureClient, *string, error) {

	var azureClient = &AzureClient{}

	maxlen := getInspectorMaxLength()

	if cloud == nil || cloud.ResourceManager == nil {
		// TODO Throw error message that helps users solve this problem
		return nil, nil, fmt.Errorf("Azure Environment not configured correctly")
	}
	resourceManagerEndpoint, _ := cloud.ResourceManager.Endpoint()
	resourceManagerAuthorizer, err := buildResourceManagerAuthorizer(ctx, newSdkAuthOptions, *cloud)
	if err != nil {
		return nil, nil, err
	}
	storageAccountAuthorizer, err := buildStorageAuthorizer(ctx, newSdkAuthOptions, *cloud)
	if err != nil {
		return nil, nil, err
	}
	dtlMetaClient := hashiDTLSDK.NewClientWithBaseURI(*resourceManagerEndpoint, func(c *autorest.Client) {
		c.Authorizer = authWrapper.AutorestAuthorizer(resourceManagerAuthorizer)
		c.UserAgent = "some-user-agent"
	})
	azureClient.DtlMetaClient = dtlMetaClient

	azureClient.GalleryImageVersionsClient = hashiGalleryImageVersionsSDK.NewGalleryImageVersionsClientWithBaseURI(*resourceManagerEndpoint)
	azureClient.GalleryImageVersionsClient.Client.Authorizer = authWrapper.AutorestAuthorizer(resourceManagerAuthorizer)
	azureClient.GalleryImageVersionsClient.Client.RequestInspector = withInspection(maxlen)
	azureClient.GalleryImageVersionsClient.Client.ResponseInspector = byConcatDecorators(byInspecting(maxlen), errorCapture(azureClient))
	azureClient.GalleryImageVersionsClient.Client.UserAgent = fmt.Sprintf("%s %s", useragent.String(version.AzurePluginVersion.FormattedVersion()), azureClient.GalleryImageVersionsClient.Client.UserAgent)
	azureClient.GalleryImageVersionsClient.Client.PollingDuration = PollingDuration

	azureClient.GalleryImagesClient = hashiGalleryImagesSDK.NewGalleryImagesClientWithBaseURI(*resourceManagerEndpoint)
	azureClient.GalleryImagesClient.Client.Authorizer = authWrapper.AutorestAuthorizer(resourceManagerAuthorizer)
	azureClient.GalleryImagesClient.Client.RequestInspector = withInspection(maxlen)
	azureClient.GalleryImagesClient.Client.ResponseInspector = byConcatDecorators(byInspecting(maxlen), errorCapture(azureClient))
	azureClient.GalleryImagesClient.Client.UserAgent = fmt.Sprintf("%s %s", useragent.String(version.AzurePluginVersion.FormattedVersion()), azureClient.GalleryImagesClient.Client.UserAgent)
	azureClient.GalleryImagesClient.Client.PollingDuration = PollingDuration

	azureClient.ImagesClient = hashiImagesSDK.NewImagesClientWithBaseURI(*resourceManagerEndpoint)
	azureClient.ImagesClient.Client.Authorizer = authWrapper.AutorestAuthorizer(resourceManagerAuthorizer)
	azureClient.ImagesClient.Client.RequestInspector = withInspection(maxlen)
	azureClient.ImagesClient.Client.ResponseInspector = byConcatDecorators(byInspecting(maxlen), errorCapture(azureClient))
	azureClient.ImagesClient.Client.UserAgent = fmt.Sprintf("%s %s", useragent.String(version.AzurePluginVersion.FormattedVersion()), azureClient.ImagesClient.Client.UserAgent)
	azureClient.ImagesClient.Client.PollingDuration = PollingDuration

	networkMetaClient, err := hashiNetworkSDK.NewClientWithBaseURI(cloud.ResourceManager, func(c *resourcemanager.Client) {
		c.Client.Authorizer = resourceManagerAuthorizer
		c.Client.UserAgent = "some-user-agent"
	})

	if err != nil {
		return nil, nil, err
	}
	azureClient.NetworkMetaClient = *networkMetaClient
	blobClient := giovanniBlobStorageSDK.New()
	azureClient.GiovanniBlobClient = blobClient
	azureClient.GiovanniBlobClient.Authorizer = authWrapper.AutorestAuthorizer(storageAccountAuthorizer)
	azureClient.GiovanniBlobClient.Client.RequestInspector = withInspection(maxlen)
	azureClient.GiovanniBlobClient.Client.ResponseInspector = byConcatDecorators(byInspecting(maxlen), errorCapture(azureClient))
	token, err := resourceManagerAuthorizer.Token(ctx, &http.Request{})
	if err != nil {
		return nil, nil, err
	}
	objectId, err := getObjectIdFromToken(token.AccessToken)
	if err != nil {
		return nil, nil, err
	}
	return azureClient, &objectId, nil
}

const (
	AuthTypeDeviceLogin     = "DeviceLogin"
	AuthTypeMSI             = "ManagedIdentity"
	AuthTypeClientSecret    = "ClientSecret"
	AuthTypeClientCert      = "ClientCertificate"
	AuthTypeClientBearerJWT = "ClientBearerJWT"
	AuthTypeAzureCLI        = "AzureCLI"
)

func buildResourceManagerAuthorizer(ctx context.Context, authOpts NewSDKAuthOptions, env environments.Environment) (auth.Authorizer, error) {
	authorizer, err := buildAuthorizer(ctx, authOpts, env, env.ResourceManager)
	if err != nil {
		return nil, fmt.Errorf("building Resource Manager authorizer from credentials: %+v", err)
	}
	return authorizer, nil
}

func buildStorageAuthorizer(ctx context.Context, authOpts NewSDKAuthOptions, env environments.Environment) (auth.Authorizer, error) {
	authorizer, err := buildAuthorizer(ctx, authOpts, env, env.Storage)
	if err != nil {
		return nil, fmt.Errorf("building Storage authorizer from credentials: %+v", err)
	}
	return authorizer, nil
}

func buildAuthorizer(ctx context.Context, authOpts NewSDKAuthOptions, env environments.Environment, api environments.Api) (auth.Authorizer, error) {
	var authConfig auth.Credentials
	switch authOpts.AuthType {
	case AuthTypeDeviceLogin:
		return nil, fmt.Errorf("DeviceLogin is not supported in v2 of the Azure Packer Plugin, however you can use the Azure CLI `az login --use-device-code` to use a device code, and then use CLI authentication")
	case AuthTypeAzureCLI:
		authConfig = auth.Credentials{
			Environment:                       env,
			EnableAuthenticatingUsingAzureCLI: true,
		}
	case AuthTypeMSI:
		authConfig = auth.Credentials{
			Environment:                              env,
			EnableAuthenticatingUsingManagedIdentity: true,
		}
	case AuthTypeClientSecret:
		authConfig = auth.Credentials{
			Environment:                           env,
			EnableAuthenticatingUsingClientSecret: true,
			ClientID:                              authOpts.ClientID,
			ClientSecret:                          authOpts.ClientSecret,
			TenantID:                              authOpts.TenantID,
		}
	case AuthTypeClientCert:
		authConfig = auth.Credentials{
			Environment: env,
			EnableAuthenticatingUsingClientCertificate: true,
			ClientID:                  authOpts.ClientID,
			ClientCertificatePath:     authOpts.ClientCertPath,
			ClientCertificatePassword: "",
		}
	case AuthTypeClientBearerJWT:
		authConfig = auth.Credentials{
			Environment:                   env,
			EnableAuthenticationUsingOIDC: true,
			ClientID:                      authOpts.ClientID,
			TenantID:                      authOpts.TenantID,
			OIDCAssertionToken:            authOpts.ClientJWT,
		}
	default:
		panic("AuthType not set")
	}
	authorizer, err := auth.NewAuthorizerFromCredentials(ctx, authConfig, api)
	if err != nil {
		return nil, err
	}
	return authorizer, nil
}
func getObjectIdFromToken(token string) (string, error) {
	claims := jwt.MapClaims{}
	var p jwt.Parser

	var err error

	_, _, err = p.ParseUnverified(token, claims)

	if err != nil {
		return "", err
	}
	return claims["oid"].(string), nil
}
func getInspectorMaxLength() int64 {
	value, ok := os.LookupEnv(EnvPackerLogAzureMaxLen)
	if !ok {
		return math.MaxInt64
	}

	i, err := strconv.ParseInt(value, 10, 64)
	if err != nil {
		return 0
	}

	if i < 0 {
		return 0
	}

	return i
}
