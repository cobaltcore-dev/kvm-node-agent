/*
SPDX-FileCopyrightText: Copyright 2024 SAP SE or an SAP affiliate company and cobaltcore-dev contributors
SPDX-License-Identifier: Apache-2.0

Licensed under the Apache License, LibVirtVersion 2.0 (the "License");
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
	"fmt"
	"os"
	"sync"
	"time"

	v1 "github.com/cobaltcore-dev/openstack-hypervisor-operator/api/v1"
	"github.com/digitalocean/go-libvirt"
	"github.com/digitalocean/go-libvirt/socket/dialers"
	"k8s.io/apimachinery/pkg/api/resource"
	"sigs.k8s.io/controller-runtime/pkg/client"
	"sigs.k8s.io/controller-runtime/pkg/log"

	"github.com/cobaltcore-dev/kvm-node-agent/internal/libvirt/capabilities"
	"github.com/cobaltcore-dev/kvm-node-agent/internal/libvirt/domcapabilities"
	"github.com/cobaltcore-dev/kvm-node-agent/internal/libvirt/dominfo"
)

type LibVirt struct {
	virt          *libvirt.Libvirt
	client        client.Client
	migrationJobs map[string]context.CancelFunc
	migrationLock sync.Mutex
	version       string
	domains       map[libvirt.ConnectListAllDomainsFlags][]libvirt.Domain

	// Client that connects to libvirt and fetches capabilities of the
	// hypervisor. The capabilities client abstracts the xml parsing away.
	capabilitiesClient capabilities.Client
	// Client that connects to libvirt and fetches domain capabilities
	// of the hypervisor. The domain capabilities client abstracts the
	// xml parsing away.
	domainCapabilitiesClient domcapabilities.Client
	// Client that connects to libvirt and fetches domain information.
	// The domain information client abstracts the xml parsing away.
	domainInfoClient dominfo.Client
}

func NewLibVirt(k client.Client) *LibVirt {
	socketPath := os.Getenv("LIBVIRT_SOCKET")
	if socketPath == "" {
		socketPath = "/run/libvirt/libvirt-sock"
	}
	log.Log.Info("Using libvirt unix domain socket", "socket", socketPath)
	return &LibVirt{
		libvirt.NewWithDialer(
			dialers.NewLocal(
				dialers.WithSocket(socketPath),
				dialers.WithLocalTimeout(15*time.Second),
			),
		),
		k,
		make(map[string]context.CancelFunc),
		sync.Mutex{},
		"N/A",
		make(map[libvirt.ConnectListAllDomainsFlags][]libvirt.Domain, 2),
		capabilities.NewClient(),
		domcapabilities.NewClient(),
		dominfo.NewClient(),
	}
}

func (l *LibVirt) Connect() error {
	// Check if already connected
	if l.virt.IsConnected() {
		return nil
	}

	var libVirtUri = libvirt.ConnectURI("ch:///system")
	if uri, present := os.LookupEnv("LIBVIRT_DEFAULT_URI"); present {
		libVirtUri = libvirt.ConnectURI(uri)
	}
	err := l.virt.ConnectToURI(libVirtUri)
	if err == nil {
		// Update the version
		if version, err := l.virt.ConnectGetVersion(); err != nil {
			log.Log.Error(err, "unable to fetch libvirt version")
		} else {
			major, minor, release := version/1000000, (version/1000)%1000, version%1000
			l.version = fmt.Sprintf("%d.%d.%d", major, minor, release)
		}

		// Run the migration listener in a goroutine
		ctx := log.IntoContext(context.Background(), log.Log.WithName("libvirt-migration-listener"))
		go l.runMigrationListener(ctx)

		// Periodic status thread
		ctx = log.IntoContext(context.Background(), log.Log.WithName("libvirt-status-thread"))
		go l.runStatusThread(ctx)
	}

	return err
}

func (l *LibVirt) Close() error {
	return l.virt.Disconnect()
}

// Add information extracted from the libvirt socket to the hypervisor instance.
// If an error occurs, the instance is returned unmodified. The libvirt
// connection needs to be established before calling this function.
func (l *LibVirt) Process(hv v1.Hypervisor) (v1.Hypervisor, error) {
	processors := []func(v1.Hypervisor) (v1.Hypervisor, error){
		l.addVersion,
		l.addInstancesInfo,
		l.addCapabilities,
		l.addDomainCapabilities,
		l.addAllocationCapacity,
	}
	var err error
	for _, processor := range processors {
		if hv, err = processor(hv); err != nil {
			log.Log.Error(err, "failed to process hypervisor", "step", processor)
			return hv, err
		}
	}
	return hv, nil
}

// Add the libvirt version to the hypervisor instance.
func (l *LibVirt) addVersion(old v1.Hypervisor) (v1.Hypervisor, error) {
	newHv := old
	newHv.Status.LibVirtVersion = l.version
	return newHv, nil
}

// Add the domain flags to the hypervisor instance, i.e. how many
// instances are running and how many are inactive.
func (l *LibVirt) addInstancesInfo(old v1.Hypervisor) (v1.Hypervisor, error) {
	newHv := old
	var instances []v1.Instance

	flags := []libvirt.ConnectListAllDomainsFlags{libvirt.ConnectListDomainsActive, libvirt.ConnectListDomainsInactive}
	for _, flag := range flags {
		for _, domain := range l.domains[flag] {
			instances = append(instances, v1.Instance{
				ID:     GetOpenstackUUID(domain),
				Name:   domain.Name,
				Active: flag == libvirt.ConnectListDomainsActive,
			})
		}
	}

	newHv.Status.Instances = instances
	newHv.Status.NumInstances = len(l.domains)
	return newHv, nil
}

// Call the libvirt capabilities API and add the resulting information
// to the hypervisor capabilities status.
func (l *LibVirt) addCapabilities(old v1.Hypervisor) (v1.Hypervisor, error) {
	newHv := old
	caps, err := l.capabilitiesClient.Get(l.virt)
	if err != nil {
		return old, err
	}
	newHv.Status.Capabilities.HostCpuArch = caps.Host.CPU.Arch
	// Loop over all numa cells to get the total memory + vcpus capacity.
	totalMemory := resource.NewQuantity(0, resource.BinarySI)
	totalCpus := resource.NewQuantity(0, resource.DecimalSI)
	for _, cell := range caps.Host.Topology.CellSpec.Cells {
		mem, err := MemoryToResource(cell.Memory.Value, cell.Memory.Unit)
		if err != nil {
			return old, err
		}
		totalMemory.Add(mem)
		cpu := resource.NewQuantity(cell.CPUs.Num, resource.DecimalSI)
		if cpu == nil {
			return old, fmt.Errorf("invalid CPU count for cell %d", cell.ID)
		}
		totalCpus.Add(*cpu)
	}
	newHv.Status.Capabilities.HostMemory = *totalMemory
	newHv.Status.Capabilities.HostCpus = *totalCpus
	return newHv, nil
}

// Call the libvirt domcapabilities api and add the resulting information
// to the hypervisor domain capabilities status.
func (l *LibVirt) addDomainCapabilities(old v1.Hypervisor) (v1.Hypervisor, error) {
	newHv := old
	domCapabilities, err := l.domainCapabilitiesClient.Get(l.virt)
	if err != nil {
		return old, err
	}

	newHv.Status.DomainCapabilities.Arch = domCapabilities.Arch
	newHv.Status.DomainCapabilities.HypervisorType = domCapabilities.Domain

	// Convert the supported cpu modes into a flat list of supported cpu types.
	// - <mode name="example" supported="yes"><enum name="1"/></mode> becomes
	// "mode/example" and "mode/example/1"
	// - <mode name="example2" supported="no"><enum name="1"/></mode> is ignored
	// - <mode name="example3" supported="yes"></mode> becomes "mode/example3"
	newHv.Status.DomainCapabilities.SupportedCpuModes = []string{}
	for _, cpuMode := range domCapabilities.CPU.Modes {
		if cpuMode.Supported != "yes" {
			continue
		}
		newHv.Status.DomainCapabilities.SupportedCpuModes = append(
			newHv.Status.DomainCapabilities.SupportedCpuModes,
			"mode/"+cpuMode.Name,
		)
		for _, enum := range cpuMode.Enums {
			for _, cpuType := range enum.Values {
				newHv.Status.DomainCapabilities.SupportedCpuModes = append(
					newHv.Status.DomainCapabilities.SupportedCpuModes,
					fmt.Sprintf("mode/%s/%s", cpuMode.Name, cpuType),
				)
			}
		}
	}

	// Convert the supported devices into a flat list.
	// - <video supported="yes"><enum name="v1"/></video>
	// becomes "video" and "video/v1"
	// - <video supported="no"><enum name="v2"/></video> is ignored
	// - <video supported="yes"></video> becomes "video".
	newHv.Status.DomainCapabilities.SupportedDevices = []string{}
	for _, device := range domCapabilities.Devices.Devices {
		if device.Supported != "yes" {
			continue
		}
		newHv.Status.DomainCapabilities.SupportedDevices = append(
			newHv.Status.DomainCapabilities.SupportedDevices,
			device.XMLName.Local,
		)
		for _, enum := range device.Enums {
			for _, deviceType := range enum.Values {
				newHv.Status.DomainCapabilities.SupportedDevices = append(
					newHv.Status.DomainCapabilities.SupportedDevices,
					fmt.Sprintf("%s/%s", device.XMLName.Local, deviceType),
				)
			}
		}
	}

	// Convert the supported features into a flat list.
	newHv.Status.DomainCapabilities.SupportedFeatures = []string{}
	for _, feature := range domCapabilities.Features.Features {
		if feature.Supported == "yes" {
			newHv.Status.DomainCapabilities.SupportedFeatures = append(
				newHv.Status.DomainCapabilities.SupportedFeatures,
				feature.XMLName.Local,
			)
		}
	}

	return newHv, nil
}

// Add total allocation, total capacity, and numa cell information
// to the hypervisor instance, by combining domain infos and hypervisor
// capabilities in libvirt.
func (l *LibVirt) addAllocationCapacity(old v1.Hypervisor) (v1.Hypervisor, error) {
	newHv := old

	// First get all the numa cells from the capabilities
	caps, err := l.capabilitiesClient.Get(l.virt)
	if err != nil {
		return old, err
	}
	totalMemoryCapacity := resource.NewQuantity(0, resource.BinarySI)
	totalCpuCapacity := resource.NewQuantity(0, resource.DecimalSI)
	cellsById := make(map[uint64]v1.Cell)
	for _, cell := range caps.Host.Topology.CellSpec.Cells {
		memoryCapacity, err := MemoryToResource(
			cell.Memory.Value,
			cell.Memory.Unit,
		)
		if err != nil {
			return old, err
		}
		totalMemoryCapacity.Add(memoryCapacity)

		cpuCapacity := *resource.NewQuantity(
			cell.CPUs.Num,
			resource.DecimalSI,
		)
		totalCpuCapacity.Add(cpuCapacity)

		cellsById[cell.ID] = v1.Cell{
			CellID: cell.ID,
			Allocation: map[string]resource.Quantity{
				// Will be updated below when we look at the domain infos.
				"memory": *resource.NewQuantity(0, resource.BinarySI),
				"cpu":    *resource.NewQuantity(0, resource.DecimalSI),
			},
			Capacity: map[string]resource.Quantity{
				"memory": memoryCapacity,
				"cpu":    cpuCapacity,
			},
		}
	}

	// Now get all domain infos to calculate the total allocation.
	domInfos, err := l.domainInfoClient.Get(l.virt)
	if err != nil {
		return old, err
	}
	totalMemoryAlloc := resource.NewQuantity(0, resource.BinarySI)
	totalCpuAlloc := resource.NewQuantity(0, resource.DecimalSI)
	for _, domInfo := range domInfos {
		memAlloc, err := MemoryToResource(
			domInfo.Memory.Value,
			domInfo.Memory.Unit,
		)
		if err != nil {
			return old, err
		}
		totalMemoryAlloc.Add(memAlloc)

		if domInfo.CPUTune == nil {
			return old, fmt.Errorf("missing cpu tune for dom %s", domInfo.Name)
		}
		cpuAlloc := *resource.NewQuantity(
			int64(len(domInfo.CPUTune.VCPUPins)),
			resource.DecimalSI,
		)
		totalCpuAlloc.Add(cpuAlloc)

		// Add memory allocation to the cells this domain is using.
		for _, memoryNode := range domInfo.NumaTune.MemNodes {
			cell, ok := cellsById[memoryNode.CellID]
			if !ok {
				return old, fmt.Errorf(
					"domain %s uses unknown memory cell %d",
					domInfo.Name, memoryNode.CellID,
				)
			}
			memAllocCell := cell.Allocation["memory"]
			memAllocCell.Add(memAlloc)
			cell.Allocation["memory"] = memAllocCell
			cellsById[memoryNode.CellID] = cell
		}

		// Add cpu allocation to the cells this domain is using.
		if domInfo.CPU.Numa == nil {
			return old, fmt.Errorf("missing cpu numa for dom %s", domInfo.Name)
		}
		for _, cpuCell := range domInfo.CPU.Numa.Cells {
			cell, ok := cellsById[cpuCell.ID]
			if !ok {
				return old, fmt.Errorf(
					"domain %s uses unknown cpu cell %d",
					domInfo.Name, cpuCell.ID,
				)
			}
			cpuAllocCell := cell.Allocation["cpu"]
			cpuAllocCell.Add(cpuAlloc)
			cell.Allocation["cpu"] = cpuAllocCell
			cellsById[cpuCell.ID] = cell
		}
	}
	cellsAsSlice := []v1.Cell{}
	for _, cell := range cellsById {
		cellsAsSlice = append(cellsAsSlice, cell)
	}

	newHv.Status.Capacity = make(map[string]resource.Quantity)
	newHv.Status.Capacity["memory"] = *totalMemoryCapacity
	newHv.Status.Capacity["cpu"] = *totalCpuCapacity
	newHv.Status.Allocation = make(map[string]resource.Quantity)
	newHv.Status.Allocation["memory"] = *totalMemoryAlloc
	newHv.Status.Allocation["cpu"] = *totalCpuAlloc
	newHv.Status.Cells = cellsAsSlice
	return newHv, nil
}
