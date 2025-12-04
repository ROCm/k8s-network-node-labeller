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

package testutils

import (
	"strings"
	"testing"
)

func TestValidateYamlConfig(t *testing.T) {
	tests := []struct {
		name    string
		config  TestSysfsConfig
		wantErr bool
		errMsg  string
	}{
		{
			name: "valid configuration with root device only",
			config: TestSysfsConfig{
				Devices: []TestPCIDevice{
					{
						ID:         "device1",
						PCIAddress: "0000:00:02.0",
					},
				},
			},
			wantErr: false,
		},
		{
			name: "valid configuration with parent-child on same bus",
			config: TestSysfsConfig{
				Devices: []TestPCIDevice{
					{
						ID:         "parent",
						PCIAddress: "0000:05:00.0",
					},
					{
						ID:            "child",
						PCIAddress:    "0000:05:01.0",
						ParentAddress: "0000:05:00.0",
					},
				},
			},
			wantErr: false,
		},
		{
			name: "invalid configuration - empty PCI address",
			config: TestSysfsConfig{
				Devices: []TestPCIDevice{
					{
						ID:         "device1",
						PCIAddress: "",
					},
				},
			},
			wantErr: true,
			errMsg:  "has empty pci_address",
		},
		{
			name: "invalid configuration - invalid PCI address format",
			config: TestSysfsConfig{
				Devices: []TestPCIDevice{
					{
						ID:         "device1",
						PCIAddress: "invalid:address",
					},
				},
			},
			wantErr: true,
			errMsg:  "has invalid pci_address",
		},
		{
			name: "invalid configuration - duplicate PCI addresses",
			config: TestSysfsConfig{
				Devices: []TestPCIDevice{
					{
						ID:         "device1",
						PCIAddress: "0000:05:00.0",
					},
					{
						ID:         "device2",
						PCIAddress: "0000:05:00.0",
					},
				},
			},
			wantErr: true,
			errMsg:  "duplicate PCI address",
		},
		{
			name: "invalid configuration - invalid parent address format",
			config: TestSysfsConfig{
				Devices: []TestPCIDevice{
					{
						ID:            "child",
						PCIAddress:    "0000:05:01.0",
						ParentAddress: "invalid:parent",
					},
				},
			},
			wantErr: true,
			errMsg:  "has invalid parent_address",
		},
		{
			name: "invalid configuration - non-existent parent device",
			config: TestSysfsConfig{
				Devices: []TestPCIDevice{
					{
						ID:            "child",
						PCIAddress:    "0000:05:01.0",
						ParentAddress: "0000:05:00.0",
					},
				},
			},
			wantErr: true,
			errMsg:  "references non-existent parent device",
		},
		{
			name: "valid configuration - parent and child on different buses",
			config: TestSysfsConfig{
				Devices: []TestPCIDevice{
					{
						ID:         "parent",
						PCIAddress: "0000:05:00.0",
					},
					{
						ID:            "child",
						PCIAddress:    "0000:06:01.0",
						ParentAddress: "0000:05:00.0",
					},
				},
			},
			wantErr: false,
		},
		{
			name: "valid configuration - parent and child on different domains",
			config: TestSysfsConfig{
				Devices: []TestPCIDevice{
					{
						ID:         "parent",
						PCIAddress: "0000:05:00.0",
					},
					{
						ID:            "child",
						PCIAddress:    "0001:05:01.0",
						ParentAddress: "0000:05:00.0",
					},
				},
			},
			wantErr: false,
		},
		{
			name: "valid configuration with multiple levels of hierarchy",
			config: TestSysfsConfig{
				Devices: []TestPCIDevice{
					{
						ID:         "root",
						PCIAddress: "0000:05:00.0",
					},
					{
						ID:            "level1",
						PCIAddress:    "0000:05:01.0",
						ParentAddress: "0000:05:00.0",
					},
					{
						ID:            "level2",
						PCIAddress:    "0000:05:02.0",
						ParentAddress: "0000:05:01.0",
					},
				},
			},
			wantErr: false,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			err := validateYamlConfig(&tt.config)
			if (err != nil) != tt.wantErr {
				t.Errorf("validateYamlConfig() error = %v, wantErr %v", err, tt.wantErr)
				return
			}
			if tt.wantErr && err != nil {
				if tt.errMsg != "" && !strings.Contains(err.Error(), tt.errMsg) {
					t.Errorf("validateYamlConfig() error = %v, expected to contain %s", err.Error(), tt.errMsg)
				}
			}
		})
	}
}

func TestParsePCIAddress(t *testing.T) {
	tests := []struct {
		name       string
		address    string
		wantDomain string
		wantBus    string
		wantDevice string
		wantFunc   string
		wantErr    bool
	}{
		{
			name:       "valid PCI address",
			address:    "0000:05:00.0",
			wantDomain: "0000",
			wantBus:    "05",
			wantDevice: "00",
			wantFunc:   "0",
			wantErr:    false,
		},
		{
			name:       "valid PCI address with hex values",
			address:    "0001:ff:1a.7",
			wantDomain: "0001",
			wantBus:    "ff",
			wantDevice: "1a",
			wantFunc:   "7",
			wantErr:    false,
		},
		{
			name:    "invalid PCI address - wrong format",
			address: "invalid",
			wantErr: true,
		},
		{
			name:    "invalid PCI address - missing parts",
			address: "0000:05:00",
			wantErr: true,
		},
		{
			name:    "invalid PCI address - wrong separator",
			address: "0000-05-00.0",
			wantErr: true,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			gotDomain, gotBus, gotDevice, gotFunc, err := parsePCIAddress(tt.address)
			if (err != nil) != tt.wantErr {
				t.Errorf("parsePCIAddress() error = %v, wantErr %v", err, tt.wantErr)
				return
			}
			if !tt.wantErr {
				if gotDomain != tt.wantDomain {
					t.Errorf("parsePCIAddress() gotDomain = %v, want %v", gotDomain, tt.wantDomain)
				}
				if gotBus != tt.wantBus {
					t.Errorf("parsePCIAddress() gotBus = %v, want %v", gotBus, tt.wantBus)
				}
				if gotDevice != tt.wantDevice {
					t.Errorf("parsePCIAddress() gotDevice = %v, want %v", gotDevice, tt.wantDevice)
				}
				if gotFunc != tt.wantFunc {
					t.Errorf("parsePCIAddress() gotFunc = %v, want %v", gotFunc, tt.wantFunc)
				}
			}
		})
	}
}
