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
	"os"
	"path/filepath"
	"strings"
	"testing"

	"github.com/ROCm/k8s-network-node-labeller/internal/testutils"
)

func TestCollectPCIDevices(t *testing.T) {
	// Create temporary directory and set up mock sysfs
	tempDir := t.TempDir()
	yamlPath := filepath.Join("testdata", "pci_devices_mixed.yaml")
	if err := testutils.CreateFakeSysfsFromFile(tempDir, yamlPath); err != nil {
		t.Fatalf("Failed to create mock sysfs: %v", err)
	}

	client, err := NewSysfsClient(tempDir)
	if err != nil {
		t.Fatalf("Failed to create SysfsClient: %v", err)
	}

	tests := []struct {
		name                 string
		filter               PCIDevice
		expectedPCIAddresses []string
	}{
		{
			name:   "Filter by AMD Pensando Vendor ID",
			filter: PCIDevice{PCIVendorID: "1dd8"},
			expectedPCIAddresses: []string{
				"0000:13:00.0", // nic1-virtual-downstream-port
				"0000:14:00.0", // nic1-pf
				"0000:14:00.1", // nic1-vf
				"0000:15:00.0", // nic2-virtual-downstream-port
				"0000:16:00.0", // nic2-pf
			},
		},
		{
			name:   "Filter by Network Controller Class",
			filter: PCIDevice{DeviceClass: "020000"},
			expectedPCIAddresses: []string{
				"0000:14:00.0", // nic1-pf
				"0000:14:00.1", // nic1-vf
				"0000:16:00.0", // nic2-pf
				"0000:17:00.0", // dummy-network-controller-1
			},
		},
		{
			name:   "Filter by Subsystem Device ID 5201",
			filter: PCIDevice{SubsystemDeviceID: "5201"},
			expectedPCIAddresses: []string{
				"0000:14:00.0", // nic1-pf
				"0000:14:00.1", // nic1-vf
				"0000:16:00.0", // nic2-pf
			},
		},
		{
			name:   "Combined Filter: AMD Pensando Network Controllers",
			filter: PCIDevice{PCIVendorID: "1dd8", DeviceClass: "020000"},
			expectedPCIAddresses: []string{
				"0000:14:00.0", // nic1-pf
				"0000:14:00.1", // nic1-vf
				"0000:16:00.0", // nic2-pf
			},
		},
		{
			name:   "Filter by PCI Bridge Class",
			filter: PCIDevice{DeviceClass: "060400"},
			expectedPCIAddresses: []string{
				"0000:00:04.0", // dummy bridge
				"0000:13:00.0", // AMD bridge 1
				"0000:15:00.0", // AMD bridge 2
			},
		},
		{
			name:                 "Filter by Non-existent Vendor ID",
			filter:               PCIDevice{PCIVendorID: "ffff"},
			expectedPCIAddresses: []string{},
		},
		{
			name:   "Filter by Specific Device ID",
			filter: PCIDevice{PCIDeviceID: "1002"},
			expectedPCIAddresses: []string{
				"0000:14:00.0", // nic1-pf
				"0000:16:00.0", // nic2-pf
			},
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			devices, err := client.collectPCIDevices(tt.filter)
			if err != nil {
				t.Fatalf("collectPCIDevices failed: %v", err)
			}

			expectedCount := len(tt.expectedPCIAddresses)
			if len(devices) != expectedCount {
				t.Errorf("Expected %d devices, got %d", expectedCount, len(devices))
				t.Logf("Filter: %+v", tt.filter)
				t.Logf("Found devices:")
				for i, device := range devices {
					t.Logf("  [%d] %s", i, device.String())
				}
			}

			// Check for expected PCI addresses in the SysFS paths if provided
			if len(tt.expectedPCIAddresses) > 0 {
				foundAddresses := make(map[string]bool)
				for _, device := range devices {
					// Extract PCI address from SysFS path (e.g., /tmp/xyz/bus/pci/devices/0000:14:00.0 -> 0000:14:00.0)
					pathParts := strings.Split(device.SysFSPath, "/")
					if len(pathParts) > 0 {
						pciAddr := pathParts[len(pathParts)-1]
						foundAddresses[pciAddr] = true
					}
				}

				for _, expectedAddr := range tt.expectedPCIAddresses {
					if !foundAddresses[expectedAddr] {
						t.Errorf("Expected PCI address %s not found in results", expectedAddr)
						t.Logf("Found addresses: %v", foundAddresses)
					}
				}
			}
		})
	}
}

func TestNewSysfsClient(t *testing.T) {
	tests := []struct {
		name        string
		sysfsRoot   string
		expectError bool
		setupFunc   func() string // Function to set up test path if needed
		cleanupFunc func(string)  // Function to clean up after test
	}{
		{
			name:        "Valid sysfs root - temp directory",
			expectError: false,
			setupFunc: func() string {
				return t.TempDir() // Create a valid temporary directory
			},
		},
		{
			name:        "Empty sysfs root",
			sysfsRoot:   "",
			expectError: true,
		},
		{
			name:        "Non-existent path",
			sysfsRoot:   "/non/existent/path/that/does/not/exist",
			expectError: true,
		},
		{
			name:        "Path is a file, not directory",
			expectError: true,
			setupFunc: func() string {
				tempDir := t.TempDir()
				filePath := filepath.Join(tempDir, "testfile")
				// Create a file instead of a directory
				if err := os.WriteFile(filePath, []byte("test"), 0644); err != nil {
					t.Fatalf("Failed to create test file: %v", err)
				}
				return filePath
			},
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			var testPath string

			if tt.setupFunc != nil {
				testPath = tt.setupFunc()
				if tt.cleanupFunc != nil {
					defer tt.cleanupFunc(testPath)
				}
			} else {
				testPath = tt.sysfsRoot
			}

			client, err := NewSysfsClient(testPath)

			if tt.expectError {
				if err == nil {
					t.Errorf("Expected error, got none")
				}
				if client != nil {
					t.Errorf("Expected nil client on error, got %v", client)
				}
				return
			}

			if err != nil {
				t.Errorf("Expected no error, got: %v", err)
				return
			}

			if client == nil {
				t.Errorf("Expected client to be created, got nil")
			}
		})
	}
}
