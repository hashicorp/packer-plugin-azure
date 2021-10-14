package chroot

import (
	"bytes"
	"context"
	"io/ioutil"
	"net/http"
	"strings"
	"testing"

	"github.com/Azure/azure-sdk-for-go/services/compute/mgmt/2019-12-01/compute"
	"github.com/Azure/go-autorest/autorest"
	"github.com/hashicorp/packer-plugin-azure/builder/azure/common/client"
	"github.com/hashicorp/packer-plugin-sdk/multistep"
	packersdk "github.com/hashicorp/packer-plugin-sdk/packer"
	"github.com/hashicorp/packer-plugin-sdk/packerbuilderdata"
)

func TestChrootStepGetSourceImageName(t *testing.T) {
	ui := &packersdk.BasicUi{
		Reader: new(bytes.Buffer),
		Writer: new(bytes.Buffer),
	}
	state := new(multistep.BasicStateBag)

	state.Put("azureclient", &client.AzureClientSetMock{SubscriptionIDMock: "1234"})
	state.Put("ui", ui)

	tc := []struct {
		name     string
		step     *StepGetSourceImageName
		expected string
	}{
		{
			name: "SourceOSDisk",
			step: &StepGetSourceImageName{
				SourceOSDiskResourceID: "https://azure/vhd",
				GeneratedData:          &packerbuilderdata.GeneratedData{State: state},
			},
			expected: "https://azure/vhd",
		},
		{
			name: "MarketPlaceImage",
			step: &StepGetSourceImageName{
				SourcePlatformImage: &client.PlatformImage{
					Publisher: "Microsoft",
					Offer:     "Server",
					Sku:       "0",
					Version:   "2019",
				},
				Location:      "west",
				GeneratedData: &packerbuilderdata.GeneratedData{State: state},
			},
			expected: "/subscriptions/1234/providers/Microsoft.Compute/locations/west/publishers/Microsoft/ArtifactTypes/vmimage/offers/Server/skus/0/versions/2019",
		},
	}
	for _, tt := range tc {
		tt := tt
		t.Run(tt.name, func(t *testing.T) {
			tt.step.Run(context.TODO(), state)
			got := state.Get("generated_data").(map[string]interface{})
			v, ok := got["SourceImageName"]
			if !ok {
				t.Errorf("expected SourceImageName to be set in generatedData")
			}

			if v != tt.expected {
				t.Errorf("expected SourceImageName to be set to %q but got %q", tt.expected, v)
			}
		})
	}

}

func TestChrootStepGetSourceImageName_SharedImage(t *testing.T) {
	ui := &packersdk.BasicUi{
		Reader: new(bytes.Buffer),
		Writer: new(bytes.Buffer),
	}
	state := new(multistep.BasicStateBag)
	state.Put("ui", ui)

	genData := packerbuilderdata.GeneratedData{State: state}

	tc := []struct {
		name     string
		step     *StepGetSourceImageName
		expected string
	}{
		{
			name: "SharedImageWithMangedImageSource",
			step: &StepGetSourceImageName{
				SourceImageResourceID: "/subscriptions/1234/resourceGroups/bar/providers/Microsoft.Compute/galleries/test/images/foo/versions/1.0.6",
				GeneratedData:         &genData,
			},
			expected: "/subscription/resource/managed/image/name/as/source",
		},
	}
	for _, tt := range tc {
		tt := tt
		t.Run(tt.name, func(t *testing.T) {

			state.Put("azureclient", &client.AzureClientSetMock{
				SubscriptionIDMock:             "1234",
				GalleryImageVersionsClientMock: MockGalleryImageClient("1234"),
			})
			tt.step.Run(context.TODO(), state)
			got := state.Get("generated_data").(map[string]interface{})
			v, ok := got["SourceImageName"]
			if !ok {
				t.Errorf("expected SourceImageName to be set in generatedData")
			}

			if v != tt.expected {
				t.Errorf("expected SourceImageName to be set to %q but got %q", tt.expected, v)
			}
		})
	}

}

func MockGalleryImageClient(subID string) compute.GalleryImageVersionsClient {
	giv := compute.NewGalleryImageVersionsClient(subID)
	giv.Sender = autorest.SenderFunc(func(r *http.Request) (*http.Response, error) {
		return &http.Response{
			Request: r,
			Body: ioutil.NopCloser(strings.NewReader(`{
							"properties": { "storageProfile": {
								"Source": {
								"ID": "/subscription/resource/managed/image/name/as/source"
								}
							} }
						}`)),
			StatusCode: 200,
		}, nil
	})

	return giv
}
