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

package label

import (
	"regexp"
	"testing"
)

// TestIsManagedLabel tests the IsManagedLabel function
func TestIsManagedLabel(t *testing.T) {
	tests := []struct {
		name     string
		key      string
		expected bool
	}{
		{
			name:     "valid nic count label",
			key:      "amd.com/nic.count",
			expected: true,
		},
		{
			name:     "valid sku specific nic count label",
			key:      "amd.com/nic.some-card-123.count",
			expected: true,
		},
		{
			name:     "valid product name label",
			key:      "amd.com/nic.product-name",
			expected: true,
		},
		{
			name:     "valid sku specific product name label",
			key:      "amd.com/nic.some-card-123.product-name",
			expected: true,
		},
		{
			name:     "valid port speed label",
			key:      "amd.com/nic.port-speed",
			expected: true,
		},
		{
			name:     "valid port speed for multi-port card label",
			key:      "amd.com/nic.some-card-123.port0-speed",
			expected: true,
		},
		{
			name:     "valid port speed for multiport card homogenous node label",
			key:      "amd.com/nic.port0-speed",
			expected: true,
		},
		{
			name:     "valid driver version label",
			key:      "amd.com/nic.driver-version",
			expected: true,
		},
		{
			name:     "valid profile label",
			key:      "amd.com/nic.profile",
			expected: true,
		},
		{
			name:     "valid sku specific profile label",
			key:      "amd.com/nic.dss-w600.profile",
			expected: true,
		},
		{
			name:     "invalid custom label",
			key:      "custom.label",
			expected: false,
		},
		{
			name:     "invalic nic label",
			key:      "amd.com/nic.some-user-defined-label",
			expected: false,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			result := IsManagedLabel(tt.key)
			if result != tt.expected {
				t.Errorf("IsManagedLabel(%s) = %v, want %v", tt.key, result, tt.expected)
			}
		})
	}
}

func TestCompileRegexps(t *testing.T) {
	// This test ensures that all regex patterns compile correctly
	for _, regex := range managedLabelRegexps {
		if _, err := regexp.Compile(regex); err != nil {
			t.Errorf("Failed to compile regex %q: %v", regex, err)
		}
	}
}
