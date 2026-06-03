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
	"errors"
	"os"
	"path/filepath"
	"strings"
	"testing"
)

func TestMockNicctlClient_GetCardsWithPorts_Default(t *testing.T) {
	mock := NewMockNicctlClient()

	cards, err := mock.GetCardsWithPorts()

	if err != nil {
		t.Errorf("Expected no error, got %v", err)
	}

	if len(cards) != 0 {
		t.Errorf("Expected empty slice, got %d cards", len(cards))
	}
}

func TestMockNicctlClient_GetCardsWithPorts_Custom(t *testing.T) {
	expectedCards := []NIC{
		{
			ID:              "test-nic-1",
			ProductName:     "Test NIC",
			SKU:             "TEST-SKU",
			FirmwareVersion: "1.0.0",
			Ports: []Port{
				{
					Spec: PortSpec{
						ID:        "port1",
						Name:      "eth0",
						PortType:  "ethernet",
						PortSpeed: "100Gbps",
					},
					Status: PortStatus{
						PhysicalPort:      "phys1",
						OperationalStatus: "up",
					},
				},
			},
		},
	}

	mock := NewMockNicctlClient()
	mock.GetCardsWithPortsFunc = func() ([]NIC, error) {
		return expectedCards, nil
	}

	cards, err := mock.GetCardsWithPorts()

	if err != nil {
		t.Errorf("Expected no error, got %v", err)
	}

	if len(cards) != 1 {
		t.Errorf("Expected 1 card, got %d", len(cards))
	}

	if cards[0].ID != "test-nic-1" {
		t.Errorf("Expected card ID 'test-nic-1', got '%s'", cards[0].ID)
	}

	if cards[0].ProductName != "Test NIC" {
		t.Errorf("Expected product name 'Test NIC', got '%s'", cards[0].ProductName)
	}

	if len(cards[0].Ports) != 1 {
		t.Errorf("Expected 1 port, got %d", len(cards[0].Ports))
	}
}

func TestMockNicctlClient_GetCardsWithPorts_Error(t *testing.T) {
	expectedError := errors.New("test error")

	mock := NewMockNicctlClient()
	mock.GetCardsWithPortsFunc = func() ([]NIC, error) {
		return nil, expectedError
	}

	cards, err := mock.GetCardsWithPorts()

	if err != expectedError {
		t.Errorf("Expected error '%v', got '%v'", expectedError, err)
	}

	if cards != nil {
		t.Errorf("Expected nil cards, got %v", cards)
	}
}

func TestMockNicctlClient_GetIonicDriverVersion_Custom(t *testing.T) {
	expectedVersion := "2.5.7"

	mock := NewMockNicctlClient()
	mock.GetIonicDriverVersionFunc = func() (string, error) {
		return expectedVersion, nil
	}

	version, err := mock.GetIonicDriverVersion()

	if err != nil {
		t.Errorf("Expected no error, got %v", err)
	}

	if version != expectedVersion {
		t.Errorf("Expected version '%s', got '%s'", expectedVersion, version)
	}
}

func TestMockNicctlClient_GetCardProfiles(t *testing.T) {
	profileError := errors.New("profile error")

	tests := []struct {
		name             string
		setup            func(*MockNicctlClient)
		wantErr          error
		wantErrContains  string
		expectedProfiles map[string]string
	}{
		{
			name:             "default returns empty map",
			setup:            func(m *MockNicctlClient) {},
			expectedProfiles: map[string]string{},
		},
		{
			name: "custom func override",
			setup: func(m *MockNicctlClient) {
				m.GetCardProfilesFunc = func() (map[string]string, error) {
					return map[string]string{"nic1": "pf1_vf1"}, nil
				}
			},
			expectedProfiles: map[string]string{"nic1": "pf1_vf1"},
		},
		{
			name: "func override error",
			setup: func(m *MockNicctlClient) {
				m.GetCardProfilesFunc = func() (map[string]string, error) {
					return nil, profileError
				}
			},
			wantErr: profileError,
		},
		{
			name: "yaml profile error trigger",
			setup: func(m *MockNicctlClient) {
				m.testData = &TestData{
					Profiles: map[string]string{"nic1": ProfileErrorTrigger},
				}
			},
			wantErrContains: "profile command failed",
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			mock := NewMockNicctlClient()
			tt.setup(mock)

			profiles, err := mock.GetCardProfiles()

			if tt.wantErr != nil || tt.wantErrContains != "" {
				if tt.wantErr != nil {
					if err != tt.wantErr {
						t.Errorf("Expected error '%v', got '%v'", tt.wantErr, err)
					}
				} else if err == nil || !strings.Contains(err.Error(), tt.wantErrContains) {
					t.Errorf("Expected error containing %q, got %v", tt.wantErrContains, err)
				}
				if profiles != nil {
					t.Errorf("Expected nil profiles, got %v", profiles)
				}
				return
			}

			if err != nil {
				t.Fatalf("Expected no error, got %v", err)
			}
			if len(profiles) != len(tt.expectedProfiles) {
				t.Fatalf("Expected %d profiles, got %d: %v", len(tt.expectedProfiles), len(profiles), profiles)
			}
			for nicID, want := range tt.expectedProfiles {
				if got := profiles[nicID]; got != want {
					t.Errorf("Expected profile %s for %s, got %s", want, nicID, got)
				}
			}
		})
	}
}

func TestMockNicctlClient_GetIonicDriverVersion_Error(t *testing.T) {
	expectedError := errors.New("driver version error")

	mock := NewMockNicctlClient()
	mock.GetIonicDriverVersionFunc = func() (string, error) {
		return "", expectedError
	}

	version, err := mock.GetIonicDriverVersion()

	if err != expectedError {
		t.Errorf("Expected error '%v', got '%v'", expectedError, err)
	}

	if version != "" {
		t.Errorf("Expected empty version, got '%s'", version)
	}
}

func TestNewMockNicctlClient(t *testing.T) {
	mock := NewMockNicctlClient()

	if mock == nil {
		t.Fatal("Expected mock client to be created, got nil")
	}

	// Test that it implements the NicctlClient interface
	var _ NicctlClient = mock
}

func TestMockNicctlClient_LoadTestDataFromYAML(t *testing.T) {
	// Create a temporary YAML file
	yamlContent := `
nics:
  - id: "yaml-nic"
    product_name: "YAML Test NIC"
    sku: "YAML-SKU"
    firmware_version: "2.1.0"
    port:
      - spec:
          id: "port1"
          name: "eth0"
          port_type: "ethernet"
          port_speed: "50Gbps"
        status:
          physical_port: "phys1"
          operational_status: "up"
profiles:
  yaml-nic: "pf1_vf1"
driver_version: "3.0.0"
`

	tmpDir := t.TempDir()
	yamlFile := filepath.Join(tmpDir, "test.yaml")

	err := os.WriteFile(yamlFile, []byte(yamlContent), 0644)
	if err != nil {
		t.Fatalf("Failed to write test YAML file: %v", err)
	}

	mock := NewMockNicctlClient()
	err = mock.LoadTestDataFromYAML(yamlFile)
	if err != nil {
		t.Fatalf("Failed to load test data from YAML: %v", err)
	}

	// Test the loaded data
	nics, err := mock.GetCardsWithPorts()
	if err != nil {
		t.Errorf("Expected no error, got %v", err)
	}
	if len(nics) != 1 {
		t.Errorf("Expected 1 NIC, got %d", len(nics))
	}
	if nics[0].ID != "yaml-nic" {
		t.Errorf("Expected NIC ID 'yaml-nic', got '%s'", nics[0].ID)
	}
	if nics[0].SKU != "YAML-SKU" {
		t.Errorf("Expected NIC SKU 'YAML-SKU', got '%s'", nics[0].SKU)
	}
	if nics[0].ProductName != "YAML Test NIC" {
		t.Errorf("Expected product name 'YAML Test NIC', got '%s'", nics[0].ProductName)
	}
	if len(nics[0].Ports) != 1 {
		t.Errorf("Expected 1 port, got %d", len(nics[0].Ports))
	}
	if nics[0].Ports[0].Spec.PortSpeed != "50Gbps" {
		t.Errorf("Expected port speed '50Gbps', got '%s'", nics[0].Ports[0].Spec.PortSpeed)
	}

	version, err := mock.GetIonicDriverVersion()
	if err != nil {
		t.Errorf("Expected no error, got %v", err)
	}
	if version != "3.0.0" {
		t.Errorf("Expected driver version '3.0.0', got '%s'", version)
	}

	profiles, err := mock.GetCardProfiles()
	if err != nil {
		t.Errorf("Expected no error, got %v", err)
	}
	if profiles["yaml-nic"] != "pf1_vf1" {
		t.Errorf("Expected profile pf1_vf1 for yaml-nic, got '%s'", profiles["yaml-nic"])
	}
}

func TestMockNicctlClient_LoadTestDataFromYAML_FileNotFound(t *testing.T) {
	mock := NewMockNicctlClient()
	err := mock.LoadTestDataFromYAML("nonexistent.yaml")

	if err == nil {
		t.Error("Expected error for nonexistent file, got nil")
	}
}

func TestNewMockNicctlClientFromYAML_InvalidYAML(t *testing.T) {
	// Create a temporary invalid YAML file
	invalidYamlContent := `nics:
  - id: "test"
    invalid: yaml: content: here`

	tmpDir := t.TempDir()
	yamlFile := filepath.Join(tmpDir, "invalid.yaml")

	err := os.WriteFile(yamlFile, []byte(invalidYamlContent), 0644)
	if err != nil {
		t.Fatalf("Failed to write test YAML file: %v", err)
	}

	_, err = NewMockNicctlClientFromYAML(yamlFile)
	if err == nil {
		t.Error("Expected error for invalid YAML, got nil")
	}
}
