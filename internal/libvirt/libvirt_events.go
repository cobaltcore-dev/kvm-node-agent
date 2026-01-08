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
	"errors"
	"fmt"
	"strings"
	"time"

	"github.com/digitalocean/go-libvirt"
	metav1 "k8s.io/apimachinery/pkg/apis/meta/v1"
	"sigs.k8s.io/controller-runtime/pkg/client"
	logger "sigs.k8s.io/controller-runtime/pkg/log"

	"github.com/cobaltcore-dev/kvm-node-agent/api/v1alpha1"
	"github.com/cobaltcore-dev/kvm-node-agent/internal/sys"
)

const (
	VIR_DOMAIN_JOB_NONE      = iota /* No job is active (Since: 0.7.7) */
	VIR_DOMAIN_JOB_BOUNDED          /* Job with a finite completion time (Since: 0.7.7) */
	VIR_DOMAIN_JOB_UNBOUNDED        /* Job without a finite completion time (Since: 0.7.7) */
	VIR_DOMAIN_JOB_COMPLETED        /* Job has finished, but isn't cleaned up (Since: 0.7.7) */
	VIR_DOMAIN_JOB_FAILED           /* Job hit error, but isn't cleaned up (Since: 0.7.7) */
	VIR_DOMAIN_JOB_CANCELLED        /* Job was aborted, but isn't cleaned up (Since: 0.7.7) */
)

const (
	VIR_DOMAIN_JOB_OPERATION_UNKNOWN         = iota /* (Since: 3.3.0) */
	VIR_DOMAIN_JOB_OPERATION_START                  /* (Since: 3.3.0) */
	VIR_DOMAIN_JOB_OPERATION_SAVE                   /* (Since: 3.3.0) */
	VIR_DOMAIN_JOB_OPERATION_RESTORE                /* (Since: 3.3.0) */
	VIR_DOMAIN_JOB_OPERATION_MIGRATION_IN           /* (Since: 3.3.0) */
	VIR_DOMAIN_JOB_OPERATION_MIGRATION_OUT          /* (Since: 3.3.0) */
	VIR_DOMAIN_JOB_OPERATION_SNAPSHOT               /* (Since: 3.3.0) */
	VIR_DOMAIN_JOB_OPERATION_SNAPSHOT_REVERT        /* (Since: 3.3.0) */
	VIR_DOMAIN_JOB_OPERATION_DUMP                   /* (Since: 3.3.0) */
	VIR_DOMAIN_JOB_OPERATION_BACKUP                 /* (Since: 6.0.0) */
	VIR_DOMAIN_JOB_OPERATION_SNAPSHOT_DELETE        /* (Since: 9.0.0) */
)

var errDomainNotFoud = errors.New("domain not found")

func GetOpenstackUUID(domain libvirt.Domain) string {
	return UUID(domain.UUID).String()
}

func (l *LibVirt) onMigrationIteration(ctx context.Context, event any) {
	log := logger.FromContext(ctx).WithName("libvirt-migration-listener")
	e := event.(*libvirt.DomainEventCallbackMigrationIterationMsg)
	domain := e.Dom
	uuid := GetOpenstackUUID(domain)
	serverLog := log.WithValues("server", uuid)
	serverLog.Info("migration iteration", "iteration", e.Iteration)

	// migration started
	if err := l.startMigrationWatch(ctx, domain); err != nil {
		serverLog.Error(err, "failed to starting migration watch")
	}
}

func (l *LibVirt) onJobCompleted(ctx context.Context, event any) {
	log := logger.FromContext(ctx).WithName("libvirt-migration-listener")
	e := event.(*libvirt.DomainEventCallbackJobCompletedMsg)
	uuid := GetOpenstackUUID(e.Dom)
	log.Info("job completed", "server", uuid, "params", e.Params)
}

func (l *LibVirt) onLifecycleEvent(ctx context.Context, event any) {
	log := logger.FromContext(ctx).WithName("libvirt-migration-listener")
	e := event.(*libvirt.DomainEventCallbackLifecycleMsg)
	domain := e.Msg.Dom
	serverLog := log.WithValues("server", GetOpenstackUUID(domain))

	switch e.Msg.Event {
	case int32(libvirt.DomainEventDefined):
		switch e.Msg.Detail {
		case int32(libvirt.DomainEventDefinedAdded):
			serverLog.Info("domain added")
		case int32(libvirt.DomainEventDefinedUpdated):
			serverLog.Info("domain updated")
		case int32(libvirt.DomainEventDefinedRenamed):
			serverLog.Info("domain renamed")
		case int32(libvirt.DomainEventDefinedFromSnapshot):
			serverLog.Info("domain defined from snapshot")
		}
	case int32(libvirt.DomainEventUndefined):
		serverLog.Info("domain undefined")
	case int32(libvirt.DomainEventStarted):
		switch e.Msg.Detail {
		case int32(libvirt.DomainEventStartedBooted):
			serverLog.Info("domain booted")
		case int32(libvirt.DomainEventStartedMigrated):
			serverLog.Info("incoming migration started")
		case int32(libvirt.DomainEventStartedRestored):
			serverLog.Info("domain restored")
		case int32(libvirt.DomainEventStartedFromSnapshot):
			serverLog.Info("domain started from snapshot")
		case int32(libvirt.DomainEventStartedWakeup):
			serverLog.Info("domain woken up")
		}
	case int32(libvirt.DomainEventSuspended):
		serverLog.Info("domain suspended")
	case int32(libvirt.DomainEventResumed):
		serverLog.Info("domain resumed")
		// incoming migration completed, finalize migration status
		if err := l.patchMigration(ctx, domain, true); client.IgnoreNotFound(err) != nil {
			serverLog.Error(err, "failed to update migration status")
		}
	case int32(libvirt.DomainEventStopped):
		serverLog.Info("domain stopped")
		l.stopMigrationWatch(ctx, domain)
	case int32(libvirt.DomainEventShutdown):
		serverLog.Info("domain shutdown")
		l.stopMigrationWatch(ctx, domain)
	case int32(libvirt.DomainEventPmsuspended):
		serverLog.Info("domain PM suspended")
	case int32(libvirt.DomainEventCrashed):
		serverLog.Info("domain crashed")
	}
}

func (l *LibVirt) startMigrationWatch(ctx context.Context, domain libvirt.Domain) error {
	log := logger.FromContext(ctx, "server", GetOpenstackUUID(domain))

	// ensure migration object exists
	migr := v1alpha1.Migration{
		ObjectMeta: metav1.ObjectMeta{
			Name:      GetOpenstackUUID(domain),
			Namespace: sys.Namespace,
		},
	}
	if err := l.client.Create(ctx, &migr); client.IgnoreAlreadyExists(err) != nil {
		return fmt.Errorf("failed to create migration object: %w", err)
	}

	// ensure we have only one job running, due to external asynchronous callback from libvirt
	l.migrationLock.Lock()
	defer l.migrationLock.Unlock()

	// check if migration watch is already running
	if _, ok := l.migrationJobs[domain.Name]; ok {
		return nil
	}

	log.Info("starting migration watch, timeout=60m")

	// Updating migration start time
	object := client.ObjectKey{
		Name:      GetOpenstackUUID(domain),
		Namespace: sys.Namespace,
	}
	var original v1alpha1.Migration
	if err := l.client.Get(ctx, object, &original); err != nil {
		return fmt.Errorf("failed to get migration status: %w", err)
	}
	patched := original.DeepCopy()
	patched.Status.Started = metav1.Now()
	patched.Status.Origin = sys.NodeLabelName
	if err := l.client.Status().Patch(ctx, patched, client.MergeFrom(&original)); err != nil {
		return fmt.Errorf("failed to patch migration status time: %w", err)
	}

	// start migration watch
	timeoutCtx, cancel := context.WithTimeout(context.Background(), 60*time.Minute)
	l.migrationJobs[domain.Name] = cancel
	go l.watchMigrationLoop(timeoutCtx, cancel, domain)
	return nil
}

func (l *LibVirt) stopMigrationWatch(ctx context.Context, domain libvirt.Domain) {
	if cancel, ok := l.migrationJobs[domain.Name]; ok {
		logger.FromContext(ctx).Info("stopping migration watch", "server", GetOpenstackUUID(domain))
		cancel()
		delete(l.migrationJobs, domain.Name)
	}
}

func (l *LibVirt) patchMigration(ctx context.Context, domain libvirt.Domain, completed bool) error {
	object := client.ObjectKey{
		Name:      GetOpenstackUUID(domain),
		Namespace: sys.Namespace,
	}

	var original v1alpha1.Migration
	if err := l.client.Get(ctx, object, &original); err != nil {
		return fmt.Errorf("failed to get migration status: %w", err)
	}

	migration := original.DeepCopy()
	if err := l.populateDomainJobInfo(domain, migration, completed); err != nil {
		// ignore domain not running error due to race condition with cancel job
		if strings.HasSuffix(err.Error(), "domain is not running") {
			return nil
		}

		// quirk if the domain job details have been reaped, set migration type to completed
		if completed && errors.Is(err, errDomainNotFoud) {
			logger.FromContext(ctx).Info("migration job details reaped, setting migration status to completed")
			migration.Status.Type = "completed"
		}
	}

	// patch migration status
	if err := l.client.Status().Patch(ctx, migration, client.MergeFrom(&original)); err != nil {
		return fmt.Errorf("failed to patch migration status: %w", err)
	}

	return nil
}

// watchMigrationLoop watches the migration progress of a domain on the source hypervisor
func (l *LibVirt) watchMigrationLoop(ctx context.Context, cancel context.CancelFunc, domain libvirt.Domain) {
	defer cancel()
	log := logger.FromContext(ctx, "server", GetOpenstackUUID(domain))

	// Watch migration progress in a loop
	for {
		select {
		case <-ctx.Done():
			log.Info("migration watch stopped")
			return
		case <-time.After(1 * time.Second):
			if ctx.Err() != nil {
				return
			}

			// Patch migration status
			if err := l.patchMigration(ctx, domain, false); err != nil {
				if errors.Is(err, errDomainNotFoud) {
					// quirk if the domain job details have been reaped, stop migration watch
					// could happen if the migration fails
					log.Info("migration job details reaped, stopping migration watch")
					return
				}
				if !errors.Is(err, context.Canceled) {
					log.Error(err, "failed updating migration status")
				}
			}
		}
	}
}

func (l *LibVirt) populateDomainJobInfo(domain libvirt.Domain, migration *v1alpha1.Migration, completed bool) error {
	var err error
	var flags libvirt.DomainGetJobStatsFlags

	if completed {
		flags = libvirt.DomainJobStatsCompleted
	}

	migration.Status.Destination = sys.NodeLabelName
	rType, params, err := l.virt.DomainGetJobStats(domain, flags)
	if err != nil {
		return err
	}

	switch rType {
	case VIR_DOMAIN_JOB_NONE:
		return errDomainNotFoud
	case VIR_DOMAIN_JOB_BOUNDED:
		migration.Status.Type = "bounded"
	case VIR_DOMAIN_JOB_UNBOUNDED:
		migration.Status.Type = "unbounded"
	case VIR_DOMAIN_JOB_COMPLETED:
		migration.Status.Type = "completed"
	case VIR_DOMAIN_JOB_FAILED:
		migration.Status.Type = "failed"
	case VIR_DOMAIN_JOB_CANCELLED:
		migration.Status.Type = "cancelled"
	}

	for _, param := range params {
		switch param.Field {
		case "operation":
			switch param.Value.I.(int32) {
			case VIR_DOMAIN_JOB_OPERATION_UNKNOWN:
				migration.Status.Operation = "unknown"
			case VIR_DOMAIN_JOB_OPERATION_START:
				migration.Status.Operation = "start"
			case VIR_DOMAIN_JOB_OPERATION_SAVE:
				migration.Status.Operation = "save"
			case VIR_DOMAIN_JOB_OPERATION_RESTORE:
				migration.Status.Operation = "restore"
			case VIR_DOMAIN_JOB_OPERATION_MIGRATION_IN:
				migration.Status.Operation = "migration_in"
			case VIR_DOMAIN_JOB_OPERATION_MIGRATION_OUT:
				migration.Status.Operation = "migration_out"
			case VIR_DOMAIN_JOB_OPERATION_SNAPSHOT:
				migration.Status.Operation = "snapshot"
			case VIR_DOMAIN_JOB_OPERATION_SNAPSHOT_REVERT:
				migration.Status.Operation = "snapshot_revert"
			case VIR_DOMAIN_JOB_OPERATION_DUMP:
				migration.Status.Operation = "dump"
			case VIR_DOMAIN_JOB_OPERATION_BACKUP:
				migration.Status.Operation = "backup"
			case VIR_DOMAIN_JOB_OPERATION_SNAPSHOT_DELETE:
				migration.Status.Operation = "snapshot_delete"
			}
		case "time_elapsed":
			migration.Status.TimeElapsed = time.Duration(param.Value.I.(int64) * 1000 * 1000).String()
		case "time_remaining":
			migration.Status.TimeRemaining = time.Duration(param.Value.I.(uint32) * 1000 * 1000).String()
		case "downtime":
			migration.Status.Downtime = time.Duration(param.Value.I.(int64) * 1000 * 1000).String()
		case "setup_time":
			migration.Status.SetupTime = time.Duration(param.Value.I.(int64) * 1000 * 1000).String()
		case "data_total":
			migration.Status.DataTotal = ByteCountIEC(param.Value.I.(uint64))
		case "data_processed":
			migration.Status.DataProcessed = ByteCountIEC(param.Value.I.(uint64))
		case "data_remaining":
			migration.Status.DataRemaining = ByteCountIEC(param.Value.I.(uint64))
		case "memory_total":
			migration.Status.MemTotal = ByteCountIEC(param.Value.I.(uint64))
		case "memory_processed":
			migration.Status.MemProcessed = ByteCountIEC(param.Value.I.(uint64))
		case "memory_remaining":
			migration.Status.MemRemaining = ByteCountIEC(param.Value.I.(uint64))
		case "memory_constant":
			migration.Status.MemConstant = param.Value.I.(uint64)
		case "memory_normal":
			migration.Status.MemNormal = param.Value.I.(uint64)
		case "memory_normal_bytes":
			migration.Status.MemNormalBytes = ByteCountIEC(param.Value.I.(uint64))
		case "memory_bps":
			migration.Status.MemBps = ByteCountIEC(param.Value.I.(uint64)) + "/s"
		case "memory_dirty_rate":
			migration.Status.MemDirtyRate = fmt.Sprintf("%d/s", param.Value.I.(uint64))
		case "memory_page_size":
			migration.Status.MemPageSize = ByteCountIEC(param.Value.I.(uint64))
		case "memory_iteration":
			migration.Status.MemIteration = param.Value.I.(uint64)
		case "memory_postcopy_requests":
			migration.Status.MemPostcopyRequests = param.Value.I.(uint64)
		case "disk_total":
			migration.Status.DiskTotal = ByteCountIEC(param.Value.I.(uint64))
		case "disk_processed":
			migration.Status.DiskProcessed = ByteCountIEC(param.Value.I.(uint64))
		case "disk_remaining":
			migration.Status.DiskRemaining = ByteCountIEC(param.Value.I.(uint64))
		case "disk_bps":
			migration.Status.DiskBps = ByteCountIEC(param.Value.I.(uint64)) + "/s"
		case "auto_converge_throttle":
			migration.Status.AutoConvergeThrottle = fmt.Sprintf("%d%%", param.Value.I.(uint64))
		case "success":
			migration.Status.Type = "success"
		case "errmsg":
			migration.Status.ErrMsg = param.Value.I.(string)
		}
	}
	return err
}
