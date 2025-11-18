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

func TestLabel_String(t *testing.T) {
	tests := []struct {
		name     string
		label    Label
		expected string
	}{
		{
			name:     "basic label",
			label:    Label{Key: "key", Value: "value"},
			expected: "key=value",
		},
		{
			name:     "empty key and value",
			label:    Label{Key: "", Value: ""},
			expected: "=",
		},
		{
			name:     "empty key",
			label:    Label{Key: "", Value: "value"},
			expected: "=value",
		},
		{
			name:     "empty value",
			label:    Label{Key: "key", Value: ""},
			expected: "key=",
		},
		{
			name:     "special characters in key and value",
			label:    Label{Key: "app.kubernetes.io/name", Value: "my-app"},
			expected: "app.kubernetes.io/name=my-app",
		},
		{
			name:     "numeric values",
			label:    Label{Key: "version", Value: "123"},
			expected: "version=123",
		},
		{
			name:     "label with spaces",
			label:    Label{Key: "environment", Value: "production cluster"},
			expected: "environment=production cluster",
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			result := tt.label.String()
			if result != tt.expected {
				t.Errorf("Label.String() = %q, want %q", result, tt.expected)
			}
		})
	}
}

func TestNewLabel(t *testing.T) {
	tests := []struct {
		name          string
		key           string
		value         string
		expectedKey   string
		expectedValue string
	}{
		{
			name:          "basic label creation",
			key:           "environment",
			value:         "production",
			expectedKey:   "environment",
			expectedValue: "production",
		},
		{
			name:          "empty key and value",
			key:           "",
			value:         "",
			expectedKey:   "",
			expectedValue: "",
		},
		{
			name:          "special characters",
			key:           "app.kubernetes.io/name",
			value:         "my-app-123",
			expectedKey:   "app.kubernetes.io/name",
			expectedValue: "my-app-123",
		},
		{
			name:          "numeric key and value",
			key:           "123",
			value:         "456",
			expectedKey:   "123",
			expectedValue: "456",
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			label := NewLabel(tt.key, tt.value)
			if label.Key != tt.expectedKey {
				t.Errorf("NewLabel(%q, %q).Key = %q, want %q", tt.key, tt.value, label.Key, tt.expectedKey)
			}
			if label.Value != tt.expectedValue {
				t.Errorf("NewLabel(%q, %q).Value = %q, want %q", tt.key, tt.value, label.Value, tt.expectedValue)
			}
		})
	}
}

func TestLabel_Integration(t *testing.T) {
	// Test that NewLabel and String work together properly
	key := "app.kubernetes.io/component"
	value := "database"

	label := NewLabel(key, value)
	expected := "app.kubernetes.io/component=database"

	result := label.String()
	if result != expected {
		t.Errorf("Integration test failed: NewLabel(%q, %q).String() = %q, want %q", key, value, result, expected)
	}
}
