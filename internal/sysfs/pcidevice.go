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

import (
	"fmt"
	"strings"
)

// PCIDevice represents a PCI device with its identifiers and sysfs path
type PCIDevice struct {
	SysFSPath         string
	PCIVendorID       string
	PCIDeviceID       string
	SubsystemVendorID string
	SubsystemDeviceID string
	DeviceClass       string
}

// NewPCIDevice creates a new PCIDevice with sanitized field values
// It removes "0x" prefixes and trims whitespace from all string fields
func NewPCIDevice(sysFSPath, vendorID, deviceID, subsystemVendorID, subsystemDeviceID, deviceClass string) PCIDevice {
	return PCIDevice{
		SysFSPath:         sysFSPath,
		PCIVendorID:       sanitizedPCIField(vendorID),
		PCIDeviceID:       sanitizedPCIField(deviceID),
		SubsystemVendorID: sanitizedPCIField(subsystemVendorID),
		SubsystemDeviceID: sanitizedPCIField(subsystemDeviceID),
		DeviceClass:       sanitizedPCIField(deviceClass),
	}
}

// sanitizedPCIField removes "0x" prefix and trims whitespace from a PCI field value
func sanitizedPCIField(value string) string {
	return strings.TrimSpace(strings.TrimPrefix(value, "0x"))
}

// MatchesFilter checks if a device matches the given filter
// Empty fields in the filter are ignored. Filter fields are sanitized before comparison.
func (d PCIDevice) MatchesFilter(filter PCIDevice) bool {
	if filter.PCIVendorID != "" && d.PCIVendorID != sanitizedPCIField(filter.PCIVendorID) {
		return false
	}

	if filter.PCIDeviceID != "" && d.PCIDeviceID != sanitizedPCIField(filter.PCIDeviceID) {
		return false
	}

	if filter.SubsystemVendorID != "" && d.SubsystemVendorID != sanitizedPCIField(filter.SubsystemVendorID) {
		return false
	}

	if filter.SubsystemDeviceID != "" && d.SubsystemDeviceID != sanitizedPCIField(filter.SubsystemDeviceID) {
		return false
	}

	if filter.DeviceClass != "" && d.DeviceClass != sanitizedPCIField(filter.DeviceClass) {
		return false
	}

	if filter.SysFSPath != "" && d.SysFSPath != filter.SysFSPath {
		return false
	}

	return true
}

// String returns a formatted string representation of the PCI device
func (d PCIDevice) String() string {
	return fmt.Sprintf("PCIDevice{SysFSPath: %s, VendorID: %s, DeviceID: %s, SubsysVendorID: %s, SubsysDeviceID: %s, DeviceClass: %s}",
		d.SysFSPath, d.PCIVendorID, d.PCIDeviceID, d.SubsystemVendorID, d.SubsystemDeviceID, d.DeviceClass)
}
