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
	"github.com/ROCm/k8s-network-node-labeller/internal/nicctl"
)

func TestNewNicctlDiscoverer(t *testing.T) {
	mockClient := &nicctl.MockNicctlClient{}
	discoverer := NewNicctlDiscoverer(mockClient)

	if discoverer == nil {
		t.Fatal("Expected discoverer to be created, got nil")
	}

	if discoverer.client != mockClient {
		t.Fatal("Expected discoverer to use provided client")
	}
}

func TestNicctlDiscoverer_Discover_FromYAML(t *testing.T) {
	tests := []struct {
		name        string
		nicMockFile string
		expected    map[string]string
	}{
		{
			name:        "No NICs",
			nicMockFile: "nic_empty.yaml",
			expected: map[string]string{
				"amd.com/nic.count": "0",
			},
		},
		{
			name:        "Single NIC",
			nicMockFile: "nic_single.yaml",
			expected: map[string]string{
				"amd.com/nic.count":            "1",
				"amd.com/nic.firmware-version": "1.2.3",
				"amd.com/nic.port-count":       "1",
				"amd.com/nic.port-speed":       "100G",
			},
		},
		{
			name:        "Single NIC MultiPort Same Speed",
			nicMockFile: "nic_single_multi_port_same_speed.yaml",
			expected: map[string]string{
				"amd.com/nic.count":            "1",
				"amd.com/nic.firmware-version": "1.2.3",
				"amd.com/nic.port-count":       "2",
				"amd.com/nic.port-speed":       "100G",
			},
		},
		{
			name:        "Single NIC MultiPort Different Speed",
			nicMockFile: "nic_single_multi_port_different_speeds.yaml",
			expected: map[string]string{
				"amd.com/nic.count":            "1",
				"amd.com/nic.firmware-version": "1.2.3",
				"amd.com/nic.port-count":       "2",
				"amd.com/nic.port0-speed":      "100G",
				"amd.com/nic.port1-speed":      "25G",
			},
		},
		{
			name:        "Multiple NICs Different Models Single Port",
			nicMockFile: "nic_multiple_different_models_single_port.yaml",
			expected: map[string]string{
				"amd.com/nic.count":                     "2",
				"amd.com/nic.dss-w600.count":            "1",
				"amd.com/nic.dss-w400.count":            "1",
				"amd.com/nic.dss-w600.firmware-version": "1.2.3",
				"amd.com/nic.dss-w400.firmware-version": "1.3.4",
				"amd.com/nic.dss-w600.port-count":       "1",
				"amd.com/nic.dss-w400.port-count":       "1",
				"amd.com/nic.dss-w600.port-speed":       "100G",
				"amd.com/nic.dss-w400.port-speed":       "25G",
			},
		},
		{
			name:        "Multiple NICs Different Model Multi Port",
			nicMockFile: "nic_multiple_different_models_multi_port.yaml",
			expected: map[string]string{
				"amd.com/nic.count":                     "2",
				"amd.com/nic.dss-w650.count":            "1",
				"amd.com/nic.dss-w450.count":            "1",
				"amd.com/nic.dss-w650.firmware-version": "1.2.3",
				"amd.com/nic.dss-w450.firmware-version": "1.3.4",
				"amd.com/nic.dss-w650.port-count":       "2",
				"amd.com/nic.dss-w450.port-count":       "2",
				"amd.com/nic.dss-w650.port-speed":       "100G",
				"amd.com/nic.dss-w450.port-speed":       "25G",
			},
		},
		{
			name:        "Multiple NICs Different Model Multi Port Different Speeds",
			nicMockFile: "nic_multiple_different_models_multi_port_different_speeds.yaml",
			expected: map[string]string{
				"amd.com/nic.count":                     "2",
				"amd.com/nic.dss-w650.count":            "1",
				"amd.com/nic.dss-w450.count":            "1",
				"amd.com/nic.dss-w650.firmware-version": "1.2.3",
				"amd.com/nic.dss-w450.firmware-version": "1.3.4",
				"amd.com/nic.dss-w650.port-count":       "2",
				"amd.com/nic.dss-w450.port-count":       "2",
				"amd.com/nic.dss-w650.port0-speed":      "100G",
				"amd.com/nic.dss-w650.port1-speed":      "400G",
				"amd.com/nic.dss-w450.port-speed":       "25G",
			},
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			yamlFile := filepath.Join("testdata", "nicctldiscoverer", tt.nicMockFile)
			mockClient, err := nicctl.NewMockNicctlClientFromYAML(yamlFile)
			if err != nil {
				t.Fatalf("Failed to create mock client from YAML: %v", err)
			}

			discoverer := NewNicctlDiscoverer(mockClient)

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
