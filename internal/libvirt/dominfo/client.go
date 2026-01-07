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

import (
	"encoding/xml"

	libvirt "github.com/digitalocean/go-libvirt"
	"sigs.k8s.io/controller-runtime/pkg/log"
)

// Client that returns information for all domains on our host.
type Client interface {
	// Return information for all domains on our host.
	Get(virt *libvirt.Libvirt) ([]DomainInfo, error)
}

// Implementation of the Client interface.
type client struct{}

// Create a new domain info client.
func NewClient() Client {
	return &client{}
}

// Return information for all domains on our host.
func (m *client) Get(virt *libvirt.Libvirt) ([]DomainInfo, error) {
	domains, _, err := virt.ConnectListAllDomains(1,
		libvirt.ConnectListDomainsActive|libvirt.ConnectListDomainsInactive)
	if err != nil {
		log.Log.Error(err, "failed to list all domains")
		return nil, err
	}
	var domainInfos []DomainInfo
	for _, domain := range domains {
		domainXML, err := virt.DomainGetXMLDesc(domain, 0)
		if err != nil {
			log.Log.Error(err, "failed to get domain xml", "domain", domain.Name)
			return nil, err
		}
		var domainInfo DomainInfo
		if err := xml.Unmarshal([]byte(domainXML), &domainInfo); err != nil {
			log.Log.Error(err, "failed to unmarshal domain xml", "domain", domain.Name)
			return nil, err
		}
		domainInfos = append(domainInfos, domainInfo)
	}
	return domainInfos, nil
}

// Emulated domain info client returning an embedded domain xml.
type clientEmulator struct{}

// Create a new emulated domain info client.
func NewClientEmulator() Client {
	return &clientEmulator{}
}

// Get the domain infos of the host we are mounted on.
func (c *clientEmulator) Get(virt *libvirt.Libvirt) ([]DomainInfo, error) {
	var info DomainInfo
	if err := xml.Unmarshal(exampleXML, &info); err != nil {
		log.Log.Error(err, "failed to unmarshal example capabilities")
		return nil, err
	}
	return []DomainInfo{info}, nil
}
