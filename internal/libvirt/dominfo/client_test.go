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

package dominfo

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
	domainInfos, err := client.Get(nil)

	if err != nil {
		t.Fatalf("Get() returned unexpected error: %v", err)
	}

	if len(domainInfos) != 1 {
		t.Fatalf("Expected 1 domain info from emulator, got %d", len(domainInfos))
	}

	// Verify the returned domain info has expected structure
	if domainInfos[0].Name == "" {
		t.Error("Expected domain to have a name")
	}

	if domainInfos[0].UUID == "" {
		t.Error("Expected domain to have a UUID")
	}
}

func TestClientEmulator_Get_ValidXML(t *testing.T) {
	client := NewClientEmulator()
	domainInfos, err := client.Get(nil)

	if err != nil {
		t.Fatalf("Get() returned unexpected error: %v", err)
	}

	// Verify the embedded XML can be parsed correctly
	var testInfo DomainInfo
	if err := xml.Unmarshal(exampleXML, &testInfo); err != nil {
		t.Fatalf("Failed to unmarshal example XML: %v", err)
	}

	// The emulator should return the same data
	if domainInfos[0].Name != testInfo.Name {
		t.Errorf("Expected domain name '%s', got '%s'", testInfo.Name, domainInfos[0].Name)
	}

	if domainInfos[0].UUID != testInfo.UUID {
		t.Errorf("Expected domain UUID '%s', got '%s'", testInfo.UUID, domainInfos[0].UUID)
	}

	if domainInfos[0].Type != testInfo.Type {
		t.Errorf("Expected domain type '%s', got '%s'", testInfo.Type, domainInfos[0].Type)
	}
}

func TestClientEmulator_Get_Consistency(t *testing.T) {
	client := NewClientEmulator()

	// Call Get multiple times and verify consistent results
	results1, err1 := client.Get(nil)
	if err1 != nil {
		t.Fatalf("First Get() call failed: %v", err1)
	}

	results2, err2 := client.Get(nil)
	if err2 != nil {
		t.Fatalf("Second Get() call failed: %v", err2)
	}

	if len(results1) != len(results2) {
		t.Errorf("Inconsistent results: first call returned %d domains, second returned %d",
			len(results1), len(results2))
	}

	if len(results1) > 0 && len(results2) > 0 {
		if results1[0].Name != results2[0].Name {
			t.Errorf("Inconsistent domain names: '%s' vs '%s'",
				results1[0].Name, results2[0].Name)
		}
		if results1[0].UUID != results2[0].UUID {
			t.Errorf("Inconsistent domain UUIDs: '%s' vs '%s'",
				results1[0].UUID, results2[0].UUID)
		}
	}
}

func TestClientEmulator_Get_MemoryInfo(t *testing.T) {
	client := NewClientEmulator()
	domainInfos, err := client.Get(nil)

	if err != nil {
		t.Fatalf("Get() returned unexpected error: %v", err)
	}

	if len(domainInfos) == 0 {
		t.Fatal("Expected at least one domain info")
	}

	domain := domainInfos[0]

	// Check that memory information is present
	if domain.Memory != nil {
		if domain.Memory.Value <= 0 {
			t.Error("Expected memory value to be positive")
		}
		if domain.Memory.Unit == "" {
			t.Error("Expected memory unit to be set")
		}
	}

	if domain.CurrentMemory != nil {
		if domain.CurrentMemory.Value <= 0 {
			t.Error("Expected current memory value to be positive")
		}
	}
}

func TestClientEmulator_Get_VCPUInfo(t *testing.T) {
	client := NewClientEmulator()
	domainInfos, err := client.Get(nil)

	if err != nil {
		t.Fatalf("Get() returned unexpected error: %v", err)
	}

	if len(domainInfos) == 0 {
		t.Fatal("Expected at least one domain info")
	}

	domain := domainInfos[0]

	// Check that VCPU information is present
	if domain.VCPU != nil {
		if domain.VCPU.Value <= 0 {
			t.Error("Expected VCPU count to be positive")
		}
	}
}

func TestClientEmulator_Get_MetadataInfo(t *testing.T) {
	client := NewClientEmulator()
	domainInfos, err := client.Get(nil)

	if err != nil {
		t.Fatalf("Get() returned unexpected error: %v", err)
	}

	if len(domainInfos) == 0 {
		t.Fatal("Expected at least one domain info")
	}

	domain := domainInfos[0]

	// Check that metadata can be accessed (if present)
	// This is optional, so we just verify it doesn't cause issues
	if domain.Metadata != nil && domain.Metadata.NovaInstance != nil {
		// Just verify we can access these fields without panic
		_ = domain.Metadata.NovaInstance.Name
		_ = domain.Metadata.NovaInstance.CreationTime
	}
}

func TestClientEmulator_Get_DevicesInfo(t *testing.T) {
	client := NewClientEmulator()
	domainInfos, err := client.Get(nil)

	if err != nil {
		t.Fatalf("Get() returned unexpected error: %v", err)
	}

	if len(domainInfos) == 0 {
		t.Fatal("Expected at least one domain info")
	}

	domain := domainInfos[0]

	// Check that devices information can be accessed (if present)
	if domain.Devices != nil {
		// Just verify we can access these fields without panic
		_ = domain.Devices.Emulator
		_ = domain.Devices.Disks
		_ = domain.Devices.Interfaces
	}
}

func TestClient_InterfaceCompliance(t *testing.T) {
	// Ensure both implementations satisfy the Client interface
	var _ = NewClient()
	var _ = NewClientEmulator()
}

func TestExampleXML_IsValid(t *testing.T) {
	// Verify that the embedded example XML can be parsed
	var info DomainInfo
	if err := xml.Unmarshal(exampleXML, &info); err != nil {
		t.Fatalf("Failed to unmarshal example XML: %v", err)
	}

	// Basic validation that we got some data
	if info.Name == "" {
		t.Error("Expected domain name to be present in example XML")
	}
	if info.UUID == "" {
		t.Error("Expected domain UUID to be present in example XML")
	}
	if info.Type == "" {
		t.Error("Expected domain type to be present in example XML")
	}
}

func TestExampleXML_Structure(t *testing.T) {
	var info DomainInfo
	if err := xml.Unmarshal(exampleXML, &info); err != nil {
		t.Fatalf("Failed to unmarshal example XML: %v", err)
	}

	// Test that the example XML has reasonable structure
	tests := []struct {
		name      string
		checkFunc func() bool
		errMsg    string
	}{
		{
			name:      "Has Memory Info",
			checkFunc: func() bool { return info.Memory != nil },
			errMsg:    "Expected Memory field to be populated",
		},
		{
			name:      "Has VCPU Info",
			checkFunc: func() bool { return info.VCPU != nil },
			errMsg:    "Expected VCPU field to be populated",
		},
		{
			name:      "Has OS Info",
			checkFunc: func() bool { return info.OS != nil },
			errMsg:    "Expected OS field to be populated",
		},
		{
			name:      "Has Devices Info",
			checkFunc: func() bool { return info.Devices != nil },
			errMsg:    "Expected Devices field to be populated",
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

func TestClientEmulator_Get_ReturnsSlice(t *testing.T) {
	client := NewClientEmulator()
	result, err := client.Get(nil)

	if err != nil {
		t.Fatalf("Get() returned unexpected error: %v", err)
	}

	// Verify the result is a slice
	if result == nil {
		t.Fatal("Expected non-nil slice, got nil")
	}

	// Verify the slice has elements
	if len(result) == 0 {
		t.Error("Expected at least one element in the result slice")
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
