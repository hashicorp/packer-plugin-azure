// Copyright (c) HashiCorp, Inc.
// SPDX-License-Identifier: MPL-2.0

package chroot

import (
	"context"
	"errors"
	"log"
	"strings"
	"time"

	hashiVMSDK "github.com/hashicorp/go-azure-sdk/resource-manager/compute/2022-03-01/virtualmachines"
	"github.com/hashicorp/packer-plugin-azure/builder/azure/common/client"
	packersdk "github.com/hashicorp/packer-plugin-sdk/packer"
)

type DiskAttacher interface {
	AttachDisk(ctx context.Context, disk string) (lun int64, err error)
	WaitForDevice(ctx context.Context, i int64) (device string, err error)
	DetachDisk(ctx context.Context, disk string) (err error)
	WaitForDetach(ctx context.Context, diskID string) error
}

var NewDiskAttacher = func(azureClient client.AzureClientSet, ui packersdk.Ui) DiskAttacher {
	return &diskAttacher{
		azcli: azureClient,
		ui:    ui,
	}
}

type diskAttacher struct {
	azcli client.AzureClientSet

	vm *client.ComputeInfo // store info about this VM so that we don't have to ask metadata service on every call
	ui packersdk.Ui
}

var DiskNotFoundError = errors.New("Disk not found")
var AzureAPIDiskError = errors.New("Azure API returned invalid disk")

func (da *diskAttacher) DetachDisk(ctx context.Context, diskID string) error {
	log.Println("Fetching list of disks currently attached to VM")
	currentDisks, err := da.getDisks(ctx)
	if err != nil {
		log.Printf("DetachDisk.getDisks: error: %+v\n", err)
		return err
	}

	log.Printf("Removing %q from list of disks currently attached to VM", diskID)
	newDisks := []hashiVMSDK.DataDisk{}
	for _, disk := range currentDisks {
		if disk.ManagedDisk != nil {
			if disk.ManagedDisk.Id == nil {
				log.Println("DetatchDisks failure: Azure Client returned a disk without an ID")
				return AzureAPIDiskError
			}
			if !strings.EqualFold(*disk.ManagedDisk.Id, diskID) {
				newDisks = append(newDisks, disk)
			}
		}
	}
	if len(currentDisks) == len(newDisks) {
		return DiskNotFoundError
	}

	log.Println("Updating new list of disks attached to VM")
	err = da.setDisks(ctx, newDisks)
	if err != nil {
		log.Printf("DetachDisk.setDisks: error: %+v\n", err)
		return err

	}

	return nil
}

func (da *diskAttacher) WaitForDetach(ctx context.Context, diskID string) error {
	for { // loop until disk is not attached, timeout or error
		list, err := da.getDisks(ctx)
		if err != nil {
			log.Printf("WaitForDetach.getDisks: error: %+v\n", err)
			return err
		}
		if findDiskInList(list, diskID) == nil {
			log.Println("Disk is no longer in VM model, assuming detached")
			return nil
		}

		select {
		case <-time.After(time.Second): //continue
		case <-ctx.Done():
			return ctx.Err()
		}
	}
}

func (da *diskAttacher) AttachDisk(ctx context.Context, diskID string) (int64, error) {
	dataDisks, err := da.getDisks(ctx)
	if err != nil {
		log.Printf("AttachDisk.getDisks: error: %+v\n", err)
		return -1, err
	}

	// check to see if disk is already attached, remember lun if found
	if disk := findDiskInList(dataDisks, diskID); disk != nil {
		// disk is already attached, just take this lun
		if disk.Lun == 0 {
			return -1, errors.New("disk is attached, but lun was not set in VM model (possibly an error in the Azure APIs)")
		}
		return disk.Lun, nil
	}

	// disk was not found on VM, go and actually attach it

	// TODO This assignment looks like it would do nothing, consider removing it
	//nolint
	var lun int64 = -1
findFreeLun:
	for lun = 0; lun < 64; lun++ {
		for _, v := range dataDisks {
			if v.Lun == lun {
				continue findFreeLun
			}
		}
		// no datadisk is using this lun
		break
	}

	// append new data disk to collection
	dataDisks = append(dataDisks, hashiVMSDK.DataDisk{
		CreateOption: hashiVMSDK.DiskCreateOptionTypesAttach,
		ManagedDisk: &hashiVMSDK.ManagedDiskParameters{
			Id: &diskID,
		},
		Lun: lun,
	})

	// prepare resource object for update operation
	err = da.setDisks(ctx, dataDisks)
	if err != nil {
		log.Printf("AttachDisk.setDisks: error: %+v\n", err)
		return -1, err
	}

	return lun, nil
}

func (da *diskAttacher) getThisVM(ctx context.Context) (hashiVMSDK.VirtualMachine, error) {
	// getting resource info for this VM
	if da.vm == nil {
		vm, err := da.azcli.MetadataClient().GetComputeInfo()
		if err != nil {
			return hashiVMSDK.VirtualMachine{}, err
		}
		da.vm = vm
	}

	vmID := hashiVMSDK.NewVirtualMachineID(da.azcli.SubscriptionID(), da.vm.ResourceGroupName, da.vm.Name)
	// retrieve actual VM
	vmResource, err := da.azcli.VirtualMachinesClient().Get(ctx, vmID, hashiVMSDK.DefaultGetOperationOptions())
	if err != nil {
		return hashiVMSDK.VirtualMachine{}, err
	}
	if vmResource.Model.Properties.StorageProfile == nil {
		return hashiVMSDK.VirtualMachine{}, errors.New("properties.storageProfile is not set on VM, this is unexpected")
	}

	return *vmResource.Model, nil
}

func (da diskAttacher) getDisks(ctx context.Context) ([]hashiVMSDK.DataDisk, error) {
	vmResource, err := da.getThisVM(ctx)
	if err != nil {
		return []hashiVMSDK.DataDisk{}, err
	}

	return *vmResource.Properties.StorageProfile.DataDisks, nil
}

func (da diskAttacher) setDisks(ctx context.Context, disks []hashiVMSDK.DataDisk) error {
	vmResource, err := da.getThisVM(ctx)
	if err != nil {
		return err
	}

	vmResource.Properties.StorageProfile.DataDisks = &disks
	vmResource.Resources = nil

	vmID := hashiVMSDK.NewVirtualMachineID(da.azcli.SubscriptionID(), da.vm.ResourceGroupName, da.vm.Name)
	// update the VM resource, attach disk
	_, err = da.azcli.VirtualMachinesClient().CreateOrUpdate(ctx, vmID, vmResource)

	return err
}

func findDiskInList(list []hashiVMSDK.DataDisk, diskID string) *hashiVMSDK.DataDisk {
	for _, disk := range list {
		if disk.ManagedDisk != nil &&
			strings.EqualFold(*(disk.ManagedDisk.Id), diskID) {
			return &disk
		}
	}
	return nil
}
