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
	"errors"
	"fmt"
	"os"
	"sync"
	"time"

	v1 "github.com/cobaltcore-dev/openstack-hypervisor-operator/api/v1"
	"github.com/digitalocean/go-libvirt"
	"github.com/digitalocean/go-libvirt/socket/dialers"
	"k8s.io/apimachinery/pkg/api/resource"
	"sigs.k8s.io/controller-runtime/pkg/client"
	logger "sigs.k8s.io/controller-runtime/pkg/log"

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

	// Event channels for domains by their libvirt event id.
	domEventChs map[libvirt.DomainEventID]<-chan any
	// Event listeners for domain events by their own identifier.
	domEventChangeHandlers map[libvirt.DomainEventID]map[string]func(context.Context, any)

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
	logger.Log.Info("Using libvirt unix domain socket", "socket", socketPath)
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
		make(map[libvirt.DomainEventID]<-chan any),
		make(map[libvirt.DomainEventID]map[string]func(context.Context, any)),
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
	if err != nil {
		return err
	}

	// Update the version
	if version, err := l.virt.ConnectGetVersion(); err != nil {
		logger.Log.Error(err, "unable to fetch libvirt version")
	} else {
		major, minor, release := version/1000000, (version/1000)%1000, version%1000
		l.version = fmt.Sprintf("%d.%d.%d", major, minor, release)
	}

	l.WatchDomainChanges(
		libvirt.DomainEventIDLifecycle,
		"lifecycle-handler",
		l.onLifecycleEvent,
	)
	l.WatchDomainChanges(
		libvirt.DomainEventIDMigrationIteration,
		"migration-iteration-handler",
		l.onMigrationIteration,
	)
	l.WatchDomainChanges(
		libvirt.DomainEventIDJobCompleted,
		"job-completed-handler",
		l.onJobCompleted,
	)

	// Start the event loop
	go l.runEventLoop(context.Background())

	return nil
}

func (l *LibVirt) Close() error {
	if err := l.virt.ConnectRegisterCloseCallback(); err != nil {
		return err
	}
	return l.virt.Disconnect()
}

// Run a loop which listens for new events on the subscribed libvirt event
// channels and distributes them to the subscribed listeners.
func (l *LibVirt) runEventLoop(ctx context.Context) {
	log := logger.FromContext(ctx, "libvirt", "event-loop")
	for {
		for eventId, ch := range l.domEventChs {
			select {
			case <-ctx.Done():
				return
			case <-l.virt.Disconnected():
				log.Error(errors.New("libvirt disconnected"), "waiting for reconnection")
				time.Sleep(5 * time.Second)
			case eventPayload, ok := <-ch:
				if !ok {
					err := errors.New("libvirt event channel closed")
					log.Error(err, "eventId", eventId)
					continue
				}
				handlers, exists := l.domEventChangeHandlers[eventId]
				if !exists {
					continue
				}
				for _, handler := range handlers {
					// Process each handler sequentially.
					handler(ctx, eventPayload)
				}
			default:
				// No event available, continue
			}
		}
	}
}

// Watch libvirt domain changes and notify the provided handler.
//
// The provided handlerId should be unique per handler, and is used to
// disambiguate multiple handlers for the same eventId.
//
// Note that the handler is called in a blocking manner, so long-running handlers
// should spawn goroutines if needed.
func (l *LibVirt) WatchDomainChanges(
	eventId libvirt.DomainEventID,
	handlerId string,
	handler func(context.Context, any),
) {

	// Register the handler so that it is called when an event with the provided
	// eventId is received.
	if _, exists := l.domEventChangeHandlers[eventId]; !exists {
		l.domEventChangeHandlers[eventId] = make(map[string]func(context.Context, any))
	}
	l.domEventChangeHandlers[eventId][handlerId] = handler

	// If we are already subscribed to this eventId, nothing more to do.
	// Note: subscribing more than once will be blocked by the libvirt client.
	if _, exists := l.domEventChs[eventId]; exists {
		return
	}
	ch, err := l.virt.SubscribeEvents(context.Background(), eventId, libvirt.OptDomain{})
	if err != nil {
		logger.Log.Error(err, "failed to subscribe to libvirt event", "eventId", eventId)
		return
	}
	l.domEventChs[eventId] = ch
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
			logger.Log.Error(err, "failed to process hypervisor", "step", processor)
			return hv, err
		}
	}
	return hv, nil
}

// Add the libvirt version to the hypervisor instance.
func (l *LibVirt) addVersion(old v1.Hypervisor) (v1.Hypervisor, error) {
	newHv := *old.DeepCopy()
	newHv.Status.LibVirtVersion = l.version
	return newHv, nil
}

// Add the domains to the hypervisor instance, i.e. how many
// instances are running and how many are inactive.
func (l *LibVirt) addInstancesInfo(old v1.Hypervisor) (v1.Hypervisor, error) {
	newHv := *old.DeepCopy()
	var instances []v1.Instance

	flags := []libvirt.ConnectListAllDomainsFlags{
		libvirt.ConnectListDomainsActive,
		libvirt.ConnectListDomainsInactive,
	}

	for _, flag := range flags {
		domains, err := l.domainInfoClient.Get(l.virt, flag)
		if err != nil {
			return old, err
		}
		for _, domain := range domains {
			instances = append(instances, v1.Instance{
				ID:     domain.UUID,
				Name:   domain.Name,
				Active: flag == libvirt.ConnectListDomainsActive,
			})
		}
	}

	newHv.Status.Instances = instances
	newHv.Status.NumInstances = len(instances)
	return newHv, nil
}

// Call the libvirt capabilities API and add the resulting information
// to the hypervisor capabilities status.
func (l *LibVirt) addCapabilities(old v1.Hypervisor) (v1.Hypervisor, error) {
	newHv := *old.DeepCopy()
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
	newHv := *old.DeepCopy()
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
	newHv := *old.DeepCopy()

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
