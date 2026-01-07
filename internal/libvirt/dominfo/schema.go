/*
SPDX-FileCopyrightText: Copyright 2025 SAP SE or an SAP affiliate company and cobaltcore-dev contributors
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

package dominfo

import "encoding/xml"

// DomainInfo as returned from the libvirt dumpxml api.
//
// The format is the same as returned when executing `virsh dumpxml`.
// See: https://www.libvirt.org/manpages/virsh.html#dumpxml
// For another reference see: https://gitlab.com/libvirt/libvirt-go-xml-module/-/blob/v1.11010.0/domain.go#L3237
type DomainInfo struct {
	Type          string               `xml:"type,attr"`
	ID            string               `xml:"id,attr,omitempty"`
	Name          string               `xml:"name"`
	UUID          string               `xml:"uuid"`
	Metadata      *DomainMetadata      `xml:"metadata,omitempty"`
	Memory        *DomainMemory        `xml:"memory,omitempty"`
	CurrentMemory *DomainMemory        `xml:"currentMemory,omitempty"`
	MemoryBacking *DomainMemoryBacking `xml:"memoryBacking,omitempty"`
	VCPU          *DomainVCPU          `xml:"vcpu,omitempty"`
	CPUTune       *DomainCPUTune       `xml:"cputune,omitempty"`
	NumaTune      *DomainNumaTune      `xml:"numatune,omitempty"`
	Resource      *DomainResource      `xml:"resource,omitempty"`
	OS            *DomainOS            `xml:"os,omitempty"`
	CPU           *DomainCPU           `xml:"cpu,omitempty"`
	Clock         *DomainClock         `xml:"clock,omitempty"`
	OnPoweroff    string               `xml:"on_poweroff,omitempty"`
	OnReboot      string               `xml:"on_reboot,omitempty"`
	OnCrash       string               `xml:"on_crash,omitempty"`
	Devices       *DomainDevices       `xml:"devices,omitempty"`
}

// DomainMetadata represents the metadata section containing OpenStack Nova information.
type DomainMetadata struct {
	NovaInstance *NovaInstance `xml:"instance"`
}

// NovaInstance represents OpenStack Nova instance metadata.
type NovaInstance struct {
	XMLName      xml.Name     `xml:"http://openstack.org/xmlns/libvirt/nova/1.1 instance"`
	Package      *NovaPackage `xml:"package,omitempty"`
	Name         string       `xml:"name,omitempty"`
	CreationTime string       `xml:"creationTime,omitempty"`
	Flavor       *NovaFlavor  `xml:"flavor,omitempty"`
	Owner        *NovaOwner   `xml:"owner,omitempty"`
	Root         *NovaRoot    `xml:"root,omitempty"`
	Ports        *NovaPorts   `xml:"ports,omitempty"`
}

// NovaPackage represents the Nova package version.
type NovaPackage struct {
	Version string `xml:"version,attr"`
}

// NovaFlavor represents the instance flavor.
type NovaFlavor struct {
	Name      string `xml:"name,attr"`
	Memory    int    `xml:"memory"`
	Disk      int    `xml:"disk"`
	Swap      int    `xml:"swap"`
	Ephemeral int    `xml:"ephemeral"`
	VCPUs     int    `xml:"vcpus"`
}

// NovaOwner represents the instance owner.
type NovaOwner struct {
	User    *NovaUser    `xml:"user,omitempty"`
	Project *NovaProject `xml:"project,omitempty"`
}

// NovaUser represents the user who owns the instance.
type NovaUser struct {
	UUID  string `xml:"uuid,attr"`
	Value string `xml:",chardata"`
}

// NovaProject represents the project that owns the instance.
type NovaProject struct {
	UUID  string `xml:"uuid,attr"`
	Value string `xml:",chardata"`
}

// NovaRoot represents the root image.
type NovaRoot struct {
	Type string `xml:"type,attr"`
	UUID string `xml:"uuid,attr"`
}

// NovaPorts represents the network ports.
type NovaPorts struct {
	Ports []NovaPort `xml:"port"`
}

// NovaPort represents a network port.
type NovaPort struct {
	UUID string   `xml:"uuid,attr"`
	IPs  []NovaIP `xml:"ip"`
}

// NovaIP represents an IP address.
type NovaIP struct {
	Type      string `xml:"type,attr"`
	Address   string `xml:"address,attr"`
	IPVersion string `xml:"ipVersion,attr"`
}

// DomainMemory represents memory configuration.
type DomainMemory struct {
	Unit  string `xml:"unit,attr"`
	Value int64  `xml:",chardata"`
}

// DomainMemoryBacking represents memory backing configuration.
type DomainMemoryBacking struct {
	HugePages *DomainHugePages `xml:"hugepages,omitempty"`
}

// DomainHugePages represents huge pages configuration.
type DomainHugePages struct {
	Pages []DomainPage `xml:"page"`
}

// DomainPage represents a huge page configuration.
type DomainPage struct {
	Size    string `xml:"size,attr"`
	Unit    string `xml:"unit,attr"`
	Nodeset string `xml:"nodeset,attr,omitempty"`
}

// DomainVCPU represents virtual CPU configuration.
type DomainVCPU struct {
	Placement string `xml:"placement,attr,omitempty"`
	Value     int    `xml:",chardata"`
}

// DomainCPUTune represents CPU tuning configuration.
type DomainCPUTune struct {
	VCPUPins    []DomainVCPUPin `xml:"vcpupin,omitempty"`
	EmulatorPin *DomainCPUPin   `xml:"emulatorpin,omitempty"`
}

// DomainVCPUPin represents a VCPU pinning configuration.
type DomainVCPUPin struct {
	VCPU   int    `xml:"vcpu,attr"`
	CPUSet string `xml:"cpuset,attr"`
}

// DomainCPUPin represents a CPU pinning configuration.
type DomainCPUPin struct {
	CPUSet string `xml:"cpuset,attr"`
}

// DomainNumaTune represents NUMA tuning configuration.
type DomainNumaTune struct {
	Memory   *DomainNumaMemory   `xml:"memory,omitempty"`
	MemNodes []DomainNumaMemNode `xml:"memnode,omitempty"`
}

// DomainNumaMemory represents NUMA memory configuration.
type DomainNumaMemory struct {
	Mode    string `xml:"mode,attr"`
	Nodeset string `xml:"nodeset,attr"`
}

// DomainNumaMemNode represents a NUMA memory node configuration.
type DomainNumaMemNode struct {
	CellID  uint64 `xml:"cellid,attr"`
	Mode    string `xml:"mode,attr"`
	Nodeset string `xml:"nodeset,attr"`
}

// DomainResource represents resource configuration.
type DomainResource struct {
	Partition string `xml:"partition,omitempty"`
}

// DomainOS represents OS configuration.
type DomainOS struct {
	Type   *DomainOSType `xml:"type,omitempty"`
	Kernel string        `xml:"kernel,omitempty"`
	Boot   *DomainBoot   `xml:"boot,omitempty"`
}

// DomainOSType represents the OS type.
type DomainOSType struct {
	Arch  string `xml:"arch,attr"`
	Value string `xml:",chardata"`
}

// DomainBoot represents boot configuration.
type DomainBoot struct {
	Dev string `xml:"dev,attr"`
}

// DomainCPU represents CPU configuration.
type DomainCPU struct {
	Mode     string             `xml:"mode,attr,omitempty"`
	Topology *DomainCPUTopology `xml:"topology,omitempty"`
	Numa     *DomainCPUNuma     `xml:"numa,omitempty"`
}

// DomainCPUTopology represents CPU topology.
type DomainCPUTopology struct {
	Sockets  int `xml:"sockets,attr"`
	Dies     int `xml:"dies,attr"`
	Clusters int `xml:"clusters,attr"`
	Cores    int `xml:"cores,attr"`
	Threads  int `xml:"threads,attr"`
}

// DomainCPUNuma represents CPU NUMA configuration.
type DomainCPUNuma struct {
	Cells []DomainCPUNumaCell `xml:"cell"`
}

// DomainCPUNumaCell represents a NUMA cell.
type DomainCPUNumaCell struct {
	ID        uint64 `xml:"id,attr"`
	CPUs      string `xml:"cpus,attr"`
	Memory    uint64 `xml:"memory,attr"`
	Unit      string `xml:"unit,attr"`
	MemAccess string `xml:"memAccess,attr,omitempty"`
}

// DomainClock represents clock configuration.
type DomainClock struct {
	Offset string `xml:"offset,attr"`
}

// DomainDevices represents all devices.
type DomainDevices struct {
	Emulator   string            `xml:"emulator,omitempty"`
	Disks      []DomainDisk      `xml:"disk,omitempty"`
	Interfaces []DomainInterface `xml:"interface,omitempty"`
	Serials    []DomainSerial    `xml:"serial,omitempty"`
}

// DomainDisk represents a disk device.
type DomainDisk struct {
	Type   string            `xml:"type,attr"`
	Device string            `xml:"device,attr"`
	Driver *DomainDiskDriver `xml:"driver,omitempty"`
	Source *DomainDiskSource `xml:"source,omitempty"`
	Target *DomainDiskTarget `xml:"target,omitempty"`
	Alias  *DomainAlias      `xml:"alias,omitempty"`
}

// DomainDiskDriver represents disk driver configuration.
type DomainDiskDriver struct {
	Type    string `xml:"type,attr"`
	Cache   string `xml:"cache,attr,omitempty"`
	Discard string `xml:"discard,attr,omitempty"`
}

// DomainDiskSource represents disk source.
type DomainDiskSource struct {
	File string `xml:"file,attr,omitempty"`
}

// DomainDiskTarget represents disk target.
type DomainDiskTarget struct {
	Dev string `xml:"dev,attr"`
	Bus string `xml:"bus,attr"`
}

// DomainInterface represents a network interface.
type DomainInterface struct {
	Type    string                 `xml:"type,attr"`
	MAC     *DomainInterfaceMAC    `xml:"mac,omitempty"`
	Source  *DomainInterfaceSource `xml:"source,omitempty"`
	Target  *DomainInterfaceTarget `xml:"target,omitempty"`
	Model   *DomainInterfaceModel  `xml:"model,omitempty"`
	Driver  *DomainInterfaceDriver `xml:"driver,omitempty"`
	MTU     *DomainInterfaceMTU    `xml:"mtu,omitempty"`
	Alias   *DomainAlias           `xml:"alias,omitempty"`
	Address *DomainAddress         `xml:"address,omitempty"`
}

// DomainInterfaceMAC represents MAC address.
type DomainInterfaceMAC struct {
	Address string `xml:"address,attr"`
}

// DomainInterfaceSource represents network source.
type DomainInterfaceSource struct {
	Bridge string `xml:"bridge,attr,omitempty"`
}

// DomainInterfaceTarget represents network target.
type DomainInterfaceTarget struct {
	Dev string `xml:"dev,attr"`
}

// DomainInterfaceModel represents network model.
type DomainInterfaceModel struct {
	Type string `xml:"type,attr"`
}

// DomainInterfaceDriver represents network driver.
type DomainInterfaceDriver struct {
	Queues string `xml:"queues,attr,omitempty"`
	Packed string `xml:"packed,attr,omitempty"`
}

// DomainInterfaceMTU represents MTU configuration.
type DomainInterfaceMTU struct {
	Size int `xml:"size,attr"`
}

// DomainSerial represents a serial device.
type DomainSerial struct {
	Type     string                `xml:"type,attr"`
	Source   *DomainSerialSource   `xml:"source,omitempty"`
	Protocol *DomainSerialProtocol `xml:"protocol,omitempty"`
	Log      *DomainSerialLog      `xml:"log,omitempty"`
	Target   *DomainSerialTarget   `xml:"target,omitempty"`
}

// DomainSerialSource represents serial source.
type DomainSerialSource struct {
	Mode    string `xml:"mode,attr"`
	Host    string `xml:"host,attr"`
	Service string `xml:"service,attr"`
}

// DomainSerialProtocol represents serial protocol.
type DomainSerialProtocol struct {
	Type string `xml:"type,attr"`
}

// DomainSerialLog represents serial log configuration.
type DomainSerialLog struct {
	File   string `xml:"file,attr"`
	Append string `xml:"append,attr"`
}

// DomainSerialTarget represents serial target.
type DomainSerialTarget struct {
	Port int `xml:"port,attr"`
}

// DomainAlias represents a device alias.
type DomainAlias struct {
	Name string `xml:"name,attr"`
}

// DomainAddress represents a device address.
type DomainAddress struct {
	Type     string `xml:"type,attr"`
	Domain   string `xml:"domain,attr,omitempty"`
	Bus      string `xml:"bus,attr,omitempty"`
	Slot     string `xml:"slot,attr,omitempty"`
	Function string `xml:"function,attr,omitempty"`
}
