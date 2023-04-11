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
	"net/url"

	"github.com/Azure/azure-sdk-for-go/services/compute/mgmt/2021-11-01/compute"
	"github.com/Azure/azure-sdk-for-go/services/keyvault/mgmt/2018-02-14/keyvault"
	"github.com/Azure/azure-sdk-for-go/services/network/mgmt/2018-01-01/network"
	"github.com/Azure/azure-sdk-for-go/services/resources/mgmt/2018-02-01/resources"
	armStorage "github.com/Azure/azure-sdk-for-go/services/storage/mgmt/2017-10-01/storage"
	"github.com/Azure/azure-sdk-for-go/storage"
	"github.com/Azure/go-autorest/autorest"
	"github.com/Azure/go-autorest/autorest/adal"
	"github.com/Azure/go-autorest/autorest/azure"
	hashiGallerySDK "github.com/hashicorp/go-azure-sdk/resource-manager/compute/2021-07-01/galleryimages"
	hashiVMSDK "github.com/hashicorp/go-azure-sdk/resource-manager/compute/2022-03-01/virtualmachines"
	hashiDisksSDK "github.com/hashicorp/go-azure-sdk/resource-manager/compute/2022-03-02/disks"
	"github.com/hashicorp/go-azure-sdk/sdk/auth"
	authWrapper "github.com/hashicorp/go-azure-sdk/sdk/auth/autorest"
	"github.com/hashicorp/go-azure-sdk/sdk/environments"
	"github.com/hashicorp/packer-plugin-azure/builder/azure/common"
	"github.com/hashicorp/packer-plugin-azure/version"
	"github.com/hashicorp/packer-plugin-sdk/useragent"
)

const (
	EnvPackerLogAzureMaxLen = "PACKER_LOG_AZURE_MAXLEN"
)

type AzureClient struct {
	storage.BlobStorageClient
	resources.DeploymentsClient
	resources.DeploymentOperationsClient
	resources.GroupsClient
	network.PublicIPAddressesClient
	network.InterfacesClient
	network.SubnetsClient
	network.VirtualNetworksClient
	network.SecurityGroupsClient
	compute.ImagesClient
	hashiVMSDK.VirtualMachinesClient
	common.VaultClient
	armStorage.AccountsClient
	hashiDisksSDK.DisksClient
	compute.SnapshotsClient
	compute.GalleryImageVersionsClient
	hashiGallerySDK.GalleryImagesClient

	InspectorMaxLength int
	Template           *CaptureTemplate
	LastError          azureErrorResponse
	VaultClientDelete  keyvault.VaultsClient
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
	TenantID       string
	SubscriptionID string
}

func NewAzureClient(subscriptionID, sigSubscriptionID, resourceGroupName, storageAccountName string,
	cloud *azure.Environment, sharedGalleryTimeout time.Duration, pollingDuration time.Duration, newSdkAuthOptions NewSDKAuthOptions,
	servicePrincipalToken, servicePrincipalTokenVault *adal.ServicePrincipalToken) (*AzureClient, error) {

	var azureClient = &AzureClient{}

	maxlen := getInspectorMaxLength()

	azureClient.DeploymentsClient = resources.NewDeploymentsClientWithBaseURI(cloud.ResourceManagerEndpoint, subscriptionID)
	azureClient.DeploymentsClient.Authorizer = autorest.NewBearerAuthorizer(servicePrincipalToken)
	azureClient.DeploymentsClient.RequestInspector = withInspection(maxlen)
	azureClient.DeploymentsClient.ResponseInspector = byConcatDecorators(byInspecting(maxlen), errorCapture(azureClient))
	azureClient.DeploymentsClient.UserAgent = fmt.Sprintf("%s %s", useragent.String(version.AzurePluginVersion.FormattedVersion()), azureClient.DeploymentsClient.UserAgent)
	azureClient.DeploymentsClient.Client.PollingDuration = pollingDuration

	azureClient.DeploymentOperationsClient = resources.NewDeploymentOperationsClientWithBaseURI(cloud.ResourceManagerEndpoint, subscriptionID)
	azureClient.DeploymentOperationsClient.Authorizer = autorest.NewBearerAuthorizer(servicePrincipalToken)
	azureClient.DeploymentOperationsClient.RequestInspector = withInspection(maxlen)
	azureClient.DeploymentOperationsClient.ResponseInspector = byConcatDecorators(byInspecting(maxlen), errorCapture(azureClient))
	azureClient.DeploymentOperationsClient.UserAgent = fmt.Sprintf("%s %s", useragent.String(version.AzurePluginVersion.FormattedVersion()), azureClient.DeploymentOperationsClient.UserAgent)
	azureClient.DeploymentOperationsClient.Client.PollingDuration = pollingDuration

	authorizer, err := buildAuthorizer(context.TODO(), newSdkAuthOptions)
	if err != nil {
		return nil, err
	}
	azureClient.DisksClient = hashiDisksSDK.NewDisksClientWithBaseURI(cloud.ResourceManagerEndpoint)
	azureClient.DisksClient.Client.Authorizer = authWrapper.AutorestAuthorizer(authorizer)
	azureClient.DisksClient.Client.RequestInspector = withInspection(maxlen)
	azureClient.DisksClient.Client.ResponseInspector = byConcatDecorators(byInspecting(maxlen), errorCapture(azureClient))
	azureClient.DisksClient.Client.UserAgent = fmt.Sprintf("%s %s", useragent.String(version.AzurePluginVersion.FormattedVersion()), azureClient.DisksClient.Client.UserAgent)
	azureClient.DisksClient.Client.PollingDuration = pollingDuration

	azureClient.GroupsClient = resources.NewGroupsClientWithBaseURI(cloud.ResourceManagerEndpoint, subscriptionID)
	azureClient.GroupsClient.Authorizer = autorest.NewBearerAuthorizer(servicePrincipalToken)
	azureClient.GroupsClient.RequestInspector = withInspection(maxlen)
	azureClient.GroupsClient.ResponseInspector = byConcatDecorators(byInspecting(maxlen), errorCapture(azureClient))
	azureClient.GroupsClient.UserAgent = fmt.Sprintf("%s %s", useragent.String(version.AzurePluginVersion.FormattedVersion()), azureClient.GroupsClient.UserAgent)
	azureClient.GroupsClient.Client.PollingDuration = pollingDuration

	azureClient.ImagesClient = compute.NewImagesClientWithBaseURI(cloud.ResourceManagerEndpoint, subscriptionID)
	azureClient.ImagesClient.Authorizer = autorest.NewBearerAuthorizer(servicePrincipalToken)
	azureClient.ImagesClient.RequestInspector = withInspection(maxlen)
	azureClient.ImagesClient.ResponseInspector = byConcatDecorators(byInspecting(maxlen), errorCapture(azureClient))
	azureClient.ImagesClient.UserAgent = fmt.Sprintf("%s %s", useragent.String(version.AzurePluginVersion.FormattedVersion()), azureClient.ImagesClient.UserAgent)
	azureClient.ImagesClient.Client.PollingDuration = pollingDuration

	azureClient.InterfacesClient = network.NewInterfacesClientWithBaseURI(cloud.ResourceManagerEndpoint, subscriptionID)
	azureClient.InterfacesClient.Authorizer = autorest.NewBearerAuthorizer(servicePrincipalToken)
	azureClient.InterfacesClient.RequestInspector = withInspection(maxlen)
	azureClient.InterfacesClient.ResponseInspector = byConcatDecorators(byInspecting(maxlen), errorCapture(azureClient))
	azureClient.InterfacesClient.UserAgent = fmt.Sprintf("%s %s", useragent.String(version.AzurePluginVersion.FormattedVersion()), azureClient.InterfacesClient.UserAgent)
	azureClient.InterfacesClient.Client.PollingDuration = pollingDuration

	azureClient.SubnetsClient = network.NewSubnetsClientWithBaseURI(cloud.ResourceManagerEndpoint, subscriptionID)
	azureClient.SubnetsClient.Authorizer = autorest.NewBearerAuthorizer(servicePrincipalToken)
	azureClient.SubnetsClient.RequestInspector = withInspection(maxlen)
	azureClient.SubnetsClient.ResponseInspector = byConcatDecorators(byInspecting(maxlen), errorCapture(azureClient))
	azureClient.SubnetsClient.UserAgent = fmt.Sprintf("%s %s", useragent.String(version.AzurePluginVersion.FormattedVersion()), azureClient.SubnetsClient.UserAgent)
	azureClient.SubnetsClient.Client.PollingDuration = pollingDuration

	azureClient.VirtualNetworksClient = network.NewVirtualNetworksClientWithBaseURI(cloud.ResourceManagerEndpoint, subscriptionID)
	azureClient.VirtualNetworksClient.Authorizer = autorest.NewBearerAuthorizer(servicePrincipalToken)
	azureClient.VirtualNetworksClient.RequestInspector = withInspection(maxlen)
	azureClient.VirtualNetworksClient.ResponseInspector = byConcatDecorators(byInspecting(maxlen), errorCapture(azureClient))
	azureClient.VirtualNetworksClient.UserAgent = fmt.Sprintf("%s %s", useragent.String(version.AzurePluginVersion.FormattedVersion()), azureClient.VirtualNetworksClient.UserAgent)
	azureClient.VirtualNetworksClient.Client.PollingDuration = pollingDuration

	azureClient.SecurityGroupsClient = network.NewSecurityGroupsClientWithBaseURI(cloud.ResourceManagerEndpoint, subscriptionID)
	azureClient.SecurityGroupsClient.Authorizer = autorest.NewBearerAuthorizer(servicePrincipalToken)
	azureClient.SecurityGroupsClient.RequestInspector = withInspection(maxlen)
	azureClient.SecurityGroupsClient.ResponseInspector = byConcatDecorators(byInspecting(maxlen), errorCapture(azureClient))
	azureClient.SecurityGroupsClient.UserAgent = fmt.Sprintf("%s %s", useragent.String(version.AzurePluginVersion.FormattedVersion()), azureClient.SecurityGroupsClient.UserAgent)

	azureClient.PublicIPAddressesClient = network.NewPublicIPAddressesClientWithBaseURI(cloud.ResourceManagerEndpoint, subscriptionID)
	azureClient.PublicIPAddressesClient.Authorizer = autorest.NewBearerAuthorizer(servicePrincipalToken)
	azureClient.PublicIPAddressesClient.RequestInspector = withInspection(maxlen)
	azureClient.PublicIPAddressesClient.ResponseInspector = byConcatDecorators(byInspecting(maxlen), errorCapture(azureClient))
	azureClient.PublicIPAddressesClient.UserAgent = fmt.Sprintf("%s %s", useragent.String(version.AzurePluginVersion.FormattedVersion()), azureClient.PublicIPAddressesClient.UserAgent)
	azureClient.PublicIPAddressesClient.Client.PollingDuration = pollingDuration

	azureClient.VirtualMachinesClient = hashiVMSDK.NewVirtualMachinesClientWithBaseURI(cloud.ResourceManagerEndpoint)
	azureClient.VirtualMachinesClient.Client.Authorizer = autorest.NewBearerAuthorizer(servicePrincipalToken)
	azureClient.VirtualMachinesClient.Client.RequestInspector = withInspection(maxlen)
	azureClient.VirtualMachinesClient.Client.ResponseInspector = byConcatDecorators(byInspecting(maxlen), templateCapture(azureClient), errorCapture(azureClient))
	azureClient.VirtualMachinesClient.Client.UserAgent = fmt.Sprintf("%s %s", useragent.String(version.AzurePluginVersion.FormattedVersion()), azureClient.VirtualMachinesClient.Client.UserAgent)
	azureClient.VirtualMachinesClient.Client.PollingDuration = pollingDuration

	azureClient.SnapshotsClient = compute.NewSnapshotsClientWithBaseURI(cloud.ResourceManagerEndpoint, subscriptionID)
	azureClient.SnapshotsClient.Authorizer = autorest.NewBearerAuthorizer(servicePrincipalToken)
	azureClient.SnapshotsClient.RequestInspector = withInspection(maxlen)
	azureClient.SnapshotsClient.ResponseInspector = byConcatDecorators(byInspecting(maxlen), errorCapture(azureClient))
	azureClient.SnapshotsClient.UserAgent = fmt.Sprintf("%s %s", useragent.String(version.AzurePluginVersion.FormattedVersion()), azureClient.SnapshotsClient.UserAgent)
	azureClient.SnapshotsClient.Client.PollingDuration = pollingDuration

	azureClient.AccountsClient = armStorage.NewAccountsClientWithBaseURI(cloud.ResourceManagerEndpoint, subscriptionID)
	azureClient.AccountsClient.Authorizer = autorest.NewBearerAuthorizer(servicePrincipalToken)
	azureClient.AccountsClient.RequestInspector = withInspection(maxlen)
	azureClient.AccountsClient.ResponseInspector = byConcatDecorators(byInspecting(maxlen), errorCapture(azureClient))
	azureClient.AccountsClient.UserAgent = fmt.Sprintf("%s %s", useragent.String(version.AzurePluginVersion.FormattedVersion()), azureClient.AccountsClient.UserAgent)
	azureClient.AccountsClient.Client.PollingDuration = pollingDuration

	azureClient.GalleryImageVersionsClient = compute.NewGalleryImageVersionsClientWithBaseURI(cloud.ResourceManagerEndpoint, subscriptionID)
	azureClient.GalleryImageVersionsClient.Authorizer = autorest.NewBearerAuthorizer(servicePrincipalToken)
	azureClient.GalleryImageVersionsClient.RequestInspector = withInspection(maxlen)
	azureClient.GalleryImageVersionsClient.ResponseInspector = byConcatDecorators(byInspecting(maxlen), errorCapture(azureClient))
	azureClient.GalleryImageVersionsClient.UserAgent = fmt.Sprintf("%s %s", useragent.String(version.AzurePluginVersion.FormattedVersion()), azureClient.GalleryImageVersionsClient.UserAgent)
	azureClient.GalleryImageVersionsClient.Client.PollingDuration = sharedGalleryTimeout
	if sigSubscriptionID != "" {
		azureClient.GalleryImageVersionsClient.SubscriptionID = sigSubscriptionID
	}

	azureClient.GalleryImagesClient = hashiGallerySDK.NewGalleryImagesClientWithBaseURI(cloud.ResourceManagerEndpoint)
	azureClient.GalleryImagesClient.Client.Authorizer = authWrapper.AutorestAuthorizer(authorizer)
	azureClient.GalleryImagesClient.Client.RequestInspector = withInspection(maxlen)
	azureClient.GalleryImagesClient.Client.ResponseInspector = byConcatDecorators(byInspecting(maxlen), errorCapture(azureClient))
	azureClient.GalleryImagesClient.Client.UserAgent = fmt.Sprintf("%s %s", useragent.String(version.AzurePluginVersion.FormattedVersion()), azureClient.GalleryImagesClient.Client.UserAgent)
	azureClient.GalleryImagesClient.Client.PollingDuration = pollingDuration

	keyVaultURL, err := url.Parse(cloud.KeyVaultEndpoint)
	if err != nil {
		return nil, err
	}

	azureClient.VaultClient = common.NewVaultClient(*keyVaultURL)
	azureClient.VaultClient.Authorizer = autorest.NewBearerAuthorizer(servicePrincipalTokenVault)
	azureClient.VaultClient.RequestInspector = withInspection(maxlen)
	azureClient.VaultClient.ResponseInspector = byConcatDecorators(byInspecting(maxlen), errorCapture(azureClient))
	azureClient.VaultClient.UserAgent = fmt.Sprintf("%s %s", useragent.String(version.AzurePluginVersion.FormattedVersion()), azureClient.VaultClient.UserAgent)
	azureClient.VaultClient.Client.PollingDuration = pollingDuration

	// This client is different than the above because it manages the vault
	// itself rather than the contents of the vault.
	azureClient.VaultClientDelete = keyvault.NewVaultsClientWithBaseURI(cloud.ResourceManagerEndpoint, subscriptionID)
	azureClient.VaultClientDelete.Authorizer = autorest.NewBearerAuthorizer(servicePrincipalToken)
	azureClient.VaultClientDelete.RequestInspector = withInspection(maxlen)
	azureClient.VaultClientDelete.ResponseInspector = byConcatDecorators(byInspecting(maxlen), errorCapture(azureClient))
	azureClient.VaultClientDelete.UserAgent = fmt.Sprintf("%s %s", useragent.String(version.AzurePluginVersion.FormattedVersion()), azureClient.VaultClientDelete.UserAgent)
	azureClient.VaultClientDelete.Client.PollingDuration = pollingDuration

	// If this is a managed disk build, this should be ignored.
	if resourceGroupName != "" && storageAccountName != "" {
		accountKeys, err := azureClient.AccountsClient.ListKeys(context.TODO(), resourceGroupName, storageAccountName)
		if err != nil {
			return nil, err
		}

		storageClient, err := storage.NewClient(
			storageAccountName,
			*(*accountKeys.Keys)[0].Value,
			cloud.StorageEndpointSuffix,
			storage.DefaultAPIVersion,
			true /*useHttps*/)

		if err != nil {
			return nil, err
		}

		azureClient.BlobStorageClient = storageClient.GetBlobService()
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
	case AuthTypeAzureCLI:
		authConfig = auth.Credentials{
			Environment:                       *env,
			EnableAuthenticatingUsingAzureCLI: true,
		}
	case AuthTypeMSI:
	case AuthTypeClientSecret:
		authConfig = auth.Credentials{
			Environment:                           *env,
			EnableAuthenticatingUsingClientSecret: true,
			ClientID:                              authOpts.ClientID,
			ClientSecret:                          authOpts.ClientSecret,
			TenantID:                              authOpts.TenantID,
		}
	case AuthTypeClientCert:
	case AuthTypeClientBearerJWT:
	default:
		panic("AuthType not set, call FillParameters, or set explicitly")
	}

	authorizer, err := auth.NewAuthorizerFromCredentials(ctx, authConfig, env.ResourceManager)
	if err != nil {
		return nil, fmt.Errorf("building authorizer from credentials: %+v", err)
	}
	return authorizer, nil
}
