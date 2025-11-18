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
	"fmt"
	"os"
	"path/filepath"
	"regexp"

	"gopkg.in/yaml.v3"
)

// TestSysfsConfig represents the YAML structure for test sysfs configuration file
type TestSysfsConfig struct {
	Devices []TestPCIDevice `yaml:"devices"`
	Drivers []TestDriver    `yaml:"drivers"`
}

// TestPCIDevice represents a PCI device in the test configuration file
type TestPCIDevice struct {
	ID                string `yaml:"id"`
	PCIAddress        string `yaml:"pci_address"`    // e.g., "0000:05:00.0"
	ParentAddress     string `yaml:"parent_address"` // e.g., "0000:00:02.0" (optional)
	PCIVendorID       string `yaml:"pci_vendor_id"`
	PCIDeviceID       string `yaml:"pci_device_id"`
	SubsystemVendorID string `yaml:"subsystem_vendor_id"`
	SubsystemDeviceID string `yaml:"subsystem_device_id"`
	DeviceClass       string `yaml:"device_class"`
	DriverName        string `yaml:"driver"`
}

// TestDriver represents a driver in the test configuration file
type TestDriver struct {
	Name    string `yaml:"name"`
	Version string `yaml:"version"`
}

// parsePCIAddress extracts domain, bus, device, and function from PCI address
func parsePCIAddress(address string) (domain, bus, device, function string, err error) {
	// Parse address like "0000:05:00.0"
	re := regexp.MustCompile(`^([0-9a-fA-F]{4}):([0-9a-fA-F]{2}):([0-9a-fA-F]{2})\.([0-9a-fA-F])$`)
	matches := re.FindStringSubmatch(address)
	if len(matches) != 5 {
		return "", "", "", "", fmt.Errorf("invalid PCI address format: %s", address)
	}
	return matches[1], matches[2], matches[3], matches[4], nil
}

// validateYamlConfig validates the consistency of the YAML configuration
func validateYamlConfig(config *TestSysfsConfig) error {
	// Check for duplicate PCI addresses
	addressMap := make(map[string]string) // pci_address -> device ID

	for _, device := range config.Devices {
		// Validate PCI address format
		if device.PCIAddress == "" {
			return fmt.Errorf("device %s has empty pci_address", device.ID)
		}

		_, _, _, _, err := parsePCIAddress(device.PCIAddress)
		if err != nil {
			return fmt.Errorf("device %s has invalid pci_address: %w", device.ID, err)
		}

		// Check for duplicate addresses
		if existingID, exists := addressMap[device.PCIAddress]; exists {
			return fmt.Errorf("duplicate PCI address %s found in devices %s and %s",
				device.PCIAddress, existingID, device.ID)
		}
		addressMap[device.PCIAddress] = device.ID
	}

	// Validate parent-child relationships
	for _, device := range config.Devices {
		if device.ParentAddress != "" {
			// Validate parent address format
			_, _, _, _, err := parsePCIAddress(device.ParentAddress)
			if err != nil {
				return fmt.Errorf("device %s has invalid parent_address: %w", device.ID, err)
			}

			// Check if parent device exists
			parentExists := false
			for _, parentDevice := range config.Devices {
				if parentDevice.PCIAddress == device.ParentAddress {
					parentExists = true
					break
				}
			}
			if !parentExists {
				return fmt.Errorf("device %s references non-existent parent device %s",
					device.ID, device.ParentAddress)
			}
		}
	}

	return nil
}

// getDeviceHierarchyPath constructs the full device path in /sys/devices
func getDeviceHierarchyPath(baseDir string, device TestPCIDevice) (string, error) {
	domain, bus, _, _, err := parsePCIAddress(device.PCIAddress)
	if err != nil {
		return "", err
	}

	// Start with the PCI bus path
	busDir := fmt.Sprintf("pci%s:%c0", domain, bus[0])
	basePath := filepath.Join(baseDir, "devices", busDir)

	// If no parent, this is a root device
	if device.ParentAddress == "" {
		return filepath.Join(basePath, device.PCIAddress), nil
	}

	// If there's a parent, construct the nested path
	// For now, we'll assume single-level nesting, but this can be extended
	_, _, _, _, err = parsePCIAddress(device.ParentAddress)
	if err != nil {
		return "", fmt.Errorf("invalid parent PCI address: %w", err)
	}

	parentDevicePath := filepath.Join(basePath, device.ParentAddress)

	// Child device path includes parent hierarchy
	childPath := filepath.Join(parentDevicePath, device.PCIAddress)
	return childPath, nil
}

// CreateFakeSysfs creates a fake sysfs directory structure based on the provided YAML configuration
func CreateFakeSysfs(baseDir string, config TestSysfsConfig) error {
	// Create base sysfs structure
	busPciDevicesDir := filepath.Join(baseDir, "bus", "pci", "devices")
	driversDir := filepath.Join(baseDir, "drivers")

	if err := os.MkdirAll(busPciDevicesDir, 0755); err != nil {
		return fmt.Errorf("failed to create devices directory: %w", err)
	}
	if err := os.MkdirAll(driversDir, 0755); err != nil {
		return fmt.Errorf("failed to create drivers directory: %w", err)
	}

	// Create /sys/devices base directory
	sysDevicesDir := filepath.Join(baseDir, "devices")
	if err := os.MkdirAll(sysDevicesDir, 0755); err != nil {
		return fmt.Errorf("failed to create sys/devices directory: %w", err)
	}

	// Create drivers first
	driverPaths := make(map[string]string)
	for _, driver := range config.Drivers {
		driverPath := filepath.Join(driversDir, driver.Name)
		if err := os.MkdirAll(driverPath, 0755); err != nil {
			return fmt.Errorf("failed to create driver directory for %s: %w", driver.Name, err)
		}

		// Only create version file if version is specified in config
		if driver.Version != "" {
			// Create module directory and version file
			moduleDir := filepath.Join(driverPath, "module")
			if err := os.MkdirAll(moduleDir, 0755); err != nil {
				return fmt.Errorf("failed to create module directory for %s: %w", driver.Name, err)
			}

			versionFile := filepath.Join(moduleDir, "version")
			if err := os.WriteFile(versionFile, []byte(driver.Version+"\n"), 0644); err != nil {
				return fmt.Errorf("failed to create version file for %s: %w", driver.Name, err)
			}
		}

		driverPaths[driver.Name] = driverPath
	}

	// Track created device paths for hierarchy validation
	deviceHierarchyPaths := make(map[string]string)

	// Create devices in hierarchy order (parents first)
	sortedDevices := sortDevicesByHierarchy(config.Devices)

	for _, device := range sortedDevices {
		// Get the real device path in /sys/devices/pci...
		realDevicePath, err := getDeviceHierarchyPath(baseDir, device)
		if err != nil {
			return fmt.Errorf("failed to get device hierarchy path for %s: %w", device.ID, err)
		}

		// Create the real device directory
		if err := os.MkdirAll(realDevicePath, 0755); err != nil {
			return fmt.Errorf("failed to create real device directory for %s: %w", device.ID, err)
		}

		deviceHierarchyPaths[device.PCIAddress] = realDevicePath

		// Create device attribute files in the real device directory
		deviceFiles := map[string]string{
			"vendor":           device.PCIVendorID,
			"device":           device.PCIDeviceID,
			"subsystem_vendor": device.SubsystemVendorID,
			"subsystem_device": device.SubsystemDeviceID,
			"class":            device.DeviceClass,
		}

		for filename, content := range deviceFiles {
			if content != "" {
				filePath := filepath.Join(realDevicePath, filename)
				if err := os.WriteFile(filePath, []byte(content+"\n"), 0644); err != nil {
					return fmt.Errorf("failed to create device file %s for %s: %w", filename, device.ID, err)
				}
			}
		}

		// Create driver symlink in the real device directory if driver is specified
		if device.DriverName != "" {
			driverPath, exists := driverPaths[device.DriverName]
			if !exists {
				return fmt.Errorf("driver %s not found for device %s", device.DriverName, device.ID)
			}

			driverSymlink := filepath.Join(realDevicePath, "driver")
			if err := os.Symlink(driverPath, driverSymlink); err != nil {
				return fmt.Errorf("failed to create driver symlink for %s: %w", device.ID, err)
			}
		}

		// Create symlink in /sys/bus/pci/devices pointing to the real device
		busDeviceSymlink := filepath.Join(busPciDevicesDir, device.PCIAddress)
		if err := os.Symlink(realDevicePath, busDeviceSymlink); err != nil {
			return fmt.Errorf("failed to create bus device symlink for %s: %w", device.ID, err)
		}
	}

	return nil
}

// sortDevicesByHierarchy sorts devices so parents come before children
func sortDevicesByHierarchy(devices []TestPCIDevice) []TestPCIDevice {
	var sorted []TestPCIDevice
	var remaining []TestPCIDevice

	// First pass: add devices without parents (root devices)
	for _, device := range devices {
		if device.ParentAddress == "" {
			sorted = append(sorted, device)
		} else {
			remaining = append(remaining, device)
		}
	}

	// Additional passes: add devices whose parents are already in sorted list
	for len(remaining) > 0 {
		var stillRemaining []TestPCIDevice
		addedInThisPass := false

		for _, device := range remaining {
			parentExists := false
			for _, sortedDevice := range sorted {
				if sortedDevice.PCIAddress == device.ParentAddress {
					parentExists = true
					break
				}
			}

			if parentExists {
				sorted = append(sorted, device)
				addedInThisPass = true
			} else {
				stillRemaining = append(stillRemaining, device)
			}
		}

		remaining = stillRemaining

		// Prevent infinite loop if there are circular dependencies
		if !addedInThisPass {
			// Add remaining devices anyway (orphaned devices)
			sorted = append(sorted, remaining...)
			break
		}
	}

	return sorted
}

// printDirectoryTree prints the directory structure recursively without following symlinks
func printDirectoryTree(dirPath string, prefix string) error {
	entries, err := os.ReadDir(dirPath)
	if err != nil {
		return err
	}

	for i, entry := range entries {
		isLast := i == len(entries)-1

		// Choose the appropriate tree characters
		var currentPrefix, nextPrefix string
		if isLast {
			currentPrefix = prefix + "└── "
			nextPrefix = prefix + "    "
		} else {
			currentPrefix = prefix + "├── "
			nextPrefix = prefix + "│   "
		}

		// Get file info to check if it's a symlink
		fullPath := filepath.Join(dirPath, entry.Name())
		info, err := os.Lstat(fullPath) // Use Lstat to not follow symlinks
		if err != nil {
			fmt.Printf("%s%s (error: %v)\n", currentPrefix, entry.Name(), err)
			continue
		}

		// Print the entry with appropriate indicator
		if info.Mode()&os.ModeSymlink != 0 {
			// It's a symlink - show target but don't recurse
			target, err := os.Readlink(fullPath)
			if err != nil {
				fmt.Printf("%s%s -> (error reading link: %v)\n", currentPrefix, entry.Name(), err)
			} else {
				fmt.Printf("%s%s -> %s\n", currentPrefix, entry.Name(), target)
			}
		} else if info.IsDir() {
			// It's a directory - print and recurse
			fmt.Printf("%s%s/\n", currentPrefix, entry.Name())
			if err := printDirectoryTree(fullPath, nextPrefix); err != nil {
				fmt.Printf("%s(error reading subdirectory: %v)\n", nextPrefix, err)
			}
		} else {
			// It's a regular file - show size
			fmt.Printf("%s%s (%d bytes)\n", currentPrefix, entry.Name(), info.Size())
		}
	}

	return nil
}

// CreateFakeSysfsFromFile creates a fake sysfs directory structure from a YAML file
func CreateFakeSysfsFromFile(baseDir string, yamlFilePath string) error {
	yamlData, err := os.ReadFile(yamlFilePath)
	if err != nil {
		return fmt.Errorf("failed to read YAML file %s: %w", yamlFilePath, err)
	}

	var config TestSysfsConfig
	if err := yaml.Unmarshal([]byte(yamlData), &config); err != nil {
		return fmt.Errorf("failed to parse YAML config: %w", err)
	}

	// Validate the configuration before processing
	if err := validateYamlConfig(&config); err != nil {
		return fmt.Errorf("YAML configuration validation failed: %w", err)
	}

	err = CreateFakeSysfs(baseDir, config)
	if err != nil {
		return err
	}

	return nil
}
