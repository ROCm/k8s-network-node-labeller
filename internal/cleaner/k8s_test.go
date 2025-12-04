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
	"strings"
	"testing"

	v1 "k8s.io/api/core/v1"
	metav1 "k8s.io/apimachinery/pkg/apis/meta/v1"
	"k8s.io/apimachinery/pkg/runtime"
	"k8s.io/client-go/kubernetes"
	"k8s.io/client-go/kubernetes/fake"
	k8stesting "k8s.io/client-go/testing"

	"github.com/ROCm/k8s-network-node-labeller/internal/label"
)

func TestNewKubernetesCleaner(t *testing.T) {
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
			cleaner, err := NewKubernetesCleaner(tt.clientset, tt.nodeName)

			if tt.expectError {
				if err == nil {
					t.Errorf("expected error but got none")
					return
				}
				if err.Error() != tt.errorMsg {
					t.Errorf("expected error message %q, got %q", tt.errorMsg, err.Error())
				}
				return
			}

			if err != nil {
				t.Errorf("unexpected error: %v", err)
				return
			}

			if cleaner == nil {
				t.Error("expected non-nil cleaner")
			}
		})
	}
}

func TestKubernetesCleaner_CleanByRegexps(t *testing.T) {
	tests := []struct {
		name           string
		nodeName       string
		regexps        []string
		initialLabels  map[string]string
		expectedLabels map[string]string
		expectError    bool
		nodeExists     bool
	}{
		{
			name:     "clean managed labels",
			nodeName: "test-node",
			regexps:  label.DefaultNICPrefixedLabelRegexps,
			initialLabels: map[string]string{
				"amd.com/nic.count":        "1",
				"amd.com/nic.product-name": "Test_NIC_1",
				"kubernetes.io/hostname":   "test-node",
				"beta.kubernetes.io/arch":  "amd64",
			},
			expectedLabels: map[string]string{
				"kubernetes.io/hostname":  "test-node",
				"beta.kubernetes.io/arch": "amd64",
			},
			expectError: false,
			nodeExists:  true,
		},
		{
			name:     "labels should not be cleaned",
			nodeName: "test-node",
			regexps:  label.DefaultNICPrefixedLabelRegexps,
			initialLabels: map[string]string{
				"kubernetes.io/hostname":  "test-node",
				"beta.kubernetes.io/arch": "amd64",

				// These labels are not managed by k8s-network-node-labeller
				// They should also not get cleaned
				"amd.com/nic.some-custom-label": "1",
				"amd.com/some-other-label":      "dummy-value",
				"amd.com/nic.pollara":           "true",
				"amd.com/nic.gen2":              "true",
				"amd.com/nic.rccl":              "true",
				"beta.amd.com/nic.some-label":   "val1",
				"beta.amd.com/some-other-label": "val2",
			},
			expectedLabels: map[string]string{
				"kubernetes.io/hostname":        "test-node",
				"beta.kubernetes.io/arch":       "amd64",
				"amd.com/nic.some-custom-label": "1",
				"amd.com/some-other-label":      "dummy-value",
				"amd.com/nic.pollara":           "true",
				"amd.com/nic.gen2":              "true",
				"amd.com/nic.rccl":              "true",
				"beta.amd.com/nic.some-label":   "val1",
				"beta.amd.com/some-other-label": "val2",
			},
			expectError: false,
			nodeExists:  true,
		},
		{
			name:           "node with no labels",
			nodeName:       "test-node",
			regexps:        label.DefaultNICPrefixedLabelRegexps,
			initialLabels:  nil,
			expectedLabels: map[string]string{},
			expectError:    false,
			nodeExists:     true,
		},
		{
			name:        "empty regexps",
			nodeName:    "test-node",
			regexps:     []string{},
			expectError: true,
			nodeExists:  true,
		},
		{
			name:        "node does not exist",
			nodeName:    "non-existent-node",
			regexps:     label.DefaultNICPrefixedLabelRegexps,
			expectError: true,
			nodeExists:  false,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			// Create fake clientset
			var objects []runtime.Object
			if tt.nodeExists {
				node := &v1.Node{
					ObjectMeta: metav1.ObjectMeta{
						Name:   tt.nodeName,
						Labels: make(map[string]string),
					},
				}
				// Copy initial labels
				if tt.initialLabels != nil {
					for k, v := range tt.initialLabels {
						node.Labels[k] = v
					}
				}
				objects = append(objects, node)
			}

			fakeClientset := fake.NewSimpleClientset(objects...)

			// Create cleaner using the updated constructor that accepts kubernetes.Interface
			cleaner, err := NewKubernetesCleaner(fakeClientset, tt.nodeName)
			if err != nil {
				t.Fatalf("failed to create cleaner: %v", err)
			}

			// Execute CleanByRegexps
			ctx := context.Background()
			err = cleaner.CleanByRegexps(ctx, tt.regexps)

			if tt.expectError {
				if err == nil {
					t.Error("expected error but got none")
				}
				return
			}

			if err != nil {
				t.Errorf("unexpected error: %v", err)
				return
			}

			// Verify the node labels after cleaning
			updatedNode, err := fakeClientset.CoreV1().Nodes().Get(ctx, tt.nodeName, metav1.GetOptions{})
			if err != nil {
				t.Fatalf("failed to get updated node: %v", err)
			}

			// Check that the expected labels match
			if tt.expectedLabels == nil && updatedNode.Labels != nil {
				t.Errorf("expected no labels, but got: %v", updatedNode.Labels)
				return
			}

			if tt.expectedLabels != nil {
				if len(updatedNode.Labels) != len(tt.expectedLabels) {
					t.Errorf("expected %d labels, got %d. Expected: %v, Got: %v",
						len(tt.expectedLabels), len(updatedNode.Labels), tt.expectedLabels, updatedNode.Labels)
					return
				}

				for key, expectedValue := range tt.expectedLabels {
					if actualValue, exists := updatedNode.Labels[key]; !exists {
						t.Errorf("expected label %q not found", key)
					} else if actualValue != expectedValue {
						t.Errorf("label %q: expected %q, got %q", key, expectedValue, actualValue)
					}
				}

				// Also check that no unexpected labels exist
				for key := range updatedNode.Labels {
					if _, expected := tt.expectedLabels[key]; !expected {
						t.Errorf("unexpected label found: %q = %q", key, updatedNode.Labels[key])
					}
				}
			}
		})
	}
}

func TestKubernetesCleaner_CleanByPrefix_UpdateError(t *testing.T) {
	nodeName := "test-node"
	node := &v1.Node{
		ObjectMeta: metav1.ObjectMeta{
			Name: nodeName,
			Labels: map[string]string{
				"amd.com/nic.count": "1",
				"other-label":       "value",
			},
		},
	}

	fakeClientset := fake.NewSimpleClientset(node)

	// Add a reactor to simulate update failure
	fakeClientset.PrependReactor("update", "nodes", func(action k8stesting.Action) (handled bool, ret runtime.Object, err error) {
		return true, nil, fmt.Errorf("simulated update error")
	})

	cleaner, err := NewKubernetesCleaner(fakeClientset, nodeName)
	if err != nil {
		t.Fatalf("failed to create cleaner: %v", err)
	}

	ctx := context.Background()
	err = cleaner.CleanByRegexps(ctx, label.DefaultNICPrefixedLabelRegexps)

	if err == nil {
		t.Error("expected error due to update failure, but got none")
	}

	expectedErrorSubstring := "failed to update node"
	if err != nil && !strings.Contains(err.Error(), expectedErrorSubstring) {
		t.Errorf("expected error to contain %q, got: %v", expectedErrorSubstring, err)
	}
}
