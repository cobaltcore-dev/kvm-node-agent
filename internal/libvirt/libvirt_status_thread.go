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
	"context"
	"fmt"
	"time"

	"github.com/digitalocean/go-libvirt"
	logger "sigs.k8s.io/controller-runtime/pkg/log"
)

func (l *LibVirt) updateDomains() error {
	flags := []libvirt.ConnectListAllDomainsFlags{
		libvirt.ConnectListDomainsActive,
		libvirt.ConnectListDomainsInactive,
	}

	// updates all domains (active / inactive)
	for _, flag := range flags {
		domains, _, err := l.virt.ConnectListAllDomains(1, flag)
		if err != nil {
			return fmt.Errorf("flag %s: %w", fmt.Sprintf("%T", flag), err)
		}

		// update the domains
		l.domains[flag] = domains
	}
	return nil
}

func (l *LibVirt) runStatusThread(ctx context.Context) {
	log := logger.FromContext(ctx)
	log.Info("starting status thread")

	// run immediately, and every minute after
	_ = l.updateDomains()

	for {
		select {
		case <-time.After(1 * time.Minute):
			if err := l.updateDomains(); err != nil {
				log.Error(err, "failed to update domains")
			}
		case <-ctx.Done():
			log.Info("shutting down status thread")
			return
		case <-l.virt.Disconnected():
			log.Info("libvirt disconnected, shutting down status thread")
			return
		}
	}
}
