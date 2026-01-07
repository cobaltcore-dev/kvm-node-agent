/*
SPDX-FileCopyrightText: Copyright 2025 SAP SE or an SAP affiliate company and cobaltcore-dev contributors
SPDX-License-Identifier: Apache-2.0

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

package libvirt

import (
	"testing"

	"k8s.io/apimachinery/pkg/api/resource"
)

func TestMemoryToResourceKiB(t *testing.T) {
	tests := []struct {
		name          string
		value         int64
		unit          string
		expectedBytes int64
	}{
		{
			name:          "1 KiB",
			value:         1,
			unit:          "KiB",
			expectedBytes: 1024,
		},
		{
			name:          "1024 KiB (1 MiB)",
			value:         1024,
			unit:          "KiB",
			expectedBytes: 1024 * 1024,
		},
		{
			name:          "Zero KiB",
			value:         0,
			unit:          "KiB",
			expectedBytes: 0,
		},
		{
			name:          "Large value KiB",
			value:         1048576, // 1 GiB in KiB
			unit:          "KiB",
			expectedBytes: 1024 * 1024 * 1024,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			result, err := MemoryToResource(tt.value, tt.unit)
			if err != nil {
				t.Fatalf("MemoryToResource() returned unexpected error: %v", err)
			}

			expectedQuantity := resource.NewQuantity(tt.expectedBytes, resource.BinarySI)
			if !result.Equal(*expectedQuantity) {
				t.Errorf("Expected quantity %s, got %s", expectedQuantity.String(), result.String())
			}

			// Verify the value in bytes
			resultBytes, ok := result.AsInt64()
			if !ok {
				t.Fatal("Failed to convert result to int64")
			}
			if resultBytes != tt.expectedBytes {
				t.Errorf("Expected %d bytes, got %d bytes", tt.expectedBytes, resultBytes)
			}
		})
	}
}

func TestMemoryToResourceMiB(t *testing.T) {
	tests := []struct {
		name          string
		value         int64
		unit          string
		expectedBytes int64
	}{
		{
			name:          "1 MiB",
			value:         1,
			unit:          "MiB",
			expectedBytes: 1024 * 1024,
		},
		{
			name:          "1024 MiB (1 GiB)",
			value:         1024,
			unit:          "MiB",
			expectedBytes: 1024 * 1024 * 1024,
		},
		{
			name:          "Zero MiB",
			value:         0,
			unit:          "MiB",
			expectedBytes: 0,
		},
		{
			name:          "Large value MiB",
			value:         16384, // 16 GiB in MiB
			unit:          "MiB",
			expectedBytes: 16 * 1024 * 1024 * 1024,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			result, err := MemoryToResource(tt.value, tt.unit)
			if err != nil {
				t.Fatalf("MemoryToResource() returned unexpected error: %v", err)
			}

			expectedQuantity := resource.NewQuantity(tt.expectedBytes, resource.BinarySI)
			if !result.Equal(*expectedQuantity) {
				t.Errorf("Expected quantity %s, got %s", expectedQuantity.String(), result.String())
			}

			// Verify the value in bytes
			resultBytes, ok := result.AsInt64()
			if !ok {
				t.Fatal("Failed to convert result to int64")
			}
			if resultBytes != tt.expectedBytes {
				t.Errorf("Expected %d bytes, got %d bytes", tt.expectedBytes, resultBytes)
			}
		})
	}
}

func TestMemoryToResourceGiB(t *testing.T) {
	tests := []struct {
		name          string
		value         int64
		unit          string
		expectedBytes int64
	}{
		{
			name:          "1 GiB",
			value:         1,
			unit:          "GiB",
			expectedBytes: 1024 * 1024 * 1024,
		},
		{
			name:          "8 GiB",
			value:         8,
			unit:          "GiB",
			expectedBytes: 8 * 1024 * 1024 * 1024,
		},
		{
			name:          "Zero GiB",
			value:         0,
			unit:          "GiB",
			expectedBytes: 0,
		},
		{
			name:          "Large value GiB",
			value:         128,
			unit:          "GiB",
			expectedBytes: 128 * 1024 * 1024 * 1024,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			result, err := MemoryToResource(tt.value, tt.unit)
			if err != nil {
				t.Fatalf("MemoryToResource() returned unexpected error: %v", err)
			}

			expectedQuantity := resource.NewQuantity(tt.expectedBytes, resource.BinarySI)
			if !result.Equal(*expectedQuantity) {
				t.Errorf("Expected quantity %s, got %s", expectedQuantity.String(), result.String())
			}

			// Verify the value in bytes
			resultBytes, ok := result.AsInt64()
			if !ok {
				t.Fatal("Failed to convert result to int64")
			}
			if resultBytes != tt.expectedBytes {
				t.Errorf("Expected %d bytes, got %d bytes", tt.expectedBytes, resultBytes)
			}
		})
	}
}

func TestMemoryToResourceTiB(t *testing.T) {
	tests := []struct {
		name          string
		value         int64
		unit          string
		expectedBytes int64
	}{
		{
			name:          "1 TiB",
			value:         1,
			unit:          "TiB",
			expectedBytes: 1024 * 1024 * 1024 * 1024,
		},
		{
			name:          "2 TiB",
			value:         2,
			unit:          "TiB",
			expectedBytes: 2 * 1024 * 1024 * 1024 * 1024,
		},
		{
			name:          "Zero TiB",
			value:         0,
			unit:          "TiB",
			expectedBytes: 0,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			result, err := MemoryToResource(tt.value, tt.unit)
			if err != nil {
				t.Fatalf("MemoryToResource() returned unexpected error: %v", err)
			}

			expectedQuantity := resource.NewQuantity(tt.expectedBytes, resource.BinarySI)
			if !result.Equal(*expectedQuantity) {
				t.Errorf("Expected quantity %s, got %s", expectedQuantity.String(), result.String())
			}

			// Verify the value in bytes
			resultBytes, ok := result.AsInt64()
			if !ok {
				t.Fatal("Failed to convert result to int64")
			}
			if resultBytes != tt.expectedBytes {
				t.Errorf("Expected %d bytes, got %d bytes", tt.expectedBytes, resultBytes)
			}
		})
	}
}

func TestMemoryToResourceInvalidUnit(t *testing.T) {
	tests := []struct {
		name  string
		value int64
		unit  string
	}{
		{
			name:  "Invalid unit KB",
			value: 1024,
			unit:  "KB",
		},
		{
			name:  "Invalid unit MB",
			value: 1024,
			unit:  "MB",
		},
		{
			name:  "Invalid unit GB",
			value: 1024,
			unit:  "GB",
		},
		{
			name:  "Invalid unit TB",
			value: 1024,
			unit:  "TB",
		},
		{
			name:  "Invalid unit bytes",
			value: 1024,
			unit:  "bytes",
		},
		{
			name:  "Empty unit",
			value: 1024,
			unit:  "",
		},
		{
			name:  "Random string",
			value: 1024,
			unit:  "invalid",
		},
		{
			name:  "Case sensitive - kib",
			value: 1024,
			unit:  "kib",
		},
		{
			name:  "Case sensitive - mib",
			value: 1024,
			unit:  "mib",
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			result, err := MemoryToResource(tt.value, tt.unit)
			if err == nil {
				t.Errorf("Expected error for invalid unit '%s', but got result: %s", tt.unit, result.String())
			}

			expectedError := "unknown memory unit " + tt.unit
			if err.Error() != expectedError {
				t.Errorf("Expected error message '%s', got '%s'", expectedError, err.Error())
			}

			// Verify the result is an empty Quantity
			if !result.IsZero() {
				t.Errorf("Expected zero quantity for error case, got %s", result.String())
			}
		})
	}
}

func TestMemoryToResourceBinaryFormat(t *testing.T) {
	// Verify that the returned quantities use BinarySI format
	result, err := MemoryToResource(1, "GiB")
	if err != nil {
		t.Fatalf("MemoryToResource() returned unexpected error: %v", err)
	}

	// BinarySI should format as "1Gi" not "1073741824"
	resultString := result.String()
	if resultString != "1Gi" {
		t.Errorf("Expected BinarySI format '1Gi', got '%s'", resultString)
	}
}

func TestMemoryToResourceRealWorldScenarios(t *testing.T) {
	tests := []struct {
		name        string
		value       int64
		unit        string
		expectedStr string
		description string
	}{
		{
			name:        "Typical VM memory - 8GB",
			value:       8192,
			unit:        "MiB",
			expectedStr: "8Gi",
			description: "8GB RAM for a typical VM",
		},
		{
			name:        "Large VM memory - 64GB",
			value:       64,
			unit:        "GiB",
			expectedStr: "64Gi",
			description: "64GB RAM for a large VM",
		},
		{
			name:        "Memory from example XML - ~24GB",
			value:       25149440,
			unit:        "KiB",
			expectedStr: "24560Mi",
			description: "Memory value from the example domain XML",
		},
		{
			name:        "Small container memory - 512MB",
			value:       524288,
			unit:        "KiB",
			expectedStr: "512Mi",
			description: "512MB memory for a small container",
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			result, err := MemoryToResource(tt.value, tt.unit)
			if err != nil {
				t.Fatalf("MemoryToResource() returned unexpected error: %v", err)
			}

			resultString := result.String()
			if resultString != tt.expectedStr {
				t.Errorf("Expected '%s', got '%s' for %s", tt.expectedStr, resultString, tt.description)
			}
		})
	}
}

func TestMemoryToResourceNegativeValues(t *testing.T) {
	// Test behavior with negative values (edge case)
	tests := []struct {
		name  string
		value int64
		unit  string
	}{
		{
			name:  "Negative KiB",
			value: -1024,
			unit:  "KiB",
		},
		{
			name:  "Negative MiB",
			value: -512,
			unit:  "MiB",
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			// The function doesn't explicitly check for negative values,
			// so it will create a negative quantity
			result, err := MemoryToResource(tt.value, tt.unit)
			if err != nil {
				t.Fatalf("MemoryToResource() returned unexpected error: %v", err)
			}

			// Verify it's negative
			if result.Sign() >= 0 {
				t.Errorf("Expected negative quantity for negative input, got %s", result.String())
			}
		})
	}
}
