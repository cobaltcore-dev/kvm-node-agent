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

func TestDomainInfoDeserialization(t *testing.T) {
	// Unmarshal the XML into our DomainInfo struct
	var domainInfo DomainInfo
	err := xml.Unmarshal(exampleXML, &domainInfo)
	if err != nil {
		t.Fatalf("Failed to unmarshal XML: %v", err)
	}

	// Verify domain type and basic attributes
	if domainInfo.Type != "kvm" {
		t.Errorf("Expected domain type to be 'kvm', got '%s'", domainInfo.Type)
	}
	if domainInfo.ID != "654321" {
		t.Errorf("Expected domain ID to be '654321', got '%s'", domainInfo.ID)
	}
	if domainInfo.Name != "instance-12345-abc" {
		t.Errorf("Expected domain name to be 'instance-12345-abc', got '%s'", domainInfo.Name)
	}
	if domainInfo.UUID != "12345-abc" {
		t.Errorf("Expected domain UUID to be '12345-abc', got '%s'", domainInfo.UUID)
	}

	// Verify metadata section
	if domainInfo.Metadata == nil {
		t.Fatal("Expected metadata to be present")
	}
	if domainInfo.Metadata.NovaInstance == nil {
		t.Fatal("Expected nova instance metadata to be present")
	}

	nova := domainInfo.Metadata.NovaInstance
	if nova.Name != "example-12345-abc" {
		t.Errorf("Expected nova name to be 'example-12345-abc', got '%s'", nova.Name)
	}
	if nova.CreationTime != "2025-12-18 00:49:23" {
		t.Errorf("Expected creation time to be '2025-12-18 00:49:23', got '%s'", nova.CreationTime)
	}

	// Verify nova package
	if nova.Package == nil {
		t.Fatal("Expected nova package to be present")
	}
	if nova.Package.Version != "28.1.1" {
		t.Errorf("Expected package version to be '28.1.1', got '%s'", nova.Package.Version)
	}

	// Verify nova flavor
	if nova.Flavor == nil {
		t.Fatal("Expected nova flavor to be present")
	}
	if nova.Flavor.Name != "g_k_c6_m24_v2" {
		t.Errorf("Expected flavor name to be 'g_k_c6_m24_v2', got '%s'", nova.Flavor.Name)
	}
	if nova.Flavor.Memory != 24560 {
		t.Errorf("Expected flavor memory to be 24560, got %d", nova.Flavor.Memory)
	}
	if nova.Flavor.VCPUs != 6 {
		t.Errorf("Expected flavor vcpus to be 6, got %d", nova.Flavor.VCPUs)
	}

	// Verify nova owner
	if nova.Owner == nil {
		t.Fatal("Expected nova owner to be present")
	}
	if nova.Owner.User == nil {
		t.Fatal("Expected nova user to be present")
	}
	if nova.Owner.User.UUID != "12345-abc" {
		t.Errorf("Expected user UUID to be '12345-abc', got '%s'", nova.Owner.User.UUID)
	}
	if nova.Owner.User.Value != "example-user" {
		t.Errorf("Expected user value to be 'example-user', got '%s'", nova.Owner.User.Value)
	}
	if nova.Owner.Project == nil {
		t.Fatal("Expected nova project to be present")
	}
	if nova.Owner.Project.UUID != "12345-abc" {
		t.Errorf("Expected project UUID to be '12345-abc', got '%s'", nova.Owner.Project.UUID)
	}
	if nova.Owner.Project.Value != "example-project" {
		t.Errorf("Expected project value to be 'example-project', got '%s'", nova.Owner.Project.Value)
	}

	// Verify nova root
	if nova.Root == nil {
		t.Fatal("Expected nova root to be present")
	}
	if nova.Root.Type != "image" {
		t.Errorf("Expected root type to be 'image', got '%s'", nova.Root.Type)
	}
	if nova.Root.UUID != "12345-abc" {
		t.Errorf("Expected root UUID to be '12345-abc', got '%s'", nova.Root.UUID)
	}

	// Verify nova ports
	if nova.Ports == nil {
		t.Fatal("Expected nova ports to be present")
	}
	if len(nova.Ports.Ports) != 1 {
		t.Fatalf("Expected 1 nova port, got %d", len(nova.Ports.Ports))
	}
	port := nova.Ports.Ports[0]
	if port.UUID != "12345-abc" {
		t.Errorf("Expected port UUID to be '12345-abc', got '%s'", port.UUID)
	}
	if len(port.IPs) != 1 {
		t.Fatalf("Expected 1 IP address, got %d", len(port.IPs))
	}
	ip := port.IPs[0]
	if ip.Type != "fixed" {
		t.Errorf("Expected IP type to be 'fixed', got '%s'", ip.Type)
	}
	if ip.Address != "0.0.0.0" {
		t.Errorf("Expected IP address to be '0.0.0.0', got '%s'", ip.Address)
	}
	if ip.IPVersion != "4" {
		t.Errorf("Expected IP version to be '4', got '%s'", ip.IPVersion)
	}

	// Verify memory section
	if domainInfo.Memory == nil {
		t.Fatal("Expected memory to be present")
	}
	if domainInfo.Memory.Unit != "KiB" {
		t.Errorf("Expected memory unit to be 'KiB', got '%s'", domainInfo.Memory.Unit)
	}
	if domainInfo.Memory.Value != 25149440 {
		t.Errorf("Expected memory value to be 25149440, got %d", domainInfo.Memory.Value)
	}

	// Verify current memory
	if domainInfo.CurrentMemory == nil {
		t.Fatal("Expected current memory to be present")
	}
	if domainInfo.CurrentMemory.Value != 25149440 {
		t.Errorf("Expected current memory value to be 25149440, got %d", domainInfo.CurrentMemory.Value)
	}

	// Verify memory backing
	if domainInfo.MemoryBacking == nil {
		t.Fatal("Expected memory backing to be present")
	}
	if domainInfo.MemoryBacking.HugePages == nil {
		t.Fatal("Expected huge pages to be present")
	}
	if len(domainInfo.MemoryBacking.HugePages.Pages) != 1 {
		t.Fatalf("Expected 1 huge page configuration, got %d", len(domainInfo.MemoryBacking.HugePages.Pages))
	}
	page := domainInfo.MemoryBacking.HugePages.Pages[0]
	if page.Size != "2048" {
		t.Errorf("Expected page size to be '2048', got '%s'", page.Size)
	}
	if page.Unit != "KiB" {
		t.Errorf("Expected page unit to be 'KiB', got '%s'", page.Unit)
	}
	if page.Nodeset != "0" {
		t.Errorf("Expected page nodeset to be '0', got '%s'", page.Nodeset)
	}

	// Verify VCPU
	if domainInfo.VCPU == nil {
		t.Fatal("Expected VCPU to be present")
	}
	if domainInfo.VCPU.Placement != "static" {
		t.Errorf("Expected VCPU placement to be 'static', got '%s'", domainInfo.VCPU.Placement)
	}
	if domainInfo.VCPU.Value != 6 {
		t.Errorf("Expected VCPU value to be 6, got %d", domainInfo.VCPU.Value)
	}

	// Verify CPU tune
	if domainInfo.CPUTune == nil {
		t.Fatal("Expected CPU tune to be present")
	}
	if len(domainInfo.CPUTune.VCPUPins) != 6 {
		t.Fatalf("Expected 6 VCPU pins, got %d", len(domainInfo.CPUTune.VCPUPins))
	}
	vcpuPin := domainInfo.CPUTune.VCPUPins[0]
	if vcpuPin.VCPU != 0 {
		t.Errorf("Expected first VCPU pin to be for VCPU 0, got %d", vcpuPin.VCPU)
	}
	if vcpuPin.CPUSet != "32-63,160-191" {
		t.Errorf("Expected first VCPU pin cpuset to be '32-63,160-191', got '%s'", vcpuPin.CPUSet)
	}
	if domainInfo.CPUTune.EmulatorPin == nil {
		t.Fatal("Expected emulator pin to be present")
	}
	if domainInfo.CPUTune.EmulatorPin.CPUSet != "32-63,160-191" {
		t.Errorf("Expected emulator pin cpuset to be '32-63,160-191', got '%s'", domainInfo.CPUTune.EmulatorPin.CPUSet)
	}

	// Verify NUMA tune
	if domainInfo.NumaTune == nil {
		t.Fatal("Expected NUMA tune to be present")
	}
	if domainInfo.NumaTune.Memory == nil {
		t.Fatal("Expected NUMA memory to be present")
	}
	if domainInfo.NumaTune.Memory.Mode != "strict" {
		t.Errorf("Expected NUMA memory mode to be 'strict', got '%s'", domainInfo.NumaTune.Memory.Mode)
	}
	if domainInfo.NumaTune.Memory.Nodeset != "1" {
		t.Errorf("Expected NUMA memory nodeset to be '1', got '%s'", domainInfo.NumaTune.Memory.Nodeset)
	}
	if len(domainInfo.NumaTune.MemNodes) != 1 {
		t.Fatalf("Expected 1 NUMA memory node, got %d", len(domainInfo.NumaTune.MemNodes))
	}
	memNode := domainInfo.NumaTune.MemNodes[0]
	if memNode.CellID != 0 {
		t.Errorf("Expected memory node cell ID to be 0, got %d", memNode.CellID)
	}
	if memNode.Mode != "strict" {
		t.Errorf("Expected memory node mode to be 'strict', got '%s'", memNode.Mode)
	}
	if memNode.Nodeset != "1" {
		t.Errorf("Expected memory node nodeset to be '1', got '%s'", memNode.Nodeset)
	}

	// Verify resource
	if domainInfo.Resource == nil {
		t.Fatal("Expected resource to be present")
	}
	if domainInfo.Resource.Partition != "/machine" {
		t.Errorf("Expected resource partition to be '/machine', got '%s'", domainInfo.Resource.Partition)
	}

	// Verify OS
	if domainInfo.OS == nil {
		t.Fatal("Expected OS to be present")
	}
	if domainInfo.OS.Type == nil {
		t.Fatal("Expected OS type to be present")
	}
	if domainInfo.OS.Type.Arch != "x86_64" {
		t.Errorf("Expected OS arch to be 'x86_64', got '%s'", domainInfo.OS.Type.Arch)
	}
	if domainInfo.OS.Type.Value != "hvm" {
		t.Errorf("Expected OS type value to be 'hvm', got '%s'", domainInfo.OS.Type.Value)
	}
	if domainInfo.OS.Kernel != "/usr/share/cloud-hypervisor/CLOUDHV_EFI.fd" {
		t.Errorf("Expected OS kernel path, got '%s'", domainInfo.OS.Kernel)
	}
	if domainInfo.OS.Boot == nil {
		t.Fatal("Expected OS boot to be present")
	}
	if domainInfo.OS.Boot.Dev != "hd" {
		t.Errorf("Expected boot dev to be 'hd', got '%s'", domainInfo.OS.Boot.Dev)
	}

	// Verify CPU
	if domainInfo.CPU == nil {
		t.Fatal("Expected CPU to be present")
	}
	if domainInfo.CPU.Mode != "host-passthrough" {
		t.Errorf("Expected CPU mode to be 'host-passthrough', got '%s'", domainInfo.CPU.Mode)
	}
	if domainInfo.CPU.Topology == nil {
		t.Fatal("Expected CPU topology to be present")
	}
	topo := domainInfo.CPU.Topology
	if topo.Sockets != 6 {
		t.Errorf("Expected 6 sockets, got %d", topo.Sockets)
	}
	if topo.Dies != 1 {
		t.Errorf("Expected 1 die, got %d", topo.Dies)
	}
	if topo.Clusters != 1 {
		t.Errorf("Expected 1 cluster, got %d", topo.Clusters)
	}
	if topo.Cores != 1 {
		t.Errorf("Expected 1 core, got %d", topo.Cores)
	}
	if topo.Threads != 1 {
		t.Errorf("Expected 1 thread, got %d", topo.Threads)
	}

	// Verify CPU NUMA
	if domainInfo.CPU.Numa == nil {
		t.Fatal("Expected CPU NUMA to be present")
	}
	if len(domainInfo.CPU.Numa.Cells) != 1 {
		t.Fatalf("Expected 1 NUMA cell, got %d", len(domainInfo.CPU.Numa.Cells))
	}
	cell := domainInfo.CPU.Numa.Cells[0]
	if cell.ID != 0 {
		t.Errorf("Expected cell ID to be 0, got %d", cell.ID)
	}
	if cell.CPUs != "0-5" {
		t.Errorf("Expected cell CPUs to be '0-5', got '%s'", cell.CPUs)
	}
	if cell.Memory != 25149440 {
		t.Errorf("Expected cell memory to be 25149440, got %d", cell.Memory)
	}
	if cell.Unit != "KiB" {
		t.Errorf("Expected cell unit to be 'KiB', got '%s'", cell.Unit)
	}
	if cell.MemAccess != "shared" {
		t.Errorf("Expected cell memAccess to be 'shared', got '%s'", cell.MemAccess)
	}

	// Verify clock
	if domainInfo.Clock == nil {
		t.Fatal("Expected clock to be present")
	}
	if domainInfo.Clock.Offset != "utc" {
		t.Errorf("Expected clock offset to be 'utc', got '%s'", domainInfo.Clock.Offset)
	}

	// Verify lifecycle actions
	if domainInfo.OnPoweroff != "destroy" {
		t.Errorf("Expected on_poweroff to be 'destroy', got '%s'", domainInfo.OnPoweroff)
	}
	if domainInfo.OnReboot != "restart" {
		t.Errorf("Expected on_reboot to be 'restart', got '%s'", domainInfo.OnReboot)
	}
	if domainInfo.OnCrash != "destroy" {
		t.Errorf("Expected on_crash to be 'destroy', got '%s'", domainInfo.OnCrash)
	}

	// Verify devices
	if domainInfo.Devices == nil {
		t.Fatal("Expected devices to be present")
	}
	if domainInfo.Devices.Emulator != "/usr/bin/cloud-hypervisor" {
		t.Errorf("Expected emulator to be '/usr/bin/cloud-hypervisor', got '%s'", domainInfo.Devices.Emulator)
	}

	// Verify disks
	if len(domainInfo.Devices.Disks) != 1 {
		t.Fatalf("Expected 1 disk, got %d", len(domainInfo.Devices.Disks))
	}
	disk := domainInfo.Devices.Disks[0]
	if disk.Type != "file" {
		t.Errorf("Expected disk type to be 'file', got '%s'", disk.Type)
	}
	if disk.Device != "disk" {
		t.Errorf("Expected disk device to be 'disk', got '%s'", disk.Device)
	}
	if disk.Driver == nil {
		t.Fatal("Expected disk driver to be present")
	}
	if disk.Driver.Type != "raw" {
		t.Errorf("Expected disk driver type to be 'raw', got '%s'", disk.Driver.Type)
	}
	if disk.Driver.Cache != "none" {
		t.Errorf("Expected disk driver cache to be 'none', got '%s'", disk.Driver.Cache)
	}
	if disk.Driver.Discard != "unmap" {
		t.Errorf("Expected disk driver discard to be 'unmap', got '%s'", disk.Driver.Discard)
	}
	if disk.Source == nil {
		t.Fatal("Expected disk source to be present")
	}
	if disk.Source.File != "/var/lib/nova/instances/12345-abc/disk" {
		t.Errorf("Expected disk source file path, got '%s'", disk.Source.File)
	}
	if disk.Target == nil {
		t.Fatal("Expected disk target to be present")
	}
	if disk.Target.Dev != "vda" {
		t.Errorf("Expected disk target dev to be 'vda', got '%s'", disk.Target.Dev)
	}
	if disk.Target.Bus != "virtio" {
		t.Errorf("Expected disk target bus to be 'virtio', got '%s'", disk.Target.Bus)
	}
	if disk.Alias == nil {
		t.Fatal("Expected disk alias to be present")
	}
	if disk.Alias.Name != "virtio-disk0" {
		t.Errorf("Expected disk alias to be 'virtio-disk0', got '%s'", disk.Alias.Name)
	}

	// Verify interfaces
	if len(domainInfo.Devices.Interfaces) != 1 {
		t.Fatalf("Expected 1 interface, got %d", len(domainInfo.Devices.Interfaces))
	}
	iface := domainInfo.Devices.Interfaces[0]
	if iface.Type != "bridge" {
		t.Errorf("Expected interface type to be 'bridge', got '%s'", iface.Type)
	}
	if iface.MAC == nil {
		t.Fatal("Expected interface MAC to be present")
	}
	if iface.MAC.Address != "ab:cd:ef:12:34:56" {
		t.Errorf("Expected MAC address to be 'ab:cd:ef:12:34:56', got '%s'", iface.MAC.Address)
	}
	if iface.Source == nil {
		t.Fatal("Expected interface source to be present")
	}
	if iface.Source.Bridge != "abcdef" {
		t.Errorf("Expected interface bridge to be 'abcdef', got '%s'", iface.Source.Bridge)
	}
	if iface.Target == nil {
		t.Fatal("Expected interface target to be present")
	}
	if iface.Target.Dev != "abcdef" {
		t.Errorf("Expected interface target dev to be 'abcdef', got '%s'", iface.Target.Dev)
	}
	if iface.Model == nil {
		t.Fatal("Expected interface model to be present")
	}
	if iface.Model.Type != "virtio" {
		t.Errorf("Expected interface model type to be 'virtio', got '%s'", iface.Model.Type)
	}
	if iface.Driver == nil {
		t.Fatal("Expected interface driver to be present")
	}
	if iface.Driver.Queues != "1" {
		t.Errorf("Expected interface driver queues to be '1', got '%s'", iface.Driver.Queues)
	}
	if iface.Driver.Packed != "on" {
		t.Errorf("Expected interface driver packed to be 'on', got '%s'", iface.Driver.Packed)
	}
	if iface.MTU == nil {
		t.Fatal("Expected interface MTU to be present")
	}
	if iface.MTU.Size != 8950 {
		t.Errorf("Expected interface MTU size to be 8950, got %d", iface.MTU.Size)
	}
	if iface.Alias == nil {
		t.Fatal("Expected interface alias to be present")
	}
	if iface.Alias.Name != "net_0" {
		t.Errorf("Expected interface alias to be 'net_0', got '%s'", iface.Alias.Name)
	}
	if iface.Address == nil {
		t.Fatal("Expected interface address to be present")
	}
	if iface.Address.Type != "pci" {
		t.Errorf("Expected interface address type to be 'pci', got '%s'", iface.Address.Type)
	}

	// Verify serials
	if len(domainInfo.Devices.Serials) != 1 {
		t.Fatalf("Expected 1 serial, got %d", len(domainInfo.Devices.Serials))
	}
	serial := domainInfo.Devices.Serials[0]
	if serial.Type != "tcp" {
		t.Errorf("Expected serial type to be 'tcp', got '%s'", serial.Type)
	}
	if serial.Source == nil {
		t.Fatal("Expected serial source to be present")
	}
	if serial.Source.Mode != "bind" {
		t.Errorf("Expected serial source mode to be 'bind', got '%s'", serial.Source.Mode)
	}
	if serial.Source.Host != "10.245.239.50" {
		t.Errorf("Expected serial source host to be '10.245.239.50', got '%s'", serial.Source.Host)
	}
	if serial.Source.Service != "10000" {
		t.Errorf("Expected serial source service to be '10000', got '%s'", serial.Source.Service)
	}
	if serial.Protocol == nil {
		t.Fatal("Expected serial protocol to be present")
	}
	if serial.Protocol.Type != "raw" {
		t.Errorf("Expected serial protocol type to be 'raw', got '%s'", serial.Protocol.Type)
	}
	if serial.Log == nil {
		t.Fatal("Expected serial log to be present")
	}
	if serial.Log.File != "/var/lib/nova/instances/12345-abc/console.log" {
		t.Errorf("Expected serial log file path, got '%s'", serial.Log.File)
	}
	if serial.Log.Append != "off" {
		t.Errorf("Expected serial log append to be 'off', got '%s'", serial.Log.Append)
	}
	if serial.Target == nil {
		t.Fatal("Expected serial target to be present")
	}
	if serial.Target.Port != 0 {
		t.Errorf("Expected serial target port to be 0, got %d", serial.Target.Port)
	}
}

func TestDomainInfoRoundTrip(t *testing.T) {
	// Unmarshal into struct
	var domainInfo DomainInfo
	err := xml.Unmarshal(exampleXML, &domainInfo)
	if err != nil {
		t.Fatalf("Failed to unmarshal XML: %v", err)
	}

	// Marshal back to XML
	marshaledXML, err := xml.MarshalIndent(domainInfo, "", "  ")
	if err != nil {
		t.Fatalf("Failed to marshal back to XML: %v", err)
	}

	// Unmarshal the marshaled XML to verify it's still valid
	var roundTripDomainInfo DomainInfo
	err = xml.Unmarshal(marshaledXML, &roundTripDomainInfo)
	if err != nil {
		t.Fatalf("Failed to unmarshal round-trip XML: %v", err)
	}

	// Verify key fields are preserved
	if domainInfo.Type != roundTripDomainInfo.Type {
		t.Errorf("Type mismatch after round trip: expected '%s', got '%s'",
			domainInfo.Type, roundTripDomainInfo.Type)
	}
	if domainInfo.ID != roundTripDomainInfo.ID {
		t.Errorf("ID mismatch after round trip: expected '%s', got '%s'",
			domainInfo.ID, roundTripDomainInfo.ID)
	}
	if domainInfo.Name != roundTripDomainInfo.Name {
		t.Errorf("Name mismatch after round trip: expected '%s', got '%s'",
			domainInfo.Name, roundTripDomainInfo.Name)
	}
	if domainInfo.UUID != roundTripDomainInfo.UUID {
		t.Errorf("UUID mismatch after round trip: expected '%s', got '%s'",
			domainInfo.UUID, roundTripDomainInfo.UUID)
	}

	// Verify nested structures
	if domainInfo.Memory != nil && roundTripDomainInfo.Memory != nil {
		if domainInfo.Memory.Value != roundTripDomainInfo.Memory.Value {
			t.Errorf("Memory value mismatch after round trip: expected %d, got %d",
				domainInfo.Memory.Value, roundTripDomainInfo.Memory.Value)
		}
	}

	if domainInfo.VCPU != nil && roundTripDomainInfo.VCPU != nil {
		if domainInfo.VCPU.Value != roundTripDomainInfo.VCPU.Value {
			t.Errorf("VCPU value mismatch after round trip: expected %d, got %d",
				domainInfo.VCPU.Value, roundTripDomainInfo.VCPU.Value)
		}
	}

	if domainInfo.Devices != nil && roundTripDomainInfo.Devices != nil {
		if len(domainInfo.Devices.Disks) != len(roundTripDomainInfo.Devices.Disks) {
			t.Errorf("Disks count mismatch after round trip: expected %d, got %d",
				len(domainInfo.Devices.Disks), len(roundTripDomainInfo.Devices.Disks))
		}
		if len(domainInfo.Devices.Interfaces) != len(roundTripDomainInfo.Devices.Interfaces) {
			t.Errorf("Interfaces count mismatch after round trip: expected %d, got %d",
				len(domainInfo.Devices.Interfaces), len(roundTripDomainInfo.Devices.Interfaces))
		}
		if len(domainInfo.Devices.Serials) != len(roundTripDomainInfo.Devices.Serials) {
			t.Errorf("Serials count mismatch after round trip: expected %d, got %d",
				len(domainInfo.Devices.Serials), len(roundTripDomainInfo.Devices.Serials))
		}
	}
}
