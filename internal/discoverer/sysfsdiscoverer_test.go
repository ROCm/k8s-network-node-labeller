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

package discoverer

import (
	"path/filepath"
	"testing"

	"github.com/ROCm/k8s-network-node-labeller/internal/label"
	"github.com/ROCm/k8s-network-node-labeller/internal/sysfs"
	"github.com/ROCm/k8s-network-node-labeller/internal/testutils"
)

func TestNewSysfsDiscoverer(t *testing.T) {
	tempDir := t.TempDir()
	mockClient, _ := sysfs.NewSysfsClient(tempDir)
	discoverer := NewSysfsDiscoverer(mockClient)

	if discoverer == nil {
		t.Fatal("Expected discoverer to be created, got nil")
	}

	if discoverer.client != mockClient {
		t.Fatal("Expected discoverer to use provided client")
	}
}

func TestSysfsDiscoverer_Discover_FromYAML(t *testing.T) {
	testProductInfoMap := map[string]sysfs.ProductInfo{
		"5201": {
			ProductName: "POLLARA_1x400G_QSFP112",
			SKU:         "POLLARA-1Q400P",
		},
		"6201": {
			ProductName: "FUTURE_2x200G_BEAST",
			SKU:         "FUTURE-2Q200P",
		},
	}

	tests := []struct {
		name        string
		nicMockFile string
		expected    map[string]string
	}{
		{
			name:        "Multi NIC Heterogenous",
			nicMockFile: "multi_nic_heterogenous.yaml",
			expected: map[string]string{
				"amd.com/nic.pollara-1q400p.product-name":   "POLLARA_1x400G_QSFP112",
				"amd.com/nic.pollara-1q400p.driver-name":    "ionic",
				"amd.com/nic.pollara-1q400p.driver-version": "1.51.0-k",
				"amd.com/nic.future-2q200p.product-name":    "FUTURE_2x200G_BEAST",
				"amd.com/nic.future-2q200p.driver-name":     "amd-rocks",
				"amd.com/nic.future-2q200p.driver-version":  "5.0.0",
			},
		},
		{
			name:        "Multi NIC Homogenous",
			nicMockFile: "multi_nic_homogenous.yaml",
			expected: map[string]string{
				"amd.com/nic.product-name":   "POLLARA_1x400G_QSFP112",
				"amd.com/nic.driver-name":    "ionic",
				"amd.com/nic.driver-version": "1.51.0-k",
			},
		},
		{
			name:        "Only VFs Heterogenous",
			nicMockFile: "only_vfs_heterogenous.yaml",
			expected: map[string]string{
				"amd.com/nic.pollara-1q400p.product-name":   "POLLARA_1x400G_QSFP112",
				"amd.com/nic.pollara-1q400p.driver-name":    "ionic",
				"amd.com/nic.pollara-1q400p.driver-version": "1.51.0-k",
				"amd.com/nic.future-2q200p.product-name":    "FUTURE_2x200G_BEAST",
				"amd.com/nic.future-2q200p.driver-name":     "amd-rocks",
				"amd.com/nic.future-2q200p.driver-version":  "5.0.0",
			},
		},
		{
			name:        "Only VFs Homogenous",
			nicMockFile: "only_vfs_homogenous.yaml",
			expected: map[string]string{
				"amd.com/nic.product-name":   "POLLARA_1x400G_QSFP112",
				"amd.com/nic.driver-name":    "ionic",
				"amd.com/nic.driver-version": "1.51.0-k",
			},
		},
		{
			name:        "Single NIC Two PF One VF",
			nicMockFile: "single_nic_two_pf_one_vf.yaml",
			expected: map[string]string{
				"amd.com/nic.product-name":   "POLLARA_1x400G_QSFP112",
				"amd.com/nic.driver-name":    "ionic-super-cool",
				"amd.com/nic.driver-version": "2.0.0",
			},
		},
		{
			name:        "Single NIC Two PFs",
			nicMockFile: "single_nic_two_pfs.yaml",
			expected: map[string]string{
				"amd.com/nic.product-name":   "POLLARA_1x400G_QSFP112",
				"amd.com/nic.driver-name":    "ionic",
				"amd.com/nic.driver-version": "1.51.0-k",
			},
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			// Create temporary directory and set up mock sysfs
			tempDir := t.TempDir()
			yamlFile := filepath.Join("testdata", "sysfsdiscoverer", tt.nicMockFile)
			if err := testutils.CreateFakeSysfsFromFile(tempDir, yamlFile); err != nil {
				t.Fatalf("Failed to create mock sysfs: %v", err)
			}

			client, err := sysfs.NewSysfsClientWithProductInfoMap(tempDir, testProductInfoMap)
			if err != nil {
				t.Fatalf("Failed to create SysfsClient: %v", err)
			}

			discoverer := NewSysfsDiscoverer(client)

			labels, err := discoverer.Discover()

			if err != nil {
				t.Fatalf("Expected no error, got %v", err)
			}

			if len(labels) == 0 {
				t.Fatal("Expected labels to be returned, got empty slice")
			}

			discoveredLabels := make(map[string]string)
			for _, l := range labels {
				discoveredLabels[l.Key] = l.Value
			}

			// Check if all discovered labels are managed by us
			// This is a sanity check to ensure we are not discovering unexpected labels
			for discoveredKey := range discoveredLabels {
				if !label.IsManagedLabel(discoveredKey) {
					t.Errorf("Discovered label %s is not a managed label, add to the list of managed label regexps in internal/label/regexps.go if necessary", discoveredKey)
				}
			}

			// Check if we have all expected labels
			for expectedKey, expectedValue := range tt.expected {
				if actualValue, exists := discoveredLabels[expectedKey]; !exists {
					t.Errorf("Expected label %s not found", expectedKey)
				} else if actualValue != expectedValue {
					t.Errorf("Expected label %s=%s, got %s=%s", expectedKey, expectedValue, expectedKey, actualValue)
				}
			}
		})
	}
}
