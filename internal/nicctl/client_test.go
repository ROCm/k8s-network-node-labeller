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

package nicctl

import (
	"os"
	"path/filepath"
	"strings"
	"testing"
)

func TestValidateBinary(t *testing.T) {
	// Create a temporary directory for testing
	tempDir := t.TempDir()

	// Create all different types of test files/directories upfront
	nonExistentPath := filepath.Join(tempDir, "nonexistent_binary")

	// Non-executable binary (644 permissions)
	nonExecutablePath := filepath.Join(tempDir, "non_executable_binary")
	nonExecutableContent := `This won't execute man!`
	err := os.WriteFile(nonExecutablePath, []byte(nonExecutableContent), 0644)
	if err != nil {
		t.Fatalf("Failed to create non-executable binary: %v", err)
	}

	// Directory instead of regular file
	directoryPath := filepath.Join(tempDir, "directory_not_file")
	err = os.Mkdir(directoryPath, 0755)
	if err != nil {
		t.Fatalf("Failed to create directory: %v", err)
	}

	// Executable binary that fails on any command
	failingBinaryPath := filepath.Join(tempDir, "failing_binary")
	failingBinaryContent := `#!/bin/sh
echo "Oops! :_)" >&2
exit 1
`
	err = os.WriteFile(failingBinaryPath, []byte(failingBinaryContent), 0755)
	if err != nil {
		t.Fatalf("Failed to create failing binary: %v", err)
	}

	// Executable binary that returns invalid JSON on show cards -j command
	invalidJsonBinaryPath := filepath.Join(tempDir, "invalid_json_binary")
	invalidJsonBinaryContent := `#!/bin/bash
if [ "$1" == "show" ] && [ "$2" == "cards" ] && [ "$3" == "-j" ]; then
	echo "{invalid json here"
	exit 0
else
	exit 1
fi
`
	err = os.WriteFile(invalidJsonBinaryPath, []byte(invalidJsonBinaryContent), 0755)
	if err != nil {
		t.Fatalf("Failed to create invalid JSON binary: %v", err)
	}

	// Valid executable binary
	validBinaryPath := filepath.Join(tempDir, "valid_binary")
	validBinaryContent := `#!/bin/bash
if [ "$1" == "show" ] && [ "$2" == "card" ] && [ "$3" == "-j" ]; then
	echo "{\"nics\": []}"
	exit 0
else
	exit 1
fi
`
	err = os.WriteFile(validBinaryPath, []byte(validBinaryContent), 0755)
	if err != nil {
		t.Fatalf("Failed to create valid binary: %v", err)
	}

	type testCase struct {
		name         string
		binaryPath   string
		wantErr      bool
		errorMessage string
	}

	testCases := []testCase{
		{
			name:         "binary not found",
			binaryPath:   nonExistentPath,
			wantErr:      true,
			errorMessage: "nicctl binary not found",
		},
		{
			name:         "binary not executable",
			binaryPath:   nonExecutablePath,
			wantErr:      true,
			errorMessage: "is not executable",
		},
		{
			name:         "binary not a regular file",
			binaryPath:   directoryPath,
			wantErr:      true,
			errorMessage: "is not a regular file",
		},
		{
			name:         "executable binary with failing show cards command",
			binaryPath:   failingBinaryPath,
			wantErr:      true,
			errorMessage: "nicctl found but seems non-functional",
		},
		{
			name:         "executable binary with invalid json on show cards command",
			binaryPath:   invalidJsonBinaryPath,
			wantErr:      true,
			errorMessage: "nicctl found but seems non-functional",
		},
		{
			name:       "valid executable binary",
			binaryPath: validBinaryPath,
			wantErr:    false,
		},
	}

	for _, tc := range testCases {
		t.Run(tc.name, func(t *testing.T) {
			// Execute the function under test with the specified binary path
			err := validateBinary(tc.binaryPath)

			// Verify the results
			if tc.wantErr {
				if err == nil {
					t.Fatalf("Expected error for test case '%s', but got nil", tc.name)
				}

				if !strings.Contains(err.Error(), tc.errorMessage) {
					t.Errorf("Expected error message to contain '%s', but got: %s", tc.errorMessage, err.Error())
				}
			} else {
				if err != nil {
					t.Errorf("Expected no error for test case '%s', but got: %v", tc.name, err)
				}
			}
		})
	}
}
