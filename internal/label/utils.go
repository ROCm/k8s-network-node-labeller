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

import (
	"log/slog"
	"regexp"
	"strings"

	"github.com/ROCm/k8s-network-node-labeller/internal/utils"
)

// NormalizeString normalizes a string by converting to be able to use it as a label key or value.
func NormalizeString(s string) string {
	// Convert to lower case
	s = strings.ToLower(s)
	// Replace spaces and underscores with dashes
	s = strings.ReplaceAll(s, " ", "-")
	s = strings.ReplaceAll(s, "_", "-")
	// Remove any non-alphanumeric characters except dashes and dots
	re := regexp.MustCompile(`[^a-z0-9-.]`)
	s = re.ReplaceAllString(s, "")
	return s
}

// HashLabels computes a hash of the given labels using chain hashing
func HashLabels(labels []Label) string {
	// Convert labels to strings
	labelStrings := make([]string, len(labels))
	for i, label := range labels {
		labelStrings[i] = label.String()
	}

	// Use utils.HashArray for the actual hashing
	return utils.HashArray(labelStrings)
}

// PrintLabels prints the given labels in a structured format at the specified log level
func PrintLabels(labels []Label, level slog.Level) {
	if len(labels) == 0 {
		slog.Log(nil, level, "No labels to print")
		return
	}

	for _, l := range labels {
		slog.Log(nil, level, "Label", "key", l.Key, "value", l.Value)
	}
}

// PrintLabelsInfo prints the given labels at Info log level
func PrintLabelsInfo(labels []Label) {
	PrintLabels(labels, slog.LevelInfo)
}

// PrintLabelsDebug prints the given labels at Debug log level
func PrintLabelsDebug(labels []Label) {
	PrintLabels(labels, slog.LevelDebug)
}
