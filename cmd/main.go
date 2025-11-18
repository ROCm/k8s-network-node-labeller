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

package main

import (
	"context"
	"fmt"
	"log/slog"
	"os"
	"os/signal"
	"strings"
	"syscall"

	"github.com/ROCm/k8s-network-node-labeller/internal/cleaner"
	"github.com/ROCm/k8s-network-node-labeller/internal/discoverer"
	"github.com/ROCm/k8s-network-node-labeller/internal/nicctl"
	"github.com/ROCm/k8s-network-node-labeller/internal/nodelabeller"
	"github.com/ROCm/k8s-network-node-labeller/internal/publisher"
	"github.com/ROCm/k8s-network-node-labeller/internal/sysfs"
	"github.com/ROCm/k8s-network-node-labeller/internal/utils"
	"k8s.io/client-go/kubernetes"
	"k8s.io/client-go/rest"
	"k8s.io/client-go/tools/clientcmd"
)

// initLogger initializes slog with the LOG_LEVEL environment variable
func initLogger() {
	logLevel := getLogLevelFromEnv()

	opts := &slog.HandlerOptions{
		Level: logLevel,
	}

	handler := slog.NewTextHandler(os.Stdout, opts)
	logger := slog.New(handler)

	// Set as default logger
	slog.SetDefault(logger)
}

// getLogLevelFromEnv returns the log level from LOG_LEVEL environment variable
// Defaults to INFO if not set or invalid
func getLogLevelFromEnv() slog.Level {
	logLevelStr := strings.ToUpper(strings.TrimSpace(os.Getenv("LOG_LEVEL")))

	switch logLevelStr {
	case "DEBUG":
		return slog.LevelDebug
	case "INFO":
		return slog.LevelInfo
	case "WARN", "WARNING":
		return slog.LevelWarn
	case "ERROR":
		return slog.LevelError
	default:
		// Default to INFO
		return slog.LevelInfo
	}
}

func createNodeLabeller() (*nodelabeller.NodeLabeller, error) {
	nodeName, err := utils.GetNodeNameFromEnv()
	if err != nil {
		return nil, fmt.Errorf("failed to get node name: %w", err)
	}
	slog.Info("Node name", "nodeName", nodeName)

	// Create shared Kubernetes client
	clientset, err := createKubernetesClient()
	if err != nil {
		return nil, fmt.Errorf("failed to create kubernetes client: %w", err)
	}

	// Create sysfs client
	sysfsClient, err := createSysfsClient()
	if err != nil {
		return nil, fmt.Errorf("failed to create sysfs client: %w", err)
	}

	// Create cleaner with shared kubernetes client
	kubeCleaner, err := cleaner.NewKubernetesCleaner(clientset, nodeName)
	if err != nil {
		return nil, fmt.Errorf("failed to create kubernetes cleaner: %w", err)
	}

	// Create publisher with shared kubernetes client
	kubePublisher, err := publisher.NewKubernetesPublisher(clientset, nodeName)
	if err != nil {
		return nil, fmt.Errorf("failed to create kubernetes publisher: %w", err)
	}

	sysfsDiscoverer := discoverer.NewSysfsDiscoverer(sysfsClient)

	discoverers := []discoverer.Discoverer{
		sysfsDiscoverer,
	}
	// create nicctl client and discoverer based on environment
	nicctlClient, err := createNicctlClient()
	if err != nil {
		return nil, fmt.Errorf("error creating nicctl client: %w", err)
	}
	if nicctlClient != nil {
		nicctlDiscoverer := discoverer.NewNicctlDiscoverer(nicctlClient)
		discoverers = append(discoverers, nicctlDiscoverer)
	}

	publishers := []publisher.Publisher{
		kubePublisher,
	}

	cleaners := []cleaner.Cleaner{
		kubeCleaner,
	}

	return nodelabeller.NewNodeLabeller(
		&cleaners,
		&discoverers,
		&publishers,
	), nil
}

// createKubernetesClient creates a Kubernetes client
func createKubernetesClient() (kubernetes.Interface, error) {
	// Try to create in-cluster config first
	config, err := rest.InClusterConfig()
	if err != nil {
		// Fall back to kubeconfig file
		config, err = clientcmd.BuildConfigFromFlags("", clientcmd.RecommendedHomeFile)
		if err != nil {
			return nil, fmt.Errorf("failed to create kubernetes config: %w", err)
		}
	}

	// Create the clientset
	clientset, err := kubernetes.NewForConfig(config)
	if err != nil {
		return nil, fmt.Errorf("failed to create kubernetes clientset: %w", err)
	}

	return clientset, nil
}

// createNicctlClient creates a nicctl client based on the environment
func createNicctlClient() (nicctl.NicctlClient, error) {
	// If SIM_ENABLE environment variable is set, create a mock client
	if _, present := os.LookupEnv("SIM_ENABLE"); present {
		slog.Info("SIM_ENABLE is set, using MockNicctlClient")
		mockClient := nicctl.NewMockNicctlClient()
		mockClient.GetIonicDriverVersionFunc = func() (string, error) {
			return "", nil
		}
		return mockClient, nil
	}

	// Check if it is a bare metal node
	if utils.IsNodeBareMetal() {
		nicctlClient, err := nicctl.NewNicctlCommandClient(discoverer.NicctlBinaryPath)
		if err != nil {
			return nil, fmt.Errorf("failed to create nicctl client: %w", err)
		}
		return nicctlClient, nil
	} else {
		slog.Info("Node is not bare metal, skipping nicctl discoverer creation")
	}

	return nil, nil
}

// createSysfsClient creates a sysfs client
func createSysfsClient() (*sysfs.SysfsClient, error) {
	client, err := sysfs.NewSysfsClient(discoverer.DefaultSysfsBasePath)
	if err != nil {
		return nil, fmt.Errorf("failed to create sysfs client: %w", err)
	}
	return client, nil
}

func main() {
	// Initialize logger with LOG_LEVEL environment variable support
	initLogger()

	ctx, cancel := context.WithCancel(context.Background())
	defer cancel()

	// Create and start NodeLabeller
	labeller, err := createNodeLabeller()
	if err != nil {
		slog.Error("Failed to create NodeLabeller", "error", err)
		os.Exit(1)
	}

	// Set up signal handling for graceful shutdown
	sigChan := make(chan os.Signal, 1)
	signal.Notify(sigChan, syscall.SIGINT, syscall.SIGTERM)

	// Start the labeller in a goroutine
	errChan := make(chan error, 1)
	go func() {
		if err := labeller.Run(ctx); err != nil {
			errChan <- err
		}
	}()

	// Wait for shutdown signal or error
	select {
	case sig := <-sigChan:
		slog.Info("Received signal, shutting down gracefully", "signal", sig)
		cancel()
		labeller.Stop()
	case err := <-errChan:
		slog.Error("NodeLabeller error", "error", err)
		cancel()
		labeller.Stop()
	}

	slog.Info("NodeLabeller shutdown complete")
}
