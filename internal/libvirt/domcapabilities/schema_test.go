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

func TestDomainCapabilitiesDeserialization(t *testing.T) {
	// Unmarshal the XML into our DomainCapabilities struct
	var domainCapabilities DomainCapabilities
	err := xml.Unmarshal(exampleXML, &domainCapabilities)
	if err != nil {
		t.Fatalf("Failed to unmarshal XML: %v", err)
	}

	// Verify domain and arch
	if domainCapabilities.Domain != "ch" {
		t.Errorf("Expected domain to be 'ch', got '%s'", domainCapabilities.Domain)
	}
	if domainCapabilities.Arch != "x86_64" {
		t.Errorf("Expected arch to be 'x86_64', got '%s'", domainCapabilities.Arch)
	}

	// Verify OS section
	if domainCapabilities.OS.Supported != "yes" {
		t.Errorf("Expected OS supported to be 'yes', got '%s'", domainCapabilities.OS.Supported)
	}
	if domainCapabilities.OS.Loader.Supported != "yes" {
		t.Errorf("Expected OS loader supported to be 'yes', got '%s'", domainCapabilities.OS.Loader.Supported)
	}
	if len(domainCapabilities.OS.Loader.Enums) != 1 {
		t.Errorf("Expected 1 loader enum, got %d", len(domainCapabilities.OS.Loader.Enums))
	}
	if domainCapabilities.OS.Loader.Enums[0].Name != "secure" {
		t.Errorf("Expected loader enum name to be 'secure', got '%s'", domainCapabilities.OS.Loader.Enums[0].Name)
	}
	if len(domainCapabilities.OS.Loader.Enums[0].Values) != 1 {
		t.Errorf("Expected 1 value in secure enum, got %d", len(domainCapabilities.OS.Loader.Enums[0].Values))
	}
	if domainCapabilities.OS.Loader.Enums[0].Values[0] != "no" {
		t.Errorf("Expected secure enum value to be 'no', got '%s'", domainCapabilities.OS.Loader.Enums[0].Values[0])
	}

	// Verify CPU section
	if len(domainCapabilities.CPU.Modes) != 4 {
		t.Errorf("Expected 4 CPU modes, got %d", len(domainCapabilities.CPU.Modes))
	}

	// Verify host-passthrough mode
	hostPassthroughMode := domainCapabilities.CPU.Modes[0]
	if hostPassthroughMode.Name != "host-passthrough" {
		t.Errorf("Expected first CPU mode name to be 'host-passthrough', got '%s'", hostPassthroughMode.Name)
	}
	if hostPassthroughMode.Supported != "yes" {
		t.Errorf("Expected host-passthrough mode supported to be 'yes', got '%s'", hostPassthroughMode.Supported)
	}
	if len(hostPassthroughMode.Enums) != 1 {
		t.Errorf("Expected 1 enum for host-passthrough mode, got %d", len(hostPassthroughMode.Enums))
	}
	if hostPassthroughMode.Enums[0].Name != "hostPassthroughMigratable" {
		t.Errorf("Expected enum name to be 'hostPassthroughMigratable', got '%s'", hostPassthroughMode.Enums[0].Name)
	}

	// Verify maximum mode
	maximumMode := domainCapabilities.CPU.Modes[1]
	if maximumMode.Name != "maximum" {
		t.Errorf("Expected second CPU mode name to be 'maximum', got '%s'", maximumMode.Name)
	}
	if maximumMode.Supported != "no" {
		t.Errorf("Expected maximum mode supported to be 'no', got '%s'", maximumMode.Supported)
	}

	// Verify host-model mode
	hostModelMode := domainCapabilities.CPU.Modes[2]
	if hostModelMode.Name != "host-model" {
		t.Errorf("Expected third CPU mode name to be 'host-model', got '%s'", hostModelMode.Name)
	}
	if hostModelMode.Supported != "no" {
		t.Errorf("Expected host-model mode supported to be 'no', got '%s'", hostModelMode.Supported)
	}

	// Verify custom mode
	customMode := domainCapabilities.CPU.Modes[3]
	if customMode.Name != "custom" {
		t.Errorf("Expected fourth CPU mode name to be 'custom', got '%s'", customMode.Name)
	}
	if customMode.Supported != "no" {
		t.Errorf("Expected custom mode supported to be 'no', got '%s'", customMode.Supported)
	}

	// Verify devices section
	if len(domainCapabilities.Devices.Devices) != 1 {
		t.Errorf("Expected 1 device, got %d", len(domainCapabilities.Devices.Devices))
	}
	videoDevice := domainCapabilities.Devices.Devices[0]
	if videoDevice.XMLName.Local != "video" {
		t.Errorf("Expected device name to be 'video', got '%s'", videoDevice.XMLName.Local)
	}
	if videoDevice.Supported != "yes" {
		t.Errorf("Expected video device supported to be 'yes', got '%s'", videoDevice.Supported)
	}
	if len(videoDevice.Enums) != 1 {
		t.Errorf("Expected 1 enum for video device, got %d", len(videoDevice.Enums))
	}
	if videoDevice.Enums[0].Name != "modelType" {
		t.Errorf("Expected video enum name to be 'modelType', got '%s'", videoDevice.Enums[0].Name)
	}
	if len(videoDevice.Enums[0].Values) != 1 {
		t.Errorf("Expected 1 value in modelType enum, got %d", len(videoDevice.Enums[0].Values))
	}
	if videoDevice.Enums[0].Values[0] != "none" {
		t.Errorf("Expected modelType value to be 'none', got '%s'", videoDevice.Enums[0].Values[0])
	}

	// Verify features section
	if len(domainCapabilities.Features.Features) != 2 {
		t.Errorf("Expected 2 features, got %d", len(domainCapabilities.Features.Features))
	}

	// Verify sev feature
	sevFeature := domainCapabilities.Features.Features[0]
	if sevFeature.XMLName.Local != "sev" {
		t.Errorf("Expected first feature name to be 'sev', got '%s'", sevFeature.XMLName.Local)
	}
	if sevFeature.Supported != "no" {
		t.Errorf("Expected sev feature supported to be 'no', got '%s'", sevFeature.Supported)
	}

	// Verify sgx feature
	sgxFeature := domainCapabilities.Features.Features[1]
	if sgxFeature.XMLName.Local != "sgx" {
		t.Errorf("Expected second feature name to be 'sgx', got '%s'", sgxFeature.XMLName.Local)
	}
	if sgxFeature.Supported != "no" {
		t.Errorf("Expected sgx feature supported to be 'no', got '%s'", sgxFeature.Supported)
	}
}

func TestDomainCapabilitiesRoundTrip(t *testing.T) {
	// Unmarshal into struct
	var domainCapabilities DomainCapabilities
	err := xml.Unmarshal(exampleXML, &domainCapabilities)
	if err != nil {
		t.Fatalf("Failed to unmarshal XML: %v", err)
	}

	// Marshal back to XML
	marshaledXML, err := xml.MarshalIndent(domainCapabilities, "", "  ")
	if err != nil {
		t.Fatalf("Failed to marshal back to XML: %v", err)
	}

	// Unmarshal the marshaled XML to verify it's still valid
	var roundTripDomainCapabilities DomainCapabilities
	err = xml.Unmarshal(marshaledXML, &roundTripDomainCapabilities)
	if err != nil {
		t.Fatalf("Failed to unmarshal round-trip XML: %v", err)
	}

	// Verify key fields are preserved
	if domainCapabilities.Domain != roundTripDomainCapabilities.Domain {
		t.Errorf("Domain mismatch after round trip: expected '%s', got '%s'",
			domainCapabilities.Domain, roundTripDomainCapabilities.Domain)
	}
	if domainCapabilities.Arch != roundTripDomainCapabilities.Arch {
		t.Errorf("Arch mismatch after round trip: expected '%s', got '%s'",
			domainCapabilities.Arch, roundTripDomainCapabilities.Arch)
	}
	if domainCapabilities.OS.Supported != roundTripDomainCapabilities.OS.Supported {
		t.Errorf("OS supported mismatch after round trip: expected '%s', got '%s'",
			domainCapabilities.OS.Supported, roundTripDomainCapabilities.OS.Supported)
	}
	if domainCapabilities.OS.Loader.Supported != roundTripDomainCapabilities.OS.Loader.Supported {
		t.Errorf("OS loader supported mismatch after round trip: expected '%s', got '%s'",
			domainCapabilities.OS.Loader.Supported, roundTripDomainCapabilities.OS.Loader.Supported)
	}
	if len(domainCapabilities.CPU.Modes) != len(roundTripDomainCapabilities.CPU.Modes) {
		t.Errorf("CPU modes count mismatch after round trip: expected %d, got %d",
			len(domainCapabilities.CPU.Modes), len(roundTripDomainCapabilities.CPU.Modes))
	}
	if len(domainCapabilities.Devices.Devices) != len(roundTripDomainCapabilities.Devices.Devices) {
		t.Errorf("Devices count mismatch after round trip: expected %d, got %d",
			len(domainCapabilities.Devices.Devices), len(roundTripDomainCapabilities.Devices.Devices))
	}
	if len(domainCapabilities.Features.Features) != len(roundTripDomainCapabilities.Features.Features) {
		t.Errorf("Features count mismatch after round trip: expected %d, got %d",
			len(domainCapabilities.Features.Features), len(roundTripDomainCapabilities.Features.Features))
	}
}
