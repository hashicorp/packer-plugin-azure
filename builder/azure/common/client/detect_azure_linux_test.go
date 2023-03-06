// Copyright (c) HashiCorp, Inc.
// SPDX-License-Identifier: MPL-2.0

package client

import (
	"io/ioutil"
	"os"
	"testing"

	"github.com/stretchr/testify/assert"
)

func TestIsAzure(t *testing.T) {
	f, err := ioutil.TempFile("", "TestIsAzure*")
	if err != nil {
		t.Fatal(err)
	}
	defer os.Remove(f.Name())

	_, err = f.Seek(0, 0)
	if err != nil {
		t.Fatal(err)
	}

	err = f.Truncate(0)
	if err != nil {
		t.Fatal(err)
	}

	_, err = f.Write([]byte("not the azure assettag"))
	if err != nil {
		t.Fatal(err)
	}

	assert.False(t, isAzureAssetTag(f.Name()), "asset tag is not Azure's")

	_, err = f.Seek(0, 0)
	if err != nil {
		t.Fatal(err)
	}

	err = f.Truncate(0)
	if err != nil {
		t.Fatal(err)
	}

	_, err = f.Write(azureAssetTag)
	if err != nil {
		t.Fatal(err)
	}

	assert.True(t, isAzureAssetTag(f.Name()), "asset tag is Azure's")
}
