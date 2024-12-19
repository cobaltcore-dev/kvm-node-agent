/*
SPDX-FileCopyrightText: Copyright 2024 SAP SE or an SAP affiliate company and cobaltcore-dev contributors
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
	"encoding/xml"
	"strconv"
	"strings"
	"time"

	"github.com/Tinkoff/libvirt-exporter/libvirtSchema"
	"github.com/digitalocean/go-libvirt"
	"github.com/google/uuid"
	"github.com/prometheus/client_golang/prometheus"
	"sigs.k8s.io/controller-runtime/pkg/log"
)

const DELAY = 60

func (l *LibVirt) statsCollector() {
	log.Log.Info("Stats collector started")
	for {
		if l.IsConnected() {
			l.collectAllDomainStats()
		}
		time.Sleep(DELAY * time.Second)
	}
}
func (l *LibVirt) collectAllDomainStats() {
	domains, _, err := l.virt.ConnectListAllDomains(1, libvirt.ConnectListDomainsActive|libvirt.ConnectListDomainsInactive)
	log.Log.Info("Collecting stats for all domains")
	if err != nil {
		return
	}

	for _, domain := range domains {
		go func(d libvirt.Domain) {
			l.collectDomainStats(d)
			l.collectDomainData(d)
			l.collectCpuStats(d)
			l.collectBlockStats(d)
		}(domain)
	}
}

func (l *LibVirt) collectDomainStats(domain libvirt.Domain) {
	state, maxmem, rmem, nvir, cputime, err := l.virt.DomainGetInfo(domain)

	if err != nil {
		return
	}
	prometheus.MustNewConstMetric(
		libvirtDomainInfoMaxMemBytesDesc,
		prometheus.GaugeValue,
		float64(maxmem)*1024,
		domain.Name)
	prometheus.MustNewConstMetric(
		libvirtDomainInfoMemoryUsageBytesDesc,
		prometheus.GaugeValue,
		float64(rmem)*1024,
		domain.Name)
	prometheus.MustNewConstMetric(
		libvirtDomainInfoNrVirtCPUDesc,
		prometheus.GaugeValue,
		float64(nvir),
		domain.Name)
	prometheus.MustNewConstMetric(
		libvirtDomainInfoCPUTimeDesc,
		prometheus.CounterValue,
		float64(cputime)/1000/1000/1000, // From nsec to sec
		domain.Name)
	prometheus.MustNewConstMetric(
		libvirtDomainInfoVirDomainState,
		prometheus.GaugeValue,
		float64(state),
		domain.Name)
}

func (l *LibVirt) collectDomainData(domain libvirt.Domain) {
	xmlDesc, err := l.virt.DomainGetXMLDesc(domain, 0)
	if err != nil {
		return
	}
	var desc libvirtSchema.Domain
	err = xml.Unmarshal([]byte(xmlDesc), &desc)
	if err != nil {
		return
	}
	var u uuid.UUID
	u, err = uuid.FromBytes(domain.UUID[:])
	if err != nil {
		return
	}
	prometheus.MustNewConstMetric(
		libvirtDomainInfoMetaDesc,
		prometheus.GaugeValue,
		float64(1),
		domain.Name,
		u.String(),
		desc.Metadata.NovaInstance.NovaName,
		desc.Metadata.NovaInstance.NovaFlavor.FlavorName,
		desc.Metadata.NovaInstance.NovaOwner.NovaUser.UserName,
		desc.Metadata.NovaInstance.NovaOwner.NovaUser.UserUUID,
		desc.Metadata.NovaInstance.NovaOwner.NovaProject.ProjectName,
		desc.Metadata.NovaInstance.NovaOwner.NovaProject.ProjectUUID,
		desc.Metadata.NovaInstance.NovaRoot.RootType,
		desc.Metadata.NovaInstance.NovaRoot.RootUUID)
}

func (l *LibVirt) collectBlockStats(domain libvirt.Domain) {
	statsType := libvirt.DomainStatsState | libvirt.DomainStatsCPUTotal | libvirt.DomainStatsBalloon | libvirt.DomainStatsVCPU | libvirt.DomainStatsInterface | libvirt.DomainStatsBlock | libvirt.DomainStatsPerf | libvirt.DomainStatsIothread | libvirt.DomainStatsMemory | libvirt.DomainStatsDirtyrate

	flags := libvirt.ConnectGetAllDomainsStatsRunning | libvirt.ConnectGetAllDomainsStatsShutoff
	stats, err := l.virt.ConnectGetAllDomainStats([]libvirt.Domain{domain}, uint32(statsType), uint32(flags))
	if err != nil {
		return
	}
	if stats == nil {
		return
	}
	statsBlockMap := make(map[string]*blockStats)

	for _, par := range stats[0].Params {
		data := strings.Split(par.Field, ".")
		if len(data) < 3 {
			continue
		}
		switch data[0] {
		case "block":
			if _, ok := statsBlockMap[data[1]]; !ok {
				statsBlockMap[data[1]] = &blockStats{
					id: data[1],
				}
			}
			switch data[2] {
			case "name":
				statsBlockMap[data[1]].name, _ = par.Value.I.(string)
			case "physical":
				statsBlockMap[data[1]].physical, _ = par.Value.I.(string)
			case "capacity":
				statsBlockMap[data[1]].capacity, _ = par.Value.I.(string)
			case "allocation":
				statsBlockMap[data[1]].allocation, _ = par.Value.I.(string)
			case "path":
				statsBlockMap[data[1]].path, _ = par.Value.I.(string)
			}
		}
	}

	for _, blockstat := range statsBlockMap {
		if blockstat.name == "hdc" || blockstat.name == "hda" {
			continue
		}
		prometheus.MustNewConstMetric(
			libvirtDomainMetaBlockDesc,
			prometheus.GaugeValue,
			float64(1),
			domain.Name,
			blockstat.name,
			blockstat.path,
			blockstat.allocation,
			blockstat.capacity,
			blockstat.physical,
		)
	}

}

func (l *LibVirt) collectCpuStats(domain libvirt.Domain) {
	stats, _, err := l.virt.DomainGetVcpus(domain, 0, 0)
	if err != nil {
		return
	}

	for _, cpustat := range stats {
		prometheus.MustNewConstMetric(
			libvirtDomainVcpuStateDesc,
			prometheus.GaugeValue,
			float64(cpustat.State),
			domain.Name,
			strconv.FormatInt(int64(cpustat.Number), 10))

		prometheus.MustNewConstMetric(
			libvirtDomainVcpuTimeDesc,
			prometheus.CounterValue,
			float64(cpustat.CPUTime)/1000/1000/1000, // From nsec to sec
			domain.Name,
			strconv.FormatInt(int64(cpustat.Number), 10))

		prometheus.MustNewConstMetric(
			libvirtDomainVcpuCPUDesc,
			prometheus.GaugeValue,
			float64(cpustat.CPU),
			domain.Name,
			strconv.FormatInt(int64(cpustat.Number), 10))

	}
}
