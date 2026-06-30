// Copyright IBM Corp. 2013, 2025
// SPDX-License-Identifier: MPL-2.0

package dtl

import (
	"context"
	"fmt"
	"math"
	"net/http"
	"os"
	"strconv"
	"time"

	"github.com/hashicorp/go-azure-sdk/resource-manager/compute/2023-07-03/galleryimages"
	"github.com/hashicorp/go-azure-sdk/resource-manager/compute/2023-07-03/galleryimageversions"
	"github.com/hashicorp/go-azure-sdk/resource-manager/compute/2024-03-01/images"
	dtl "github.com/hashicorp/go-azure-sdk/resource-manager/devtestlab/2018-09-15"
	"github.com/hashicorp/go-azure-sdk/resource-manager/keyvault/2023-07-01/vaults"
	networks "github.com/hashicorp/go-azure-sdk/resource-manager/network/2023-09-01"
	"github.com/hashicorp/go-azure-sdk/sdk/client"
	"github.com/hashicorp/go-azure-sdk/sdk/client/resourcemanager"
	"github.com/hashicorp/go-azure-sdk/sdk/environments"
	"github.com/hashicorp/packer-plugin-azure/builder/azure/common"
	commonclient "github.com/hashicorp/packer-plugin-azure/builder/azure/common/client"
	"github.com/hashicorp/packer-plugin-azure/version"
	"github.com/hashicorp/packer-plugin-sdk/useragent"
)

const (
	EnvPackerLogAzureMaxLen = "PACKER_LOG_AZURE_MAXLEN"
)

type AzureClient struct {
	InspectorMaxLength int
	LastError          azureErrorResponse

	images.ImagesClient
	vaults.VaultsClient
	NetworkMetaClient networks.Client
	galleryimageversions.GalleryImageVersionsClient
	galleryimages.GalleryImagesClient
	DtlMetaClient dtl.Client

	PollingDuration           time.Duration
	CustomImageCaptureTimeout time.Duration
	SharedGalleryTimeout      time.Duration
	ObjectId                  string
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
func NewAzureClient(ctx context.Context, subscriptionID string,
	cloud *environments.Environment, SharedGalleryTimeout time.Duration, CustomImageCaptureTimeout time.Duration, PollingDuration time.Duration, authOptions commonclient.AzureAuthOptions) (*AzureClient, error) {

	var azureClient = &AzureClient{}

	maxlen := getInspectorMaxLength()
	responseMiddleware := []client.ResponseMiddleware{common.ByInspecting(maxlen), errorCapture(azureClient)}
	requestMiddleware := []client.RequestMiddleware{common.WithInspection(maxlen)}

	// All requests made using go-azure-sdk require a context with a duration set for polling purposes (even when not polling)
	// These three values are used to set the duration of these contexts for each request
	azureClient.CustomImageCaptureTimeout = CustomImageCaptureTimeout
	azureClient.PollingDuration = PollingDuration
	azureClient.SharedGalleryTimeout = SharedGalleryTimeout

	if cloud == nil || cloud.ResourceManager == nil {
		return nil, fmt.Errorf("Azure Environment not configured correctly")
	}
	resourceManagerAuthorizer, err := commonclient.BuildResourceManagerAuthorizer(ctx, authOptions, *cloud)
	if err != nil {
		return nil, err
	}
	dtlMetaClient, err := dtl.NewClientWithBaseURI(cloud.ResourceManager, func(c *resourcemanager.Client) {
		c.Authorizer = resourceManagerAuthorizer
		c.UserAgent = fmt.Sprintf("%s %s", useragent.String(version.AzurePluginVersion.FormattedVersion()), "go-azure-sdk Meta Client")
		c.Client.ResponseMiddlewares = &responseMiddleware
		c.Client.RequestMiddlewares = &requestMiddleware
	})
	if err != nil {
		return nil, err
	}
	azureClient.DtlMetaClient = *dtlMetaClient

	galleryImageVersionsClient, err := galleryimageversions.NewGalleryImageVersionsClientWithBaseURI(cloud.ResourceManager)
	if err != nil {
		return nil, err
	}
	galleryImageVersionsClient.Client.Authorizer = resourceManagerAuthorizer
	galleryImageVersionsClient.Client.ResponseMiddlewares = &responseMiddleware
	galleryImageVersionsClient.Client.RequestMiddlewares = &requestMiddleware
	galleryImageVersionsClient.Client.UserAgent = fmt.Sprintf("%s %s", useragent.String(version.AzurePluginVersion.FormattedVersion()), galleryImageVersionsClient.Client.UserAgent)

	azureClient.GalleryImageVersionsClient = *galleryImageVersionsClient

	galleryImagesClient, err := galleryimages.NewGalleryImagesClientWithBaseURI(cloud.ResourceManager)
	if err != nil {
		return nil, err
	}
	galleryImagesClient.Client.Authorizer = resourceManagerAuthorizer
	galleryImagesClient.Client.ResponseMiddlewares = &responseMiddleware
	galleryImagesClient.Client.RequestMiddlewares = &requestMiddleware
	galleryImagesClient.Client.UserAgent = fmt.Sprintf("%s %s", useragent.String(version.AzurePluginVersion.FormattedVersion()), galleryImagesClient.Client.UserAgent)
	azureClient.GalleryImagesClient = *galleryImagesClient

	imagesClient, err := images.NewImagesClientWithBaseURI(cloud.ResourceManager)
	if err != nil {
		return nil, err
	}
	imagesClient.Client.Authorizer = resourceManagerAuthorizer
	imagesClient.Client.ResponseMiddlewares = &responseMiddleware
	imagesClient.Client.RequestMiddlewares = &requestMiddleware
	imagesClient.Client.UserAgent = fmt.Sprintf("%s %s", useragent.String(version.AzurePluginVersion.FormattedVersion()), imagesClient.Client.UserAgent)
	azureClient.ImagesClient = *imagesClient

	networkMetaClient, err := networks.NewClientWithBaseURI(cloud.ResourceManager, func(c *resourcemanager.Client) {
		c.Client.Authorizer = resourceManagerAuthorizer
		c.Client.UserAgent = "some-user-agent"
		c.Client.RequestMiddlewares = &requestMiddleware
		c.Client.ResponseMiddlewares = &responseMiddleware
	})

	if err != nil {
		return nil, err
	}
	azureClient.NetworkMetaClient = *networkMetaClient
	token, err := resourceManagerAuthorizer.Token(ctx, &http.Request{})
	if err != nil {
		return nil, err
	}
	objectId, err := commonclient.GetObjectIdFromToken(token.AccessToken)
	if err != nil {
		return nil, err
	}
	azureClient.ObjectId = objectId
	return azureClient, nil
}

const (
	AuthTypeDeviceLogin     = "DeviceLogin"
	AuthTypeMSI             = "ManagedIdentity"
	AuthTypeClientSecret    = "ClientSecret"
	AuthTypeClientCert      = "ClientCertificate"
	AuthTypeClientBearerJWT = "ClientBearerJWT"
	AuthTypeAzureCLI        = "AzureCLI"
)

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
