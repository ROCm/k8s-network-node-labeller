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
	"bytes"
	"context"
	"fmt"
	"log/slog"
	"strings"
	"testing"

	v1 "k8s.io/api/core/v1"
	apierrors "k8s.io/apimachinery/pkg/api/errors"
	metav1 "k8s.io/apimachinery/pkg/apis/meta/v1"
	"k8s.io/apimachinery/pkg/runtime"
	"k8s.io/client-go/kubernetes"
	"k8s.io/client-go/kubernetes/fake"
	k8stesting "k8s.io/client-go/testing"

	"github.com/ROCm/k8s-network-node-labeller/internal/label"
)

func TestNewKubernetesPublisher(t *testing.T) {
	tests := []struct {
		name        string
		clientset   kubernetes.Interface
		nodeName    string
		expectError bool
		errorMsg    string
	}{
		{
			name:        "valid parameters",
			clientset:   fake.NewSimpleClientset(),
			nodeName:    "test-node",
			expectError: false,
		},
		{
			name:        "nil clientset",
			clientset:   nil,
			nodeName:    "test-node",
			expectError: true,
			errorMsg:    "clientset cannot be nil",
		},
		{
			name:        "empty node name",
			clientset:   fake.NewSimpleClientset(),
			nodeName:    "",
			expectError: true,
			errorMsg:    "node name cannot be empty",
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			publisher, err := NewKubernetesPublisher(tt.clientset, tt.nodeName)

			if tt.expectError {
				if err == nil {
					t.Errorf("expected error but got none")
					return
				}
				if err.Error() != tt.errorMsg {
					t.Errorf("expected error message '%s', got '%s'", tt.errorMsg, err.Error())
				}
				if publisher != nil {
					t.Errorf("expected nil publisher when error occurs")
				}
			} else {
				if err != nil {
					t.Errorf("unexpected error: %v", err)
					return
				}
				if publisher == nil {
					t.Errorf("expected non-nil publisher")
					return
				}
				if publisher.nodeName != tt.nodeName {
					t.Errorf("expected node name '%s', got '%s'", tt.nodeName, publisher.nodeName)
				}
				// Verify that previousLabelsHash is initialized to hash of empty labels
				expectedInitialHash := label.HashLabels([]label.Label{})
				if publisher.previousLabelsHash != expectedInitialHash {
					t.Errorf("expected previousLabelsHash to be initialized to '%s', got '%s'", expectedInitialHash, publisher.previousLabelsHash)
				}
			}
		})
	}
}

func TestKubernetesPublisher_Publish(t *testing.T) {
	tests := []struct {
		name           string
		nodeName       string
		existingNode   *v1.Node
		labels         []label.Label
		expectError    bool
		errorContains  string
		expectedLabels map[string]string
	}{
		{
			name:     "publish labels to existing node",
			nodeName: "test-node",
			existingNode: &v1.Node{
				ObjectMeta: metav1.ObjectMeta{
					Name: "test-node",
					Labels: map[string]string{
						"existing-label": "existing-value",
					},
				},
			},
			labels: []label.Label{
				{Key: "amd.com/nic.product-name", Value: "Test_NIC_1"},
				{Key: "amd.com/nic.port-speed", Value: "100G"},
			},
			expectError: false,
			expectedLabels: map[string]string{
				"existing-label":           "existing-value",
				"amd.com/nic.product-name": "Test_NIC_1",
				"amd.com/nic.port-speed":   "100G",
			},
		},
		{
			name:     "publish labels to node with nil labels",
			nodeName: "test-node",
			existingNode: &v1.Node{
				ObjectMeta: metav1.ObjectMeta{
					Name: "test-node",
				},
			},
			labels: []label.Label{
				{Key: "amd.com/nic.driver-name", Value: "ionic"},
			},
			expectError: false,
			expectedLabels: map[string]string{
				"amd.com/nic.driver-name": "ionic",
			},
		},
		{
			name:     "publish empty labels list",
			nodeName: "test-node",
			existingNode: &v1.Node{
				ObjectMeta: metav1.ObjectMeta{
					Name: "test-node",
					Labels: map[string]string{
						"existing-label": "existing-value",
					},
				},
			},
			labels:      []label.Label{},
			expectError: false,
			expectedLabels: map[string]string{
				"existing-label": "existing-value",
			},
		},
		{
			name:     "overwrite existing label",
			nodeName: "test-node",
			existingNode: &v1.Node{
				ObjectMeta: metav1.ObjectMeta{
					Name: "test-node",
					Labels: map[string]string{
						"amd.com/nic.product-name": "old-product",
					},
				},
			},
			labels: []label.Label{
				{Key: "amd.com/nic.product-name", Value: "Test_NIC_1"},
			},
			expectError: false,
			expectedLabels: map[string]string{
				"amd.com/nic.product-name": "Test_NIC_1",
			},
		},
		{
			name:          "node not found",
			nodeName:      "non-existent-node",
			existingNode:  nil,
			labels:        []label.Label{{Key: "amd.com/nic.count", Value: "1"}},
			expectError:   true,
			errorContains: "failed to get node non-existent-node",
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			// Create fake clientset
			var clientset *fake.Clientset
			if tt.existingNode != nil {
				clientset = fake.NewSimpleClientset(tt.existingNode)
			} else {
				clientset = fake.NewSimpleClientset()
			}

			// Create publisher
			publisher, err := NewKubernetesPublisher(clientset, tt.nodeName)
			if err != nil {
				t.Fatalf("failed to create publisher: %v", err)
			}

			// Call Publish
			ctx := context.Background()
			err = publisher.Publish(ctx, tt.labels)

			if tt.expectError {
				if err == nil {
					t.Errorf("expected error but got none")
					return
				}
				if tt.errorContains != "" && !strings.Contains(err.Error(), tt.errorContains) {
					t.Errorf("expected error to contain '%s', got '%s'", tt.errorContains, err.Error())
				}
				return
			}

			if err != nil {
				t.Errorf("unexpected error: %v", err)
				return
			}

			// Verify the node was updated correctly
			if tt.existingNode != nil {
				updatedNode, err := clientset.CoreV1().Nodes().Get(ctx, tt.nodeName, metav1.GetOptions{})
				if err != nil {
					t.Errorf("failed to get updated node: %v", err)
					return
				}

				for key, expectedValue := range tt.expectedLabels {
					if actualValue, exists := updatedNode.Labels[key]; !exists {
						t.Errorf("expected label '%s' not found in node", key)
					} else if actualValue != expectedValue {
						t.Errorf("expected label '%s' to have value '%s', got '%s'", key, expectedValue, actualValue)
					}
				}

				// Verify no unexpected labels were added
				for key := range updatedNode.Labels {
					if _, expected := tt.expectedLabels[key]; !expected {
						t.Errorf("unexpected label '%s' found in node", key)
					}
				}
			}
		})
	}
}

func TestKubernetesPublisher_PublishWithRetryFailure(t *testing.T) {
	// Create a fake clientset
	clientset := fake.NewSimpleClientset(&v1.Node{
		ObjectMeta: metav1.ObjectMeta{
			Name: "test-node",
		},
	})

	// Add a reactor to simulate update failures
	clientset.PrependReactor("update", "nodes", func(action k8stesting.Action) (handled bool, ret runtime.Object, err error) {
		return true, nil, apierrors.NewInternalError(fmt.Errorf("simulated update failure"))
	})

	publisher, err := NewKubernetesPublisher(clientset, "test-node")
	if err != nil {
		t.Fatalf("failed to create publisher: %v", err)
	}

	labels := []label.Label{
		{Key: "amd.com/nic.firmware-version", Value: "1.2.3"},
	}

	ctx := context.Background()
	err = publisher.Publish(ctx, labels)

	if err == nil {
		t.Errorf("expected error due to simulated update failure")
		return
	}

	if !strings.Contains(err.Error(), "failed to update node") {
		t.Errorf("expected error to contain 'failed to update node', got '%s'", err.Error())
	}
}

func TestKubernetesPublisher_PublishWithGetFailure(t *testing.T) {
	// Create a fake clientset
	clientset := fake.NewSimpleClientset()

	// Add a reactor to simulate get failures
	clientset.PrependReactor("get", "nodes", func(action k8stesting.Action) (handled bool, ret runtime.Object, err error) {
		return true, nil, apierrors.NewInternalError(fmt.Errorf("simulated get failure"))
	})

	publisher, err := NewKubernetesPublisher(clientset, "test-node")
	if err != nil {
		t.Fatalf("failed to create publisher: %v", err)
	}

	labels := []label.Label{
		{Key: "amd.com/nic.driver-version", Value: "2.1.0"},
	}

	ctx := context.Background()
	err = publisher.Publish(ctx, labels)

	if err == nil {
		t.Errorf("expected error due to simulated get failure")
		return
	}

	if !strings.Contains(err.Error(), "failed to get node") {
		t.Errorf("expected error to contain 'failed to get node', got '%s'", err.Error())
	}
}

func TestKubernetesPublisher_PublishIntegration(t *testing.T) {
	// Create a node with some existing labels
	existingNode := &v1.Node{
		ObjectMeta: metav1.ObjectMeta{
			Name: "integration-test-node",
			Labels: map[string]string{
				"kubernetes.io/hostname":           "integration-test-node",
				"node.kubernetes.io/instance-type": "standard",
			},
		},
		Spec: v1.NodeSpec{},
		Status: v1.NodeStatus{
			Phase: v1.NodeRunning,
		},
	}

	clientset := fake.NewSimpleClientset(existingNode)
	publisher, err := NewKubernetesPublisher(clientset, "integration-test-node")
	if err != nil {
		t.Fatalf("failed to create publisher: %v", err)
	}

	// Test multiple label operations
	labels1 := []label.Label{
		{Key: "amd.com/nic.count", Value: "1"},
		{Key: "amd.com/nic.product-name", Value: "Test_NIC_1"},
		{Key: "amd.com/nic.port-speed", Value: "100G"},
	}

	labels2 := []label.Label{
		{Key: "amd.com/nic.firmware-version", Value: "1.2.3"},
		{Key: "amd.com/nic.driver-name", Value: "ionic"},
		{Key: "amd.com/nic.driver-version", Value: "2.1.0"},
	}

	ctx := context.Background()

	// First publish
	err = publisher.Publish(ctx, labels1)
	if err != nil {
		t.Errorf("first publish failed: %v", err)
		return
	}

	// Second publish (should add more labels)
	err = publisher.Publish(ctx, labels2)
	if err != nil {
		t.Errorf("second publish failed: %v", err)
		return
	}

	// Verify all labels are present
	node, err := clientset.CoreV1().Nodes().Get(ctx, "integration-test-node", metav1.GetOptions{})
	if err != nil {
		t.Errorf("failed to get node: %v", err)
		return
	}

	expectedLabels := map[string]string{
		"kubernetes.io/hostname":           "integration-test-node",
		"node.kubernetes.io/instance-type": "standard",
		"amd.com/nic.count":                "1",
		"amd.com/nic.product-name":         "Test_NIC_1",
		"amd.com/nic.port-speed":           "100G",
		"amd.com/nic.firmware-version":     "1.2.3",
		"amd.com/nic.driver-name":          "ionic",
		"amd.com/nic.driver-version":       "2.1.0",
	}

	for key, expectedValue := range expectedLabels {
		if actualValue, exists := node.Labels[key]; !exists {
			t.Errorf("expected label '%s' not found", key)
		} else if actualValue != expectedValue {
			t.Errorf("expected label '%s' to have value '%s', got '%s'", key, expectedValue, actualValue)
		}
	}
}

func TestKubernetesPublisher_logLabelsChanged(t *testing.T) {
	// Test case structure
	testCases := []struct {
		name             string
		nodeName         string
		firstCallLabels  []label.Label
		secondCallLabels []label.Label
		expectHashChange bool
		logExpectations  []struct {
			call             int // 1 for first call, 2 for second call
			shouldContain    []string
			shouldNotContain []string
		}
	}{
		{
			name:     "initial call with labels then same labels",
			nodeName: "test-node-1",
			firstCallLabels: []label.Label{
				{Key: "test.com/nic.driver", Value: "ionic"},
				{Key: "test.com/nic.count", Value: "2"},
			},
			secondCallLabels: []label.Label{
				{Key: "test.com/nic.driver", Value: "ionic"},
				{Key: "test.com/nic.count", Value: "2"},
			},
			expectHashChange: false,
			logExpectations: []struct {
				call             int
				shouldContain    []string
				shouldNotContain []string
			}{
				{
					call:          1,
					shouldContain: []string{"Found new labels to Publish", "test-node-1"},
				},
				{
					call:          2,
					shouldContain: []string{"No new labels to Publish, Hash unchanged", "test-node-1"},
				},
			},
		},
		{
			name:     "initial call then different labels",
			nodeName: "test-node-2",
			firstCallLabels: []label.Label{
				{Key: "test.com/nic.driver", Value: "ionic"},
				{Key: "test.com/nic.version", Value: "1.0"},
			},
			secondCallLabels: []label.Label{
				{Key: "test.com/nic.driver", Value: "mlx5"},
				{Key: "test.com/nic.version", Value: "2.0"},
			},
			expectHashChange: true,
			logExpectations: []struct {
				call             int
				shouldContain    []string
				shouldNotContain []string
			}{
				{
					call:          1,
					shouldContain: []string{"Found new labels to Publish", "test-node-2"},
				},
				{
					call:          2,
					shouldContain: []string{"Found new labels to Publish", "test-node-2"},
				},
			},
		},
		{
			name:            "empty labels then non-empty labels",
			nodeName:        "test-node-3",
			firstCallLabels: []label.Label{},
			secondCallLabels: []label.Label{
				{Key: "test.com/nic.added", Value: "true"},
			},
			expectHashChange: true, // Hash changes between first and second call
			logExpectations: []struct {
				call             int
				shouldContain    []string
				shouldNotContain []string
			}{
				{
					call:          1,
					shouldContain: []string{"No new labels to Publish, Hash unchanged", "test-node-3"},
				},
				{
					call:          2,
					shouldContain: []string{"Found new labels to Publish", "test-node-3"},
				},
			},
		},
		{
			name:             "nil labels then empty labels",
			nodeName:         "test-node-4",
			firstCallLabels:  nil,
			secondCallLabels: []label.Label{},
			expectHashChange: false, // Both nil and empty should produce same hash
			logExpectations: []struct {
				call             int
				shouldContain    []string
				shouldNotContain []string
			}{
				{
					call:          1,
					shouldContain: []string{"No new labels to Publish, Hash unchanged", "test-node-4"},
				},
				{
					call:          2,
					shouldContain: []string{"No new labels to Publish, Hash unchanged", "test-node-4"},
				},
			},
		},
		{
			name:     "verify labels get printed when changed",
			nodeName: "test-node-print",
			firstCallLabels: []label.Label{
				{Key: "test.com/driver", Value: "ionic"},
				{Key: "test.com/version", Value: "1.5.0"},
			},
			secondCallLabels: []label.Label{
				{Key: "test.com/driver", Value: "ionic"},
				{Key: "test.com/version", Value: "1.5.0"},
			},
			expectHashChange: false,
			logExpectations: []struct {
				call             int
				shouldContain    []string
				shouldNotContain []string
			}{
				{
					call: 1,
					shouldContain: []string{
						"Found new labels to Publish",
						"test-node-print",

						"key=test.com/driver",
						"value=ionic",
						"key=test.com/version",
						"value=1.5.0",
					},
				},
				{
					call:             2,
					shouldContain:    []string{"No new labels to Publish, Hash unchanged", "test-node-print"},
					shouldNotContain: []string{"key=test.com/driver", "value=ionic"},
				},
			},
		},
	}

	for _, tc := range testCases {
		t.Run(tc.name, func(t *testing.T) {
			// Setup
			clientset := fake.NewSimpleClientset()
			publisher, err := NewKubernetesPublisher(clientset, tc.nodeName)
			if err != nil {
				t.Fatalf("failed to create publisher: %v", err)
			}

			// Verify initial state - previousLabelsHash should be hash of empty labels
			expectedInitialHash := label.HashLabels([]label.Label{})
			if publisher.previousLabelsHash != expectedInitialHash {
				t.Fatalf("expected initial previousLabelsHash to be '%s', got '%s'", expectedInitialHash, publisher.previousLabelsHash)
			}

			// Capture log output (capture both Debug and Info levels)
			var logBuf bytes.Buffer
			logger := slog.New(slog.NewTextHandler(&logBuf, &slog.HandlerOptions{
				Level: slog.LevelDebug, // This captures Debug, Info, Warn, Error
			}))
			originalLogger := slog.Default()
			slog.SetDefault(logger)
			defer slog.SetDefault(originalLogger)

			// First call
			logBuf.Reset()
			publisher.logLabelsChanged(tc.firstCallLabels)
			firstCallLog := logBuf.String()
			hashAfterFirst := publisher.previousLabelsHash

			// Verify first call expectations
			for _, expectation := range tc.logExpectations {
				if expectation.call == 1 {
					for _, shouldContain := range expectation.shouldContain {
						if !strings.Contains(firstCallLog, shouldContain) {
							t.Errorf("first call log should contain '%s', but got: %s", shouldContain, firstCallLog)
						}
					}
					for _, shouldNotContain := range expectation.shouldNotContain {
						if strings.Contains(firstCallLog, shouldNotContain) {
							t.Errorf("first call log should not contain '%s', but got: %s", shouldNotContain, firstCallLog)
						}
					}
				}
			}

			// Verify hash was set after first call
			expectedFirstHash := label.HashLabels(tc.firstCallLabels)
			if hashAfterFirst != expectedFirstHash {
				t.Errorf("after first call, expected hash '%s', got '%s'", expectedFirstHash, hashAfterFirst)
			}

			// Second call
			logBuf.Reset()
			publisher.logLabelsChanged(tc.secondCallLabels)
			secondCallLog := logBuf.String()
			hashAfterSecond := publisher.previousLabelsHash

			// Verify second call expectations
			for _, expectation := range tc.logExpectations {
				if expectation.call == 2 {
					for _, shouldContain := range expectation.shouldContain {
						if !strings.Contains(secondCallLog, shouldContain) {
							t.Errorf("second call log should contain '%s', but got: %s", shouldContain, secondCallLog)
						}
					}
					for _, shouldNotContain := range expectation.shouldNotContain {
						if strings.Contains(secondCallLog, shouldNotContain) {
							t.Errorf("second call log should not contain '%s', but got: %s", shouldNotContain, secondCallLog)
						}
					}
				}
			}

			// Verify hash behavior
			expectedSecondHash := label.HashLabels(tc.secondCallLabels)
			if hashAfterSecond != expectedSecondHash {
				t.Errorf("after second call, expected hash '%s', got '%s'", expectedSecondHash, hashAfterSecond)
			}

			// Verify hash change expectation
			if tc.expectHashChange {
				if hashAfterFirst == hashAfterSecond {
					t.Errorf("expected hash to change between calls, but both were '%s'", hashAfterFirst)
				}
			} else {
				if hashAfterFirst != hashAfterSecond {
					t.Errorf("expected hash to remain same between calls, but got '%s' then '%s'", hashAfterFirst, hashAfterSecond)
				}
			}
		})
	}
}
