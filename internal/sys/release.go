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

package sys

import (
	"bufio"
	"context"
	"os"

	logger "sigs.k8s.io/controller-runtime/pkg/log"
)

func GetOSVersion(ctx context.Context) string {
	log := logger.FromContext(ctx, "os", "GetOSVersion")

	f, err := os.Open("/etc/os-release")
	if err != nil {
		log.Error(err, "failed to open /etc/os-release")
		return "unknown"
	}
	defer func() { _ = f.Close() }()

	buf := bufio.NewScanner(f)
	for buf.Scan() {
		// skip comments
		if len(buf.Text()) > 0 && buf.Text()[0] == '#' {
			continue
		}

		// check if line starts with IMAGE_VERSION
		if len(buf.Text()) > 0 && buf.Text()[0] == 'I' {
			if len(buf.Text()) > 13 && buf.Text()[:13] == "IMAGE_VERSION" {
				return buf.Text()[14:]
			}
		}
	}
	return "unknown"
}
