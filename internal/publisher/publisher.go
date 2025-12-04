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

	"github.com/ROCm/k8s-network-node-labeller/internal/label"
)

// Publisher defines the interface for publishing labels
type Publisher interface {
	// Publish publishes the given labels
	Publish(ctx context.Context, labels []label.Label) error

	// GetName returns the name of the publisher for logging purposes
	GetName() string
}
