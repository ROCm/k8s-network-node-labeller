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
	"bytes"
	"log/slog"
	"strings"
	"testing"
)

func TestNormalizeString(t *testing.T) {
	tests := []struct {
		name     string
		input    string
		expected string
	}{
		{
			name:     "uppercase to lowercase",
			input:    "UPPERCASE",
			expected: "uppercase",
		},
		{
			name:     "spaces to dashes",
			input:    "hello world",
			expected: "hello-world",
		},
		{
			name:     "underscores to dashes",
			input:    "hello_world",
			expected: "hello-world",
		},
		{
			name:     "mixed spaces and underscores",
			input:    "hello world_test string",
			expected: "hello-world-test-string",
		},
		{
			name:     "remove special characters",
			input:    "hello@world#test$string%",
			expected: "helloworldteststring",
		},
		{
			name:     "preserve dots",
			input:    "hello.world.test",
			expected: "hello.world.test",
		},
		{
			name:     "preserve dashes",
			input:    "hello-world-test",
			expected: "hello-world-test",
		},
		{
			name:     "preserve alphanumeric",
			input:    "hello123world456",
			expected: "hello123world456",
		},
		{
			name:     "complex string with all transformations",
			input:    "Hello_World Test@123.Example#String$",
			expected: "hello-world-test123.examplestring",
		},
		{
			name:     "empty string",
			input:    "",
			expected: "",
		},
		{
			name:     "string with only special characters",
			input:    "@#$%^&*()",
			expected: "",
		},
		{
			name:     "string with only spaces and underscores",
			input:    "_ _ _ _",
			expected: "-------",
		},
		{
			name:     "numbers only",
			input:    "123456",
			expected: "123456",
		},
		{
			name:     "dots only",
			input:    "...",
			expected: "...",
		},
		{
			name:     "dashes only",
			input:    "---",
			expected: "---",
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			result := NormalizeString(tt.input)
			if result != tt.expected {
				t.Errorf("NormalizeString(%q) = %q, want %q", tt.input, result, tt.expected)
			}
		})
	}
}

func TestHashLabels(t *testing.T) {
	tests := []struct {
		name   string
		labels []Label
	}{
		{
			name:   "empty labels",
			labels: []Label{},
		},
		{
			name: "single label",
			labels: []Label{
				{Key: "test.key", Value: "test-value"},
			},
		},
		{
			name: "multiple labels",
			labels: []Label{
				{Key: "key1", Value: "value1"},
				{Key: "key2", Value: "value2"},
			},
		},
		{
			name: "duplicate labels",
			labels: []Label{
				{Key: "key1", Value: "value1"},
				{Key: "key1", Value: "value1"},
			},
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			result := HashLabels(tt.labels)

			if tt.name == "empty labels" {
				if result != "" {
					t.Errorf("HashLabels(%v) = %q, want empty string", tt.labels, result)
				}
			} else {
				// For non-empty labels, ensure we get a non-empty hash
				if result == "" {
					t.Errorf("HashLabels(%v) returned empty string, expected non-empty hash", tt.labels)
				}

				// Test consistency: same input should produce same output
				result2 := HashLabels(tt.labels)
				if result != result2 {
					t.Errorf("HashLabels is not consistent: first call returned %q, second call returned %q", result, result2)
				}

				// Verify hash length is consistent (should be 16 bytes for murmur3 128-bit hash)
				if len(result) != 16 {
					t.Errorf("HashLabels(%v) returned hash of length %d, expected 16", tt.labels, len(result))
				}
			}
		})
	}

	// Test that different label sets produce different hashes
	labels1 := []Label{{Key: "key1", Value: "value1"}}
	labels2 := []Label{{Key: "key2", Value: "value2"}}

	hash1 := HashLabels(labels1)
	hash2 := HashLabels(labels2)

	if hash1 == hash2 {
		t.Errorf("Different label sets produced same hash: %q", hash1)
	}
}

func TestPrintLabels(t *testing.T) {
	// Capture log output by creating a custom logger
	var buf bytes.Buffer
	logger := slog.New(slog.NewTextHandler(&buf, &slog.HandlerOptions{
		Level: slog.LevelDebug,
	}))

	// Set the default logger to our custom one
	oldLogger := slog.Default()
	slog.SetDefault(logger)
	defer slog.SetDefault(oldLogger)

	tests := []struct {
		name          string
		labels        []Label
		level         slog.Level
		expectedParts []string
	}{
		{
			name:          "empty labels",
			labels:        []Label{},
			level:         slog.LevelInfo,
			expectedParts: []string{"No labels to print"},
		},
		{
			name: "single label",
			labels: []Label{
				{Key: "test.key", Value: "test-value"},
			},
			level:         slog.LevelInfo,
			expectedParts: []string{"test.key", "test-value"},
		},
		{
			name: "multiple labels",
			labels: []Label{
				{Key: "key1", Value: "value1"},
				{Key: "key2", Value: "value2"},
			},
			level:         slog.LevelDebug,
			expectedParts: []string{"key1", "value1", "key2", "value2"},
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			buf.Reset()

			PrintLabels(tt.labels, tt.level)

			output := buf.String()
			for _, expectedPart := range tt.expectedParts {
				if !strings.Contains(output, expectedPart) {
					t.Errorf("Expected output to contain %q, but got: %s", expectedPart, output)
				}
			}
		})
	}
}

func TestPrintLabelsInfo(t *testing.T) {
	// Capture log output
	var buf bytes.Buffer
	logger := slog.New(slog.NewTextHandler(&buf, &slog.HandlerOptions{
		Level: slog.LevelDebug,
	}))

	oldLogger := slog.Default()
	slog.SetDefault(logger)
	defer slog.SetDefault(oldLogger)

	labels := []Label{
		{Key: "test.key1", Value: "test-value1"},
		{Key: "test.key2", Value: "test-value2"},
	}

	PrintLabelsInfo(labels)

	output := buf.String()
	lines := strings.Split(strings.TrimSpace(output), "\n")

	// Should have one line for each label
	expectedLineCount := len(labels)
	if len(lines) != expectedLineCount {
		t.Errorf("Expected %d log lines, got %d. Output: %s", expectedLineCount, len(lines), output)
	}

	// Check each label has its own line with INFO level
	for i, label := range labels {
		lineIndex := i
		if lineIndex >= len(lines) {
			t.Errorf("Missing log line for label %d: %v", i, label)
			continue
		}

		line := lines[lineIndex]

		// Verify the line contains the label key and value
		if !strings.Contains(line, label.Key) {
			t.Errorf("Label line %d should contain key %q, got: %s", i, label.Key, line)
		}
		if !strings.Contains(line, label.Value) {
			t.Errorf("Label line %d should contain value %q, got: %s", i, label.Value, line)
		}

		// Verify the line has INFO level
		if !strings.Contains(line, "level=INFO") {
			t.Errorf("Label line %d should have INFO level, got: %s", i, line)
		}
	}
}

func TestPrintLabelsDebug(t *testing.T) {
	// Capture log output
	var buf bytes.Buffer
	logger := slog.New(slog.NewTextHandler(&buf, &slog.HandlerOptions{
		Level: slog.LevelDebug,
	}))

	oldLogger := slog.Default()
	slog.SetDefault(logger)
	defer slog.SetDefault(oldLogger)

	labels := []Label{
		{Key: "debug.key1", Value: "debug-value1"},
		{Key: "debug.key2", Value: "debug-value2"},
		{Key: "debug.key3", Value: "debug-value3"},
	}

	PrintLabelsDebug(labels)

	output := buf.String()
	lines := strings.Split(strings.TrimSpace(output), "\n")

	// Should have one line for each label
	expectedLineCount := len(labels)
	if len(lines) != expectedLineCount {
		t.Errorf("Expected %d log lines, got %d. Output: %s", expectedLineCount, len(lines), output)
	}

	// Check each label has its own line with DEBUG level
	for i, label := range labels {
		lineIndex := i
		if lineIndex >= len(lines) {
			t.Errorf("Missing log line for label %d: %v", i, label)
			continue
		}

		line := lines[lineIndex]

		// Verify the line contains the label key and value
		if !strings.Contains(line, label.Key) {
			t.Errorf("Label line %d should contain key %q, got: %s", i, label.Key, line)
		}
		if !strings.Contains(line, label.Value) {
			t.Errorf("Label line %d should contain value %q, got: %s", i, label.Value, line)
		}

		// Verify the line has DEBUG level
		if !strings.Contains(line, "level=DEBUG") {
			t.Errorf("Label line %d should have DEBUG level, got: %s", i, line)
		}
	}
}
