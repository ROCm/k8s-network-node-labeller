/*
Copyright (c) 2025 Advanced Micro Devices, Inc. All rights reserved.

Licensed under the Apache License, Version 2.0 (the "License");
you may not use this file except in compliance with the License.
You may obtain a copy of the License at

    http://www.apache.org/licenses/LICENSE-2.0

Unless required by applicable law or agreed to in writing, software
distributed under the License is distributed on an "AS IS" BASIS,
WITHOUT WARRANTIES OR CONDITIONS OF ANY KIND, either express or implied.
See the License for the specific language governing permissions and
limitations under the License.
*/

package sysfs

import (
	"fmt"
	"log/slog"
	"os"
	"path/filepath"
	"strings"
)

// ProductInfo contains product information for a NIC
type ProductInfo struct {
	ProductName string
	SKU         string
}

var DefaultSubsystemIDtoProductInfoMap = map[string]ProductInfo{
	"5201": {
		ProductName: "POLLARA 1x400G QSFP112",
		SKU:         "POLLARA-1Q400P",
	},
}

// SysfsClient provides methods to interact with sysfs
type SysfsClient struct {
	sysfsRoot      string
	productInfoMap map[string]ProductInfo
}

// NewSysfsClient creates a new sysfs client with the specified root path using the default product info map
func NewSysfsClient(sysfsRoot string) (*SysfsClient, error) {
	return NewSysfsClientWithProductInfoMap(sysfsRoot, DefaultSubsystemIDtoProductInfoMap)
}

// NewSysfsClientWithProductInfoMap creates a new sysfs client with the specified root path and product info map
func NewSysfsClientWithProductInfoMap(sysfsRoot string, productInfoMap map[string]ProductInfo) (*SysfsClient, error) {
	if sysfsRoot == "" {
		return nil, fmt.Errorf("sysfs root path cannot be empty")
	}
	info, err := os.Stat(sysfsRoot)
	if err != nil {
		return nil, fmt.Errorf("failed to access sysfs root path: %w", err)
	}
	if !info.IsDir() {
		return nil, fmt.Errorf("sysfs root path is not a directory")
	}

	return &SysfsClient{
		productInfoMap: productInfoMap,
		sysfsRoot:      sysfsRoot,
	}, nil
}

// collectPCIDevices scans sysfs and returns all PCI devices matching the given filter
// The filter parameter is a PCIDevice with some fields set to filter by, empty fields are ignored
func (c *SysfsClient) collectPCIDevices(filter PCIDevice) ([]PCIDevice, error) {
	pciDevicesPath := filepath.Join(c.sysfsRoot, "bus", "pci", "devices")

	var devices []PCIDevice

	// Read the PCI devices directory
	entries, err := os.ReadDir(pciDevicesPath)
	if err != nil {
		return nil, fmt.Errorf("failed to read PCI devices directory: %w", err)
	}

	for _, entry := range entries {
		entryPath := filepath.Join(pciDevicesPath, entry.Name())

		// Resolve symlink to get actual path and check if it's a directory
		actualPath, err := filepath.EvalSymlinks(entryPath)
		if err != nil {
			// Continue processing other devices even if symlink resolution fails
			continue
		}

		// Check if the resolved path is actually a directory
		info, err := os.Stat(actualPath)
		if err != nil || !info.IsDir() {
			continue
		}

		device, err := c.pciDeviceFromSysfsPath(actualPath)
		if err != nil {
			// Continue processing other devices even if one fails
			continue
		}

		// Apply filter
		if device.MatchesFilter(filter) {
			devices = append(devices, device)
		}
	}

	return devices, nil
}

// pciDeviceFromSysfsPath reads PCI device information from sysfs
func (c *SysfsClient) pciDeviceFromSysfsPath(devicePath string) (PCIDevice, error) {
	// Read vendorID ID
	vendorID, err := os.ReadFile(filepath.Join(devicePath, "vendor"))
	if err != nil {
		return PCIDevice{}, fmt.Errorf("failed to read vendor: %w", err)
	}

	// Read device ID
	deviceID, err := os.ReadFile(filepath.Join(devicePath, "device"))
	if err != nil {
		return PCIDevice{}, fmt.Errorf("failed to read device: %w", err)
	}

	// Read subsystem vendor ID
	subsysVendorData, err := os.ReadFile(filepath.Join(devicePath, "subsystem_vendor"))
	var subsysVendorID string
	if err != nil {
		// Subsystem vendor might not exist for all devices, set empty
		subsysVendorID = ""
	} else {
		subsysVendorID = string(subsysVendorData)
	}

	// Read subsystem device ID
	subsysDeviceData, err := os.ReadFile(filepath.Join(devicePath, "subsystem_device"))
	var subsysDeviceID string
	if err != nil {
		// Subsystem device might not exist for all devices, set empty
		subsysDeviceID = ""
	} else {
		subsysDeviceID = string(subsysDeviceData)
	}

	// Read device class
	pciClass, err := os.ReadFile(filepath.Join(devicePath, "class"))
	if err != nil {
		return PCIDevice{}, fmt.Errorf("failed to read class: %w", err)
	}

	// Create device using NewPCIDevice which handles sanitization
	device := NewPCIDevice(
		devicePath,
		string(vendorID),
		string(deviceID),
		subsysVendorID,
		subsysDeviceID,
		string(pciClass),
	)

	return device, nil
}

// collectAmdNicPciFuncs returns all AMD NIC specific PCI Functions
func (c *SysfsClient) collectAmdNicPciFuncs() ([]PCIDevice, error) {
	var allDevices []PCIDevice

	// Iterate through each NICs of interest
	for subsystemID, _ := range c.productInfoMap {
		filter := PCIDevice{
			PCIVendorID:       "1dd8",      // AMD Pensando
			SubsystemDeviceID: subsystemID, // Use the PCI ID from the map
			DeviceClass:       "020000",    // Network controller class
		}

		devices, err := c.collectPCIDevices(filter)
		if err != nil {
			return nil, fmt.Errorf("failed to collect devices for Subsystem ID %s: %w", subsystemID, err)
		}

		allDevices = append(allDevices, devices...)
	}

	return allDevices, nil
}

// GetNICs returns an array of NICs by collecting AMD NIC PFs and VFs and grouping them
func (c *SysfsClient) GetNICs() ([]NIC, error) {
	// Collect all AMD NIC PF and VFs
	funcs, err := c.collectAmdNicPciFuncs()
	if err != nil {
		return nil, fmt.Errorf("failed to collect AMD NIC Functions: %w", err)
	}

	// Group PFs and VFs into NICs
	return c.groupFuncsToNICs(funcs), nil
}

// groupFuncsToNICs groups PCI Functions into NICs
func (c *SysfsClient) groupFuncsToNICs(funcs []PCIDevice) []NIC {
	// Create a map to group PCI functions by their parent path
	parentGroups := make(map[string][]PCIDevice)

	for _, device := range funcs {
		// Extract parent path from SysFSPath
		// For PCI devices, the parent is typically the directory containing the device
		parentPath := filepath.Dir(device.SysFSPath)
		parent := filepath.Base(parentPath)

		// Parent is not an actual PCI device if it starts with "pci" (like pci0000:00)
		// This can happen in case of VMs, where VFs are directly mounted on bus
		if strings.HasPrefix(parent, "pci") {
			// Use the device itself as its own parent
			parent = filepath.Base(device.SysFSPath)
		}

		parentGroups[parent] = append(parentGroups[parent], device)
	}

	// Create NICs from each group of related PCI functions
	var nics []NIC
	for _, group := range parentGroups {
		nic, err := NewNICFromPCIFuncs(group, c.productInfoMap)
		if err != nil {
			slog.Warn("failed to create NIC from PCI functions", "error", err)
			continue
		}
		nics = append(nics, nic)
	}

	return nics
}
