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

func TestGenerateProfileLabelsAll_HeterogeneousSKUKeys(t *testing.T) {
	d := NewNicctlDiscoverer(nil)
	cards := []nicctl.NIC{
		{ID: "nic1", SKU: "DSS-W600"},
		{ID: "nic2", SKU: "DSS-W400"},
	}
	profiles := map[string]string{
		"nic1": "pf1_vf1",
	}

	labels := d.generateProfileLabelsAll(cards, profiles)

	if len(labels) != 1 {
		t.Fatalf("Expected 1 profile label, got %d", len(labels))
	}

	expectedKey := label.DefaultNICPrefixedKey("dss-w600.profile")
	if labels[0].Key != expectedKey {
		t.Errorf("Expected key %s, got %s", expectedKey, labels[0].Key)
	}
	if labels[0].Value != "pf1_vf1" {
		t.Errorf("Expected value pf1_vf1, got %s", labels[0].Value)
	}
}

func TestNicctlDiscoverer_Discover_FromYAML(t *testing.T) {
	tests := []struct {
		name        string
		nicMockFile string
		expected    map[string]string
		absentKeys  []string
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
				"amd.com/nic.profile":          "pf1_vf1",
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
				"amd.com/nic.profile":          "pf1_vf1",
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
				"amd.com/nic.profile":          "pf1_vf1",
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
				"amd.com/nic.dss-w600.profile":          "pf1_vf1",
				"amd.com/nic.dss-w400.profile":          "hnic_pf1_vf8",
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
				"amd.com/nic.dss-w650.profile":          "pf1_vf1",
				"amd.com/nic.dss-w450.profile":          "hnic_pf1_vf8",
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
				"amd.com/nic.dss-w650.profile":          "pf1_vf1",
				"amd.com/nic.dss-w450.profile":          "hnic_pf1_vf8",
			},
		},
		{
			name:        "Multiple NICs Same Model Mixed Profiles",
			nicMockFile: "nic_multiple_same_model_mixed_profiles.yaml",
			expected: map[string]string{
				"amd.com/nic.count":            "2",
				"amd.com/nic.firmware-version": "1.2.3",
				"amd.com/nic.port-count":       "2",
				"amd.com/nic.port-speed":       "100G",
				"amd.com/nic.profile":          "misconfig-mixed-profiles",
			},
		},
		{
			name:        "Profile discovery error continues without profile labels",
			nicMockFile: "nic_single_profiles_error.yaml",
			expected: map[string]string{
				"amd.com/nic.count":            "1",
				"amd.com/nic.firmware-version": "1.2.3",
				"amd.com/nic.port-count":       "1",
				"amd.com/nic.port-speed":       "100G",
			},
			absentKeys: []string{"amd.com/nic.profile"},
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

			for discoveredKey := range discoveredLabels {
				if !label.IsManagedLabel(discoveredKey) {
					t.Errorf("Discovered label %s is not a managed label, add to the list of managed label regexps in internal/label/regexps.go if necessary", discoveredKey)
				}
			}

			for expectedKey, expectedValue := range tt.expected {
				if actualValue, exists := discoveredLabels[expectedKey]; !exists {
					t.Errorf("Expected label %s not found", expectedKey)
				} else if actualValue != expectedValue {
					t.Errorf("Expected label %s=%s, got %s=%s", expectedKey, expectedValue, expectedKey, actualValue)
				}
			}

			for _, key := range tt.absentKeys {
				if _, exists := discoveredLabels[key]; exists {
					t.Errorf("Expected label %s to be absent", key)
				}
			}
		})
	}
}
