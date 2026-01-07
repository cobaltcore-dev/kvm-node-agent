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

package domcapabilities

import "encoding/xml"

// DomainCapabilities as returned from the libvirt domain capabilities api.
//
// The format is the same as returned when executing `virsh domcapabilities`.
// See: https://www.libvirt.org/manpages/virsh.html#domcapabilities
// For another reference see: https://gitlab.com/libvirt/libvirt-go-xml-module/-/blob/v1.11010.0/domain_capabilities.go
type DomainCapabilities struct {
	Domain   string                     `xml:"domain"`
	Arch     string                     `xml:"arch"`
	OS       DomainCapabilitiesOS       `xml:"os"`
	CPU      DomainCapabilitiesCPU      `xml:"cpu"`
	Devices  DomainCapabilitiesDevices  `xml:"devices"`
	Features DomainCapabilitiesFeatures `xml:"features"`
}

// DomainCapabilitiesOS represents the OS capabilities section.
type DomainCapabilitiesOS struct {
	Supported string                     `xml:"supported,attr"`
	Loader    DomainCapabilitiesOSLoader `xml:"loader"`
}

// DomainCapabilitiesOSLoader represents the loader capabilities.
type DomainCapabilitiesOSLoader struct {
	Supported string                   `xml:"supported,attr"`
	Enums     []DomainCapabilitiesEnum `xml:"enum"`
}

// DomainCapabilitiesEnum represents an enumeration of possible values.
type DomainCapabilitiesEnum struct {
	Name   string   `xml:"name,attr"`
	Values []string `xml:"value"`
}

// DomainCapabilitiesCPU represents the CPU capabilities section.
type DomainCapabilitiesCPU struct {
	Modes []DomainCapabilitiesCPUMode `xml:"mode"`
}

// DomainCapabilitiesCPUMode represents a CPU mode with its capabilities.
type DomainCapabilitiesCPUMode struct {
	Name      string                   `xml:"name,attr"`
	Supported string                   `xml:"supported,attr"`
	Enums     []DomainCapabilitiesEnum `xml:"enum"`
}

// DomainCapabilitiesDevice represents the devices capabilities section.
type DomainCapabilitiesDevice struct {
	XMLName   xml.Name                 `xml:""`
	Supported string                   `xml:"supported,attr"`
	Enums     []DomainCapabilitiesEnum `xml:"enum"`
}

// DomainCapabilitiesDevices represents the devices capabilities section.
type DomainCapabilitiesDevices struct {
	Devices []DomainCapabilitiesDevice `xml:",any"`
}

// DomainCapabilitiesFeature represents a feature with supported attribute.
type DomainCapabilitiesFeature struct {
	XMLName   xml.Name `xml:""`
	Supported string   `xml:"supported,attr"`
}

// DomainCapabilitiesFeatures represents the features capabilities section.
type DomainCapabilitiesFeatures struct {
	Features []DomainCapabilitiesFeature `xml:",any"`
}
