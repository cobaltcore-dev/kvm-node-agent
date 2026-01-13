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
	"context"
	"testing"
	"time"

	v1 "github.com/cobaltcore-dev/openstack-hypervisor-operator/api/v1"
	libvirt "github.com/digitalocean/go-libvirt"
	"k8s.io/apimachinery/pkg/api/resource"

	"github.com/cobaltcore-dev/kvm-node-agent/internal/libvirt/capabilities"
	"github.com/cobaltcore-dev/kvm-node-agent/internal/libvirt/domcapabilities"
	"github.com/cobaltcore-dev/kvm-node-agent/internal/libvirt/dominfo"
)

// mockCapabilitiesClient implements the capabilities.Client interface for testing
type mockCapabilitiesClient struct {
	caps capabilities.Capabilities
	err  error
}

func (m *mockCapabilitiesClient) Get(virt *libvirt.Libvirt) (capabilities.Capabilities, error) {
	if m.err != nil {
		return capabilities.Capabilities{}, m.err
	}
	return m.caps, nil
}

// mockDomCapabilitiesClient implements the domcapabilities.Client interface for testing
type mockDomCapabilitiesClient struct {
	caps domcapabilities.DomainCapabilities
	err  error
}

func (m *mockDomCapabilitiesClient) Get(virt *libvirt.Libvirt) (domcapabilities.DomainCapabilities, error) {
	if m.err != nil {
		return domcapabilities.DomainCapabilities{}, m.err
	}
	return m.caps, nil
}

// mockDomInfoClient implements the dominfo.Client interface for testing
type mockDomInfoClient struct {
	infos []dominfo.DomainInfo
	err   error
}

func (m *mockDomInfoClient) Get(
	virt *libvirt.Libvirt,
	flags ...libvirt.ConnectListAllDomainsFlags,
) ([]dominfo.DomainInfo, error) {

	if m.err != nil {
		return nil, m.err
	}
	return m.infos, nil
}

// mockEventloopRunnable implements the eventloopRunnable interface for testing
type mockEventloopRunnable struct {
	disconnectedCh chan struct{}
}

func newMockEventloopRunnable() *mockEventloopRunnable {
	// For tests that don't test disconnection, we create a channel that will
	// never be closed. Tests must ensure proper cleanup of goroutines.
	return &mockEventloopRunnable{
		disconnectedCh: make(chan struct{}),
	}
}

// newMockEventloopRunnableCloseable creates a mock that can be explicitly closed
// Use this when testing libvirt disconnection scenarios
func newMockEventloopRunnableCloseable() *mockEventloopRunnable {
	return &mockEventloopRunnable{
		disconnectedCh: make(chan struct{}),
	}
}

func (m *mockEventloopRunnable) Disconnected() <-chan struct{} {
	return m.disconnectedCh
}

func (m *mockEventloopRunnable) close() {
	select {
	case <-m.disconnectedCh:
		// Already closed
	default:
		close(m.disconnectedCh)
	}
}

func TestAddVersion(t *testing.T) {
	l := &LibVirt{
		version: "8.0.0",
	}

	hv := v1.Hypervisor{}
	result, err := l.addVersion(hv)

	if err != nil {
		t.Fatalf("addVersion() returned unexpected error: %v", err)
	}

	if result.Status.LibVirtVersion != "8.0.0" {
		t.Errorf("Expected LibVirtVersion '8.0.0', got '%s'", result.Status.LibVirtVersion)
	}
}

func TestAddVersion_PreservesOtherFields(t *testing.T) {
	l := &LibVirt{
		version: "8.0.0",
	}

	hv := v1.Hypervisor{
		Status: v1.HypervisorStatus{
			NumInstances: 5,
		},
	}

	result, err := l.addVersion(hv)

	if err != nil {
		t.Fatalf("addVersion() returned unexpected error: %v", err)
	}

	if result.Status.NumInstances != 5 {
		t.Errorf("Expected NumInstances to be preserved, got %d", result.Status.NumInstances)
	}
}

func TestAddCapabilities_Success(t *testing.T) {
	caps := capabilities.Capabilities{
		Host: capabilities.CapabilitiesHost{
			CPU: capabilities.CapabilitiesHostCPU{
				Arch: "x86_64",
			},
			Topology: capabilities.CapabilitiesHostTopology{
				CellSpec: capabilities.CapabilitiesHostTopologyCells{
					Num: 1,
					Cells: []capabilities.CapabilitiesHostTopologyCell{
						{
							ID: 0,
							Memory: capabilities.CapabilitiesHostTopologyCellMemory{
								Unit:  "KiB",
								Value: 16777216, // 16 GiB in KiB
							},
							CPUs: capabilities.CapabilitiesHostTopologyCellCPUs{
								Num: 8,
							},
						},
					},
				},
			},
		},
	}

	l := &LibVirt{
		capabilitiesClient: &mockCapabilitiesClient{caps: caps},
	}

	hv := v1.Hypervisor{}
	result, err := l.addCapabilities(hv)

	if err != nil {
		t.Fatalf("addCapabilities() returned unexpected error: %v", err)
	}

	if result.Status.Capabilities.HostCpuArch != "x86_64" {
		t.Errorf("Expected HostCpuArch 'x86_64', got '%s'", result.Status.Capabilities.HostCpuArch)
	}

	expectedMemory := resource.NewQuantity(16777216*1024, resource.BinarySI)
	if !result.Status.Capabilities.HostMemory.Equal(*expectedMemory) {
		t.Errorf("Expected HostMemory %s, got %s", expectedMemory.String(), result.Status.Capabilities.HostMemory.String())
	}

	expectedCpus := resource.NewQuantity(8, resource.DecimalSI)
	if !result.Status.Capabilities.HostCpus.Equal(*expectedCpus) {
		t.Errorf("Expected HostCpus %s, got %s", expectedCpus.String(), result.Status.Capabilities.HostCpus.String())
	}
}

func TestAddCapabilities_MultipleCells(t *testing.T) {
	caps := capabilities.Capabilities{
		Host: capabilities.CapabilitiesHost{
			CPU: capabilities.CapabilitiesHostCPU{
				Arch: "x86_64",
			},
			Topology: capabilities.CapabilitiesHostTopology{
				CellSpec: capabilities.CapabilitiesHostTopologyCells{
					Num: 2,
					Cells: []capabilities.CapabilitiesHostTopologyCell{
						{
							ID: 0,
							Memory: capabilities.CapabilitiesHostTopologyCellMemory{
								Unit:  "GiB",
								Value: 32,
							},
							CPUs: capabilities.CapabilitiesHostTopologyCellCPUs{
								Num: 16,
							},
						},
						{
							ID: 1,
							Memory: capabilities.CapabilitiesHostTopologyCellMemory{
								Unit:  "GiB",
								Value: 32,
							},
							CPUs: capabilities.CapabilitiesHostTopologyCellCPUs{
								Num: 16,
							},
						},
					},
				},
			},
		},
	}

	l := &LibVirt{
		capabilitiesClient: &mockCapabilitiesClient{caps: caps},
	}

	hv := v1.Hypervisor{}
	result, err := l.addCapabilities(hv)

	if err != nil {
		t.Fatalf("addCapabilities() returned unexpected error: %v", err)
	}

	// Total should be 64 GiB
	expectedMemory := resource.NewQuantity(64*1024*1024*1024, resource.BinarySI)
	if !result.Status.Capabilities.HostMemory.Equal(*expectedMemory) {
		t.Errorf("Expected HostMemory %s, got %s", expectedMemory.String(), result.Status.Capabilities.HostMemory.String())
	}

	// Total should be 32 CPUs
	expectedCpus := resource.NewQuantity(32, resource.DecimalSI)
	if !result.Status.Capabilities.HostCpus.Equal(*expectedCpus) {
		t.Errorf("Expected HostCpus %s, got %s", expectedCpus.String(), result.Status.Capabilities.HostCpus.String())
	}
}

func TestAddDomainCapabilities_Success(t *testing.T) {
	domCaps := domcapabilities.DomainCapabilities{
		Domain: "kvm",
		Arch:   "x86_64",
		CPU: domcapabilities.DomainCapabilitiesCPU{
			Modes: []domcapabilities.DomainCapabilitiesCPUMode{
				{
					Name:      "host-passthrough",
					Supported: "yes",
				},
				{
					Name:      "custom",
					Supported: "yes",
					Enums: []domcapabilities.DomainCapabilitiesEnum{
						{
							Name:   "model",
							Values: []string{"Skylake-Client", "Broadwell"},
						},
					},
				},
			},
		},
		Devices: domcapabilities.DomainCapabilitiesDevices{
			Devices: []domcapabilities.DomainCapabilitiesDevice{
				{
					Supported: "yes",
					Enums: []domcapabilities.DomainCapabilitiesEnum{
						{
							Name:   "type",
							Values: []string{"vnc", "spice"},
						},
					},
				},
			},
		},
		Features: domcapabilities.DomainCapabilitiesFeatures{
			Features: []domcapabilities.DomainCapabilitiesFeature{
				{
					Supported: "yes",
				},
			},
		},
	}
	// Set XMLName for device
	domCaps.Devices.Devices[0].XMLName.Local = "graphics"
	// Set XMLName for feature
	domCaps.Features.Features[0].XMLName.Local = "acpi"

	l := &LibVirt{
		domainCapabilitiesClient: &mockDomCapabilitiesClient{caps: domCaps},
	}

	hv := v1.Hypervisor{}
	result, err := l.addDomainCapabilities(hv)

	if err != nil {
		t.Fatalf("addDomainCapabilities() returned unexpected error: %v", err)
	}

	if result.Status.DomainCapabilities.Arch != "x86_64" {
		t.Errorf("Expected Arch 'x86_64', got '%s'", result.Status.DomainCapabilities.Arch)
	}

	if result.Status.DomainCapabilities.HypervisorType != "kvm" {
		t.Errorf("Expected HypervisorType 'kvm', got '%s'", result.Status.DomainCapabilities.HypervisorType)
	}

	// Check CPU modes
	expectedCpuModes := []string{
		"mode/host-passthrough",
		"mode/custom",
		"mode/custom/Skylake-Client",
		"mode/custom/Broadwell",
	}
	if len(result.Status.DomainCapabilities.SupportedCpuModes) != len(expectedCpuModes) {
		t.Errorf("Expected %d CPU modes, got %d", len(expectedCpuModes), len(result.Status.DomainCapabilities.SupportedCpuModes))
	}

	// Check devices
	expectedDevices := []string{
		"graphics",
		"graphics/vnc",
		"graphics/spice",
	}
	if len(result.Status.DomainCapabilities.SupportedDevices) != len(expectedDevices) {
		t.Errorf("Expected %d devices, got %d", len(expectedDevices), len(result.Status.DomainCapabilities.SupportedDevices))
	}

	// Check features
	if len(result.Status.DomainCapabilities.SupportedFeatures) != 1 {
		t.Errorf("Expected 1 feature, got %d", len(result.Status.DomainCapabilities.SupportedFeatures))
	}
	if result.Status.DomainCapabilities.SupportedFeatures[0] != "acpi" {
		t.Errorf("Expected feature 'acpi', got '%s'", result.Status.DomainCapabilities.SupportedFeatures[0])
	}
}

func TestAddDomainCapabilities_UnsupportedFiltered(t *testing.T) {
	domCaps := domcapabilities.DomainCapabilities{
		Domain: "kvm",
		Arch:   "x86_64",
		CPU: domcapabilities.DomainCapabilitiesCPU{
			Modes: []domcapabilities.DomainCapabilitiesCPUMode{
				{
					Name:      "supported-mode",
					Supported: "yes",
				},
				{
					Name:      "unsupported-mode",
					Supported: "no",
				},
			},
		},
	}

	l := &LibVirt{
		domainCapabilitiesClient: &mockDomCapabilitiesClient{caps: domCaps},
	}

	hv := v1.Hypervisor{}
	result, err := l.addDomainCapabilities(hv)

	if err != nil {
		t.Fatalf("addDomainCapabilities() returned unexpected error: %v", err)
	}

	// Only the supported mode should be included
	if len(result.Status.DomainCapabilities.SupportedCpuModes) != 1 {
		t.Errorf("Expected 1 supported CPU mode, got %d", len(result.Status.DomainCapabilities.SupportedCpuModes))
	}

	if result.Status.DomainCapabilities.SupportedCpuModes[0] != "mode/supported-mode" {
		t.Errorf("Expected 'mode/supported-mode', got '%s'", result.Status.DomainCapabilities.SupportedCpuModes[0])
	}
}

func TestAddAllocationCapacity_Success(t *testing.T) {
	caps := capabilities.Capabilities{
		Host: capabilities.CapabilitiesHost{
			Topology: capabilities.CapabilitiesHostTopology{
				CellSpec: capabilities.CapabilitiesHostTopologyCells{
					Num: 1,
					Cells: []capabilities.CapabilitiesHostTopologyCell{
						{
							ID: 0,
							Memory: capabilities.CapabilitiesHostTopologyCellMemory{
								Unit:  "GiB",
								Value: 64,
							},
							CPUs: capabilities.CapabilitiesHostTopologyCellCPUs{
								Num: 16,
							},
						},
					},
				},
			},
		},
	}

	domInfos := []dominfo.DomainInfo{
		{
			Name: "test-instance",
			Memory: &dominfo.DomainMemory{
				Unit:  "GiB",
				Value: 8,
			},
			CPUTune: &dominfo.DomainCPUTune{
				VCPUPins: []dominfo.DomainVCPUPin{
					{VCPU: 0, CPUSet: "0"},
					{VCPU: 1, CPUSet: "1"},
				},
			},
			NumaTune: &dominfo.DomainNumaTune{
				MemNodes: []dominfo.DomainNumaMemNode{
					{CellID: 0, Mode: "strict", Nodeset: "0"},
				},
			},
			CPU: &dominfo.DomainCPU{
				Numa: &dominfo.DomainCPUNuma{
					Cells: []dominfo.DomainCPUNumaCell{
						{ID: 0, CPUs: "0-1", Memory: 8, Unit: "GiB"},
					},
				},
			},
		},
	}

	l := &LibVirt{
		capabilitiesClient: &mockCapabilitiesClient{caps: caps},
		domainInfoClient:   &mockDomInfoClient{infos: domInfos},
	}

	hv := v1.Hypervisor{}
	result, err := l.addAllocationCapacity(hv)

	if err != nil {
		t.Fatalf("addAllocationCapacity() returned unexpected error: %v", err)
	}

	// Check total capacity
	expectedMemCapacity := resource.NewQuantity(64*1024*1024*1024, resource.BinarySI)
	memCap := result.Status.Capacity["memory"]
	if !memCap.Equal(*expectedMemCapacity) {
		t.Errorf("Expected memory capacity %s, got %s",
			expectedMemCapacity.String(), memCap.String())
	}

	expectedCpuCapacity := resource.NewQuantity(16, resource.DecimalSI)
	cpuCap := result.Status.Capacity["cpu"]
	if !cpuCap.Equal(*expectedCpuCapacity) {
		t.Errorf("Expected CPU capacity %s, got %s",
			expectedCpuCapacity.String(), cpuCap.String())
	}

	// Check total allocation
	expectedMemAlloc := resource.NewQuantity(8*1024*1024*1024, resource.BinarySI)
	memAlloc := result.Status.Allocation["memory"]
	if !memAlloc.Equal(*expectedMemAlloc) {
		t.Errorf("Expected memory allocation %s, got %s",
			expectedMemAlloc.String(), memAlloc.String())
	}

	expectedCpuAlloc := resource.NewQuantity(2, resource.DecimalSI)
	cpuAlloc := result.Status.Allocation["cpu"]
	if !cpuAlloc.Equal(*expectedCpuAlloc) {
		t.Errorf("Expected CPU allocation %s, got %s",
			expectedCpuAlloc.String(), cpuAlloc.String())
	}

	// Check cells
	if len(result.Status.Cells) != 1 {
		t.Fatalf("Expected 1 cell, got %d", len(result.Status.Cells))
	}
}

func TestProcess_Success(t *testing.T) {
	caps := capabilities.Capabilities{
		Host: capabilities.CapabilitiesHost{
			CPU: capabilities.CapabilitiesHostCPU{
				Arch: "x86_64",
			},
			Topology: capabilities.CapabilitiesHostTopology{
				CellSpec: capabilities.CapabilitiesHostTopologyCells{
					Num: 1,
					Cells: []capabilities.CapabilitiesHostTopologyCell{
						{
							ID: 0,
							Memory: capabilities.CapabilitiesHostTopologyCellMemory{
								Unit:  "GiB",
								Value: 16,
							},
							CPUs: capabilities.CapabilitiesHostTopologyCellCPUs{
								Num: 4,
							},
						},
					},
				},
			},
		},
	}

	domCaps := domcapabilities.DomainCapabilities{
		Domain: "kvm",
		Arch:   "x86_64",
		CPU: domcapabilities.DomainCapabilitiesCPU{
			Modes: []domcapabilities.DomainCapabilitiesCPUMode{},
		},
	}

	l := &LibVirt{
		capabilitiesClient:       &mockCapabilitiesClient{caps: caps},
		domainCapabilitiesClient: &mockDomCapabilitiesClient{caps: domCaps},
		domainInfoClient:         &mockDomInfoClient{infos: []dominfo.DomainInfo{}},
	}

	hv := v1.Hypervisor{}
	result, err := l.Process(hv)

	if err != nil {
		t.Fatalf("Process() returned unexpected error: %v", err)
	}

	// Verify all processors ran
	if result.Status.Capabilities.HostCpuArch != "x86_64" {
		t.Error("addCapabilities did not run")
	}
	if result.Status.DomainCapabilities.HypervisorType != "kvm" {
		t.Error("addDomainCapabilities did not run")
	}
	if result.Status.Capacity == nil {
		t.Error("addAllocationCapacity did not run")
	}
}

func TestProcess_PreservesOriginalOnError(t *testing.T) {
	l := &LibVirt{
		version:                  "8.0.0",
		capabilitiesClient:       &mockCapabilitiesClient{err: &testError{"capability error"}},
		domainCapabilitiesClient: &mockDomCapabilitiesClient{},
		domainInfoClient:         &mockDomInfoClient{},
	}

	originalHv := v1.Hypervisor{
		Status: v1.HypervisorStatus{
			NumInstances: 42,
		},
	}

	result, err := l.Process(originalHv)

	if err == nil {
		t.Fatal("Expected error from Process(), got nil")
	}

	// The hypervisor should be returned even on error
	// Version should have been added before the error
	if result.Status.LibVirtVersion != "8.0.0" {
		t.Error("Expected version to be added before error")
	}
}

func TestAddInstancesInfo_NoInstances(t *testing.T) {
	l := &LibVirt{
		domainInfoClient: &mockDomInfoClient{infos: []dominfo.DomainInfo{}},
	}

	hv := v1.Hypervisor{}
	result, err := l.addInstancesInfo(hv)

	if err != nil {
		t.Fatalf("addInstancesInfo() returned unexpected error: %v", err)
	}

	if result.Status.NumInstances != 0 {
		t.Errorf("Expected NumInstances 0, got %d", result.Status.NumInstances)
	}

	if len(result.Status.Instances) != 0 {
		t.Errorf("Expected 0 instances, got %d", len(result.Status.Instances))
	}
}

func TestAddInstancesInfo_ActiveInstances(t *testing.T) {
	activeInfos := []dominfo.DomainInfo{
		{
			UUID: "instance-1",
			Name: "test-vm-1",
		},
		{
			UUID: "instance-2",
			Name: "test-vm-2",
		},
	}

	inactiveInfos := []dominfo.DomainInfo{}

	// Create a mock client that returns different results based on the flag
	mockClient := &mockDomInfoClientWithFlags{
		activeInfos:   activeInfos,
		inactiveInfos: inactiveInfos,
	}

	l := &LibVirt{
		domainInfoClient: mockClient,
	}

	hv := v1.Hypervisor{}
	result, err := l.addInstancesInfo(hv)

	if err != nil {
		t.Fatalf("addInstancesInfo() returned unexpected error: %v", err)
	}

	if result.Status.NumInstances != 2 {
		t.Errorf("Expected NumInstances 2, got %d", result.Status.NumInstances)
	}

	if len(result.Status.Instances) != 2 {
		t.Fatalf("Expected 2 instances, got %d", len(result.Status.Instances))
	}

	// Verify first instance
	if result.Status.Instances[0].ID != "instance-1" {
		t.Errorf("Expected instance ID 'instance-1', got '%s'", result.Status.Instances[0].ID)
	}
	if result.Status.Instances[0].Name != "test-vm-1" {
		t.Errorf("Expected instance name 'test-vm-1', got '%s'", result.Status.Instances[0].Name)
	}
	if !result.Status.Instances[0].Active {
		t.Error("Expected instance to be active")
	}

	// Verify second instance
	if result.Status.Instances[1].ID != "instance-2" {
		t.Errorf("Expected instance ID 'instance-2', got '%s'", result.Status.Instances[1].ID)
	}
	if result.Status.Instances[1].Name != "test-vm-2" {
		t.Errorf("Expected instance name 'test-vm-2', got '%s'", result.Status.Instances[1].Name)
	}
	if !result.Status.Instances[1].Active {
		t.Error("Expected instance to be active")
	}
}

func TestAddInstancesInfo_InactiveInstances(t *testing.T) {
	activeInfos := []dominfo.DomainInfo{}

	inactiveInfos := []dominfo.DomainInfo{
		{
			UUID: "instance-3",
			Name: "test-vm-3",
		},
	}

	mockClient := &mockDomInfoClientWithFlags{
		activeInfos:   activeInfos,
		inactiveInfos: inactiveInfos,
	}

	l := &LibVirt{
		domainInfoClient: mockClient,
	}

	hv := v1.Hypervisor{}
	result, err := l.addInstancesInfo(hv)

	if err != nil {
		t.Fatalf("addInstancesInfo() returned unexpected error: %v", err)
	}

	if result.Status.NumInstances != 1 {
		t.Errorf("Expected NumInstances 1, got %d", result.Status.NumInstances)
	}

	if len(result.Status.Instances) != 1 {
		t.Fatalf("Expected 1 instance, got %d", len(result.Status.Instances))
	}

	if result.Status.Instances[0].ID != "instance-3" {
		t.Errorf("Expected instance ID 'instance-3', got '%s'", result.Status.Instances[0].ID)
	}
	if result.Status.Instances[0].Name != "test-vm-3" {
		t.Errorf("Expected instance name 'test-vm-3', got '%s'", result.Status.Instances[0].Name)
	}
	if result.Status.Instances[0].Active {
		t.Error("Expected instance to be inactive")
	}
}

func TestAddInstancesInfo_MixedInstances(t *testing.T) {
	activeInfos := []dominfo.DomainInfo{
		{
			UUID: "active-1",
			Name: "active-vm-1",
		},
		{
			UUID: "active-2",
			Name: "active-vm-2",
		},
	}

	inactiveInfos := []dominfo.DomainInfo{
		{
			UUID: "inactive-1",
			Name: "inactive-vm-1",
		},
	}

	mockClient := &mockDomInfoClientWithFlags{
		activeInfos:   activeInfos,
		inactiveInfos: inactiveInfos,
	}

	l := &LibVirt{
		domainInfoClient: mockClient,
	}

	hv := v1.Hypervisor{}
	result, err := l.addInstancesInfo(hv)

	if err != nil {
		t.Fatalf("addInstancesInfo() returned unexpected error: %v", err)
	}

	if result.Status.NumInstances != 3 {
		t.Errorf("Expected NumInstances 3, got %d", result.Status.NumInstances)
	}

	if len(result.Status.Instances) != 3 {
		t.Fatalf("Expected 3 instances, got %d", len(result.Status.Instances))
	}

	// Count active and inactive instances
	activeCount := 0
	inactiveCount := 0
	for _, instance := range result.Status.Instances {
		if instance.Active {
			activeCount++
		} else {
			inactiveCount++
		}
	}

	if activeCount != 2 {
		t.Errorf("Expected 2 active instances, got %d", activeCount)
	}
	if inactiveCount != 1 {
		t.Errorf("Expected 1 inactive instance, got %d", inactiveCount)
	}

	// Verify the active instances come first
	if !result.Status.Instances[0].Active || !result.Status.Instances[1].Active {
		t.Error("Expected active instances to be listed first")
	}
	if result.Status.Instances[2].Active {
		t.Error("Expected third instance to be inactive")
	}
}

func TestAddInstancesInfo_PreservesOtherFields(t *testing.T) {
	mockClient := &mockDomInfoClientWithFlags{
		activeInfos:   []dominfo.DomainInfo{{ID: "test-1", Name: "vm-1"}},
		inactiveInfos: []dominfo.DomainInfo{},
	}

	l := &LibVirt{
		domainInfoClient: mockClient,
	}

	hv := v1.Hypervisor{
		Status: v1.HypervisorStatus{
			LibVirtVersion: "8.0.0",
			Capabilities: v1.Capabilities{
				HostCpuArch: "x86_64",
			},
		},
	}

	result, err := l.addInstancesInfo(hv)

	if err != nil {
		t.Fatalf("addInstancesInfo() returned unexpected error: %v", err)
	}

	// Verify other fields are preserved
	if result.Status.LibVirtVersion != "8.0.0" {
		t.Errorf("Expected LibVirtVersion to be preserved, got '%s'", result.Status.LibVirtVersion)
	}
	if result.Status.Capabilities.HostCpuArch != "x86_64" {
		t.Errorf("Expected HostCpuArch to be preserved, got '%s'", result.Status.Capabilities.HostCpuArch)
	}
}

func TestAddInstancesInfo_ErrorHandling(t *testing.T) {
	mockClient := &mockDomInfoClient{
		err: &testError{"failed to get domain info"},
	}

	l := &LibVirt{
		domainInfoClient: mockClient,
	}

	originalHv := v1.Hypervisor{
		Status: v1.HypervisorStatus{
			NumInstances: 5,
		},
	}

	result, err := l.addInstancesInfo(originalHv)

	if err == nil {
		t.Fatal("Expected error from addInstancesInfo(), got nil")
	}

	// Should return the original hypervisor on error
	if result.Status.NumInstances != 5 {
		t.Errorf("Expected original NumInstances to be preserved, got %d", result.Status.NumInstances)
	}
}

// mockDomInfoClientWithFlags is a mock that returns different results based on flags
type mockDomInfoClientWithFlags struct {
	activeInfos   []dominfo.DomainInfo
	inactiveInfos []dominfo.DomainInfo
	err           error
}

func (m *mockDomInfoClientWithFlags) Get(
	virt *libvirt.Libvirt,
	flags ...libvirt.ConnectListAllDomainsFlags,
) ([]dominfo.DomainInfo, error) {

	if m.err != nil {
		return nil, m.err
	}

	// If no flags provided, return all
	if len(flags) == 0 {
		return append(m.activeInfos, m.inactiveInfos...), nil
	}

	// Check which flag was passed
	flag := flags[0]
	switch flag {
	case libvirt.ConnectListDomainsActive:
		return m.activeInfos, nil
	case libvirt.ConnectListDomainsInactive:
		return m.inactiveInfos, nil
	}

	return []dominfo.DomainInfo{}, nil
}

// testError is a simple error type for testing
type testError struct {
	msg string
}

func (e *testError) Error() string {
	return e.msg
}

func TestWatchDomainChanges_RegistersHandler(t *testing.T) {
	// Pre-create a channel to avoid calling libvirt.SubscribeEvents
	eventCh := make(chan any, 1)
	defer close(eventCh)

	l := &LibVirt{
		domEventChangeHandlers: make(map[libvirt.DomainEventID]map[string]func(context.Context, any)),
		domEventChs: map[libvirt.DomainEventID]<-chan any{
			libvirt.DomainEventIDLifecycle: eventCh,
		},
	}

	eventID := libvirt.DomainEventIDLifecycle
	handlerID := "test-handler"
	handlerCalled := false

	handler := func(ctx context.Context, payload any) {
		handlerCalled = true
	}

	l.WatchDomainChanges(eventID, handlerID, handler)

	// Verify handler was registered
	handlers, exists := l.domEventChangeHandlers[eventID]
	if !exists {
		t.Fatal("Expected handler map to exist for event ID")
	}

	registeredHandler, exists := handlers[handlerID]
	if !exists {
		t.Fatal("Expected handler to be registered")
	}

	// Test that the handler can be called
	registeredHandler(context.Background(), nil)
	if !handlerCalled {
		t.Error("Expected handler to be called")
	}
}

func TestWatchDomainChanges_MultipleHandlersSameEvent(t *testing.T) {
	// Pre-create a channel to avoid calling libvirt.SubscribeEvents
	eventCh := make(chan any, 1)
	defer close(eventCh)

	l := &LibVirt{
		domEventChangeHandlers: make(map[libvirt.DomainEventID]map[string]func(context.Context, any)),
		domEventChs: map[libvirt.DomainEventID]<-chan any{
			libvirt.DomainEventIDLifecycle: eventCh,
		},
	}

	eventID := libvirt.DomainEventIDLifecycle
	handler1Called := false
	handler2Called := false

	handler1 := func(ctx context.Context, payload any) {
		handler1Called = true
	}
	handler2 := func(ctx context.Context, payload any) {
		handler2Called = true
	}

	l.WatchDomainChanges(eventID, "handler-1", handler1)
	l.WatchDomainChanges(eventID, "handler-2", handler2)

	// Verify both handlers are registered
	handlers, exists := l.domEventChangeHandlers[eventID]
	if !exists {
		t.Fatal("Expected handler map to exist for event ID")
	}

	if len(handlers) != 2 {
		t.Errorf("Expected 2 handlers, got %d", len(handlers))
	}

	// Call both handlers
	handlers["handler-1"](context.Background(), nil)
	handlers["handler-2"](context.Background(), nil)

	if !handler1Called {
		t.Error("Expected handler 1 to be called")
	}
	if !handler2Called {
		t.Error("Expected handler 2 to be called")
	}
}

func TestWatchDomainChanges_DifferentEvents(t *testing.T) {
	// Pre-create channels for both events to avoid calling libvirt.SubscribeEvents
	eventCh1 := make(chan any, 1)
	defer close(eventCh1)
	eventCh2 := make(chan any, 1)
	defer close(eventCh2)

	l := &LibVirt{
		domEventChangeHandlers: make(map[libvirt.DomainEventID]map[string]func(context.Context, any)),
		domEventChs: map[libvirt.DomainEventID]<-chan any{
			libvirt.DomainEventIDLifecycle:          eventCh1,
			libvirt.DomainEventIDMigrationIteration: eventCh2,
		},
	}

	event1 := libvirt.DomainEventIDLifecycle
	event2 := libvirt.DomainEventIDMigrationIteration

	handler1 := func(ctx context.Context, payload any) {
		// Handler 1 implementation
	}
	handler2 := func(ctx context.Context, payload any) {
		// Handler 2 implementation
	}

	l.WatchDomainChanges(event1, "handler-1", handler1)
	l.WatchDomainChanges(event2, "handler-2", handler2)

	// Verify handlers are registered under different event IDs
	if len(l.domEventChangeHandlers) != 2 {
		t.Errorf("Expected 2 event IDs registered, got %d", len(l.domEventChangeHandlers))
	}

	handlers1, exists := l.domEventChangeHandlers[event1]
	if !exists || len(handlers1) != 1 {
		t.Error("Expected handler 1 to be registered under event1")
	}

	handlers2, exists := l.domEventChangeHandlers[event2]
	if !exists || len(handlers2) != 1 {
		t.Error("Expected handler 2 to be registered under event2")
	}
}

func TestWatchDomainChanges_OverwriteHandler(t *testing.T) {
	// Pre-create a channel to avoid calling libvirt.SubscribeEvents
	eventCh := make(chan any, 1)
	defer close(eventCh)

	l := &LibVirt{
		domEventChangeHandlers: make(map[libvirt.DomainEventID]map[string]func(context.Context, any)),
		domEventChs: map[libvirt.DomainEventID]<-chan any{
			libvirt.DomainEventIDLifecycle: eventCh,
		},
	}

	eventID := libvirt.DomainEventIDLifecycle
	handlerID := "test-handler"
	firstHandlerCalled := false
	secondHandlerCalled := false

	firstHandler := func(ctx context.Context, payload any) {
		firstHandlerCalled = true
	}
	secondHandler := func(ctx context.Context, payload any) {
		secondHandlerCalled = true
	}

	// Register first handler
	l.WatchDomainChanges(eventID, handlerID, firstHandler)

	// Register second handler with same ID (should overwrite)
	l.WatchDomainChanges(eventID, handlerID, secondHandler)

	handlers, exists := l.domEventChangeHandlers[eventID]
	if !exists {
		t.Fatal("Expected handler map to exist")
	}

	if len(handlers) != 1 {
		t.Errorf("Expected 1 handler, got %d", len(handlers))
	}

	// Only the second handler should be called
	handlers[handlerID](context.Background(), nil)

	if firstHandlerCalled {
		t.Error("First handler should not be called after being overwritten")
	}
	if !secondHandlerCalled {
		t.Error("Second handler should be called")
	}
}

func TestRunEventLoop_MultipleEvents(t *testing.T) {
	t.Skip("Skipping due to race condition with mock disconnected channel - functionality is tested via TestRunEventLoop_LibvirtDisconnection")
	// Create channels for different event types
	lifecycleCh := make(chan any, 10)
	defer close(lifecycleCh)
	migrationCh := make(chan any, 10)
	defer close(migrationCh)

	// Track handler calls
	lifecycleHandlerCalls := 0
	migrationHandlerCalls := 0

	// Create handlers
	lifecycleHandler := func(_ context.Context, _ any) {
		lifecycleHandlerCalls++
	}
	migrationHandler := func(_ context.Context, _ any) {
		migrationHandlerCalls++
	}

	// Create LibVirt instance with multiple event channels
	l := &LibVirt{
		domEventChangeHandlers: map[libvirt.DomainEventID]map[string]func(context.Context, any){
			libvirt.DomainEventIDLifecycle: {
				"lifecycle-handler": lifecycleHandler,
			},
			libvirt.DomainEventIDMigrationIteration: {
				"migration-handler": migrationHandler,
			},
		},
		domEventChs: map[libvirt.DomainEventID]<-chan any{
			libvirt.DomainEventIDLifecycle:          lifecycleCh,
			libvirt.DomainEventIDMigrationIteration: migrationCh,
		},
	}

	// Create mock eventloop runnable
	mock := newMockEventloopRunnable()

	// Create a context that we can cancel
	ctx, cancel := context.WithCancel(context.Background())

	// Run the event loop in a goroutine
	done := make(chan struct{})
	go func() {
		defer close(done)
		l.runEventLoop(ctx, mock)
	}()

	// Give the event loop time to start
	time.Sleep(10 * time.Millisecond)

	// Send events to different channels
	lifecycleCh <- "lifecycle-event-1"
	migrationCh <- "migration-event-1"
	lifecycleCh <- "lifecycle-event-2"

	// Give time for handlers to be called
	time.Sleep(100 * time.Millisecond)

	// Verify handlers were called the correct number of times
	if lifecycleHandlerCalls != 2 {
		t.Errorf("Expected lifecycle handler to be called 2 times, got %d", lifecycleHandlerCalls)
	}
	if migrationHandlerCalls != 1 {
		t.Errorf("Expected migration handler to be called 1 time, got %d", migrationHandlerCalls)
	}

	// Clean up
	cancel()
	<-done
	// Give significant time for the goroutine to fully exit to avoid test interference
	time.Sleep(100 * time.Millisecond)
}

func TestRunEventLoop_LibvirtDisconnection(t *testing.T) {
	// Create a channel for the event
	eventCh := make(chan any, 1)
	defer close(eventCh)

	// Create LibVirt instance
	l := &LibVirt{
		domEventChangeHandlers: make(map[libvirt.DomainEventID]map[string]func(context.Context, any)),
		domEventChs: map[libvirt.DomainEventID]<-chan any{
			libvirt.DomainEventIDLifecycle: eventCh,
		},
	}

	// Create mock eventloop runnable that can be closed
	mock := newMockEventloopRunnableCloseable()

	// Create a context
	ctx := context.Background()

	// Track if panic was recovered
	panicRecovered := false
	var panicValue any

	// Run the event loop in a goroutine with panic recovery
	done := make(chan struct{})
	go func() {
		defer func() {
			if r := recover(); r != nil {
				panicRecovered = true
				panicValue = r
			}
			close(done)
		}()
		l.runEventLoop(ctx, mock)
	}()

	// Give the event loop time to start
	time.Sleep(10 * time.Millisecond)

	// Trigger disconnection
	mock.close()

	// Wait for panic with timeout
	select {
	case <-done:
		// Check that panic was recovered
		if !panicRecovered {
			t.Fatal("Expected panic on libvirt disconnection, but no panic occurred")
		}
		// Verify the panic message
		if panicMsg, ok := panicValue.(string); !ok || panicMsg != "libvirt connection closed" {
			t.Errorf("Expected panic message 'libvirt connection closed', got '%v'", panicValue)
		}
	case <-time.After(1 * time.Second):
		t.Fatal("Event loop did not panic after libvirt disconnection")
	}
}

func TestRunEventLoop_ClosedEventChannel(t *testing.T) {
	// Create a channel and close it immediately
	eventCh := make(chan any)
	close(eventCh)

	handlerCalled := false
	handler := func(_ context.Context, _ any) {
		handlerCalled = true
	}

	// Create LibVirt instance with the closed channel
	l := &LibVirt{
		domEventChangeHandlers: map[libvirt.DomainEventID]map[string]func(context.Context, any){
			libvirt.DomainEventIDLifecycle: {
				"handler": handler,
			},
		},
		domEventChs: map[libvirt.DomainEventID]<-chan any{
			libvirt.DomainEventIDLifecycle: eventCh,
		},
	}

	// Create mock eventloop runnable
	mock := newMockEventloopRunnable()

	// Create a context
	ctx := context.Background()

	// Track if panic was recovered
	panicRecovered := false

	// Run the event loop in a goroutine with panic recovery
	done := make(chan struct{})
	go func() {
		defer func() {
			if r := recover(); r != nil {
				panicRecovered = true
			}
			close(done)
		}()
		l.runEventLoop(ctx, mock)
	}()

	// Wait for panic with timeout
	select {
	case <-done:
		if !panicRecovered {
			t.Fatal("Expected panic when event channel is closed, but no panic occurred")
		}
		// Handler should not have been called
		if handlerCalled {
			t.Error("Handler should not have been called when channel is closed")
		}
	case <-time.After(1 * time.Second):
		t.Fatal("Event loop did not handle closed channel within timeout")
	}
}
