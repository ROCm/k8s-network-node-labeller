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
	"fmt"
	"os"

	"gopkg.in/yaml.v3"
)

// Ensure that MockNicctlClient implements NicctlClient interface
var _ NicctlClient = (*MockNicctlClient)(nil)

// TestData represents the structure of test data in YAML files
type TestData struct {
	NICs          []NIC  `yaml:"nics" json:"nics"`
	DriverVersion string `yaml:"driver_version" json:"driver_version"`
}

// MockNicctlClient is a mock implementation of NicctlClient for testing
type MockNicctlClient struct {
	GetCardsWithPortsFunc     func() ([]NIC, error)
	GetIonicDriverVersionFunc func() (string, error)
	testData                  *TestData
}

// GetCardsWithPorts implements the NicctlClient interface
func (m *MockNicctlClient) GetCardsWithPorts() ([]NIC, error) {
	if m.GetCardsWithPortsFunc != nil {
		return m.GetCardsWithPortsFunc()
	}
	if m.testData != nil {
		return m.testData.NICs, nil
	}
	return []NIC{}, nil
}

// GetIonicDriverVersion implements the NicctlClient interface
func (m *MockNicctlClient) GetIonicDriverVersion() (string, error) {
	if m.GetIonicDriverVersionFunc != nil {
		return m.GetIonicDriverVersionFunc()
	}
	if m.testData != nil {
		return m.testData.DriverVersion, nil
	}
	return "", fmt.Errorf("failed to get Ionic driver version")
}

// NewMockNicctlClient creates a new MockNicctlClient
func NewMockNicctlClient() *MockNicctlClient {
	return &MockNicctlClient{}
}

// NewMockNicctlClientFromYAML creates a new MockNicctlClient with data loaded from a YAML file
func NewMockNicctlClientFromYAML(filePath string) (*MockNicctlClient, error) {
	data, err := os.ReadFile(filePath)
	if err != nil {
		return nil, fmt.Errorf("failed to read YAML file %s: %w", filePath, err)
	}

	var testData TestData
	if err := yaml.Unmarshal(data, &testData); err != nil {
		return nil, fmt.Errorf("failed to unmarshal YAML data from %s: %w", filePath, err)
	}

	return &MockNicctlClient{
		testData: &testData,
	}, nil
}

// LoadTestDataFromYAML loads test data from a YAML file into the mock client
func (m *MockNicctlClient) LoadTestDataFromYAML(filePath string) error {
	data, err := os.ReadFile(filePath)
	if err != nil {
		return fmt.Errorf("failed to read YAML file %s: %w", filePath, err)
	}

	var testData TestData
	if err := yaml.Unmarshal(data, &testData); err != nil {
		return fmt.Errorf("failed to unmarshal YAML data from %s: %w", filePath, err)
	}

	m.testData = &testData
	return nil
}

func (m *MockNicctlClient) GetTestdataYAML() ([]byte, error) {
	if m.testData == nil {
		return nil, fmt.Errorf("no test data available")
	}

	return yaml.Marshal(m.testData)
}
