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

package utils

import (
	"testing"
)

func TestHashArray(t *testing.T) {
	tests := []struct {
		name  string
		input []string
	}{
		{
			name:  "empty array",
			input: []string{},
		},
		{
			name:  "single element",
			input: []string{"hello"},
		},
		{
			name:  "two elements",
			input: []string{"hello", "world"},
		},
		{
			name:  "three elements",
			input: []string{"hello", "world", "test"},
		},
		{
			name:  "duplicate elements",
			input: []string{"test", "test", "test"},
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			result := HashArray(tt.input)

			if tt.name == "empty array" {
				if result != "" {
					t.Errorf("HashArray(%v) = %q, want empty string", tt.input, result)
				}
			} else {
				// For non-empty arrays, ensure we get a non-empty hash
				if result == "" {
					t.Errorf("HashArray(%v) returned empty string, expected non-empty hash", tt.input)
				}

				// Test consistency: same input should produce same output
				result2 := HashArray(tt.input)
				if result != result2 {
					t.Errorf("HashArray is not consistent: first call returned %q, second call returned %q", result, result2)
				}

				// Verify hash length is consistent (should be 16 bytes for murmur3 128-bit hash)
				if len(result) != 16 {
					t.Errorf("HashArray(%v) returned hash of length %d, expected 16", tt.input, len(result))
				}
			}
		})
	}
}

func TestHashArrayDifferentInputs(t *testing.T) {
	// Test that different arrays produce different hashes
	testCases := []struct {
		name   string
		array1 []string
		array2 []string
	}{
		{
			name:   "Test case 1",
			array1: []string{"test1", "value1"},
			array2: []string{"test2", "value2"},
		},
		{
			name:   "Test case 2",
			array1: []string{"hello", "world"},
			array2: []string{"foo", "bar"},
		},
	}

	for _, tt := range testCases {
		t.Run(tt.name, func(t *testing.T) {
			hash1 := HashArray(tt.array1)
			hash2 := HashArray(tt.array2)

			if hash1 == hash2 {
				t.Errorf("Different arrays produced same hash: array1=%v, array2=%v, hash=%q", tt.array1, tt.array2, hash1)
			}
		})
	}
}

func TestHashArrayOrderSensitivity(t *testing.T) {
	// Test that order matters - same elements in different order should produce different hashes
	testCases := []struct {
		name   string
		array1 []string
		array2 []string
	}{
		{
			name:   "Test case 1",
			array1: []string{"a", "b", "c"},
			array2: []string{"c", "a", "b"},
		},
		{
			name:   "Test case 2",
			array1: []string{"foo", "bar", "bob"},
			array2: []string{"bob", "foo", "bar"},
		},
	}

	for _, tt := range testCases {
		t.Run(tt.name, func(t *testing.T) {
			hash1 := HashArray(tt.array1)
			hash2 := HashArray(tt.array2)

			if hash1 == hash2 {
				t.Errorf("Arrays with different order produced same hash: array1=%v, array2=%v, hash=%q", tt.array1, tt.array2, hash1)
			}
		})
	}
}

func TestHashArrayLength(t *testing.T) {
	// Test that the hash length is consistent for multi-element arrays
	testArrays := [][]string{
		{"a", "b"},
		{"hello", "world"},
		{"test1", "test2", "test3"},
		{"very", "long", "array", "with", "many", "elements"},
	}

	var expectedLength int
	for i, arr := range testArrays {
		hash := HashArray(arr)
		if i == 0 {
			expectedLength = len(hash)
		} else {
			if len(hash) != expectedLength {
				t.Errorf("Hash length inconsistent: expected %d, got %d for array %v", expectedLength, len(hash), arr)
			}
		}
	}
}

func TestHashArrayChaining(t *testing.T) {
	// Test that the chain hashing works as expected
	// Each step should produce a different result
	step1 := HashArray([]string{"first"})
	step2 := HashArray([]string{"first", "second"})
	step3 := HashArray([]string{"first", "second", "third"})

	if step1 == step2 {
		t.Errorf("Adding second element didn't change hash: %q", step1)
	}

	if step2 == step3 {
		t.Errorf("Adding third element didn't change hash: %q", step2)
	}

	if step1 == step3 {
		t.Errorf("Single element hash equals three element hash: %q", step1)
	}
}
