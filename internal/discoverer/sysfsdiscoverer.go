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

package discoverer

import (
	"fmt"
	"regexp"
	"strings"

	"github.com/ROCm/k8s-network-node-labeller/internal/label"
	"github.com/ROCm/k8s-network-node-labeller/internal/sysfs"
)

const (
	// DefaultSysfsBasePath is the default base path for sysfs network interface discovery
	DefaultSysfsBasePath = "/sys"
)

// SysfsDiscoverer implements the Discoverer interface using sysfs
type SysfsDiscoverer struct {
	client *sysfs.SysfsClient
}

// NewSysfsDiscoverer creates a new SysfsDiscoverer instance
func NewSysfsDiscoverer(client *sysfs.SysfsClient) *SysfsDiscoverer {
	return &SysfsDiscoverer{
		client: client,
	}
}

// Discover discovers network interface features and returns a list of labels
func (d *SysfsDiscoverer) Discover() ([]label.Label, error) {
	// Get NICs from sysfs
	nics, err := d.client.GetNICs()
	if err != nil {
		return nil, fmt.Errorf("failed to get NICs: %w", err)
	}

	var labels []label.Label

	// Add card name labels
	cardNameLabels := d.generateCardNameLabels(nics)
	labels = append(labels, cardNameLabels...)

	// Add driver info labels
	driverInfoLabels := d.generateDriverInfoLabels(nics)
	labels = append(labels, driverInfoLabels...)

	return labels, nil
}

// generateCardNameLabels generates labels for card product names.
func (d *SysfsDiscoverer) generateCardNameLabels(nics []sysfs.NIC) []label.Label {
	var labels []label.Label

	// Group product names by SKU
	skuToProductName := make(map[string]string)
	for _, nic := range nics {
		sku := label.NormalizeString(nic.SKU)
		productName := strings.TrimSpace(nic.ProductName)
		re := regexp.MustCompile(`[^A-Za-z0-9-.]`)
		productName = re.ReplaceAllString(productName, "_")
		skuToProductName[sku] = productName
	}

	// Add labels for each SKU
	for sku, productName := range skuToProductName {
		var key string
		if len(skuToProductName) == 1 {
			key = label.DefaultNICPrefixedKey("product-name")
		} else {
			key = label.DefaultNICPrefixedKey(sku + ".product-name")
		}

		nameLabel := label.NewLabel(key, productName)
		labels = append(labels, nameLabel)
	}

	return labels
}

// generateDriverInfoLabels generates labels for driver information.
func (d *SysfsDiscoverer) generateDriverInfoLabels(nics []sysfs.NIC) []label.Label {
	var labels []label.Label

	skuToDriverName := make(map[string]string)
	skuToDriverVersion := make(map[string]string)

	// Group driver names and versions by SKU
	for _, nic := range nics {
		sku := label.NormalizeString(nic.SKU)
		skuToDriverName[sku] = label.NormalizeString(nic.DriverName)
		skuToDriverVersion[sku] = label.NormalizeString(nic.DriverVersion)
	}

	// Add labels for each SKU
	for sku := range skuToDriverName {
		driverName := skuToDriverName[sku]
		driverVersion := skuToDriverVersion[sku]

		var nameKey, versionKey string
		if len(skuToDriverName) == 1 && len(skuToDriverVersion) == 1 {
			nameKey = label.DefaultNICPrefixedKey("driver-name")
			versionKey = label.DefaultNICPrefixedKey("driver-version")
		} else {
			nameKey = label.DefaultNICPrefixedKey(sku + ".driver-name")
			versionKey = label.DefaultNICPrefixedKey(sku + ".driver-version")
		}

		nameLabel := label.NewLabel(nameKey, driverName)
		versionLabel := label.NewLabel(versionKey, driverVersion)
		labels = append(labels, nameLabel, versionLabel)
	}

	return labels
}

// GetName returns the name of the discoverer for logging purposes
func (d *SysfsDiscoverer) GetName() string {
	return "SysfsDiscoverer"
}
