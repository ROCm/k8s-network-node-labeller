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

package nodelabeller

import (
	"context"
	"fmt"
	"log/slog"
	"time"

	"github.com/ROCm/k8s-network-node-labeller/internal/cleaner"
	"github.com/ROCm/k8s-network-node-labeller/internal/discoverer"
	"github.com/ROCm/k8s-network-node-labeller/internal/label"
	"github.com/ROCm/k8s-network-node-labeller/internal/publisher"
)

const (
	// DefaultInterval is the default interval for labelling cycles
	DefaultInterval = time.Minute
)

// NodeLabeller coordinates the labelling process
type NodeLabeller struct {
	cleaners    *[]cleaner.Cleaner
	discoverers *[]discoverer.Discoverer
	publishers  *[]publisher.Publisher
}

// NewNodeLabeller creates a new NodeLabeller
func NewNodeLabeller(
	cleaners *[]cleaner.Cleaner,
	discoverers *[]discoverer.Discoverer,
	publishers *[]publisher.Publisher,
) *NodeLabeller {
	return &NodeLabeller{
		cleaners:    cleaners,
		discoverers: discoverers,
		publishers:  publishers,
	}
}

// Run starts the periodic labelling process
func (nl *NodeLabeller) Run(ctx context.Context) error {
	slog.Info("Starting NodeLabeller", "interval", DefaultInterval)

	ticker := time.NewTicker(DefaultInterval)
	defer ticker.Stop()

	// Run once immediately, exit if there's an error in the initial run
	if err := nl.runOnce(ctx); err != nil {
		slog.Error("Error during initial labelling run", "error", err)
		return err
	}

	for {
		select {
		case <-ctx.Done():
			slog.Info("NodeLabeller stopping due to context cancellation")
			return ctx.Err()
		case <-ticker.C:
			if err := nl.runOnce(ctx); err != nil {
				slog.Error("Error during labelling run", "error", err)
				// Continue running even if there's an error
			}
		}
	}
}

// runOnce performs a single labelling cycle
func (nl *NodeLabeller) runOnce(ctx context.Context) error {

	// Step 1: Clean existing labels that match managed label patterns
	regexps := label.DefaultNICPrefixedLabelRegexps
	slog.Debug("Cleaning labels matching managed label regexps")
	for _, cleaner := range *nl.cleaners {
		if err := cleaner.CleanByRegexps(ctx, regexps); err != nil {
			return fmt.Errorf("failed to clean labels by regexps: %w", err)
		}
	}
	slog.Debug("Successfully cleaned labels matching managed label regexps")

	// Step 2: Discover new labels
	var allLabels []label.Label
	for i, disc := range *nl.discoverers {
		slog.Debug("Running discoverer", "discovererIndex", i+1)
		labels, err := disc.Discover()
		if err != nil {
			return fmt.Errorf("error discovering from discoverer %d: %w", i+1, err)
		}
		slog.Debug("Discoverer found labels", "discovererName", disc.GetName(), "discovererIndex", i+1, "labelCount", len(labels))
		allLabels = append(allLabels, labels...)
	}

	slog.Debug("Generated labels total", "totalLabelCount", len(allLabels))

	// Print discovered labels for debugging
	slog.Debug("Discovered labels:")
	label.PrintLabelsDebug(allLabels)

	// Step 3: Publish labels
	if len(allLabels) == 0 {
		slog.Warn("No labels to publish")
		return nil
	}

	for i, pub := range *nl.publishers {
		slog.Debug("Publishing labels with publisher", "labelCount", len(allLabels), "publisherName", pub.GetName())
		if err := pub.Publish(ctx, allLabels); err != nil {
			return fmt.Errorf("failed to publish labels with publisher %d: %w", i+1, err)
		}
		slog.Debug("Successfully published labels with publisher", "labelCount", len(allLabels), "publisherName", pub.GetName())
	}

	slog.Debug("Labelling cycle completed")
	return nil
}

// Stop gracefully stops the NodeLabeller
func (nl *NodeLabeller) Stop() {
	slog.Info("Cleaning up managed labels when stopping NodeLabeller")
	regexps := label.DefaultNICPrefixedLabelRegexps
	for _, cleaner := range *nl.cleaners {
		if err := cleaner.CleanByRegexps(context.TODO(), regexps); err != nil {
			slog.Error("failed to clean labels by regexps", "error", err)
		}
	}
	slog.Info("Successfully cleaned up managed labels")
	slog.Info("NodeLabeller stopped")
}
