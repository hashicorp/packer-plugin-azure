// Copyright (c) HashiCorp, Inc.
// SPDX-License-Identifier: MPL-2.0

package client

import (
	"encoding/json"
	"fmt"
	"io"
	"net/http"
)

// DefaultMetadataClient is the default instance metadata client for Azure. Replace this variable for testing purposes only
var DefaultMetadataClient = NewMetadataClient()

// MetadataClientAPI holds methods that Packer uses to get information about the current VM
type MetadataClientAPI interface {
	GetComputeInfo() (*ComputeInfo, error)
}

// MetadataClientStub is an easy way to put a test hook in DefaultMetadataClient
type MetadataClientStub struct {
	ComputeInfo
}

// GetComputeInfo implements MetadataClientAPI
func (s MetadataClientStub) GetComputeInfo() (*ComputeInfo, error) {
	return &s.ComputeInfo, nil
}

// ComputeInfo defines the Azure VM metadata that is used in Packer
type ComputeInfo struct {
	Name              string
	ResourceID        string
	ResourceGroupName string
	SubscriptionID    string
	Location          string
	VmScaleSetName    string
}

// metadataClient implements MetadataClient
type metadataClient struct {
}

var _ MetadataClientAPI = metadataClient{}

const imdsURL = "http://169.254.169.254/metadata/instance?api-version=2021-02-01"

// VMResourceID returns the resource ID of the current VM
func (client metadataClient) GetComputeInfo() (*ComputeInfo, error) {
	httpClient := &http.Client{}
	req, err := http.NewRequest("GET", imdsURL, nil)
	if err != nil {
		return nil, err
	}
	req.Header.Add("Metadata", "true")
	resp, err := httpClient.Do(req)
	if err != nil {
		return nil, err
	}
	defer resp.Body.Close()
	body, err := io.ReadAll(resp.Body)
	if err != nil {
		return nil, err
	}
	var vminfo struct {
		ComputeInfo `json:"compute"`
	}
	err = json.Unmarshal(body, &vminfo)
	if err != nil {
		return nil, err
	}
	return &vminfo.ComputeInfo, nil
}

func (ci ComputeInfo) GetResourceID() string {
	return fmt.Sprintf("/%s", ci.ResourceID)
}

// NewMetadataClient creates a new instance metadata client
func NewMetadataClient() MetadataClientAPI {
	return metadataClient{}
}
