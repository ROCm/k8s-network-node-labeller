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
	"path/filepath"
	"strings"
	"testing"

	"github.com/ROCm/k8s-network-node-labeller/internal/testutils"
)

func TestGetDriverInfoForPCIDevice(t *testing.T) {
	tests := []struct {
		name        string
		yamlFile    string
		deviceID    string
		wantDriver  string
		wantVersion string
		wantErr     bool
	}{
		{
			name:        "successful driver info extraction",
			yamlFile:    "testdata/get_driver_info_for_pci_device/pci_device_with_driver.yaml",
			deviceID:    "0000:05:00.0",
			wantDriver:  "ionic",
			wantVersion: "1.51.0-k",
			wantErr:     false,
		},
		{
			name:        "driver without version file",
			yamlFile:    "testdata/get_driver_info_for_pci_device/pci_device_with_driver_without_version.yaml",
			deviceID:    "0000:05:00.0",
			wantDriver:  "ionic",
			wantVersion: "",
			wantErr:     false,
		},
		{
			name:        "missing driver symlink",
			yamlFile:    "testdata/get_driver_info_for_pci_device/pci_device_without_driver.yaml",
			deviceID:    "0000:05:00.0",
			wantDriver:  "",
			wantVersion: "",
			wantErr:     true,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			// Create temporary directory structure
			sysfsBase := t.TempDir()

			// Create fake sysfs structure from YAML configuration
			err := testutils.CreateFakeSysfsFromFile(sysfsBase, tt.yamlFile)
			if err != nil {
				t.Fatalf("Failed to create fake sysfs from YAML: %v", err)
			}

			devicePath := filepath.Join(sysfsBase, "bus", "pci", "devices", tt.deviceID)

			// Test the function
			gotName, gotVersion, err := getDriverInfoForPCIDevice(devicePath)

			// Check error expectation
			if (err != nil) != tt.wantErr {
				t.Errorf("getDriverInfoForPCIDevice() error = %v, wantErr %v", err, tt.wantErr)
				return
			}

			// Check results if no error expected
			if !tt.wantErr {
				if gotName != tt.wantDriver {
					t.Errorf("getDriverInfoForPCIDevice() gotName = %v, want %v", gotName, tt.wantDriver)
				}
				if gotVersion != tt.wantVersion {
					t.Errorf("getDriverInfoForPCIDevice() gotVersion = %v, want %v", gotVersion, tt.wantVersion)
				}
			}
		})
	}
}

func TestNewNICFromPCIFuncs(t *testing.T) {
	tests := []struct {
		name        string
		yamlFile    string
		deviceAddrs []string
		wantNIC     NIC
		wantErr     bool
		wantErrMsg  string
	}{
		{
			name:        "successful NIC creation from single PCI function",
			yamlFile:    "testdata/new_nic_from_pci_funcs/single_pci_func_with_driver.yaml",
			deviceAddrs: []string{"0000:05:00.0"},
			wantNIC: NIC{
				ProductName:   "POLLARA 1x400G QSFP112",
				DriverName:    "ionic",
				DriverVersion: "1.51.0-k",
				SKU:           "POLLARA-1Q400P",
			},
			wantErr: false,
		},
		{
			name:        "successful NIC creation from multiple PCI functions",
			yamlFile:    "testdata/new_nic_from_pci_funcs/multiple_pci_funcs_with_driver.yaml",
			deviceAddrs: []string{"0000:05:00.0", "0000:05:00.1"},
			wantNIC: NIC{
				ProductName:   "POLLARA 1x400G QSFP112",
				DriverName:    "ionic",
				DriverVersion: "1.51.0-k",
				SKU:           "POLLARA-1Q400P",
			},
			wantErr: false,
		},
		{
			name:        "successful NIC creation with driver but no version",
			yamlFile:    "testdata/new_nic_from_pci_funcs/driver_without_version.yaml",
			deviceAddrs: []string{"0000:05:00.0"},
			wantNIC: NIC{
				ProductName:   "POLLARA 1x400G QSFP112",
				DriverName:    "ionic",
				DriverVersion: "",
				SKU:           "POLLARA-1Q400P",
			},
			wantErr: false,
		},
		{
			name:        "empty PCI functions slice",
			yamlFile:    "testdata/new_nic_from_pci_funcs/single_pci_func_with_driver.yaml",
			deviceAddrs: []string{},
			wantNIC:     NIC{},
			wantErr:     true,
			wantErrMsg:  "cannot create NIC without PCI functions",
		},
		{
			name:        "unknown subsystem device ID",
			yamlFile:    "testdata/new_nic_from_pci_funcs/unknown_product_id.yaml",
			deviceAddrs: []string{"0000:05:00.0"},
			wantNIC:     NIC{},
			wantErr:     true,
			wantErrMsg:  "unknown product info for subsystem device ID: 9999",
		},
		{
			name:        "missing driver information",
			yamlFile:    "testdata/new_nic_from_pci_funcs/missing_driver.yaml",
			deviceAddrs: []string{"0000:05:00.0"},
			wantNIC:     NIC{},
			wantErr:     true,
			wantErrMsg:  "failed to get driver info for PCI device",
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			// Create temporary directory structure
			sysfsBase := t.TempDir()

			// Create fake sysfs structure from YAML configuration
			err := testutils.CreateFakeSysfsFromFile(sysfsBase, tt.yamlFile)
			if err != nil {
				t.Fatalf("Failed to create fake sysfs from YAML: %v", err)
			}

			// Create PCIDevice slice from the device addresses
			var devices []PCIDevice
			for _, addr := range tt.deviceAddrs {
				devicePath := filepath.Join(sysfsBase, "bus", "pci", "devices", addr)

				// Create a SysfsClient to help read device information
				client, err := NewSysfsClient(sysfsBase)
				if err != nil {
					t.Fatalf("Failed to create SysfsClient: %v", err)
				}

				device, err := client.pciDeviceFromSysfsPath(devicePath)
				if err != nil {
					t.Fatalf("Failed to create PCIDevice from path %s: %v", devicePath, err)
				}
				devices = append(devices, device)
			}

			// Test the function
			gotNIC, err := NewNICFromPCIFuncs(devices, DefaultSubsystemIDtoProductInfoMap)

			// Check error expectation
			if (err != nil) != tt.wantErr {
				t.Errorf("NewNICFromPCIFuncs() error = %v, wantErr %v", err, tt.wantErr)
				return
			}

			// Check error message if error is expected
			if tt.wantErr && tt.wantErrMsg != "" {
				if err == nil || !strings.Contains(err.Error(), tt.wantErrMsg) {
					t.Errorf("NewNICFromPCIFuncs() error = %v, want error containing %q", err, tt.wantErrMsg)
				}
				return
			}

			// Check results if no error expected
			if !tt.wantErr {
				if gotNIC.ProductName != tt.wantNIC.ProductName {
					t.Errorf("NewNICFromPCIFuncs() ProductName = %q, want %q", gotNIC.ProductName, tt.wantNIC.ProductName)
				}
				if gotNIC.DriverName != tt.wantNIC.DriverName {
					t.Errorf("NewNICFromPCIFuncs() DriverName = %q, want %q", gotNIC.DriverName, tt.wantNIC.DriverName)
				}
				if gotNIC.DriverVersion != tt.wantNIC.DriverVersion {
					t.Errorf("NewNICFromPCIFuncs() DriverVersion = %q, want %q", gotNIC.DriverVersion, tt.wantNIC.DriverVersion)
				}
				if gotNIC.SKU != tt.wantNIC.SKU {
					t.Errorf("NewNICFromPCIFuncs() SKU = %q, want %q", gotNIC.SKU, tt.wantNIC.SKU)
				}
			}
		})
	}
}
