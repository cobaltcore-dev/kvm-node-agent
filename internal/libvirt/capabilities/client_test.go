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

package capabilities

import (
	"encoding/xml"
	"os"
	"testing"

	"k8s.io/apimachinery/pkg/api/resource"
)

func TestNewClient(t *testing.T) {
	// Test with default socket path
	client := NewClient()
	if client == nil {
		t.Fatal("NewClient() returned nil")
	}

	// Verify it returns a client implementation
	// We can't check the concrete type since it's unexported, but we can verify it implements the interface
	if client == nil {
		t.Error("NewClient() returned nil")
	}
}

func TestNewClientWithCustomSocket(t *testing.T) {
	// Set custom socket path
	originalSocket := os.Getenv("LIBVIRT_SOCKET")
	defer func() {
		if originalSocket != "" {
			os.Setenv("LIBVIRT_SOCKET", originalSocket)
		} else {
			os.Unsetenv("LIBVIRT_SOCKET")
		}
	}()

	customSocket := "/custom/libvirt-sock"
	os.Setenv("LIBVIRT_SOCKET", customSocket)

	client := NewClient()
	if client == nil {
		t.Fatal("NewClient() returned nil with custom socket")
	}

	// Verify it returns a valid client implementation
	if client == nil {
		t.Error("NewClient() returned nil with custom socket")
	}
}

func TestNewClientEmulator(t *testing.T) {
	client := NewClientEmulator()
	if client == nil {
		t.Fatal("NewClientEmulator() returned nil")
	}

	// Verify it returns a valid client implementation
	if client == nil {
		t.Error("NewClientEmulator() returned nil")
	}
}

func TestClientEmulatorGet(t *testing.T) {
	client := NewClientEmulator()

	status, err := client.Get()
	if err != nil {
		t.Fatalf("clientEmulator.Get() returned error: %v", err)
	}

	// Verify the returned status has expected values based on example XML
	if status.HostCpuArch != "x86_64" {
		t.Errorf("Expected HostCpuArch to be 'x86_64', got '%s'", status.HostCpuArch)
	}

	// Verify memory is calculated correctly (sum of all NUMA cells)
	// From the example XML, we have 4 cells with different memory values:
	// Cell 0: 1056462864 KiB, Cell 1: 1056946772 KiB, Cell 2: 1056946772 KiB, Cell 3: 1056932756 KiB
	expectedMemoryKiB := int64(1056462864 + 1056946772 + 1056946772 + 1056932756)
	expectedMemoryBytes := expectedMemoryKiB * 1024 // Convert KiB to bytes
	expectedMemory := resource.NewQuantity(expectedMemoryBytes, resource.BinarySI)

	if !status.HostMemory.Equal(*expectedMemory) {
		t.Errorf("Expected HostMemory to be %s, got %s", expectedMemory.String(), status.HostMemory.String())
	}

	// Verify CPUs are calculated correctly (sum of all NUMA cells)
	// From the example XML, we have 4 cells with 64 CPUs each
	expectedCpus := resource.NewQuantity(4*64, resource.DecimalSI)

	if !status.HostCpus.Equal(*expectedCpus) {
		t.Errorf("Expected HostCpus to be %s, got %s", expectedCpus.String(), status.HostCpus.String())
	}
}

func TestConvert(t *testing.T) {
	// Create test capabilities data
	capabilities := Capabilities{
		Host: CapabilitiesHost{
			CPU: CapabilitiesHostCPU{
				Arch: "x86_64",
			},
			Topology: CapabilitiesHostTopology{
				CellSpec: CapabilitiesHostTopologyCells{
					Num: 2,
					Cells: []CapabilitiesHostTopologyCell{
						{
							ID: 0,
							Memory: CapabilitiesHostTopologyCellMemory{
								Unit:  "KiB",
								Value: 1024000, // 1GB in KiB
							},
							CPUs: CapabilitiesHostTopologyCellCPUs{
								Num: 4,
							},
						},
						{
							ID: 1,
							Memory: CapabilitiesHostTopologyCellMemory{
								Unit:  "KiB",
								Value: 2048000, // 2GB in KiB
							},
							CPUs: CapabilitiesHostTopologyCellCPUs{
								Num: 8,
							},
						},
					},
				},
			},
		},
	}

	status, err := convert(capabilities)
	if err != nil {
		t.Fatalf("convert() returned error: %v", err)
	}

	// Verify CPU architecture
	if status.HostCpuArch != "x86_64" {
		t.Errorf("Expected HostCpuArch to be 'x86_64', got '%s'", status.HostCpuArch)
	}

	// Verify total memory (1GB + 2GB = 3GB)
	expectedMemoryBytes := int64((1024000 + 2048000) * 1024) // Convert KiB to bytes
	expectedMemory := resource.NewQuantity(expectedMemoryBytes, resource.BinarySI)

	if !status.HostMemory.Equal(*expectedMemory) {
		t.Errorf("Expected HostMemory to be %s, got %s", expectedMemory.String(), status.HostMemory.String())
	}

	// Verify total CPUs (4 + 8 = 12)
	expectedCpus := resource.NewQuantity(12, resource.DecimalSI)

	if !status.HostCpus.Equal(*expectedCpus) {
		t.Errorf("Expected HostCpus to be %s, got %s", expectedCpus.String(), status.HostCpus.String())
	}
}

func TestConvertWithDifferentMemoryUnits(t *testing.T) {
	testCases := []struct {
		name          string
		unit          string
		value         int64
		expectedBytes int64
	}{
		{
			name:          "KiB unit",
			unit:          "KiB",
			value:         1024,
			expectedBytes: 1024 * 1024,
		},
		{
			name:          "MiB unit",
			unit:          "MiB",
			value:         1,
			expectedBytes: 1024 * 1024,
		},
		{
			name:          "GiB unit",
			unit:          "GiB",
			value:         1,
			expectedBytes: 1024 * 1024 * 1024,
		},
	}

	for _, tc := range testCases {
		t.Run(tc.name, func(t *testing.T) {
			capabilities := Capabilities{
				Host: CapabilitiesHost{
					CPU: CapabilitiesHostCPU{
						Arch: "x86_64",
					},
					Topology: CapabilitiesHostTopology{
						CellSpec: CapabilitiesHostTopologyCells{
							Num: 1,
							Cells: []CapabilitiesHostTopologyCell{
								{
									ID: 0,
									Memory: CapabilitiesHostTopologyCellMemory{
										Unit:  tc.unit,
										Value: tc.value,
									},
									CPUs: CapabilitiesHostTopologyCellCPUs{
										Num: 1,
									},
								},
							},
						},
					},
				},
			}

			status, err := convert(capabilities)
			if err != nil {
				t.Fatalf("convert() returned error: %v", err)
			}

			expectedMemory := resource.NewQuantity(tc.expectedBytes, resource.BinarySI)
			if !status.HostMemory.Equal(*expectedMemory) {
				t.Errorf("Expected HostMemory to be %s, got %s", expectedMemory.String(), status.HostMemory.String())
			}
		})
	}
}

func TestConvertWithInvalidMemoryUnit(t *testing.T) {
	capabilities := Capabilities{
		Host: CapabilitiesHost{
			CPU: CapabilitiesHostCPU{
				Arch: "x86_64",
			},
			Topology: CapabilitiesHostTopology{
				CellSpec: CapabilitiesHostTopologyCells{
					Num: 1,
					Cells: []CapabilitiesHostTopologyCell{
						{
							ID: 0,
							Memory: CapabilitiesHostTopologyCellMemory{
								Unit:  "InvalidUnit",
								Value: 1024,
							},
							CPUs: CapabilitiesHostTopologyCellCPUs{
								Num: 1,
							},
						},
					},
				},
			},
		},
	}

	_, err := convert(capabilities)
	if err == nil {
		t.Error("Expected convert() to return error for invalid memory unit, but got nil")
	}

	expectedError := "unknown memory unit InvalidUnit"
	if err.Error() != expectedError {
		t.Errorf("Expected error message '%s', got '%s'", expectedError, err.Error())
	}
}

func TestConvertWithZeroCPUs(t *testing.T) {
	capabilities := Capabilities{
		Host: CapabilitiesHost{
			CPU: CapabilitiesHostCPU{
				Arch: "x86_64",
			},
			Topology: CapabilitiesHostTopology{
				CellSpec: CapabilitiesHostTopologyCells{
					Num: 1,
					Cells: []CapabilitiesHostTopologyCell{
						{
							ID: 0,
							Memory: CapabilitiesHostTopologyCellMemory{
								Unit:  "KiB",
								Value: 1024,
							},
							CPUs: CapabilitiesHostTopologyCellCPUs{
								Num: 0,
							},
						},
					},
				},
			},
		},
	}

	status, err := convert(capabilities)
	if err != nil {
		t.Fatalf("convert() returned unexpected error: %v", err)
	}

	expectedCpus := resource.NewQuantity(0, resource.DecimalSI)
	if !status.HostCpus.Equal(*expectedCpus) {
		t.Errorf("Expected HostCpus to be %s, got %s", expectedCpus.String(), status.HostCpus.String())
	}
}

func TestConvertWithEmptyCells(t *testing.T) {
	capabilities := Capabilities{
		Host: CapabilitiesHost{
			CPU: CapabilitiesHostCPU{
				Arch: "aarch64",
			},
			Topology: CapabilitiesHostTopology{
				CellSpec: CapabilitiesHostTopologyCells{
					Num:   0,
					Cells: []CapabilitiesHostTopologyCell{},
				},
			},
		},
	}

	status, err := convert(capabilities)
	if err != nil {
		t.Fatalf("convert() returned unexpected error: %v", err)
	}

	// Verify CPU architecture is preserved
	if status.HostCpuArch != "aarch64" {
		t.Errorf("Expected HostCpuArch to be 'aarch64', got '%s'", status.HostCpuArch)
	}

	// Verify zero memory and CPUs
	expectedMemory := resource.NewQuantity(0, resource.BinarySI)
	expectedCpus := resource.NewQuantity(0, resource.DecimalSI)

	if !status.HostMemory.Equal(*expectedMemory) {
		t.Errorf("Expected HostMemory to be %s, got %s", expectedMemory.String(), status.HostMemory.String())
	}

	if !status.HostCpus.Equal(*expectedCpus) {
		t.Errorf("Expected HostCpus to be %s, got %s", expectedCpus.String(), status.HostCpus.String())
	}
}

func TestConvertWithRealExampleData(t *testing.T) {
	// Use the embedded example XML to test conversion
	var capabilities Capabilities
	err := xml.Unmarshal(exampleXML, &capabilities)
	if err != nil {
		t.Fatalf("Failed to unmarshal example XML: %v", err)
	}

	status, err := convert(capabilities)
	if err != nil {
		t.Fatalf("convert() returned error with real example data: %v", err)
	}

	// Verify the status is valid
	if status.HostCpuArch == "" {
		t.Error("HostCpuArch should not be empty")
	}

	// Memory and CPU quantities should be positive
	if status.HostMemory.IsZero() {
		t.Error("HostMemory should not be zero")
	}

	if status.HostCpus.IsZero() {
		t.Error("HostCpus should not be zero")
	}

	// Verify specific values from example XML
	if status.HostCpuArch != "x86_64" {
		t.Errorf("Expected HostCpuArch to be 'x86_64', got '%s'", status.HostCpuArch)
	}
}

// Test helper function to create a mock capabilities structure
func createMockCapabilities(arch string, cells []mockCell) Capabilities {
	var capabilitiesCells []CapabilitiesHostTopologyCell

	for _, cell := range cells {
		capabilitiesCells = append(capabilitiesCells, CapabilitiesHostTopologyCell{
			ID: cell.ID,
			Memory: CapabilitiesHostTopologyCellMemory{
				Unit:  cell.MemoryUnit,
				Value: cell.MemoryValue,
			},
			CPUs: CapabilitiesHostTopologyCellCPUs{
				Num: cell.CPUCount,
			},
		})
	}

	return Capabilities{
		Host: CapabilitiesHost{
			CPU: CapabilitiesHostCPU{
				Arch: arch,
			},
			Topology: CapabilitiesHostTopology{
				CellSpec: CapabilitiesHostTopologyCells{
					Num:   len(cells),
					Cells: capabilitiesCells,
				},
			},
		},
	}
}

type mockCell struct {
	ID          int
	MemoryUnit  string
	MemoryValue int64
	CPUCount    int64
}

func TestConvertWithMultipleCellsAndArchitectures(t *testing.T) {
	testCases := []struct {
		name           string
		arch           string
		cells          []mockCell
		expectedMemory int64 // in bytes
		expectedCPUs   int64
	}{
		{
			name: "Single cell x86_64",
			arch: "x86_64",
			cells: []mockCell{
				{ID: 0, MemoryUnit: "KiB", MemoryValue: 1024, CPUCount: 2},
			},
			expectedMemory: 1024 * 1024,
			expectedCPUs:   2,
		},
		{
			name: "Multiple cells aarch64",
			arch: "aarch64",
			cells: []mockCell{
				{ID: 0, MemoryUnit: "MiB", MemoryValue: 512, CPUCount: 4},
				{ID: 1, MemoryUnit: "MiB", MemoryValue: 1024, CPUCount: 8},
			},
			expectedMemory: (512 + 1024) * 1024 * 1024,
			expectedCPUs:   12,
		},
		{
			name: "Mixed memory units",
			arch: "ppc64le",
			cells: []mockCell{
				{ID: 0, MemoryUnit: "KiB", MemoryValue: 1048576, CPUCount: 1}, // 1GB in KiB
				{ID: 1, MemoryUnit: "MiB", MemoryValue: 1024, CPUCount: 2},    // 1GB in MiB
			},
			expectedMemory: 2 * 1024 * 1024 * 1024, // 2GB total
			expectedCPUs:   3,
		},
	}

	for _, tc := range testCases {
		t.Run(tc.name, func(t *testing.T) {
			capabilities := createMockCapabilities(tc.arch, tc.cells)

			status, err := convert(capabilities)
			if err != nil {
				t.Fatalf("convert() returned error: %v", err)
			}

			if status.HostCpuArch != tc.arch {
				t.Errorf("Expected HostCpuArch to be '%s', got '%s'", tc.arch, status.HostCpuArch)
			}

			expectedMemory := resource.NewQuantity(tc.expectedMemory, resource.BinarySI)
			if !status.HostMemory.Equal(*expectedMemory) {
				t.Errorf("Expected HostMemory to be %s, got %s", expectedMemory.String(), status.HostMemory.String())
			}

			expectedCpus := resource.NewQuantity(tc.expectedCPUs, resource.DecimalSI)
			if !status.HostCpus.Equal(*expectedCpus) {
				t.Errorf("Expected HostCpus to be %s, got %s", expectedCpus.String(), status.HostCpus.String())
			}
		})
	}
}
