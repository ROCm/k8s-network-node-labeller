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
	"encoding/json"
	"fmt"
	"os"
	"os/exec"
)

// NIC represents a network interface card
type NIC struct {
	ID              string `json:"id" yaml:"id"`
	ProductName     string `json:"product_name,omitempty" yaml:"product_name,omitempty"`
	SKU             string `json:"sku,omitempty" yaml:"sku,omitempty"`
	FirmwareVersion string `json:"firmware_version,omitempty" yaml:"firmware_version,omitempty"`
	Ports           []Port `json:"port,omitempty" yaml:"port,omitempty"`
}

// Port represents a network port on a NIC
type Port struct {
	Spec   PortSpec   `json:"spec" yaml:"spec"`
	Status PortStatus `json:"status" yaml:"status"`
}

// PortSpec contains the specification of a port
type PortSpec struct {
	ID        string `json:"id" yaml:"id"`
	Name      string `json:"name" yaml:"name"`
	PortType  string `json:"port_type" yaml:"port_type"`
	PortSpeed string `json:"port_speed" yaml:"port_speed"`
}

// PortStatus contains the status information of a port
type PortStatus struct {
	PhysicalPort      string `json:"physical_port" yaml:"physical_port"`
	OperationalStatus string `json:"operational_status" yaml:"operational_status"`
}

// NICResponse represents the JSON response from nicctl for NIC operations
type NICResponse struct {
	NICs []NIC `json:"nic" yaml:"nic"`
}

// VersionResponse represents the JSON response from nicctl for version operations
type VersionResponse struct {
	Version struct {
		IonicDriver string `json:"ionic_driver" yaml:"ionic_driver"`
	} `json:"version" yaml:"version"`
}

// NicctlClient defines the interface for interacting with nicctl
type NicctlClient interface {
	GetCardsWithPorts() ([]NIC, error)
	GetIonicDriverVersion() (string, error)
}

// NicctlCommandClient implements NicctlClient using actual nicctl commands
type NicctlCommandClient struct {
	binaryPath string
}

// NewNicctlCommandClient creates a new NicctlCommandClient
func NewNicctlCommandClient(binaryPath string) (*NicctlCommandClient, error) {
	if err := validateBinary(binaryPath); err != nil {
		return nil, fmt.Errorf("Invalid nicctl binary: %s", err)
	}

	return &NicctlCommandClient{
		binaryPath: binaryPath,
	}, nil
}

// validateBinary checks if the nicctl binary exists and is executable
func validateBinary(binaryPath string) error {
	// Check if file exists
	fileInfo, err := os.Stat(binaryPath)
	if err != nil {
		if os.IsNotExist(err) {
			return fmt.Errorf("nicctl binary not found at %s", binaryPath)
		}
		return fmt.Errorf("failed to check nicctl binary at %s: %w", binaryPath, err)
	}

	// Check if it is an executable
	mode := fileInfo.Mode()
	if !mode.IsRegular() {
		return fmt.Errorf("nicctl binary at %s is not a regular file", binaryPath)
	}
	if mode.Perm()&0111 == 0 {
		return fmt.Errorf("nicctl binary at %s is not executable", binaryPath)
	}

	// Check if nicctl show card -j gives an output, to make sure it is the right binary
	cmd := exec.Command(binaryPath, "show", "card", "-j")
	output, err := cmd.CombinedOutput()
	if err != nil {
		return fmt.Errorf("nicctl found but seems non-functional: Failed to execute nicctl binary at %s: %w\nOutput: %s", binaryPath, err, output)
	}

	// Try to parse the output as JSON to ensure it's valid
	var testResponse NICResponse
	if err := json.Unmarshal(output, &testResponse); err != nil {
		return fmt.Errorf("nicctl found but seems non-functional: nicctl binary at %s does not produce valid JSON output: %w\nOutput: %s", binaryPath, err, output)
	}

	return nil
}

// GetCardsWithPorts retrieves all NIC cards information with ports merged
func (c *NicctlCommandClient) GetCardsWithPorts() ([]NIC, error) {
	// Get card information
	cardCmd := exec.Command(c.binaryPath, "show", "card", "--json")
	cardOutput, err := cardCmd.Output()
	if err != nil {
		return nil, fmt.Errorf("failed to %s: %w", cardCmd.String(), err)
	}

	var cardResponse NICResponse
	if err := json.Unmarshal(cardOutput, &cardResponse); err != nil {
		return nil, fmt.Errorf("failed to parse nicctl card JSON output: %w", err)
	}

	// Get port information
	portCmd := exec.Command(c.binaryPath, "show", "port", "--json")
	portOutput, err := portCmd.Output()
	if err != nil {
		return nil, fmt.Errorf("failed to %s: %w", portCmd.String(), err)
	}

	var portResponse NICResponse
	if err := json.Unmarshal(portOutput, &portResponse); err != nil {
		return nil, fmt.Errorf("failed to parse nicctl port JSON output: %w", err)
	}

	// Create a map of NIC ID to ports for efficient lookup
	portsPerNIC := make(map[string][]Port)
	for _, nic := range portResponse.NICs {
		portsPerNIC[nic.ID] = nic.Ports
	}

	// Merge ports into cards
	for i, card := range cardResponse.NICs {
		if ports, exists := portsPerNIC[card.ID]; exists {
			cardResponse.NICs[i].Ports = ports
		}
	}

	return cardResponse.NICs, nil
}

// GetIonicDriverVersion retrieves the ionic driver version
func (c *NicctlCommandClient) GetIonicDriverVersion() (string, error) {
	cmd := exec.Command(c.binaryPath, "show", "version", "host-software", "--json")
	output, _ := cmd.Output()

	var response VersionResponse
	if err := json.Unmarshal(output, &response); err != nil {
		return "", fmt.Errorf("failed to parse nicctl JSON output: %w\nOutput: %s", err, output)
	}

	return response.Version.IonicDriver, nil
}
