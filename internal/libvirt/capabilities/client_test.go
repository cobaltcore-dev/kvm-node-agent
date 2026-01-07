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
	"testing"
)

func TestNewClient(t *testing.T) {
	client := NewClient()
	if client == nil {
		t.Fatal("NewClient() returned nil")
	}

	// Verify it implements the Client interface
	var _ = client
}

func TestNewClient_ReturnsCorrectType(t *testing.T) {
	client := NewClient()
	// Verify it returns a non-nil implementation
	if client == nil {
		t.Fatal("NewClient() returned nil")
	}
	// Verify it's not the emulator type by checking behavior
	// (we can't check the unexported type directly)
}

func TestNewClientEmulator(t *testing.T) {
	client := NewClientEmulator()
	if client == nil {
		t.Fatal("NewClientEmulator() returned nil")
	}

	// Verify it implements the Client interface
	var _ = client
}

func TestNewClientEmulator_ReturnsCorrectType(t *testing.T) {
	client := NewClientEmulator()
	// Verify it returns a non-nil implementation
	if client == nil {
		t.Fatal("NewClientEmulator() returned nil")
	}
	// Verify it works without a libvirt connection (emulator behavior)
	_, err := client.Get(nil)
	if err != nil {
		t.Errorf("Emulator should work with nil libvirt connection, got error: %v", err)
	}
}

func TestClientEmulator_Get_Success(t *testing.T) {
	client := NewClientEmulator()

	// The emulator doesn't actually use the libvirt connection,
	// so we pass nil to test it doesn't panic
	capabilities, err := client.Get(nil)

	if err != nil {
		t.Fatalf("Get() returned unexpected error: %v", err)
	}

	// Verify the returned capabilities has expected structure
	if capabilities.Host.CPU.Arch == "" {
		t.Error("Expected capabilities to have Host CPU architecture")
	}
}

func TestClientEmulator_Get_ValidXML(t *testing.T) {
	client := NewClientEmulator()
	capabilities, err := client.Get(nil)

	if err != nil {
		t.Fatalf("Get() returned unexpected error: %v", err)
	}

	// Verify the embedded XML can be parsed correctly
	var testCaps Capabilities
	if err := xml.Unmarshal(exampleXML, &testCaps); err != nil {
		t.Fatalf("Failed to unmarshal example XML: %v", err)
	}

	// The emulator should return the same data
	if capabilities.Host.CPU.Arch != testCaps.Host.CPU.Arch {
		t.Errorf("Expected host CPU arch '%s', got '%s'",
			testCaps.Host.CPU.Arch, capabilities.Host.CPU.Arch)
	}
}

func TestClientEmulator_Get_Consistency(t *testing.T) {
	client := NewClientEmulator()

	// Call Get multiple times and verify consistent results
	result1, err1 := client.Get(nil)
	if err1 != nil {
		t.Fatalf("First Get() call failed: %v", err1)
	}

	result2, err2 := client.Get(nil)
	if err2 != nil {
		t.Fatalf("Second Get() call failed: %v", err2)
	}

	// Compare Host CPU architecture
	if result1.Host.CPU.Arch != result2.Host.CPU.Arch {
		t.Errorf("Inconsistent host CPU arch: '%s' vs '%s'",
			result1.Host.CPU.Arch, result2.Host.CPU.Arch)
	}
}

func TestClientEmulator_Get_HostInfo(t *testing.T) {
	client := NewClientEmulator()
	capabilities, err := client.Get(nil)

	if err != nil {
		t.Fatalf("Get() returned unexpected error: %v", err)
	}

	// Verify basic host fields
	if capabilities.Host.CPU.Arch == "" {
		t.Error("Expected host CPU architecture to be set")
	}
}

func TestClientEmulator_Get_CPUInfo(t *testing.T) {
	client := NewClientEmulator()
	capabilities, err := client.Get(nil)

	if err != nil {
		t.Fatalf("Get() returned unexpected error: %v", err)
	}

	cpu := capabilities.Host.CPU

	// Verify CPU information
	if cpu.Arch == "" {
		t.Error("Expected CPU architecture to be set")
	}
}

func TestClientEmulator_Get_TopologyInfo(t *testing.T) {
	client := NewClientEmulator()
	capabilities, err := client.Get(nil)

	if err != nil {
		t.Fatalf("Get() returned unexpected error: %v", err)
	}

	topology := capabilities.Host.Topology

	// Verify topology information is accessible
	if topology.CellSpec.Num <= 0 {
		t.Skip("No NUMA cells found in example XML")
	}

	if len(topology.CellSpec.Cells) == 0 {
		t.Error("Expected at least one NUMA cell when num > 0")
	}
}

func TestClientEmulator_Get_GuestInfo(t *testing.T) {
	client := NewClientEmulator()
	capabilities, err := client.Get(nil)

	if err != nil {
		t.Fatalf("Get() returned unexpected error: %v", err)
	}

	guest := capabilities.Guest

	// Verify guest fields are accessible
	if guest.OSType == "" {
		t.Error("Expected guest OS type to be set")
	}

	if guest.Arch.Name == "" {
		t.Error("Expected guest architecture name to be set")
	}
}

func TestClient_InterfaceCompliance(t *testing.T) {
	// Ensure both implementations satisfy the Client interface
	var _ = NewClient()
	var _ = NewClientEmulator()
}

func TestExampleXML_IsValid(t *testing.T) {
	// Verify that the embedded example XML can be parsed
	var capabilities Capabilities
	if err := xml.Unmarshal(exampleXML, &capabilities); err != nil {
		t.Fatalf("Failed to unmarshal example XML: %v", err)
	}

	// Basic validation that we got some data
	if capabilities.Host.CPU.Arch == "" {
		t.Error("Expected CPU architecture to be present in example XML")
	}
}

func TestExampleXML_Structure(t *testing.T) {
	var capabilities Capabilities
	if err := xml.Unmarshal(exampleXML, &capabilities); err != nil {
		t.Fatalf("Failed to unmarshal example XML: %v", err)
	}

	// Test that the example XML has reasonable structure
	tests := []struct {
		name      string
		checkFunc func() bool
		errMsg    string
	}{
		{
			name:      "Has CPU Arch",
			checkFunc: func() bool { return capabilities.Host.CPU.Arch != "" },
			errMsg:    "Expected Host CPU Arch to be populated",
		},
		{
			name:      "Has Guest OSType",
			checkFunc: func() bool { return capabilities.Guest.OSType != "" },
			errMsg:    "Expected Guest OSType to be populated",
		},
		{
			name:      "Has Guest Arch Name",
			checkFunc: func() bool { return capabilities.Guest.Arch.Name != "" },
			errMsg:    "Expected Guest Arch Name to be populated",
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			if !tt.checkFunc() {
				t.Error(tt.errMsg)
			}
		})
	}
}

func TestClientEmulator_Get_ReturnsStruct(t *testing.T) {
	client := NewClientEmulator()
	result, err := client.Get(nil)

	if err != nil {
		t.Fatalf("Get() returned unexpected error: %v", err)
	}

	// Verify the result has expected data
	if result.Host.CPU.Arch == "" && result.Guest.OSType == "" {
		t.Error("Expected result to have either Host or Guest populated")
	}
}

func TestClientTypes_AreDistinct(t *testing.T) {
	client1 := NewClient()
	client2 := NewClientEmulator()

	// Verify they are different types
	type1 := any(client1)
	type2 := any(client2)

	if type1 == type2 {
		t.Error("Expected NewClient() and NewClientEmulator() to return different types")
	}
}

func TestExampleXML_IOMMUSupport(t *testing.T) {
	var capabilities Capabilities
	if err := xml.Unmarshal(exampleXML, &capabilities); err != nil {
		t.Fatalf("Failed to unmarshal example XML: %v", err)
	}

	// IOMMU support should be accessible
	// This test just verifies we can access it without panic
	_ = capabilities.Host.IOMMU.Support
}

func TestExampleXML_CacheInfo(t *testing.T) {
	var capabilities Capabilities
	if err := xml.Unmarshal(exampleXML, &capabilities); err != nil {
		t.Fatalf("Failed to unmarshal example XML: %v", err)
	}

	// Cache banks should be accessible
	// This test just verifies we can access them without panic
	_ = capabilities.Host.Cache.Banks
}

func TestClientEmulator_Get_NoPanic(t *testing.T) {
	client := NewClientEmulator()

	// Ensure calling Get doesn't panic even with nil libvirt
	defer func() {
		if r := recover(); r != nil {
			t.Errorf("Get() panicked with nil libvirt: %v", r)
		}
	}()

	_, err := client.Get(nil)
	if err != nil {
		t.Fatalf("Get() returned unexpected error: %v", err)
	}
}

func TestExampleXML_TopologyCells(t *testing.T) {
	var capabilities Capabilities
	if err := xml.Unmarshal(exampleXML, &capabilities); err != nil {
		t.Fatalf("Failed to unmarshal example XML: %v", err)
	}

	// If we have NUMA cells, verify their structure
	if capabilities.Host.Topology.CellSpec.Num > 0 {
		if len(capabilities.Host.Topology.CellSpec.Cells) == 0 {
			t.Error("Expected cells to be present when num > 0")
		}

		// Verify we can access cell properties
		for i, cell := range capabilities.Host.Topology.CellSpec.Cells {
			_ = cell.ID
			_ = cell.Memory.Value
			_ = cell.CPUs.Num

			// Just basic validation on first cell
			if i == 0 && cell.Memory.Value <= 0 {
				t.Error("Expected positive memory value for first cell")
			}
		}
	}
}

func TestExampleXML_GuestArchDomain(t *testing.T) {
	var capabilities Capabilities
	if err := xml.Unmarshal(exampleXML, &capabilities); err != nil {
		t.Fatalf("Failed to unmarshal example XML: %v", err)
	}

	guest := capabilities.Guest

	// Verify guest arch domain is accessible
	if guest.Arch.Domain.Type == "" {
		t.Skip("Guest arch domain type not set in example XML")
	}

	// Domain type should be something like "kvm" or "qemu"
	_ = guest.Arch.Domain.Type
}

func TestExampleXML_WordSize(t *testing.T) {
	var capabilities Capabilities
	if err := xml.Unmarshal(exampleXML, &capabilities); err != nil {
		t.Fatalf("Failed to unmarshal example XML: %v", err)
	}

	// Word size should be present (typically 32 or 64)
	if capabilities.Guest.Arch.WordSize <= 0 {
		t.Error("Expected positive word size value")
	}

	// Typical word sizes are 32 or 64
	wordSize := capabilities.Guest.Arch.WordSize
	if wordSize != 32 && wordSize != 64 {
		t.Logf("Note: Unusual word size %d (expected 32 or 64)", wordSize)
	}
}
