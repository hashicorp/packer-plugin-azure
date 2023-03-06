// Copyright (c) HashiCorp, Inc.
// SPDX-License-Identifier: MPL-2.0

package arm

import (
	"bytes"
	"context"
	"testing"

	"github.com/hashicorp/packer-plugin-azure/builder/azure/common/client"
	"github.com/hashicorp/packer-plugin-sdk/multistep"
	packersdk "github.com/hashicorp/packer-plugin-sdk/packer"
	"github.com/hashicorp/packer-plugin-sdk/packerbuilderdata"
)

func TestStepGetSourceImageName(t *testing.T) {
	ui := &packersdk.BasicUi{
		Reader: new(bytes.Buffer),
		Writer: new(bytes.Buffer),
	}
	state := new(multistep.BasicStateBag)
	genData := packerbuilderdata.GeneratedData{State: state}

	tc := []struct {
		name     string
		config   *Config
		expected string
	}{
		{
			name:     "ImageUrl",
			config:   &Config{ImageUrl: "https://azure/vhd"},
			expected: "https://azure/vhd",
		},
		{
			name: "CustomManagedImageName",
			config: &Config{
				CustomManagedImageName: "/subscription/1/resource/name",
				// During build time the custom managed id source is resolved
				// and stored as customManagedImageID
				customManagedImageID: "/subscription/0/resource/mangedimage/12",
			},
			expected: "/subscription/0/resource/mangedimage/12",
		},
		{
			name: "MarketPlaceImage",
			config: &Config{
				ClientConfig:   client.Config{SubscriptionID: "1234"},
				Location:       "west",
				ImagePublisher: "Microsoft",
				ImageOffer:     "Server",
				ImageSku:       "0",
				ImageVersion:   "2019",
			},
			expected: "/subscriptions/1234/providers/Microsoft.Compute/locations/west/publishers/Microsoft/ArtifactTypes/vmimage/offers/Server/skus/0/versions/2019",
		},
	}
	for _, tt := range tc {
		tt := tt
		t.Run(tt.name, func(t *testing.T) {
			step := &StepGetSourceImageName{
				config:        tt.config,
				GeneratedData: &genData,
				say:           ui.Say,
				error:         func(e error) {},
			}

			step.Run(context.TODO(), state)
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
