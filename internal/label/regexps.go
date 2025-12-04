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

package label

import "regexp"

var (
	// ManagedLabelRegexps is a list of regex patterns for labels managed by the k8s-network-node-labeller
	managedLabelRegexps = []string{
		// Count labels
		`count`,
		`.*\.count`, // For SKU specific count labels

		// Product labels
		`product-name`,
		`.*\.product-name`, // For SKU specific product labels

		// Firmware version labels
		`firmware-version`,
		`.*\.firmware-version`, // For SKU specific firmware version labels

		// Driver info labels
		`driver-version`,
		`.*\.driver-version`,
		`driver-name`,
		`.*\.driver-name`,

		// Port count labels
		`port-count`,
		`.*\.port-count`, // For SKU specific port count labels

		// Port speed labels
		`port[0-9]*-speed`,
		`.*\.port[0-9]*-speed`, // For SKU specific port speed labels
	}
	DefaultNICPrefixedLabelRegexps = getDefaultNICPrefixedManagedLabelRegexps()
)

// GetDefaultNICPrefixedManagedLabelRegexps returns the default prefixed regex patterns for NIC labels managed by the k8s-network-node-labeller
func getDefaultNICPrefixedManagedLabelRegexps() []string {
	prefixedRegexps := make([]string, len(managedLabelRegexps))
	for i, regex := range managedLabelRegexps {
		prefixedRegexps[i] = DefaultNICPrefixedKey(regex)
	}
	return prefixedRegexps
}

// IsManagedLabel checks if a label key matches any of the managed label regex patterns
func IsManagedLabel(key string) bool {
	for _, regex := range DefaultNICPrefixedLabelRegexps {
		if matched, _ := regexp.MatchString(regex, key); matched {
			return true
		}
	}
	return false
}
