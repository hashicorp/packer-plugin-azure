// Copyright (c) HashiCorp, Inc.
// SPDX-License-Identifier: MPL-2.0

package arm

import (
	"context"
	"fmt"
	"math"
	"os"
	"strconv"
	"time"

	"net/http"

	"github.com/hashicorp/go-azure-sdk/resource-manager/compute/2022-03-01/images"
	"github.com/hashicorp/go-azure-sdk/resource-manager/compute/2022-03-01/virtualmachines"
	"github.com/hashicorp/go-azure-sdk/resource-manager/compute/2022-03-02/disks"
	"github.com/hashicorp/go-azure-sdk/resource-manager/compute/2022-03-02/snapshots"
	"github.com/hashicorp/go-azure-sdk/resource-manager/compute/2022-03-03/galleryimages"
	"github.com/hashicorp/go-azure-sdk/resource-manager/compute/2022-03-03/galleryimageversions"
	"github.com/hashicorp/go-azure-sdk/resource-manager/keyvault/2023-07-01/secrets"
	"github.com/hashicorp/go-azure-sdk/resource-manager/keyvault/2023-07-01/vaults"
	networks "github.com/hashicorp/go-azure-sdk/resource-manager/network/2023-09-01"
	"github.com/hashicorp/go-azure-sdk/resource-manager/resources/2022-09-01/deploymentoperations"
	"github.com/hashicorp/go-azure-sdk/resource-manager/resources/2022-09-01/deployments"
	"github.com/hashicorp/go-azure-sdk/resource-manager/resources/2022-09-01/resourcegroups"
	"github.com/hashicorp/go-azure-sdk/resource-manager/storage/2023-01-01/storageaccounts"
	"github.com/hashicorp/go-azure-sdk/sdk/client"
	"github.com/hashicorp/go-azure-sdk/sdk/client/resourcemanager"
	"github.com/hashicorp/go-azure-sdk/sdk/environments"
	"github.com/hashicorp/packer-plugin-azure/builder/azure/common"
	commonclient "github.com/hashicorp/packer-plugin-azure/builder/azure/common/client"
	"github.com/hashicorp/packer-plugin-azure/version"
	"github.com/hashicorp/packer-plugin-sdk/useragent"
	giovanniBlobStorageSDK "github.com/tombuildsstuff/giovanni/storage/2020-08-04/blob/blobs"
)

const (
	EnvPackerLogAzureMaxLen = "PACKER_LOG_AZURE_MAXLEN"
)

type AzureClient struct {
	NetworkMetaClient networks.Client
	deployments.DeploymentsClient
	storageaccounts.StorageAccountsClient
	deploymentoperations.DeploymentOperationsClient
	images.ImagesClient
	virtualmachines.VirtualMachinesClient
	secrets.SecretsClient
	vaults.VaultsClient
	disks.DisksClient
	resourcegroups.ResourceGroupsClient
	snapshots.SnapshotsClient
	galleryimageversions.GalleryImageVersionsClient
	galleryimages.GalleryImagesClient
	GiovanniBlobClient giovanniBlobStorageSDK.Client
	InspectorMaxLength int
	LastError          azureErrorResponse

	ObjectID             string
	PollingDuration      time.Duration
	SharedGalleryTimeout time.Duration
}

func errorCapture(client *AzureClient) client.ResponseMiddleware {
	return func(req *http.Request, resp *http.Response) (*http.Response, error) {
		body, bodyString := common.HandleBody(resp.Body, math.MaxInt64)
		resp.Body = body

		errorResponse := newAzureErrorResponse(bodyString)
		if errorResponse != nil {
			client.LastError = *errorResponse
		}
		return resp, nil
	}
}

// Returns an Azure Client used for the Azure Resource Manager
func NewAzureClient(ctx context.Context, isVHDBuild bool, cloud *environments.Environment, sharedGalleryTimeout time.Duration, pollingDuration time.Duration, authOptions commonclient.AzureAuthOptions) (*AzureClient, error) {

	var azureClient = &AzureClient{}

	// All requests made using go-azure-sdk require a context with a duration set for polling purposes (even when not polling)
	// These two values are used to set the duration of these contexts for each request
	azureClient.PollingDuration = pollingDuration
	azureClient.SharedGalleryTimeout = sharedGalleryTimeout

	maxlen := getInspectorMaxLength()
	if cloud == nil || cloud.ResourceManager == nil {
		return nil, fmt.Errorf("azure environment not configured correctly")
	}
	resourceManagerAuthorizer, err := commonclient.BuildResourceManagerAuthorizer(ctx, authOptions, *cloud)
	if err != nil {
		return nil, err
	}

	trackTwoResponseMiddleware := []client.ResponseMiddleware{common.ByInspecting(maxlen), errorCapture(azureClient)}
	trackTwoRequestMiddleware := []client.RequestMiddleware{common.WithInspection(maxlen)}

	disksClient, err := disks.NewDisksClientWithBaseURI(cloud.ResourceManager)
	if err != nil {
		return nil, err
	}
	disksClient.Client.Authorizer = resourceManagerAuthorizer
	disksClient.Client.UserAgent = useragent.String(version.AzurePluginVersion.FormattedVersion())
	disksClient.Client.ResponseMiddlewares = &trackTwoResponseMiddleware
	disksClient.Client.RequestMiddlewares = &trackTwoRequestMiddleware
	azureClient.DisksClient = *disksClient

	virtualMachinesClient, err := virtualmachines.NewVirtualMachinesClientWithBaseURI(cloud.ResourceManager)
	if err != nil {
		return nil, err
	}
	virtualMachinesClient.Client.Authorizer = resourceManagerAuthorizer
	virtualMachinesClient.Client.ResponseMiddlewares = &trackTwoResponseMiddleware
	virtualMachinesClient.Client.RequestMiddlewares = &trackTwoRequestMiddleware
	virtualMachinesClient.Client.UserAgent = fmt.Sprintf("%s %s", useragent.String(version.AzurePluginVersion.FormattedVersion()), virtualMachinesClient.Client.UserAgent)
	azureClient.VirtualMachinesClient = *virtualMachinesClient

	snapshotsClient, err := snapshots.NewSnapshotsClientWithBaseURI(cloud.ResourceManager)
	if err != nil {
		return nil, err
	}
	snapshotsClient.Client.Authorizer = resourceManagerAuthorizer
	snapshotsClient.Client.ResponseMiddlewares = &trackTwoResponseMiddleware
	snapshotsClient.Client.RequestMiddlewares = &trackTwoRequestMiddleware
	snapshotsClient.Client.UserAgent = fmt.Sprintf("%s %s", useragent.String(version.AzurePluginVersion.FormattedVersion()), snapshotsClient.Client.UserAgent)
	azureClient.SnapshotsClient = *snapshotsClient

	vaultsClient, err := vaults.NewVaultsClientWithBaseURI(cloud.ResourceManager)
	if err != nil {
		return nil, err
	}
	vaultsClient.Client.Authorizer = resourceManagerAuthorizer
	vaultsClient.Client.ResponseMiddlewares = &trackTwoResponseMiddleware
	vaultsClient.Client.RequestMiddlewares = &trackTwoRequestMiddleware
	vaultsClient.Client.UserAgent = fmt.Sprintf("%s %s", useragent.String(version.AzurePluginVersion.FormattedVersion()), vaultsClient.Client.UserAgent)
	azureClient.VaultsClient = *vaultsClient

	secretsClient, err := secrets.NewSecretsClientWithBaseURI(cloud.ResourceManager)
	if err != nil {
		return nil, err
	}
	secretsClient.Client.Authorizer = resourceManagerAuthorizer
	secretsClient.Client.ResponseMiddlewares = &trackTwoResponseMiddleware
	secretsClient.Client.RequestMiddlewares = &trackTwoRequestMiddleware
	secretsClient.Client.UserAgent = fmt.Sprintf("%s %s", useragent.String(version.AzurePluginVersion.FormattedVersion()), vaultsClient.Client.UserAgent)
	azureClient.SecretsClient = *secretsClient

	deploymentsClient, err := deployments.NewDeploymentsClientWithBaseURI(cloud.ResourceManager)
	if err != nil {
		return nil, err
	}
	deploymentsClient.Client.ResponseMiddlewares = &trackTwoResponseMiddleware
	deploymentsClient.Client.RequestMiddlewares = &trackTwoRequestMiddleware
	deploymentsClient.Client.Authorizer = resourceManagerAuthorizer
	deploymentsClient.Client.UserAgent = fmt.Sprintf("%s %s", useragent.String(version.AzurePluginVersion.FormattedVersion()), deploymentsClient.Client.UserAgent)
	azureClient.DeploymentsClient = *deploymentsClient

	deploymentOperationsClient, err := deploymentoperations.NewDeploymentOperationsClientWithBaseURI(cloud.ResourceManager)
	if err != nil {
		return nil, err
	}
	deploymentOperationsClient.Client.Authorizer = resourceManagerAuthorizer
	deploymentOperationsClient.Client.ResponseMiddlewares = &trackTwoResponseMiddleware
	deploymentOperationsClient.Client.RequestMiddlewares = &trackTwoRequestMiddleware
	deploymentOperationsClient.Client.UserAgent = fmt.Sprintf("%s %s", useragent.String(version.AzurePluginVersion.FormattedVersion()), deploymentOperationsClient.Client.UserAgent)
	azureClient.DeploymentOperationsClient = *deploymentOperationsClient

	resourceGroupsClient, err := resourcegroups.NewResourceGroupsClientWithBaseURI(cloud.ResourceManager)
	if err != nil {
		return nil, err
	}
	resourceGroupsClient.Client.Authorizer = resourceManagerAuthorizer
	resourceGroupsClient.Client.ResponseMiddlewares = &trackTwoResponseMiddleware
	resourceGroupsClient.Client.RequestMiddlewares = &trackTwoRequestMiddleware
	resourceGroupsClient.Client.UserAgent = fmt.Sprintf("%s %s", useragent.String(version.AzurePluginVersion.FormattedVersion()), resourceGroupsClient.Client.UserAgent)
	azureClient.ResourceGroupsClient = *resourceGroupsClient

	imagesClient, err := images.NewImagesClientWithBaseURI(cloud.ResourceManager)
	if err != nil {
		return nil, err
	}
	imagesClient.Client.Authorizer = resourceManagerAuthorizer
	imagesClient.Client.ResponseMiddlewares = &trackTwoResponseMiddleware
	imagesClient.Client.RequestMiddlewares = &trackTwoRequestMiddleware
	imagesClient.Client.UserAgent = fmt.Sprintf("%s %s", useragent.String(version.AzurePluginVersion.FormattedVersion()), imagesClient.Client.UserAgent)
	azureClient.ImagesClient = *imagesClient

	storageAccountsClient, err := storageaccounts.NewStorageAccountsClientWithBaseURI(cloud.ResourceManager)
	if err != nil {
		return nil, err
	}
	storageAccountsClient.Client.Authorizer = resourceManagerAuthorizer
	storageAccountsClient.Client.ResponseMiddlewares = &trackTwoResponseMiddleware
	storageAccountsClient.Client.RequestMiddlewares = &trackTwoRequestMiddleware
	storageAccountsClient.Client.UserAgent = fmt.Sprintf("%s %s", useragent.String(version.AzurePluginVersion.FormattedVersion()), storageAccountsClient.Client.UserAgent)
	azureClient.StorageAccountsClient = *storageAccountsClient

	networkMetaClient, err := networks.NewClientWithBaseURI(cloud.ResourceManager, func(c *resourcemanager.Client) {
		c.Client.Authorizer = resourceManagerAuthorizer
		c.Client.UserAgent = useragent.String(version.AzurePluginVersion.FormattedVersion())
		c.Client.ResponseMiddlewares = &trackTwoResponseMiddleware
		c.Client.RequestMiddlewares = &trackTwoRequestMiddleware
	})
	if err != nil {
		return nil, err
	}
	azureClient.NetworkMetaClient = *networkMetaClient

	galleryImageVersionsClient, err := galleryimageversions.NewGalleryImageVersionsClientWithBaseURI(cloud.ResourceManager)
	if err != nil {
		return nil, err
	}
	galleryImageVersionsClient.Client.Authorizer = resourceManagerAuthorizer
	galleryImageVersionsClient.Client.ResponseMiddlewares = &trackTwoResponseMiddleware
	galleryImageVersionsClient.Client.RequestMiddlewares = &trackTwoRequestMiddleware
	galleryImageVersionsClient.Client.UserAgent = fmt.Sprintf("%s %s", useragent.String(version.AzurePluginVersion.FormattedVersion()), galleryImageVersionsClient.Client.UserAgent)
	azureClient.GalleryImageVersionsClient = *galleryImageVersionsClient

	galleryImagesClient, err := galleryimages.NewGalleryImagesClientWithBaseURI(cloud.ResourceManager)
	if err != nil {
		return nil, err
	}
	galleryImagesClient.Client.Authorizer = resourceManagerAuthorizer
	galleryImagesClient.Client.ResponseMiddlewares = &trackTwoResponseMiddleware
	galleryImagesClient.Client.RequestMiddlewares = &trackTwoRequestMiddleware
	galleryImagesClient.Client.UserAgent = fmt.Sprintf("%s %s", useragent.String(version.AzurePluginVersion.FormattedVersion()), galleryImagesClient.Client.UserAgent)
	azureClient.GalleryImagesClient = *galleryImagesClient

	// We only need the Blob Client to delete the OS VHD during VHD builds
	if isVHDBuild {
		storageAccountAuthorizer, err := commonclient.BuildStorageAuthorizer(ctx, authOptions, *cloud)
		if err != nil {
			return nil, err
		}

		blobClient, err := giovanniBlobStorageSDK.NewWithBaseUri(cloud.Storage.Name())
		if err != nil {
			return nil, err
		}
		blobClient.Client.Authorizer = storageAccountAuthorizer
		blobClient.Client.RequestMiddlewares = &trackTwoRequestMiddleware
		blobClient.Client.ResponseMiddlewares = &trackTwoResponseMiddleware
		azureClient.GiovanniBlobClient = *blobClient
	}

	token, err := resourceManagerAuthorizer.Token(ctx, &http.Request{})
	if err != nil {
		return nil, err
	}
	if token == nil {
		return nil, fmt.Errorf("unable to parse token from Azure Resource Manager")
	}
	objectId, err := commonclient.GetObjectIdFromToken(token.AccessToken)
	if err != nil {
		return nil, err
	}
	azureClient.ObjectID = objectId
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
