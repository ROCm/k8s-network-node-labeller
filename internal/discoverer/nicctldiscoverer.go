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
	"log/slog"
	"strconv"

	"github.com/ROCm/k8s-network-node-labeller/internal/label"
	"github.com/ROCm/k8s-network-node-labeller/internal/nicctl"
)

const (
	// NicctlBinaryPath is the path to the nicctl binary
	NicctlBinaryPath = "/usr/sbin/nicctl"
)

// NicctlDiscoverer implements the Discoverer interface using nicctl binary
type NicctlDiscoverer struct {
	client nicctl.NicctlClient
}

// NewNicctlDiscoverer creates a new NicctlDiscoverer instance
func NewNicctlDiscoverer(client nicctl.NicctlClient) *NicctlDiscoverer {
	return &NicctlDiscoverer{
		client: client,
	}
}

// Discover discovers network interface features and returns a list of labels
func (d *NicctlDiscoverer) Discover() ([]label.Label, error) {
	// Get fresh card information with ports merged
	cards, err := d.client.GetCardsWithPorts()
	if err != nil {
		return nil, fmt.Errorf("failed to get cards: %w", err)
	}

	var labels []label.Label

	// Add card count labels
	cardCountLabels := d.generateCardCountLabelsAll(cards)
	labels = append(labels, cardCountLabels...)

	// Add firmware version labels
	firmwareVersionLabels := d.generateFirmwareVersionLabelsAll(cards)
	labels = append(labels, firmwareVersionLabels...)

	// Add port count labels
	portCountLabels := d.generatePortCountLabelsAll(cards)
	labels = append(labels, portCountLabels...)

	// Add port speed labels
	portSpeedLabels := d.generatePortSpeedLabelsAll(cards)
	labels = append(labels, portSpeedLabels...)

	profiles, err := d.client.GetCardProfiles()
	if err != nil {
		slog.Warn("Failed to get card profiles, skipping profile labels", "error", err)
	} else {
		profileLabels := d.generateProfileLabelsAll(cards, profiles)
		labels = append(labels, profileLabels...)
	}

	return labels, nil
}

// generateCardCountLabelsAll generates labels for card count information.
func (d *NicctlDiscoverer) generateCardCountLabelsAll(cards []nicctl.NIC) []label.Label {
	var labels []label.Label

	// Add label for total card count
	totalCount := len(cards)
	totalCountLabel := label.NewLabel(
		label.DefaultNICPrefixedKey("count"),
		strconv.Itoa(totalCount),
	)
	labels = append(labels, totalCountLabel)

	// Count cards by SKU
	skuCounts := make(map[string]int)
	for _, nic := range cards {
		sku := label.NormalizeString(nic.SKU)
		skuCounts[sku]++
	}

	// Early return if only one SKU is present
	if len(skuCounts) == 1 {
		return labels
	}

	// Add labels for each SKU count
	for sku, count := range skuCounts {
		skuCountLabel := label.NewLabel(
			label.DefaultNICPrefixedKey(sku+".count"),
			strconv.Itoa(count),
		)
		labels = append(labels, skuCountLabel)
	}

	return labels
}

// generateFirmwareVersionLabelsAll generates labels for firmware version information.
func (d *NicctlDiscoverer) generateFirmwareVersionLabelsAll(cards []nicctl.NIC) []label.Label {
	var labels []label.Label

	// Group firmware versions by SKU
	skuToFirmwareVersion := make(map[string]string)
	for _, nic := range cards {
		normalizedSKU := label.NormalizeString(nic.SKU)
		skuToFirmwareVersion[normalizedSKU] = label.NormalizeString(nic.FirmwareVersion)
	}

	// Add labels for each SKU
	for sku, firmwareVersion := range skuToFirmwareVersion {
		var key string
		if len(skuToFirmwareVersion) == 1 {
			key = label.DefaultNICPrefixedKey("firmware-version")
		} else {
			key = label.DefaultNICPrefixedKey(sku + ".firmware-version")
		}

		firmwareLabel := label.NewLabel(key, firmwareVersion)
		labels = append(labels, firmwareLabel)
	}

	return labels
}

// generateProfileLabelsAll generates labels for NIC profile information.
// If NICs of the same SKU have different profiles, the value is set to
// "misconfig-mixed-profiles" to flag the misconfiguration.
func (d *NicctlDiscoverer) generateProfileLabelsAll(cards []nicctl.NIC, profiles map[string]string) []label.Label {
	var labels []label.Label

	skuToProfile := make(map[string]string)
	for _, nic := range cards {
		normalizedSKU := label.NormalizeString(nic.SKU)
		profile := profiles[nic.ID]
		if existing, ok := skuToProfile[normalizedSKU]; !ok {
			skuToProfile[normalizedSKU] = profile
		} else if existing != profile && profile != "" {
			skuToProfile[normalizedSKU] = "misconfig-mixed-profiles"
		}
	}

	for sku, profileName := range skuToProfile {
		if profileName == "" {
			continue
		}

		var key string
		if len(skuToProfile) == 1 {
			key = label.DefaultNICPrefixedKey("profile")
		} else {
			key = label.DefaultNICPrefixedKey(sku + ".profile")
		}

		labels = append(labels, label.NewLabel(key, profileName))
	}

	return labels
}

// generatePortCountLabelsAll generates labels for port count information.
func (d *NicctlDiscoverer) generatePortCountLabelsAll(cards []nicctl.NIC) []label.Label {
	var labels []label.Label

	// Group port counts by SKU
	skuToPortCount := make(map[string]int)
	for _, nic := range cards {
		normalizedSKU := label.NormalizeString(nic.SKU)
		skuToPortCount[normalizedSKU] += len(nic.Ports)
	}

	// Add labels for each SKU
	for sku, portCount := range skuToPortCount {
		var key string
		if len(skuToPortCount) == 1 {
			key = label.DefaultNICPrefixedKey("port-count")
		} else {
			key = label.DefaultNICPrefixedKey(sku + ".port-count")
		}

		portCountLabel := label.NewLabel(key, strconv.Itoa(portCount))
		labels = append(labels, portCountLabel)
	}

	return labels
}

// generatePortSpeedLabelsAll generates labels for port speed information.
func (d *NicctlDiscoverer) generatePortSpeedLabelsAll(cards []nicctl.NIC) []label.Label {
	var labels []label.Label

	// SKUToPort maps SKU to a list of ports
	skuToPorts := make(map[string][]nicctl.Port)
	for _, nic := range cards {
		normalizedSKU := label.NormalizeString(nic.SKU)
		skuToPorts[normalizedSKU] = nic.Ports
	}

	// Process each SKU and its ports
	for sku, ports := range skuToPorts {
		skuPrefix := ""
		if len(skuToPorts) > 1 {
			skuPrefix = sku + "."
		}

		// Check if all ports have same speed
		allSame := true
		speed := ""
		for _, p := range ports {
			if speed == "" {
				speed = p.Spec.PortSpeed
			} else if p.Spec.PortSpeed != speed {
				allSame = false
				break
			}
		}

		// Generate labels based on port speed
		for i, p := range ports {
			var key string

			if allSame {
				key = label.DefaultNICPrefixedKey(skuPrefix + "port-speed")
			} else {
				key = label.DefaultNICPrefixedKey(fmt.Sprintf("%sport%d-speed", skuPrefix, i))
			}

			portSpeedLabel := label.NewLabel(key, p.Spec.PortSpeed)
			labels = append(labels, portSpeedLabel)
		}
	}

	return labels
}

// GetName returns the name of the discoverer for logging purposes
func (d *NicctlDiscoverer) GetName() string {
	return "NicctlDiscoverer"
}
