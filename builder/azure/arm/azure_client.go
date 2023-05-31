// Copyright (c) HashiCorp, Inc.
// SPDX-License-Identifier: MPL-2.0

package arm

import (
	"context"
	"encoding/json"
	"fmt"
	"math"
	"os"
	"strconv"
	"time"

	"net/http"

	"github.com/Azure/go-autorest/autorest"
	"github.com/Azure/go-autorest/autorest/azure"
	hashiImagesSDK "github.com/hashicorp/go-azure-sdk/resource-manager/compute/2022-03-01/images"
	hashiVMSDK "github.com/hashicorp/go-azure-sdk/resource-manager/compute/2022-03-01/virtualmachines"
	hashiDisksSDK "github.com/hashicorp/go-azure-sdk/resource-manager/compute/2022-03-02/disks"
	hashiSnapshotsSDK "github.com/hashicorp/go-azure-sdk/resource-manager/compute/2022-03-02/snapshots"
	hashiGalleryImagesSDK "github.com/hashicorp/go-azure-sdk/resource-manager/compute/2022-03-03/galleryimages"
	hashiGalleryImageVersionsSDK "github.com/hashicorp/go-azure-sdk/resource-manager/compute/2022-03-03/galleryimageversions"
	hashiSecretsSDK "github.com/hashicorp/go-azure-sdk/resource-manager/keyvault/2023-02-01/secrets"
	hashiVaultsSDK "github.com/hashicorp/go-azure-sdk/resource-manager/keyvault/2023-02-01/vaults"
	hashiNetworkMetaSDK "github.com/hashicorp/go-azure-sdk/resource-manager/network/2022-09-01"
	hashiDeploymentOperationsSDK "github.com/hashicorp/go-azure-sdk/resource-manager/resources/2022-09-01/deploymentoperations"
	hashiDeploymentsSDK "github.com/hashicorp/go-azure-sdk/resource-manager/resources/2022-09-01/deployments"
	hashiGroupsSDK "github.com/hashicorp/go-azure-sdk/resource-manager/resources/2022-09-01/resourcegroups"
	hashiBlobContainersSDK "github.com/hashicorp/go-azure-sdk/resource-manager/storage/2022-09-01/blobcontainers"
	hashiStorageAccountsSDK "github.com/hashicorp/go-azure-sdk/resource-manager/storage/2022-09-01/storageaccounts"
	"github.com/hashicorp/go-azure-sdk/sdk/auth"
	authWrapper "github.com/hashicorp/go-azure-sdk/sdk/auth/autorest"
	"github.com/hashicorp/go-azure-sdk/sdk/client/resourcemanager"
	"github.com/hashicorp/go-azure-sdk/sdk/environments"
	"github.com/hashicorp/packer-plugin-azure/version"
	"github.com/hashicorp/packer-plugin-sdk/useragent"
)

const (
	EnvPackerLogAzureMaxLen = "PACKER_LOG_AZURE_MAXLEN"
)

type AzureClient struct {
	hashiBlobContainersSDK.BlobContainersClient
	NetworkMetaClient hashiNetworkMetaSDK.Client
	hashiDeploymentsSDK.DeploymentsClient
	hashiStorageAccountsSDK.StorageAccountsClient
	hashiDeploymentOperationsSDK.DeploymentOperationsClient
	hashiImagesSDK.ImagesClient
	hashiVMSDK.VirtualMachinesClient
	hashiSecretsSDK.SecretsClient
	hashiVaultsSDK.VaultsClient
	hashiDisksSDK.DisksClient
	hashiGroupsSDK.ResourceGroupsClient
	hashiSnapshotsSDK.SnapshotsClient
	hashiGalleryImageVersionsSDK.GalleryImageVersionsClient
	hashiGalleryImagesSDK.GalleryImagesClient

	InspectorMaxLength int
	Template           *CaptureTemplate
	LastError          azureErrorResponse
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

// HACK(chrboum): This method is a hack.  It was written to work around this issue
// (https://github.com/Azure/azure-sdk-for-go/issues/307) and to an extent this
// issue (https://github.com/Azure/azure-rest-api-specs/issues/188).
//
// Capturing a VM is a long running operation that requires polling.  There are
// couple different forms of polling, and the end result of a poll operation is
// discarded by the SDK.  It is expected that any discarded data can be re-fetched,
// so discarding it has minimal impact.  Unfortunately, there is no way to re-fetch
// the template returned by a capture call that I am aware of.
//
// If the second issue were fixed the VM ID would be included when GET'ing a VM.  The
// VM ID could be used to locate the captured VHD, and captured template.
// Unfortunately, the VM ID is not included so this method cannot be used either.
//
// This code captures the template and saves it to the client (the AzureClient type).
// It expects that the capture API is called only once, or rather you only care that the
// last call's value is important because subsequent requests are not persisted.  There
// is no care given to multiple threads writing this value because for our use case
// it does not matter.
func templateCapture(client *AzureClient) autorest.RespondDecorator {
	return func(r autorest.Responder) autorest.Responder {
		return autorest.ResponderFunc(func(resp *http.Response) error {
			body, bodyString := handleBody(resp.Body, math.MaxInt64)
			resp.Body = body

			captureTemplate := getCaptureResponse(bodyString)
			if captureTemplate != nil {
				client.Template = captureTemplate
			}

			return r.Respond(resp)
		})
	}
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

func NewAzureClient(resourceGroupName, storageAccountName string, cloud *azure.Environment, sharedGalleryTimeout time.Duration, pollingDuration time.Duration, newSdkAuthOptions NewSDKAuthOptions) (*AzureClient, error) {

	var azureClient = &AzureClient{}

	maxlen := getInspectorMaxLength()

	authorizer, err := buildAuthorizer(context.TODO(), newSdkAuthOptions)
	if err != nil {
		return nil, err
	}

	// Clients that have been ported to hashicorp/go-azure-sdk
	azureClient.DisksClient = hashiDisksSDK.NewDisksClientWithBaseURI(cloud.ResourceManagerEndpoint)
	azureClient.DisksClient.Client.Authorizer = authWrapper.AutorestAuthorizer(authorizer)
	azureClient.DisksClient.Client.RequestInspector = withInspection(maxlen)
	azureClient.DisksClient.Client.ResponseInspector = byConcatDecorators(byInspecting(maxlen), errorCapture(azureClient))
	azureClient.DisksClient.Client.UserAgent = fmt.Sprintf("%s %s", useragent.String(version.AzurePluginVersion.FormattedVersion()), azureClient.DisksClient.Client.UserAgent)
	azureClient.DisksClient.Client.PollingDuration = pollingDuration

	azureClient.VirtualMachinesClient = hashiVMSDK.NewVirtualMachinesClientWithBaseURI(cloud.ResourceManagerEndpoint)
	azureClient.VirtualMachinesClient.Client.Authorizer = authWrapper.AutorestAuthorizer(authorizer)
	azureClient.VirtualMachinesClient.Client.RequestInspector = withInspection(maxlen)
	azureClient.VirtualMachinesClient.Client.ResponseInspector = byConcatDecorators(byInspecting(maxlen), templateCapture(azureClient), errorCapture(azureClient))
	azureClient.VirtualMachinesClient.Client.UserAgent = fmt.Sprintf("%s %s", useragent.String(version.AzurePluginVersion.FormattedVersion()), azureClient.VirtualMachinesClient.Client.UserAgent)
	azureClient.VirtualMachinesClient.Client.PollingDuration = pollingDuration

	azureClient.SnapshotsClient = hashiSnapshotsSDK.NewSnapshotsClientWithBaseURI(cloud.ResourceManagerEndpoint)
	azureClient.SnapshotsClient.Client.Authorizer = authWrapper.AutorestAuthorizer(authorizer)
	azureClient.SnapshotsClient.Client.RequestInspector = withInspection(maxlen)
	azureClient.SnapshotsClient.Client.ResponseInspector = byConcatDecorators(byInspecting(maxlen), errorCapture(azureClient))
	azureClient.SnapshotsClient.Client.UserAgent = fmt.Sprintf("%s %s", useragent.String(version.AzurePluginVersion.FormattedVersion()), azureClient.SnapshotsClient.Client.UserAgent)
	azureClient.SnapshotsClient.Client.PollingDuration = pollingDuration

	azureClient.SecretsClient = hashiSecretsSDK.NewSecretsClientWithBaseURI(cloud.ResourceManagerEndpoint)
	azureClient.SecretsClient.Client.Authorizer = authWrapper.AutorestAuthorizer(authorizer)
	azureClient.SecretsClient.Client.RequestInspector = withInspection(maxlen)
	azureClient.SecretsClient.Client.ResponseInspector = byConcatDecorators(byInspecting(maxlen), errorCapture(azureClient))
	azureClient.SecretsClient.Client.UserAgent = fmt.Sprintf("%s %s", useragent.String(version.AzurePluginVersion.FormattedVersion()), azureClient.SecretsClient.Client.UserAgent)
	azureClient.SecretsClient.Client.PollingDuration = pollingDuration

	azureClient.VaultsClient = hashiVaultsSDK.NewVaultsClientWithBaseURI(cloud.ResourceManagerEndpoint)
	azureClient.VaultsClient.Client.Authorizer = authWrapper.AutorestAuthorizer(authorizer)
	azureClient.VaultsClient.Client.RequestInspector = withInspection(maxlen)
	azureClient.VaultsClient.Client.ResponseInspector = byConcatDecorators(byInspecting(maxlen), errorCapture(azureClient))
	azureClient.VaultsClient.Client.UserAgent = fmt.Sprintf("%s %s", useragent.String(version.AzurePluginVersion.FormattedVersion()), azureClient.VaultsClient.Client.UserAgent)
	azureClient.VaultsClient.Client.PollingDuration = pollingDuration

	azureClient.DeploymentsClient = hashiDeploymentsSDK.NewDeploymentsClientWithBaseURI(cloud.ResourceManagerEndpoint)
	azureClient.DeploymentsClient.Client.Authorizer = authWrapper.AutorestAuthorizer(authorizer)
	azureClient.DeploymentsClient.Client.RequestInspector = withInspection(maxlen)
	azureClient.DeploymentsClient.Client.ResponseInspector = byConcatDecorators(byInspecting(maxlen), errorCapture(azureClient))
	azureClient.DeploymentsClient.Client.UserAgent = fmt.Sprintf("%s %s", useragent.String(version.AzurePluginVersion.FormattedVersion()), azureClient.DeploymentsClient.Client.UserAgent)
	azureClient.DeploymentsClient.Client.PollingDuration = pollingDuration

	azureClient.DeploymentOperationsClient = hashiDeploymentOperationsSDK.NewDeploymentOperationsClientWithBaseURI(cloud.ResourceManagerEndpoint)
	azureClient.DeploymentOperationsClient.Client.Authorizer = authWrapper.AutorestAuthorizer(authorizer)
	azureClient.DeploymentOperationsClient.Client.RequestInspector = withInspection(maxlen)
	azureClient.DeploymentOperationsClient.Client.ResponseInspector = byConcatDecorators(byInspecting(maxlen), errorCapture(azureClient))
	azureClient.DeploymentOperationsClient.Client.UserAgent = fmt.Sprintf("%s %s", useragent.String(version.AzurePluginVersion.FormattedVersion()), azureClient.DeploymentOperationsClient.Client.UserAgent)
	azureClient.DeploymentOperationsClient.Client.PollingDuration = pollingDuration

	azureClient.ResourceGroupsClient = hashiGroupsSDK.NewResourceGroupsClientWithBaseURI(cloud.ResourceManagerEndpoint)
	azureClient.ResourceGroupsClient.Client.Authorizer = authWrapper.AutorestAuthorizer(authorizer)
	azureClient.ResourceGroupsClient.Client.RequestInspector = withInspection(maxlen)
	azureClient.ResourceGroupsClient.Client.ResponseInspector = byConcatDecorators(byInspecting(maxlen), errorCapture(azureClient))
	azureClient.ResourceGroupsClient.Client.UserAgent = fmt.Sprintf("%s %s", useragent.String(version.AzurePluginVersion.FormattedVersion()), azureClient.ResourceGroupsClient.Client.UserAgent)
	azureClient.ResourceGroupsClient.Client.PollingDuration = pollingDuration

	azureClient.ImagesClient = hashiImagesSDK.NewImagesClientWithBaseURI(cloud.ResourceManagerEndpoint)
	azureClient.ImagesClient.Client.Authorizer = authWrapper.AutorestAuthorizer(authorizer)
	azureClient.ImagesClient.Client.RequestInspector = withInspection(maxlen)
	azureClient.ImagesClient.Client.ResponseInspector = byConcatDecorators(byInspecting(maxlen), errorCapture(azureClient))
	azureClient.ImagesClient.Client.UserAgent = fmt.Sprintf("%s %s", useragent.String(version.AzurePluginVersion.FormattedVersion()), azureClient.ImagesClient.Client.UserAgent)
	azureClient.ImagesClient.Client.PollingDuration = pollingDuration

	// Clients that are using the existing SDK/auth logic
	azureClient.StorageAccountsClient = hashiStorageAccountsSDK.NewStorageAccountsClientWithBaseURI(cloud.ResourceManagerEndpoint)
	azureClient.StorageAccountsClient.Client.Authorizer = authWrapper.AutorestAuthorizer(authorizer)
	azureClient.StorageAccountsClient.Client.RequestInspector = withInspection(maxlen)
	azureClient.StorageAccountsClient.Client.ResponseInspector = byConcatDecorators(byInspecting(maxlen), errorCapture(azureClient))
	azureClient.StorageAccountsClient.Client.UserAgent = fmt.Sprintf("%s %s", useragent.String(version.AzurePluginVersion.FormattedVersion()), azureClient.StorageAccountsClient.Client.UserAgent)
	azureClient.StorageAccountsClient.Client.PollingDuration = pollingDuration

	api := environments.AzurePublic().ResourceManager
	networkMetaClient, err := hashiNetworkMetaSDK.NewClientWithBaseURI(api, func(c *resourcemanager.Client) {
		c.Client.Authorizer = authorizer
		c.Client.UserAgent = "some-user-agent"
	})
	if err != nil {
		return nil, err
	}
	azureClient.NetworkMetaClient = *networkMetaClient

	azureClient.GalleryImageVersionsClient = hashiGalleryImageVersionsSDK.NewGalleryImageVersionsClientWithBaseURI(cloud.ResourceManagerEndpoint)
	azureClient.GalleryImageVersionsClient.Client.Authorizer = authWrapper.AutorestAuthorizer(authorizer)
	azureClient.GalleryImageVersionsClient.Client.RequestInspector = withInspection(maxlen)
	azureClient.GalleryImageVersionsClient.Client.ResponseInspector = byConcatDecorators(byInspecting(maxlen), errorCapture(azureClient))
	azureClient.GalleryImageVersionsClient.Client.UserAgent = fmt.Sprintf("%s %s", useragent.String(version.AzurePluginVersion.FormattedVersion()), azureClient.GalleryImageVersionsClient.Client.UserAgent)
	azureClient.GalleryImageVersionsClient.Client.PollingDuration = sharedGalleryTimeout

	azureClient.GalleryImagesClient = hashiGalleryImagesSDK.NewGalleryImagesClientWithBaseURI(cloud.ResourceManagerEndpoint)
	azureClient.GalleryImagesClient.Client.Authorizer = authWrapper.AutorestAuthorizer(authorizer)
	azureClient.GalleryImagesClient.Client.RequestInspector = withInspection(maxlen)
	azureClient.GalleryImagesClient.Client.ResponseInspector = byConcatDecorators(byInspecting(maxlen), errorCapture(azureClient))
	azureClient.GalleryImagesClient.Client.UserAgent = fmt.Sprintf("%s %s", useragent.String(version.AzurePluginVersion.FormattedVersion()), azureClient.GalleryImagesClient.Client.UserAgent)
	azureClient.GalleryImagesClient.Client.PollingDuration = pollingDuration

	// The Blob Client is only used for VHD builds
	if resourceGroupName != "" && storageAccountName != "" {
		azureClient.BlobContainersClient = hashiBlobContainersSDK.NewBlobContainersClientWithBaseURI(cloud.ResourceManagerEndpoint)
		azureClient.BlobContainersClient.Client.Authorizer = authWrapper.AutorestAuthorizer(authorizer)
		azureClient.BlobContainersClient.Client.RequestInspector = withInspection(maxlen)
		azureClient.BlobContainersClient.Client.ResponseInspector = byConcatDecorators(byInspecting(maxlen), errorCapture(azureClient))
		azureClient.BlobContainersClient.Client.UserAgent = fmt.Sprintf("%s %s", useragent.String(version.AzurePluginVersion.FormattedVersion()), azureClient.GalleryImagesClient.Client.UserAgent)
		azureClient.BlobContainersClient.Client.PollingDuration = pollingDuration
	}

	return azureClient, nil
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

const (
	AuthTypeDeviceLogin     = "DeviceLogin"
	AuthTypeMSI             = "ManagedIdentity"
	AuthTypeClientSecret    = "ClientSecret"
	AuthTypeClientCert      = "ClientCertificate"
	AuthTypeClientBearerJWT = "ClientBearerJWT"
	AuthTypeAzureCLI        = "AzureCLI"
)

func buildAuthorizer(ctx context.Context, authOpts NewSDKAuthOptions) (auth.Authorizer, error) {
	env := environments.AzurePublic()
	var authConfig auth.Credentials
	switch authOpts.AuthType {
	case AuthTypeDeviceLogin:
		panic("Not implemented currently")
	case AuthTypeAzureCLI:
		authConfig = auth.Credentials{
			Environment:                       *env,
			EnableAuthenticatingUsingAzureCLI: true,
		}
	case AuthTypeMSI:
		authConfig = auth.Credentials{
			Environment:                              *env,
			EnableAuthenticatingUsingManagedIdentity: true,
		}
	case AuthTypeClientSecret:
		authConfig = auth.Credentials{
			Environment:                           *env,
			EnableAuthenticatingUsingClientSecret: true,
			ClientID:                              authOpts.ClientID,
			ClientSecret:                          authOpts.ClientSecret,
			TenantID:                              authOpts.TenantID,
		}
	case AuthTypeClientCert:
		authConfig = auth.Credentials{
			Environment: *env,
			EnableAuthenticatingUsingClientCertificate: true,
			ClientID:                  authOpts.ClientID,
			ClientCertificatePath:     authOpts.ClientCertPath,
			ClientCertificatePassword: "",
		}
	case AuthTypeClientBearerJWT:
		authConfig = auth.Credentials{
			Environment:                   *env,
			EnableAuthenticationUsingOIDC: true,
			ClientID:                      authOpts.ClientID,
			TenantID:                      authOpts.TenantID,
			OIDCAssertionToken:            authOpts.ClientJWT,
		}
	default:
		panic("AuthType not set")
	}
	authorizer, err := auth.NewAuthorizerFromCredentials(ctx, authConfig, env.ResourceManager)
	if err != nil {
		return nil, fmt.Errorf("building authorizer from credentials: %+v", err)
	}
	return authorizer, nil
}
