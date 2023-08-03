// Copyright (c) HashiCorp, Inc.
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

	"github.com/Azure/go-autorest/autorest"
	"github.com/hashicorp/go-azure-sdk/resource-manager/compute/2022-03-01/images"
	"github.com/hashicorp/go-azure-sdk/resource-manager/compute/2022-03-03/galleryimages"
	"github.com/hashicorp/go-azure-sdk/resource-manager/compute/2022-03-03/galleryimageversions"
	dtl "github.com/hashicorp/go-azure-sdk/resource-manager/devtestlab/2018-09-15"
	"github.com/hashicorp/go-azure-sdk/resource-manager/keyvault/2023-02-01/vaults"
	networks "github.com/hashicorp/go-azure-sdk/resource-manager/network/2022-09-01"
	authWrapper "github.com/hashicorp/go-azure-sdk/sdk/auth/autorest"
	"github.com/hashicorp/go-azure-sdk/sdk/client"
	"github.com/hashicorp/go-azure-sdk/sdk/client/resourcemanager"
	"github.com/hashicorp/go-azure-sdk/sdk/environments"
	azcommon "github.com/hashicorp/packer-plugin-azure/builder/azure/common/client"
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

func errorCaptureTrack2(client *AzureClient) client.ResponseMiddleware {
	return func(req *http.Request, resp *http.Response) (*http.Response, error) {
		body, bodyString := handleBody(resp.Body, math.MaxInt64)
		resp.Body = body

		errorResponse := newAzureErrorResponse(bodyString)
		if errorResponse != nil {
			client.LastError = *errorResponse
		}
		return resp, nil
	}
}

func byConcatDecorators(decorators ...autorest.RespondDecorator) autorest.RespondDecorator {
	return func(r autorest.Responder) autorest.Responder {
		return autorest.DecorateResponder(r, decorators...)
	}
}

// Returns an Azure Client used for the Azure Resource Manager
// Also returns the Azure object ID for the authentication method used in the build
func NewAzureClient(ctx context.Context, subscriptionID string,
	cloud *environments.Environment, SharedGalleryTimeout time.Duration, CustomImageCaptureTimeout time.Duration, PollingDuration time.Duration, authOptions azcommon.AzureAuthOptions) (*AzureClient, *string, error) {

	var azureClient = &AzureClient{}

	maxlen := getInspectorMaxLength()
	trackTwoResponseMiddleware := []client.ResponseMiddleware{byInspectingTrack2(maxlen), errorCaptureTrack2(azureClient)}
	trackTwoRequestMiddleware := []client.RequestMiddleware{withInspectionTrack2(maxlen)}

	azureClient.CustomImageCaptureTimeout = CustomImageCaptureTimeout
	azureClient.PollingDuration = PollingDuration
	azureClient.SharedGalleryTimeout = SharedGalleryTimeout

	if cloud == nil || cloud.ResourceManager == nil {
		return nil, nil, fmt.Errorf("Azure Environment not configured correctly")
	}
	resourceManagerEndpoint, _ := cloud.ResourceManager.Endpoint()
	resourceManagerAuthorizer, err := azcommon.BuildResourceManagerAuthorizer(ctx, authOptions, *cloud)
	if err != nil {
		return nil, nil, err
	}
	dtlMetaClient := dtl.NewClientWithBaseURI(*resourceManagerEndpoint, func(c *autorest.Client) {
		c.Authorizer = authWrapper.AutorestAuthorizer(resourceManagerAuthorizer)
		c.UserAgent = fmt.Sprintf("%s %s", useragent.String(version.AzurePluginVersion.FormattedVersion()), "go-azure-sdk Meta Client")
		c.RequestInspector = withInspection(maxlen)
		c.ResponseInspector = byConcatDecorators(byInspecting(maxlen), errorCapture(azureClient))
	})
	azureClient.DtlMetaClient = dtlMetaClient

	azureClient.GalleryImageVersionsClient = galleryimageversions.NewGalleryImageVersionsClientWithBaseURI(*resourceManagerEndpoint)
	azureClient.GalleryImageVersionsClient.Client.Authorizer = authWrapper.AutorestAuthorizer(resourceManagerAuthorizer)
	azureClient.GalleryImageVersionsClient.Client.RequestInspector = withInspection(maxlen)
	azureClient.GalleryImageVersionsClient.Client.ResponseInspector = byConcatDecorators(byInspecting(maxlen), errorCapture(azureClient))
	azureClient.GalleryImageVersionsClient.Client.UserAgent = fmt.Sprintf("%s %s", useragent.String(version.AzurePluginVersion.FormattedVersion()), azureClient.GalleryImageVersionsClient.Client.UserAgent)
	azureClient.GalleryImageVersionsClient.Client.PollingDuration = PollingDuration

	azureClient.GalleryImagesClient = galleryimages.NewGalleryImagesClientWithBaseURI(*resourceManagerEndpoint)
	azureClient.GalleryImagesClient.Client.Authorizer = authWrapper.AutorestAuthorizer(resourceManagerAuthorizer)
	azureClient.GalleryImagesClient.Client.RequestInspector = withInspection(maxlen)
	azureClient.GalleryImagesClient.Client.ResponseInspector = byConcatDecorators(byInspecting(maxlen), errorCapture(azureClient))
	azureClient.GalleryImagesClient.Client.UserAgent = fmt.Sprintf("%s %s", useragent.String(version.AzurePluginVersion.FormattedVersion()), azureClient.GalleryImagesClient.Client.UserAgent)
	azureClient.GalleryImagesClient.Client.PollingDuration = PollingDuration

	azureClient.ImagesClient = images.NewImagesClientWithBaseURI(*resourceManagerEndpoint)
	azureClient.ImagesClient.Client.Authorizer = authWrapper.AutorestAuthorizer(resourceManagerAuthorizer)
	azureClient.ImagesClient.Client.RequestInspector = withInspection(maxlen)
	azureClient.ImagesClient.Client.ResponseInspector = byConcatDecorators(byInspecting(maxlen), errorCapture(azureClient))
	azureClient.ImagesClient.Client.UserAgent = fmt.Sprintf("%s %s", useragent.String(version.AzurePluginVersion.FormattedVersion()), azureClient.ImagesClient.Client.UserAgent)
	azureClient.ImagesClient.Client.PollingDuration = PollingDuration

	networkMetaClient, err := networks.NewClientWithBaseURI(cloud.ResourceManager, func(c *resourcemanager.Client) {
		c.Client.Authorizer = resourceManagerAuthorizer
		c.Client.UserAgent = "some-user-agent"
		c.Client.RequestMiddlewares = &trackTwoRequestMiddleware
		c.Client.ResponseMiddlewares = &trackTwoResponseMiddleware
	})

	if err != nil {
		return nil, nil, err
	}
	azureClient.NetworkMetaClient = *networkMetaClient
	token, err := resourceManagerAuthorizer.Token(ctx, &http.Request{})
	if err != nil {
		return nil, nil, err
	}
	objectId, err := azcommon.GetObjectIdFromToken(token.AccessToken)
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
