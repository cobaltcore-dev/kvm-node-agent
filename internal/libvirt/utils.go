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

package libvirt

import (
	"encoding/hex"
	"fmt"
	"math"

	"k8s.io/apimachinery/pkg/api/resource"

	"github.com/cobaltcore-dev/kvm-node-agent/internal/libvirt/capabilities"
)

const (
	UnitKiB = "KiB"
	UnitMiB = "MiB"
	UnitGiB = "GiB"
	UnitTiB = "TiB"
)

type UUID [16]byte

func (uuid UUID) String() string {
	var tmp [36]byte
	hex.Encode(tmp[:], uuid[:4])
	tmp[8] = '-'
	hex.Encode(tmp[:][9:13], uuid[4:6])
	tmp[13] = '-'
	hex.Encode(tmp[:][14:18], uuid[6:8])
	tmp[18] = '-'
	hex.Encode(tmp[:][19:23], uuid[8:10])
	tmp[23] = '-'
	hex.Encode(tmp[:][24:], uuid[10:])
	return string(tmp[:])
}

func ByteCountIEC(b uint64) string {
	const unit = 1024
	if b < unit {
		return fmt.Sprintf("%d B", b)
	}
	div, exp := int64(unit), 0
	for n := b / unit; n >= unit; n /= unit {
		div *= unit
		exp++
	}
	return fmt.Sprintf("%.1f %ciB",
		float64(b)/float64(div), "KMGTPE"[exp])
}

// Get the cell memory as resource.Quantity.
func MemoryToResource(value int64, unit string) (resource.Quantity, error) {
	var quantity *resource.Quantity
	// Check the unit
	switch unit {
	case UnitKiB:
		quantity = resource.NewQuantity(value*1024, resource.BinarySI)
	case UnitMiB:
		quantity = resource.NewQuantity(value*1024*1024, resource.BinarySI)
	case UnitGiB:
		quantity = resource.NewQuantity(value*1024*1024*1024, resource.BinarySI)
	case UnitTiB:
		quantity = resource.NewQuantity(value*1024*1024*1024*1024, resource.BinarySI)
	}
	if quantity == nil {
		return resource.Quantity{}, fmt.Errorf("unknown memory unit %s", unit)
	}
	// Set the value
	return *quantity, nil
}

// HugepageMemoryToResource returns the allocatable hugepage-backed memory of a
// NUMA cell as a resource.Quantity.
//
// Unlike the cell's total physical memory, which fluctuates between otherwise
// identical hosts because linux reserves/frees varying amounts during boot, the
// number of hugepages is explicitly configured and therefore stable. All KVM
// nova flavors use hw:mem_page_size=large, so only non-base (hugepage) page
// sizes are allocatable to VMs. The base page size is the smallest page size
// reported for the cell and is excluded. A cell with no pages, or only base
// pages, yields 0.
func HugepageMemoryToResource(pages []capabilities.CapabilitiesHostTopologyCellPages) (resource.Quantity, error) {
	total := resource.NewQuantity(0, resource.BinarySI)
	if len(pages) == 0 {
		return *total, nil
	}

	// The base page size is the smallest reported page size. Do not assume it
	// is the first entry.
	baseSize := pages[0].Size
	for _, page := range pages[1:] {
		if page.Size < baseSize {
			baseSize = page.Size
		}
	}

	for _, page := range pages {
		// Skip the base page size, only hugepages are allocatable to VMs.
		if page.Size == baseSize {
			continue
		}
		// Libvirt always reports page sizes in KiB. Be defensive in case this
		// ever changes, so we don't silently report a wrong capacity.
		if page.Unit != UnitKiB {
			return resource.Quantity{}, fmt.Errorf(
				"unexpected hugepage unit %s", page.Unit,
			)
		}
		// Guard the uint64 -> int64 conversion. Real hugepage counts are far
		// below this bound, but guard anyway so an implausible value cannot
		// wrap into a negative capacity.
		if page.Value > math.MaxInt64 {
			return resource.Quantity{}, fmt.Errorf(
				"hugepage count %d exceeds int64", page.Value,
			)
		}
		bytes := int64(page.Value) * int64(page.Size) * 1024
		total.Add(*resource.NewQuantity(bytes, resource.BinarySI))
	}
	return *total, nil
}
