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

package systemd

import (
	"context"
	"errors"
	"fmt"
	"os"
	"strconv"

	"github.com/coreos/go-systemd/v22/dbus"
	dbus2 "github.com/godbus/dbus/v5"
	logger "sigs.k8s.io/controller-runtime/pkg/log"

	"github.com/cobaltcode-dev/kvm-node-agent/api/v1alpha1"
)

const (
	ACTIVE     = "active"
	ACTIVATING = "activating"
	INACTIVE   = "inactive"
	FAILED     = "failed"
)

type SystemdConn struct {
	conn *dbus.Conn
}

func NewSystemd(ctx context.Context) (*SystemdConn, error) {
	conn, err := dbus.NewConnection(func() (*dbus2.Conn, error) {
		conn, err := dbus2.SystemBusPrivate()
		if err != nil {
			return nil, err
		}
		methods := []dbus2.Auth{
			dbus2.AuthExternal("0"),
			dbus2.AuthExternal(strconv.Itoa(os.Getuid())),
			dbus2.AuthAnonymous(),
		}
		if err = conn.Auth(methods); err != nil {
			_ = conn.Close()
			return nil, err
		}
		if err = conn.Hello(); err != nil {
			_ = conn.Close()
			return nil, err
		}
		return conn, nil
	})
	if err != nil {
		return nil, err
	}

	return &SystemdConn{
		conn: conn,
	}, nil
}

func (s *SystemdConn) Close() {
	s.conn.Close()
}

func (s *SystemdConn) IsConnected() bool {
	return s.conn.Connected()
}

func (s *SystemdConn) ListUnitsByNames(ctx context.Context, units []string) ([]dbus.UnitStatus, error) {
	return s.conn.ListUnitsByNamesContext(ctx, units)
}

func (s *SystemdConn) GetUnitByName(ctx context.Context, unit string) (dbus.UnitStatus, error) {
	units, err := s.ListUnitsByNames(ctx, []string{unit})
	if err != nil {
		return dbus.UnitStatus{}, err
	}
	if len(units) == 0 {
		return dbus.UnitStatus{}, nil
	}
	return units[0], nil
}

func (s *SystemdConn) StartUnit(ctx context.Context, unit string) (int, error) {
	return s.conn.StartUnitContext(ctx, unit, "replace", nil)
}

var ErrFailed = errors.New("update has failed")

// ReconcileSysUpdate orchestrates a systemd-sysupdate via the systemd-sysupdate@.service unit.
func (s *SystemdConn) ReconcileSysUpdate(ctx context.Context, hv *v1alpha1.Hypervisor) (bool, error) {
	version := hv.Spec.OperatingSystemVersion
	log := logger.FromContext(ctx, "systemd", "reconcileSysUpdate", "version", version)

	// Needs to be connected to systemd
	if !s.IsConnected() {
		return false, fmt.Errorf("not connected to systemd")
	}

	unit := fmt.Sprintf("systemd-sysupdate@%s.service", version)
	if version == "latest" {
		unit = "systemd-sysupdate.service"
	}

	status, err := s.GetUnitByName(ctx, unit)
	if err != nil {
		return false, err
	}

	// Check if the update is already running
	if hv.Status.Update.InProgress {
		switch status.ActiveState {
		case ACTIVE, ACTIVATING:
			log.Info("update is running")
		case FAILED:
			log.Info("Update has failed")
			return false, fmt.Errorf("%s %w", version, ErrFailed)
		case INACTIVE:
			// Update has finished successfully
			if hv.Spec.Reboot {
				if _, err = s.StartUnit(ctx, "systemd-sysupdate-reboot.target"); err != nil {
					return false, err
				}
			}
		}
	} else {
		if status.ActiveState == ACTIVE {
			log.Info("An update is already running, ignoring")
		} else {
			// Start the update
			log.Info("starting update")
			if _, err = s.StartUnit(ctx, unit); err != nil {
				return false, err
			}
			return true, nil
		}
	}

	return status.ActiveState == ACTIVE, nil
}
