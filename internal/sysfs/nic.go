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

type NIC struct {
	ProductName   string
	DriverName    string
	DriverVersion string
	SKU           string
}

// NewNICFromPCIFuncs creates a NIC from a group of related PCI Functions (PFs and VFs)
func NewNICFromPCIFuncs(funcs []PCIDevice, productInfoMap map[string]ProductInfo) (NIC, error) {
	// Check if funcs slice is empty
	if len(funcs) == 0 {
		return NIC{}, fmt.Errorf("cannot create NIC without PCI functions")
	}

	f := funcs[0]

	// Get product info from subsystem device ID
	productInfo, exists := productInfoMap[f.SubsystemDeviceID]
	if !exists {
		return NIC{}, fmt.Errorf("unknown product info for subsystem device ID: %s", f.SubsystemDeviceID)
	}

	// Extract driver name and version from sysfs
	driverName, driverVersion, err := getDriverInfoForPCIDevice(f.SysFSPath)
	if err != nil {
		return NIC{}, fmt.Errorf("failed to get driver info for PCI device at %s: %w", f.SysFSPath, err)
	}

	return NIC{
		ProductName:   productInfo.ProductName,
		DriverName:    driverName,
		DriverVersion: driverVersion,
		SKU:           productInfo.SKU,
	}, nil
}

// getDriverInfoForPCIDevice extracts driver name and version for a PCI device from its sysfs path
func getDriverInfoForPCIDevice(sysFSPath string) (string, string, error) {
	// Driver symlink path
	driverLinkPath := filepath.Join(sysFSPath, "driver")

	// Follow the symlink to get the actual driver directory
	driverPath, err := filepath.EvalSymlinks(driverLinkPath)
	if err != nil {
		return "", "", fmt.Errorf("failed to follow driver symlink: %w", err)
	}

	// Get basename of the driver path (this is the driver name)
	driverName := filepath.Base(driverPath)

	// Read driver version from module/version
	versionPath := filepath.Join(driverPath, "module", "version")
	versionData, err := os.ReadFile(versionPath)
	if err != nil {
		// Version file might not exist, log and return empty version
		slog.Warn("failed to read driver version", "versionPath", versionPath, "error", err)
		return driverName, "", nil
	}

	driverVersion := strings.TrimSpace(string(versionData))

	return driverName, driverVersion, nil
}
