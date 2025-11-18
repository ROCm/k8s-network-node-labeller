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

package sysfs

import "testing"

func TestPCIDevice_MatchesFilter(t *testing.T) {
	// Sample PCIDevice for testing - using NewPCIDevice with 0x prefixed strings
	device := NewPCIDevice(
		"/sys/bus/pci/devices/0000:01:00.0",
		"0x1234",
		"0x5678",
		"0xabcd",
		"0xef01",
		"0x020000",
	)

	tests := []struct {
		name     string
		device   PCIDevice
		filter   PCIDevice
		expected bool
	}{
		{
			name:     "Empty filter matches any device",
			device:   device,
			filter:   PCIDevice{},
			expected: true,
		},
		{
			name:   "Exact match on all fields",
			device: device,
			filter: PCIDevice{
				SysFSPath:         "/sys/bus/pci/devices/0000:01:00.0",
				PCIVendorID:       "1234",
				PCIDeviceID:       "5678",
				SubsystemVendorID: "abcd",
				SubsystemDeviceID: "ef01",
				DeviceClass:       "020000",
			},
			expected: true,
		},
		{
			name:   "Exact match with 0x prefixed filter fields",
			device: device,
			filter: PCIDevice{
				SysFSPath:         "/sys/bus/pci/devices/0000:01:00.0",
				PCIVendorID:       "0x1234",
				PCIDeviceID:       "0x5678",
				SubsystemVendorID: "0xabcd",
				SubsystemDeviceID: "0xef01",
				DeviceClass:       "0x020000",
			},
			expected: true,
		},
		{
			name:   "Match on PCIVendorID only",
			device: device,
			filter: PCIDevice{
				PCIVendorID: "1234",
			},
			expected: true,
		},
		{
			name:   "Match on PCIDeviceID only",
			device: device,
			filter: PCIDevice{
				PCIDeviceID: "5678",
			},
			expected: true,
		},
		{
			name:   "Match on SubsystemVendorID only",
			device: device,
			filter: PCIDevice{
				SubsystemVendorID: "abcd",
			},
			expected: true,
		},
		{
			name:   "Match on SubsystemDeviceID only",
			device: device,
			filter: PCIDevice{
				SubsystemDeviceID: "ef01",
			},
			expected: true,
		},
		{
			name:   "Match on DeviceClass only",
			device: device,
			filter: PCIDevice{
				DeviceClass: "020000",
			},
			expected: true,
		},
		{
			name:   "Match on SysFSPath only",
			device: device,
			filter: PCIDevice{
				SysFSPath: "/sys/bus/pci/devices/0000:01:00.0",
			},
			expected: true,
		},
		{
			name:   "Match on multiple fields",
			device: device,
			filter: PCIDevice{
				PCIVendorID: "1234",
				PCIDeviceID: "5678",
				DeviceClass: "020000",
			},
			expected: true,
		},
		{
			name:   "No match - wrong PCIVendorID",
			device: device,
			filter: PCIDevice{
				PCIVendorID: "9999",
			},
			expected: false,
		},
		{
			name:   "No match - wrong PCIDeviceID",
			device: device,
			filter: PCIDevice{
				PCIDeviceID: "9999",
			},
			expected: false,
		},
		{
			name:   "No match - wrong SubsystemVendorID",
			device: device,
			filter: PCIDevice{
				SubsystemVendorID: "9999",
			},
			expected: false,
		},
		{
			name:   "No match - wrong SubsystemDeviceID",
			device: device,
			filter: PCIDevice{
				SubsystemDeviceID: "9999",
			},
			expected: false,
		},
		{
			name:   "No match - wrong DeviceClass",
			device: device,
			filter: PCIDevice{
				DeviceClass: "030000",
			},
			expected: false,
		},
		{
			name:   "No match - wrong SysFSPath",
			device: device,
			filter: PCIDevice{
				SysFSPath: "/sys/bus/pci/devices/0000:02:00.0",
			},
			expected: false,
		},
		{
			name:   "Partial match fails - one field doesn't match",
			device: device,
			filter: PCIDevice{
				PCIVendorID: "1234", // matches
				PCIDeviceID: "9999", // doesn't match
			},
			expected: false,
		},
		{
			name: "Match with different device - all empty fields",
			device: PCIDevice{
				SysFSPath:         "/sys/bus/pci/devices/0000:02:00.0",
				PCIVendorID:       "",
				PCIDeviceID:       "",
				SubsystemVendorID: "",
				SubsystemDeviceID: "",
				DeviceClass:       "",
			},
			filter: PCIDevice{
				SysFSPath: "/sys/bus/pci/devices/0000:02:00.0",
			},
			expected: true,
		},
		{
			name: "No match with empty device fields",
			device: PCIDevice{
				SysFSPath:         "/sys/bus/pci/devices/0000:02:00.0",
				PCIVendorID:       "",
				PCIDeviceID:       "",
				SubsystemVendorID: "",
				SubsystemDeviceID: "",
				DeviceClass:       "",
			},
			filter: PCIDevice{
				PCIVendorID: "1234",
			},
			expected: false,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			result := tt.device.MatchesFilter(tt.filter)
			if result != tt.expected {
				t.Errorf("MatchesFilter() = %v, expected %v", result, tt.expected)
				t.Errorf("Device: %+v", tt.device)
				t.Errorf("Filter: %+v", tt.filter)
			}
		})
	}
}
