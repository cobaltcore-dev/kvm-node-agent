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

func TestCapabilitiesDeserialization(t *testing.T) {
	// Unmarshal the XML into our Capabilities struct
	var capabilities Capabilities
	err := xml.Unmarshal(exampleXML, &capabilities)
	if err != nil {
		t.Fatalf("Failed to unmarshal XML: %v", err)
	}

	// Verify host section
	if capabilities.Host.CPU.Arch != "x86_64" {
		t.Errorf("Expected CPU arch to be 'x86_64', got '%s'", capabilities.Host.CPU.Arch)
	}
	if capabilities.Host.IOMMU.Support != "no" {
		t.Errorf("Expected IOMMU support to be 'no', got '%s'", capabilities.Host.IOMMU.Support)
	}

	// Verify topology
	if capabilities.Host.Topology.CellSpec.Num != 4 {
		t.Errorf("Expected 4 cells, got %d", capabilities.Host.Topology.CellSpec.Num)
	}
	if len(capabilities.Host.Topology.CellSpec.Cells) != 4 {
		t.Errorf("Expected 4 cells in slice, got %d", len(capabilities.Host.Topology.CellSpec.Cells))
	}

	// Verify first cell
	firstCell := capabilities.Host.Topology.CellSpec.Cells[0]
	if firstCell.ID != 0 {
		t.Errorf("Expected first cell ID to be 0, got %d", firstCell.ID)
	}
	if firstCell.Memory.Unit != "KiB" {
		t.Errorf("Expected memory unit to be 'KiB', got '%s'", firstCell.Memory.Unit)
	}
	if firstCell.Memory.Value != 1056462864 {
		t.Errorf("Expected memory value to be 1056462864, got %d", firstCell.Memory.Value)
	}
	if len(firstCell.Pages) != 3 {
		t.Errorf("Expected 3 pages entries, got %d", len(firstCell.Pages))
	}
	if firstCell.CPUs.Num != 64 {
		t.Errorf("Expected 64 CPUs, got %d", firstCell.CPUs.Num)
	}
	if len(firstCell.CPUs.CPUs) != 64 {
		t.Errorf("Expected 64 CPUs in slice, got %d", len(firstCell.CPUs.CPUs))
	}

	// Verify first CPU in first cell
	firstCPU := firstCell.CPUs.CPUs[0]
	if firstCPU.ID != 0 {
		t.Errorf("Expected first CPU ID to be 0, got %d", firstCPU.ID)
	}
	if firstCPU.SocketID != 0 {
		t.Errorf("Expected first CPU socket ID to be 0, got %d", firstCPU.SocketID)
	}
	if firstCPU.DieID != 0 {
		t.Errorf("Expected first CPU die ID to be 0, got %d", firstCPU.DieID)
	}
	if firstCPU.ClusterID != 0 {
		t.Errorf("Expected first CPU cluster ID to be 0, got %d", firstCPU.ClusterID)
	}
	if firstCPU.CoreID != 0 {
		t.Errorf("Expected first CPU core ID to be 0, got %d", firstCPU.CoreID)
	}
	if firstCPU.Siblings != "0,128" {
		t.Errorf("Expected first CPU siblings to be '0,128', got '%s'", firstCPU.Siblings)
	}

	// Verify pages information
	if firstCell.Pages[0].Unit != "KiB" {
		t.Errorf("Expected first page unit to be 'KiB', got '%s'", firstCell.Pages[0].Unit)
	}
	if firstCell.Pages[0].Size != 4 {
		t.Errorf("Expected first page size to be 4, got %d", firstCell.Pages[0].Size)
	}
	if firstCell.Pages[0].Value != 11796996 {
		t.Errorf("Expected first page value to be 11796996, got %d", firstCell.Pages[0].Value)
	}

	// Verify distances
	if len(firstCell.Distances.Siblings) != 4 {
		t.Errorf("Expected 4 distance siblings, got %d", len(firstCell.Distances.Siblings))
	}
	if firstCell.Distances.Siblings[0].ID != 0 {
		t.Errorf("Expected first distance sibling ID to be 0, got %d", firstCell.Distances.Siblings[0].ID)
	}
	if firstCell.Distances.Siblings[0].Value != 10 {
		t.Errorf("Expected first distance sibling value to be 10, got %d", firstCell.Distances.Siblings[0].Value)
	}

	// Verify interconnects
	if len(capabilities.Host.Topology.Interconnects.Latencies) == 0 {
		t.Error("Expected non-empty latencies")
	}
	if len(capabilities.Host.Topology.Interconnects.Bandwidths) == 0 {
		t.Error("Expected non-empty bandwidths")
	}

	// Verify first latency
	firstLatency := capabilities.Host.Topology.Interconnects.Latencies[0]
	if firstLatency.Initiator != 0 {
		t.Errorf("Expected first latency initiator to be 0, got %d", firstLatency.Initiator)
	}
	if firstLatency.Target != 0 {
		t.Errorf("Expected first latency target to be 0, got %d", firstLatency.Target)
	}
	if firstLatency.Type != "read" {
		t.Errorf("Expected first latency type to be 'read', got '%s'", firstLatency.Type)
	}
	if firstLatency.Value != 0 {
		t.Errorf("Expected first latency value to be 0, got %d", firstLatency.Value)
	}

	// Verify first bandwidth
	firstBandwidth := capabilities.Host.Topology.Interconnects.Bandwidths[0]
	if firstBandwidth.Initiator != 0 {
		t.Errorf("Expected first bandwidth initiator to be 0, got %d", firstBandwidth.Initiator)
	}
	if firstBandwidth.Target != 0 {
		t.Errorf("Expected first bandwidth target to be 0, got %d", firstBandwidth.Target)
	}
	if firstBandwidth.Type != "read" {
		t.Errorf("Expected first bandwidth type to be 'read', got '%s'", firstBandwidth.Type)
	}
	if firstBandwidth.Value != 288358400 {
		t.Errorf("Expected first bandwidth value to be 288358400, got %d", firstBandwidth.Value)
	}
	if firstBandwidth.Unit != "KiB" {
		t.Errorf("Expected first bandwidth unit to be 'KiB', got '%s'", firstBandwidth.Unit)
	}

	// Verify cache
	if len(capabilities.Host.Cache.Banks) == 0 {
		t.Error("Expected non-empty cache banks")
	}
	firstCacheBank := capabilities.Host.Cache.Banks[0]
	if firstCacheBank.ID != 0 {
		t.Errorf("Expected first cache bank ID to be 0, got %d", firstCacheBank.ID)
	}
	if firstCacheBank.Level != 2 {
		t.Errorf("Expected first cache bank level to be 2, got %d", firstCacheBank.Level)
	}
	if firstCacheBank.Type != "both" {
		t.Errorf("Expected first cache bank type to be 'both', got '%s'", firstCacheBank.Type)
	}
	if firstCacheBank.Size != 2 {
		t.Errorf("Expected first cache bank size to be 2, got %d", firstCacheBank.Size)
	}
	if firstCacheBank.Unit != "MiB" {
		t.Errorf("Expected first cache bank unit to be 'MiB', got '%s'", firstCacheBank.Unit)
	}
	if firstCacheBank.CPUs != "0,128" {
		t.Errorf("Expected first cache bank CPUs to be '0,128', got '%s'", firstCacheBank.CPUs)
	}

	// Verify guest section
	if capabilities.Guest.OSType != "hvm" {
		t.Errorf("Expected guest OS type to be 'hvm', got '%s'", capabilities.Guest.OSType)
	}
	if capabilities.Guest.Arch.Name != "x86_64" {
		t.Errorf("Expected guest arch name to be 'x86_64', got '%s'", capabilities.Guest.Arch.Name)
	}
	if capabilities.Guest.Arch.WordSize != 64 {
		t.Errorf("Expected guest arch word size to be 64, got %d", capabilities.Guest.Arch.WordSize)
	}
	if capabilities.Guest.Arch.Domain.Type != "kvm" {
		t.Errorf("Expected guest arch domain type to be 'kvm', got '%s'", capabilities.Guest.Arch.Domain.Type)
	}
}

func TestCapabilitiesRoundTrip(t *testing.T) {
	// Unmarshal into struct
	var capabilities Capabilities
	err := xml.Unmarshal(exampleXML, &capabilities)
	if err != nil {
		t.Fatalf("Failed to unmarshal XML: %v", err)
	}

	// Marshal back to XML
	marshaledXML, err := xml.MarshalIndent(capabilities, "", "  ")
	if err != nil {
		t.Fatalf("Failed to marshal back to XML: %v", err)
	}

	// Unmarshal the marshaled XML to verify it's still valid
	var roundTripCapabilities Capabilities
	err = xml.Unmarshal(marshaledXML, &roundTripCapabilities)
	if err != nil {
		t.Fatalf("Failed to unmarshal round-trip XML: %v", err)
	}

	// Verify key fields are preserved
	if capabilities.Host.CPU.Arch != roundTripCapabilities.Host.CPU.Arch {
		t.Errorf("CPU arch mismatch after round trip: expected '%s', got '%s'",
			capabilities.Host.CPU.Arch, roundTripCapabilities.Host.CPU.Arch)
	}
	if capabilities.Host.IOMMU.Support != roundTripCapabilities.Host.IOMMU.Support {
		t.Errorf("IOMMU support mismatch after round trip: expected '%s', got '%s'",
			capabilities.Host.IOMMU.Support, roundTripCapabilities.Host.IOMMU.Support)
	}
	if capabilities.Host.Topology.CellSpec.Num != roundTripCapabilities.Host.Topology.CellSpec.Num {
		t.Errorf("Cells num mismatch after round trip: expected %d, got %d",
			capabilities.Host.Topology.CellSpec.Num, roundTripCapabilities.Host.Topology.CellSpec.Num)
	}
	if capabilities.Guest.OSType != roundTripCapabilities.Guest.OSType {
		t.Errorf("Guest OS type mismatch after round trip: expected '%s', got '%s'",
			capabilities.Guest.OSType, roundTripCapabilities.Guest.OSType)
	}
	if capabilities.Guest.Arch.Name != roundTripCapabilities.Guest.Arch.Name {
		t.Errorf("Guest arch name mismatch after round trip: expected '%s', got '%s'",
			capabilities.Guest.Arch.Name, roundTripCapabilities.Guest.Arch.Name)
	}
}
