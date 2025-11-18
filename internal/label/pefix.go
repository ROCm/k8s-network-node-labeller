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

const (
	DefaultPrefix = "amd.com"
	BetaPrefix    = "beta.amd.com"
	NICQualifier  = "nic"
)

// DefaultPrefixedKey creates a key with default prefix
func DefaultPrefixedKey(key string) string {
	return PrefixedKey(DefaultPrefix, key)
}

// BetaPrefixedKey creates a key with beta prefix
func BetaPrefixedKey(key string) string {
	return PrefixedKey(BetaPrefix, key)
}

// PrefixedKey creates a prefixed key
func PrefixedKey(prefix, key string) string {
	return prefix + "/" + key
}

// DefaultNICPrefixedKey creates a key with default prefix for NIC properties
func DefaultNICPrefixedKey(key string) string {
	return DefaultPrefixedKey(NICQualifier + "." + key)
}
