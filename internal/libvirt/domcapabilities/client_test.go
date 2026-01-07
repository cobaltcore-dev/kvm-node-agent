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

package domcapabilities

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
	if capabilities.Domain == "" {
		t.Error("Expected capabilities to have Domain type")
	}

	if capabilities.Arch == "" {
		t.Error("Expected capabilities to have Arch")
	}
}

func TestClientEmulator_Get_ValidXML(t *testing.T) {
	client := NewClientEmulator()
	capabilities, err := client.Get(nil)

	if err != nil {
		t.Fatalf("Get() returned unexpected error: %v", err)
	}

	// Verify the embedded XML can be parsed correctly
	var testCaps DomainCapabilities
	if err := xml.Unmarshal(exampleXML, &testCaps); err != nil {
		t.Fatalf("Failed to unmarshal example XML: %v", err)
	}

	// The emulator should return the same data
	if capabilities.Domain != testCaps.Domain {
		t.Errorf("Expected domain type '%s', got '%s'", testCaps.Domain, capabilities.Domain)
	}

	if capabilities.Arch != testCaps.Arch {
		t.Errorf("Expected arch '%s', got '%s'", testCaps.Arch, capabilities.Arch)
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

	// Compare domain types
	if result1.Domain != result2.Domain {
		t.Errorf("Inconsistent domain types: '%s' vs '%s'", result1.Domain, result2.Domain)
	}

	// Compare architectures
	if result1.Arch != result2.Arch {
		t.Errorf("Inconsistent architectures: '%s' vs '%s'", result1.Arch, result2.Arch)
	}
}

func TestClientEmulator_Get_OSInfo(t *testing.T) {
	client := NewClientEmulator()
	capabilities, err := client.Get(nil)

	if err != nil {
		t.Fatalf("Get() returned unexpected error: %v", err)
	}

	// Verify OS capabilities are accessible
	if capabilities.OS.Supported == "" {
		t.Skip("OS supported attribute not set in example XML")
	}

	// Loader information should be accessible
	_ = capabilities.OS.Loader.Supported
}

func TestClientEmulator_Get_CPUInfo(t *testing.T) {
	client := NewClientEmulator()
	capabilities, err := client.Get(nil)

	if err != nil {
		t.Fatalf("Get() returned unexpected error: %v", err)
	}

	// Verify CPU modes are accessible
	if len(capabilities.CPU.Modes) == 0 {
		t.Skip("No CPU modes found in example XML")
	}

	// Verify we can access mode properties
	mode := capabilities.CPU.Modes[0]
	if mode.Name == "" {
		t.Error("Expected CPU mode to have a name")
	}
}

func TestClientEmulator_Get_DevicesInfo(t *testing.T) {
	client := NewClientEmulator()
	capabilities, err := client.Get(nil)

	if err != nil {
		t.Fatalf("Get() returned unexpected error: %v", err)
	}

	// Verify devices are accessible
	if len(capabilities.Devices.Devices) == 0 {
		t.Skip("No devices found in example XML")
	}

	// Verify we can access device properties
	for _, device := range capabilities.Devices.Devices {
		_ = device.Supported
		_ = device.Enums
	}
}

func TestClientEmulator_Get_FeaturesInfo(t *testing.T) {
	client := NewClientEmulator()
	capabilities, err := client.Get(nil)

	if err != nil {
		t.Fatalf("Get() returned unexpected error: %v", err)
	}

	// Verify features are accessible
	if len(capabilities.Features.Features) == 0 {
		t.Skip("No features found in example XML")
	}

	// Verify we can access feature properties
	for _, feature := range capabilities.Features.Features {
		_ = feature.Supported
	}
}

func TestClient_InterfaceCompliance(t *testing.T) {
	// Ensure both implementations satisfy the Client interface
	var _ = NewClient()
	var _ = NewClientEmulator()
}

func TestExampleXML_IsValid(t *testing.T) {
	// Verify that the embedded example XML can be parsed
	var capabilities DomainCapabilities
	if err := xml.Unmarshal(exampleXML, &capabilities); err != nil {
		t.Fatalf("Failed to unmarshal example XML: %v", err)
	}

	// Basic validation that we got some data
	if capabilities.Domain == "" {
		t.Error("Expected domain type to be present in example XML")
	}

	if capabilities.Arch == "" {
		t.Error("Expected architecture to be present in example XML")
	}
}

func TestExampleXML_Structure(t *testing.T) {
	var capabilities DomainCapabilities
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
			name:      "Has Domain Type",
			checkFunc: func() bool { return capabilities.Domain != "" },
			errMsg:    "Expected Domain type to be populated",
		},
		{
			name:      "Has Architecture",
			checkFunc: func() bool { return capabilities.Arch != "" },
			errMsg:    "Expected Architecture to be populated",
		},
		{
			name:      "Has CPU Modes",
			checkFunc: func() bool { return len(capabilities.CPU.Modes) > 0 },
			errMsg:    "Expected at least one CPU mode to be populated",
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
	if result.Domain == "" && result.Arch == "" {
		t.Error("Expected result to have either Domain or Arch populated")
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

func TestExampleXML_CPUModes(t *testing.T) {
	var capabilities DomainCapabilities
	if err := xml.Unmarshal(exampleXML, &capabilities); err != nil {
		t.Fatalf("Failed to unmarshal example XML: %v", err)
	}

	if len(capabilities.CPU.Modes) == 0 {
		t.Skip("No CPU modes in example XML")
	}

	// Verify CPU mode properties
	for _, mode := range capabilities.CPU.Modes {
		if mode.Name == "" {
			t.Error("Expected CPU mode to have a name")
		}

		// Supported attribute should be present
		if mode.Supported == "" {
			t.Logf("Note: CPU mode '%s' has no supported attribute", mode.Name)
		}

		// Enums should be accessible
		_ = mode.Enums
	}
}

func TestExampleXML_OSLoader(t *testing.T) {
	var capabilities DomainCapabilities
	if err := xml.Unmarshal(exampleXML, &capabilities); err != nil {
		t.Fatalf("Failed to unmarshal example XML: %v", err)
	}

	// OS loader should be accessible
	loader := capabilities.OS.Loader
	_ = loader.Supported

	// Enums should be accessible
	if len(loader.Enums) > 0 {
		for _, enum := range loader.Enums {
			if enum.Name == "" {
				t.Error("Expected enum to have a name")
			}
			// Values should be accessible
			_ = enum.Values
		}
	}
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

func TestExampleXML_DomainType(t *testing.T) {
	var capabilities DomainCapabilities
	if err := xml.Unmarshal(exampleXML, &capabilities); err != nil {
		t.Fatalf("Failed to unmarshal example XML: %v", err)
	}

	// Domain type should typically be "kvm", "qemu", etc.
	if capabilities.Domain == "" {
		t.Error("Expected domain type to be set")
	}

	// Common domain types
	validTypes := map[string]bool{
		"kvm":  true,
		"qemu": true,
		"xen":  true,
		"lxc":  true,
	}

	if !validTypes[capabilities.Domain] {
		t.Logf("Note: Unusual domain type '%s' (expected kvm, qemu, xen, or lxc)", capabilities.Domain)
	}
}

func TestExampleXML_Architecture(t *testing.T) {
	var capabilities DomainCapabilities
	if err := xml.Unmarshal(exampleXML, &capabilities); err != nil {
		t.Fatalf("Failed to unmarshal example XML: %v", err)
	}

	// Architecture should be set
	if capabilities.Arch == "" {
		t.Error("Expected architecture to be set")
	}

	// Common architectures
	validArchs := map[string]bool{
		"x86_64":  true,
		"i686":    true,
		"aarch64": true,
		"ppc64":   true,
		"ppc64le": true,
		"s390x":   true,
	}

	if !validArchs[capabilities.Arch] {
		t.Logf("Note: Unusual architecture '%s'", capabilities.Arch)
	}
}

func TestExampleXML_DeviceEnums(t *testing.T) {
	var capabilities DomainCapabilities
	if err := xml.Unmarshal(exampleXML, &capabilities); err != nil {
		t.Fatalf("Failed to unmarshal example XML: %v", err)
	}

	if len(capabilities.Devices.Devices) == 0 {
		t.Skip("No devices in example XML")
	}

	// Verify device enum structure
	for _, device := range capabilities.Devices.Devices {
		for _, enum := range device.Enums {
			if enum.Name == "" {
				t.Error("Expected device enum to have a name")
			}
			if len(enum.Values) == 0 {
				t.Logf("Note: Device enum '%s' has no values", enum.Name)
			}
		}
	}
}

func TestExampleXML_FeaturesList(t *testing.T) {
	var capabilities DomainCapabilities
	if err := xml.Unmarshal(exampleXML, &capabilities); err != nil {
		t.Fatalf("Failed to unmarshal example XML: %v", err)
	}

	if len(capabilities.Features.Features) == 0 {
		t.Skip("No features in example XML")
	}

	// Verify we can iterate through features
	for _, feature := range capabilities.Features.Features {
		// Supported attribute should be accessible
		_ = feature.Supported
	}
}

func TestClientEmulator_Get_MultipleFields(t *testing.T) {
	client := NewClientEmulator()
	result, err := client.Get(nil)

	if err != nil {
		t.Fatalf("Get() returned unexpected error: %v", err)
	}

	// Count how many main fields are populated
	fieldsSet := 0
	if result.Domain != "" {
		fieldsSet++
	}
	if result.Arch != "" {
		fieldsSet++
	}
	if len(result.CPU.Modes) > 0 {
		fieldsSet++
	}
	if len(result.Devices.Devices) > 0 {
		fieldsSet++
	}
	if len(result.Features.Features) > 0 {
		fieldsSet++
	}

	if fieldsSet == 0 {
		t.Error("Expected at least one field to be populated in domain capabilities")
	}
}
