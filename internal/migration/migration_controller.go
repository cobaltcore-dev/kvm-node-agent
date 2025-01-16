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

package migration

import (
	"context"
	"errors"
	"fmt"
	"strings"
	"time"

	"github.com/digitalocean/go-libvirt"
	"sigs.k8s.io/controller-runtime/pkg/client"

	logger "sigs.k8s.io/controller-runtime/pkg/log"

	"github.com/cobaltcode-dev/kvm-node-agent/api/v1alpha1"
	lvirt "github.com/cobaltcode-dev/kvm-node-agent/internal/libvirt"
)

var Finished = errors.New("migration finished")

func PatchMigration(ctx context.Context, r client.Client, l lvirt.Interface, domain libvirt.Domain) error {
	log := logger.FromContext(ctx, "controller", "migration")

	object := client.ObjectKey{
		Name:      lvirt.GetOpenstackUUID(domain),
		Namespace: "monsoon3",
	}

	var original v1alpha1.Migration
	if err := r.Get(ctx, object, &original); err != nil {
		return fmt.Errorf("failed to get migration status: %w", err)
	}

	migration := original.DeepCopy()
	err := l.GetDomainJobInfo(domain, migration)
	if err != nil {
		if lvirt.DomainNotFound(err) {
			log.Info("Domain migrated", "id", lvirt.GetOpenstackUUID(domain))
			return Finished
		}

		return fmt.Errorf("failed to get migration info: %w", err)
	}

	// patch migration status
	log.Info("Updating migration status", "id", lvirt.GetOpenstackUUID(domain), "status", migration.Status.Type)
	if err = r.Status().Patch(ctx, migration, client.MergeFrom(&original)); err != nil {
		return fmt.Errorf("failed to patch migration status: %w", err)
	}

	if !strings.HasSuffix(migration.Status.Type, "bounded") {
		return Finished
	}

	return nil
}

func WatchMigrationLoop(ctx context.Context, cancel context.CancelFunc, r client.Client, l lvirt.Interface, domain libvirt.Domain) {
	defer cancel()
	log := logger.FromContext(ctx, "controller", "hypervisor")
	log.Info("Watching migration", "domain", lvirt.GetOpenstackUUID(domain))

	// Watch migration progress in a loop
	for {
		select {
		case <-ctx.Done():
			log.Error(ctx.Err(), "Context canceled, stopping migration watch")
			return
		case <-time.After(1 * time.Second):
			log.Info("Updating migration status", "id", lvirt.GetOpenstackUUID(domain))

			// update migration status
			if err := PatchMigration(ctx, r, l, domain); err != nil {
				if errors.Is(err, Finished) {
					log.Info("Migration completed", "id", lvirt.GetOpenstackUUID(domain))
					return
				}
				log.Error(err, "Failed to update migration status")
				return
			}
		}
	}
}
