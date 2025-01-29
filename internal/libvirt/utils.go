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

	"github.com/digitalocean/go-libvirt"
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

func GetOpenstackUUID(domain libvirt.Domain) string {
	return UUID(domain.UUID).String()
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
