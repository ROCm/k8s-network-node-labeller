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

package cleaner

import (
	"context"
	"fmt"
	"log/slog"
	"regexp"

	metav1 "k8s.io/apimachinery/pkg/apis/meta/v1"
	"k8s.io/client-go/kubernetes"
	"k8s.io/client-go/util/retry"
)

// KubernetesCleaner implements Cleaner interface for removing labels from Kubernetes nodes
type KubernetesCleaner struct {
	clientset kubernetes.Interface
	nodeName  string
}

// NewKubernetesCleaner creates a new Kubernetes cleaner with an existing clientset
func NewKubernetesCleaner(clientset kubernetes.Interface, nodeName string) (*KubernetesCleaner, error) {
	if clientset == nil {
		return nil, fmt.Errorf("clientset cannot be nil")
	}
	if nodeName == "" {
		return nil, fmt.Errorf("node name cannot be empty")
	}

	return &KubernetesCleaner{
		clientset: clientset,
		nodeName:  nodeName,
	}, nil
}

// CleanByRegexps removes all labels with keys matching any of the given regex patterns from the Kubernetes node
func (kc *KubernetesCleaner) CleanByRegexps(ctx context.Context, regexps []string) error {
	if len(regexps) == 0 {
		return fmt.Errorf("regexps list cannot be empty")
	}

	// Compile all regex patterns
	compiledRegexps := make([]*regexp.Regexp, len(regexps))
	for i, pattern := range regexps {
		compiled, err := regexp.Compile(pattern)
		if err != nil {
			return fmt.Errorf("failed to compile regex pattern %s: %w", pattern, err)
		}
		compiledRegexps[i] = compiled
	}

	err := retry.OnError(retry.DefaultRetry, func(err error) bool {
		return true
	}, func() error {
		node, err := kc.clientset.CoreV1().Nodes().Get(ctx, kc.nodeName, metav1.GetOptions{})
		if err != nil {
			return fmt.Errorf("failed to get node %s: %w", kc.nodeName, err)
		}

		if node.Labels == nil {
			// Nothing to do
			return nil
		}

		// Remove labels that DO match any of the regex patterns
		var labelsToRemove []string
		for key := range node.Labels {
			for _, regex := range compiledRegexps {
				if regex.MatchString(key) {
					labelsToRemove = append(labelsToRemove, key)
					break
				}
			}
		}

		// Remove the labels
		for _, key := range labelsToRemove {
			delete(node.Labels, key)
		}

		// Only update if there were labels to remove
		if len(labelsToRemove) == 0 {
			return nil
		}

		_, err = kc.clientset.CoreV1().Nodes().Update(ctx, node, metav1.UpdateOptions{})
		if err != nil {
			return fmt.Errorf("failed to update node %s: %w", kc.nodeName, err)
		}
		slog.Debug("Cleaned stale labels from node", "nodeName", kc.nodeName, "removedLabelCount", len(labelsToRemove))

		return nil
	})

	return err
}

// GetName returns the name of the cleaner for logging purposes
func (kc *KubernetesCleaner) GetName() string {
	return "KubernetesCleaner"
}
