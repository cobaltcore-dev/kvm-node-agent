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

package libvirt

import (
	"errors"
	"fmt"
	"os"
	"strings"
	"time"

	"github.com/digitalocean/go-libvirt"
	"github.com/digitalocean/go-libvirt/socket/dialers"
	"github.com/google/uuid"
	"sigs.k8s.io/controller-runtime/pkg/log"

	"github.com/cobaltcode-dev/kvm-node-agent/api/v1alpha1"
	"github.com/cobaltcode-dev/kvm-node-agent/internal/sys"
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

type LibVirt struct {
	virt *libvirt.Libvirt
}

func NewLibVirt() *LibVirt {
	socketPath := os.Getenv("LIBVIRT_SOCKET")
	if socketPath == "" {
		socketPath = "/run/libvirt/libvirt-sock"
	}
	log.Log.Info("Using libvirt unix domain socket", "socket", socketPath)
	return &LibVirt{libvirt.NewWithDialer(dialers.NewLocal(dialers.WithSocket(socketPath)))}
}

func (l *LibVirt) Connect() error {
	return l.virt.Connect()
}

func (l *LibVirt) GetVersion() (string, error) {
	var version uint64
	var err error
	if version, err = l.virt.ConnectGetVersion(); err != nil {
		return "", fmt.Errorf("failed to fetch libvirt version: %w", err)
	}
	major, minor, release := version/1000000, (version/1000)%1000, version%1000
	return fmt.Sprintf("%d.%d.%d", major, minor, release), nil
}

func (l *LibVirt) Close() error {
	return l.virt.Disconnect()
}

func (l *LibVirt) GetInstances() ([]v1alpha1.Instance, error) {
	var instances []v1alpha1.Instance

	flags := []libvirt.ConnectListAllDomainsFlags{libvirt.ConnectListDomainsActive, libvirt.ConnectListDomainsInactive}
	for _, flag := range flags {
		domains, _, err := l.virt.ConnectListAllDomains(1, flag)
		if err != nil {
			return nil, err
		}

		for _, domain := range domains {
			instances = append(instances, v1alpha1.Instance{
				ID:     GetOpenstackUUID(domain),
				Name:   domain.Name,
				Active: flag == libvirt.ConnectListDomainsActive,
			})
		}
	}
	return instances, nil
}

func (l *LibVirt) GetDomainsActive() ([]libvirt.Domain, error) {
	domains, _, err := l.virt.ConnectListAllDomains(1, libvirt.ConnectListDomainsActive)
	if err != nil {
		return nil, err
	}
	return domains, nil
}

func (l *LibVirt) GetDomainJobInfo(domain libvirt.Domain, migration *v1alpha1.Migration) error {
	var err error

	rType, params, err := l.virt.DomainGetJobStats(domain, libvirt.DomainJobStatsKeepCompleted|libvirt.DomainJobStatsCompleted)
	if err != nil {
		return err
	}

	migration.Status.Host = sys.Hostname

	switch rType {
	case VIR_DOMAIN_JOB_NONE:
		migration.Status.Type = "none"
		return errors.New("Domain not found")
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
			migration.Status.TimeElapsed = time.Duration(param.Value.I.(uint64) * 1000 * 1000).String()
		case "time_remaining":
			migration.Status.TimeRemaining = time.Duration(param.Value.I.(uint32) * 1000 * 1000).String()
		case "downtime":
			migration.Status.Downtime = time.Duration(param.Value.I.(uint64) * 1000 * 1000).String()
		case "setup_time":
			migration.Status.SetupTime = time.Duration(param.Value.I.(uint64) * 1000 * 1000).String()
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

func (l *LibVirt) IsConnected() bool {
	return l.virt.IsConnected()
}

func GetOpenstackUUID(domain libvirt.Domain) string {
	u, err := uuid.FromBytes(domain.UUID[:])
	if err != nil {
		return ""
	}

	return u.String()
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

func DomainNotFound(err error) bool {
	return err != nil && strings.HasPrefix(err.Error(), "Domain not found")
}
