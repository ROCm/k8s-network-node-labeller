/*
Copyright (c) Advanced Micro Devices, Inc. All rights reserved.

Licensed under the Apache License, Version 2.0 (the \"License\");
you may not use this file except in compliance with the License.
You may obtain a copy of the License at

     http://www.apache.org/licenses/LICENSE-2.0

Unless required by applicable law or agreed to in writing, software
distributed under the License is distributed on an \"AS IS\" BASIS,
WITHOUT WARRANTIES OR CONDITIONS OF ANY KIND, either express or implied.
See the License for the specific language governing permissions and
limitations under the License.
*/

package utils

import (
	"encoding/binary"

	"github.com/prometheus/procfs"
	"github.com/spaolacci/murmur3"
)

const (
	hypervisorCPUFlag = "hypervisor"
)

func HashArray(input []string) string {
	if len(input) == 0 {
		return ""
	}

	// Start with the first string
	currentHash := []byte("")

	// Chain hash each subsequent string with the previous result
	for i := 0; i < len(input); i++ {
		// Combine current hash with next string
		combined := append(currentHash, []byte(input[i])...)
		h1, h2 := murmur3.Sum128(combined)

		// Convert the 128-bit hash to bytes
		hashBytes := make([]byte, 16)
		binary.LittleEndian.PutUint64(hashBytes[0:8], h1)
		binary.LittleEndian.PutUint64(hashBytes[8:16], h2)
		currentHash = hashBytes
	}

	return string(currentHash)
}

func IsNodeBareMetal() bool {
	fs, err := procfs.NewDefaultFS()
	if err != nil {
		return false
	}
	cpus, err := fs.CPUInfo()
	if err != nil {
		return false
	}
	for _, cpu := range cpus {
		for _, flag := range cpu.Flags {
			if flag == hypervisorCPUFlag {
				return false
			}
		}
	}
	return true
}
