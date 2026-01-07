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

package capabilities

// Capabilities as returned from the libvirt driver capabilities api.
//
// The format is the same as returned when executing `virsh capabilities`. See:
//   - https://www.libvirt.org/manpages/virsh.html#capabilities
//   - https://libvirt.org/formatcaps.html
type Capabilities struct {
	Host  CapabilitiesHost  `xml:"host"`
	Guest CapabilitiesGuest `xml:"guest"`
}

type CapabilitiesHost struct {
	CPU      CapabilitiesHostCPU      `xml:"cpu"`
	IOMMU    CapabilitiesHostIOMMU    `xml:"iommu"`
	Topology CapabilitiesHostTopology `xml:"topology"`
	Cache    CapabilitiesHostCache    `xml:"cache"`
}

type CapabilitiesHostCPU struct {
	Arch string `xml:"arch"`
}

type CapabilitiesHostIOMMU struct {
	Support string `xml:"support,attr"`
}

type CapabilitiesHostTopology struct {
	CellSpec      CapabilitiesHostTopologyCells         `xml:"cells"`
	Interconnects CapabilitiesHostTopologyInterconnects `xml:"interconnects"`
}

type CapabilitiesHostTopologyCells struct {
	Num   int                            `xml:"num,attr"`
	Cells []CapabilitiesHostTopologyCell `xml:"cell"`
}

type CapabilitiesHostTopologyCell struct {
	ID        uint64                                `xml:"id,attr"`
	Memory    CapabilitiesHostTopologyCellMemory    `xml:"memory"`
	Pages     []CapabilitiesHostTopologyCellPages   `xml:"pages"`
	Distances CapabilitiesHostTopologyCellDistances `xml:"distances"`
	CPUs      CapabilitiesHostTopologyCellCPUs      `xml:"cpus"`
}

type CapabilitiesHostTopologyCellMemory struct {
	Unit  string `xml:"unit,attr"`
	Value int64  `xml:",chardata"`
}

type CapabilitiesHostTopologyCellPages struct {
	Unit  string `xml:"unit,attr"`
	Size  int    `xml:"size,attr"`
	Value uint64 `xml:",chardata"`
}

type CapabilitiesHostTopologyCellDistances struct {
	Siblings []CapabilitiesHostTopologyCellSibling `xml:"sibling"`
}

type CapabilitiesHostTopologyCellSibling struct {
	ID    int `xml:"id,attr"`
	Value int `xml:"value,attr"`
}

type CapabilitiesHostTopologyCellCPUs struct {
	Num  int64                             `xml:"num,attr"`
	CPUs []CapabilitiesHostTopologyCellCPU `xml:"cpu"`
}

type CapabilitiesHostTopologyCellCPU struct {
	ID        int    `xml:"id,attr"`
	SocketID  int    `xml:"socket_id,attr"`
	DieID     int    `xml:"die_id,attr"`
	ClusterID int    `xml:"cluster_id,attr"`
	CoreID    int    `xml:"core_id,attr"`
	Siblings  string `xml:"siblings,attr"`
}

type CapabilitiesHostTopologyInterconnects struct {
	Latencies  []CapabilitiesHostTopologyLatency   `xml:"latency"`
	Bandwidths []CapabilitiesHostTopologyBandwidth `xml:"bandwidth"`
}

type CapabilitiesHostTopologyLatency struct {
	Initiator int    `xml:"initiator,attr"`
	Target    int    `xml:"target,attr"`
	Type      string `xml:"type,attr"`
	Value     int    `xml:"value,attr"`
}

type CapabilitiesHostTopologyBandwidth struct {
	Initiator int    `xml:"initiator,attr"`
	Target    int    `xml:"target,attr"`
	Type      string `xml:"type,attr"`
	Value     uint64 `xml:"value,attr"`
	Unit      string `xml:"unit,attr"`
}

type CapabilitiesHostCache struct {
	Banks []CapabilitiesHostCacheBank `xml:"bank"`
}

type CapabilitiesHostCacheBank struct {
	ID    int    `xml:"id,attr"`
	Level int    `xml:"level,attr"`
	Type  string `xml:"type,attr"`
	Size  int    `xml:"size,attr"`
	Unit  string `xml:"unit,attr"`
	CPUs  string `xml:"cpus,attr"`
}

type CapabilitiesGuest struct {
	OSType string                `xml:"os_type"`
	Arch   CapabilitiesGuestArch `xml:"arch"`
}

type CapabilitiesGuestArch struct {
	Name     string                      `xml:"name,attr"`
	WordSize int                         `xml:"wordsize"`
	Domain   CapabilitiesGuestArchDomain `xml:"domain"`
}

type CapabilitiesGuestArchDomain struct {
	Type string `xml:"type,attr"`
}
