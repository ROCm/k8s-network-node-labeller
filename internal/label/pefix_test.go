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
	"testing"
)

func TestDefaultPrefixedKey(t *testing.T) {
	tests := []struct {
		name     string
		key      string
		expected string
	}{
		{
			name:     "basic key",
			key:      "environment",
			expected: "amd.com/environment",
		},
		{
			name:     "empty key",
			key:      "",
			expected: "amd.com/",
		},
		{
			name:     "key with dots",
			key:      "app.version",
			expected: "amd.com/app.version",
		},
		{
			name:     "key with dashes",
			key:      "app-name",
			expected: "amd.com/app-name",
		},
		{
			name:     "key with underscores",
			key:      "app_name",
			expected: "amd.com/app_name",
		},
		{
			name:     "numeric key",
			key:      "123",
			expected: "amd.com/123",
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			result := DefaultPrefixedKey(tt.key)
			if result != tt.expected {
				t.Errorf("DefaultPrefixedKey(%q) = %q, want %q", tt.key, result, tt.expected)
			}
		})
	}
}

func TestBetaPrefixedKey(t *testing.T) {
	tests := []struct {
		name     string
		key      string
		expected string
	}{
		{
			name:     "basic key",
			key:      "environment",
			expected: "beta.amd.com/environment",
		},
		{
			name:     "empty key",
			key:      "",
			expected: "beta.amd.com/",
		},
		{
			name:     "key with dots",
			key:      "app.version",
			expected: "beta.amd.com/app.version",
		},
		{
			name:     "key with dashes",
			key:      "app-name",
			expected: "beta.amd.com/app-name",
		},
		{
			name:     "key with underscores",
			key:      "app_name",
			expected: "beta.amd.com/app_name",
		},
		{
			name:     "numeric key",
			key:      "123",
			expected: "beta.amd.com/123",
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			result := BetaPrefixedKey(tt.key)
			if result != tt.expected {
				t.Errorf("BetaPrefixedKey(%q) = %q, want %q", tt.key, result, tt.expected)
			}
		})
	}
}

func TestPrefixedKey(t *testing.T) {
	tests := []struct {
		name     string
		prefix   string
		key      string
		expected string
	}{
		{
			name:     "basic prefix and key",
			prefix:   "example.com",
			key:      "environment",
			expected: "example.com/environment",
		},
		{
			name:     "empty prefix",
			prefix:   "",
			key:      "environment",
			expected: "/environment",
		},
		{
			name:     "empty key",
			prefix:   "example.com",
			key:      "",
			expected: "example.com/",
		},
		{
			name:     "both empty",
			prefix:   "",
			key:      "",
			expected: "/",
		},
		{
			name:     "prefix with subdomain",
			prefix:   "app.kubernetes.io",
			key:      "name",
			expected: "app.kubernetes.io/name",
		},
		{
			name:     "prefix with multiple dots",
			prefix:   "a.b.c.d",
			key:      "test.key",
			expected: "a.b.c.d/test.key",
		},
		{
			name:     "special characters in key",
			prefix:   "example.com",
			key:      "app-name_version.1",
			expected: "example.com/app-name_version.1",
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			result := PrefixedKey(tt.prefix, tt.key)
			if result != tt.expected {
				t.Errorf("PrefixedKey(%q, %q) = %q, want %q", tt.prefix, tt.key, result, tt.expected)
			}
		})
	}
}

func TestDefaultNICPrefixedKey(t *testing.T) {
	tests := []struct {
		name     string
		key      string
		expected string
	}{
		{
			name:     "basic NIC property",
			key:      "model",
			expected: "amd.com/nic.model",
		},
		{
			name:     "empty key",
			key:      "",
			expected: "amd.com/nic.",
		},
		{
			name:     "key with dots",
			key:      "port.speed",
			expected: "amd.com/nic.port.speed",
		},
		{
			name:     "key with dashes",
			key:      "driver-version",
			expected: "amd.com/nic.driver-version",
		},
		{
			name:     "key with underscores",
			key:      "pci_address",
			expected: "amd.com/nic.pci_address",
		},
		{
			name:     "complex key",
			key:      "port-1.speed_gbps",
			expected: "amd.com/nic.port-1.speed_gbps",
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			result := DefaultNICPrefixedKey(tt.key)
			if result != tt.expected {
				t.Errorf("DefaultNICPrefixedKey(%q) = %q, want %q", tt.key, result, tt.expected)
			}
		})
	}
}

func TestPrefixFunctionConsistency(t *testing.T) {
	// Test that the convenience functions are consistent with the general PrefixedKey function
	key := "test-key"

	// Test DefaultPrefixedKey consistency
	defaultResult := DefaultPrefixedKey(key)
	expectedDefault := PrefixedKey(DefaultPrefix, key)
	if defaultResult != expectedDefault {
		t.Errorf("DefaultPrefixedKey inconsistent: got %q, expected %q", defaultResult, expectedDefault)
	}

	// Test BetaPrefixedKey consistency
	betaResult := BetaPrefixedKey(key)
	expectedBeta := PrefixedKey(BetaPrefix, key)
	if betaResult != expectedBeta {
		t.Errorf("BetaPrefixedKey inconsistent: got %q, expected %q", betaResult, expectedBeta)
	}

	// Test DefaultNICPrefixedKey consistency
	nicResult := DefaultNICPrefixedKey(key)
	expectedNIC := DefaultPrefixedKey(NICQualifier + "." + key)
	if nicResult != expectedNIC {
		t.Errorf("DefaultNICPrefixedKey inconsistent: got %q, expected %q", nicResult, expectedNIC)
	}
}
