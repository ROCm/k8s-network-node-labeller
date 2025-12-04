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

package publisher

import (
	"context"
	"fmt"
	"log/slog"

	metav1 "k8s.io/apimachinery/pkg/apis/meta/v1"
	"k8s.io/client-go/kubernetes"

	"github.com/ROCm/k8s-network-node-labeller/internal/label"
	"k8s.io/client-go/util/retry"
)

// KubernetesPublisher implements Publisher interface for writing labels to Kubernetes nodes
type KubernetesPublisher struct {
	clientset          kubernetes.Interface
	nodeName           string
	previousLabelsHash string
}

// NewKubernetesPublisher creates a new Kubernetes publisher with an existing clientset
func NewKubernetesPublisher(clientset kubernetes.Interface, nodeName string) (*KubernetesPublisher, error) {
	if clientset == nil {
		return nil, fmt.Errorf("clientset cannot be nil")
	}
	if nodeName == "" {
		return nil, fmt.Errorf("node name cannot be empty")
	}

	return &KubernetesPublisher{
		clientset:          clientset,
		nodeName:           nodeName,
		previousLabelsHash: label.HashLabels([]label.Label{}),
	}, nil
}

func (kp *KubernetesPublisher) logLabelsChanged(labels []label.Label) {
	// Compute hash of current labels
	currentHash := label.HashLabels(labels)

	// Check if hash has changed
	if currentHash != kp.previousLabelsHash {
		slog.Debug("Found new labels to Publish", "node", kp.nodeName, "previousHash", kp.previousLabelsHash, "currentHash", currentHash)
		slog.Info("Publishing Labels:", "node", kp.nodeName)
		label.PrintLabelsInfo(labels)
		kp.previousLabelsHash = currentHash
	} else {
		slog.Debug("No new labels to Publish, Hash unchanged", "node", kp.nodeName, "previousHash", kp.previousLabelsHash, "currentHash", currentHash)
	}
}

// Publish applies labels to the Kubernetes node
func (kp *KubernetesPublisher) Publish(ctx context.Context, labels []label.Label) error {
	if len(labels) == 0 {
		return nil
	}

	// Log labels if they have changed
	kp.logLabelsChanged(labels)

	err := retry.OnError(retry.DefaultRetry, func(err error) bool {
		return true
	}, func() error {
		// Get the node
		node, err := kp.clientset.CoreV1().Nodes().Get(ctx, kp.nodeName, metav1.GetOptions{})
		if err != nil {
			return fmt.Errorf("failed to get node %s: %w", kp.nodeName, err)
		}

		// Update node labels
		if node.Labels == nil {
			node.Labels = make(map[string]string)
		}

		// Add/update labels
		for _, l := range labels {
			node.Labels[l.Key] = l.Value
		}

		_, err = kp.clientset.CoreV1().Nodes().Update(ctx, node, metav1.UpdateOptions{})
		if err != nil {
			return fmt.Errorf("failed to update node %s: %w", kp.nodeName, err)
		}

		return nil
	})

	return err
}

// GetName returns the name of the publisher for logging purposes
func (kp *KubernetesPublisher) GetName() string {
	return "KubernetesPublisher"
}
