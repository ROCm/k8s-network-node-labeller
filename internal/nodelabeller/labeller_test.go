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
	"path/filepath"
	"testing"

	v1 "k8s.io/api/core/v1"
	metav1 "k8s.io/apimachinery/pkg/apis/meta/v1"
	"k8s.io/client-go/kubernetes/fake"

	"github.com/ROCm/k8s-network-node-labeller/internal/cleaner"
	"github.com/ROCm/k8s-network-node-labeller/internal/discoverer"
	"github.com/ROCm/k8s-network-node-labeller/internal/nicctl"
	"github.com/ROCm/k8s-network-node-labeller/internal/publisher"
	"github.com/ROCm/k8s-network-node-labeller/internal/sysfs"
	"github.com/ROCm/k8s-network-node-labeller/internal/testutils"
)

func TestNodeLabeller_IntegrationWithNicSingleYAML(t *testing.T) {
	const testNodeName = "test-node"

	// Create a fake Kubernetes clientset
	fakeClientset := fake.NewSimpleClientset()

	// Create a test node with some existing labels
	testNode := &v1.Node{
		ObjectMeta: metav1.ObjectMeta{
			Name: testNodeName,
			Labels: map[string]string{
				"kubernetes.io/hostname":       testNodeName,
				"beta.kubernetes.io/arch":      "amd64",
				"amd.com/nic.count":            "2",     // This will be updated
				"amd.com/nic.firmware-version": "3.0.0", // This will be updated
				"amd.com/nic.pollara":          "true",  // This should NOT be cleaned
			},
		},
	}

	// Add the test node to the fake clientset
	_, err := fakeClientset.CoreV1().Nodes().Create(context.Background(), testNode, metav1.CreateOptions{})
	if err != nil {
		t.Fatalf("Failed to create test node: %v", err)
	}

	// Create mock nicctl client using the nic_single.yaml testdata
	yamlFile := filepath.Join("testdata", "nic_single.yaml")
	mockNicctlClient, err := nicctl.NewMockNicctlClientFromYAML(yamlFile)
	if err != nil {
		t.Fatalf("Failed to create mock nicctl client from YAML: %v", err)
	}

	// Create product info map with the specified product info
	testProductInfoMap := map[string]sysfs.ProductInfo{
		"8201": {
			ProductName: "Test NIC 1",
			SKU:         "DSS-W600",
		},
	}

	// Create temporary directory and set up mock sysfs using nic_single_pci_funcs.yaml
	tempDir := t.TempDir()
	sysfsYamlFile := filepath.Join("testdata", "nic_single_pci_funcs.yaml")
	err = testutils.CreateFakeSysfsFromFile(tempDir, sysfsYamlFile)
	if err != nil {
		t.Fatalf("Failed to create fake sysfs: %v", err)
	}

	// Create sysfs client with custom product info
	sysfsClient, err := sysfs.NewSysfsClientWithProductInfoMap(tempDir, testProductInfoMap)
	if err != nil {
		t.Fatalf("Failed to create sysfs client: %v", err)
	}

	// Create discoverers with both mock nicctl and sysfs clients
	nicctlDiscoverer := discoverer.NewNicctlDiscoverer(mockNicctlClient)
	sysfsDiscoverer := discoverer.NewSysfsDiscoverer(sysfsClient)

	// Create cleaner with the fake clientset
	kubeCleaner, err := cleaner.NewKubernetesCleaner(fakeClientset, testNodeName)
	if err != nil {
		t.Fatalf("Failed to create kubernetes cleaner: %v", err)
	}

	// Create publisher with the fake clientset
	kubePublisher, err := publisher.NewKubernetesPublisher(fakeClientset, testNodeName)
	if err != nil {
		t.Fatalf("Failed to create kubernetes publisher: %v", err)
	}

	// Create NodeLabeller with injected dependencies for testing
	cleaners := []cleaner.Cleaner{kubeCleaner}
	discoverers := []discoverer.Discoverer{nicctlDiscoverer, sysfsDiscoverer}
	publishers := []publisher.Publisher{kubePublisher}

	nodeLabeller := NewNodeLabeller(
		&cleaners,
		&discoverers,
		&publishers,
	)

	// Run a single labelling cycle
	ctx := context.Background()
	err = nodeLabeller.runOnce(ctx)
	if err != nil {
		t.Fatalf("Failed to run labelling cycle: %v", err)
	}

	// Verify the labels on the node
	updatedNode, err := fakeClientset.CoreV1().Nodes().Get(ctx, testNodeName, metav1.GetOptions{})
	if err != nil {
		t.Fatalf("Failed to get updated node: %v", err)
	}

	// Expected labels based on nic_single.yaml testdata
	expectedLabels := map[string]string{
		"kubernetes.io/hostname":       testNodeName,
		"beta.kubernetes.io/arch":      "amd64",
		"amd.com/nic.count":            "1",
		"amd.com/nic.product-name":     "Test_NIC_1",
		"amd.com/nic.firmware-version": "1.2.3",
		"amd.com/nic.profile":          "pf1_vf1",
		"amd.com/nic.port-count":       "1",
		"amd.com/nic.port-speed":       "100G",
		"amd.com/nic.driver-version":   "2.1.0",
		"amd.com/nic.driver-name":      "ionic",
		"amd.com/nic.pollara":          "true",
	}

	// Check that expected labels exist with correct values
	for expectedKey, expectedValue := range expectedLabels {
		if actualValue, exists := updatedNode.Labels[expectedKey]; !exists {
			t.Errorf("Expected label %s not found on node", expectedKey)
		} else if actualValue != expectedValue {
			t.Errorf("Expected label %s to have value %s, got %s", expectedKey, expectedValue, actualValue)
		}
	}

	t.Logf("Successfully verified %d expected labels on node %s", len(expectedLabels), testNodeName)

	// Print all labels for debugging
	t.Log("Final node labels:")
	for key, value := range updatedNode.Labels {
		t.Logf("  %s: %s", key, value)
	}
}
