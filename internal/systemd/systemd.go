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
	"encoding/json"
	"errors"
	"fmt"
	"os"
	"strconv"
	"syscall"

	v1 "github.com/cobaltcore-dev/openstack-hypervisor-operator/api/v1"
	systemd "github.com/coreos/go-systemd/v22/dbus"
	"github.com/godbus/dbus/v5"
	logger "sigs.k8s.io/controller-runtime/pkg/log"
)

const (
	ACTIVE     = "active"
	ACTIVATING = "activating"
	INACTIVE   = "inactive"
	FAILED     = "failed"
)

type SystemdConn struct {
	// go-systemd dbus connection
	conn *systemd.Conn

	// godbus dbus connection for poweroff inhibition
	login1conn *dbus.Conn

	// godbus dbus object for poweroff inhibition
	login1obj dbus.BusObject

	// channel for shutdown signal
	prepareForShutdownSignal chan *dbus.Signal

	// channel for shutdown goroutine
	shutdownCh chan bool

	// file descriptor for inhibition
	fd int
}

var systemdConn *SystemdConn

func dialBus() (*dbus.Conn, error) {
	conn, err := dbus.SystemBusPrivate()
	if err != nil {
		return nil, err
	}
	methods := []dbus.Auth{
		dbus.AuthExternal("0"),
		dbus.AuthExternal(strconv.Itoa(os.Getuid())),
		dbus.AuthAnonymous(),
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
}

func NewSystemd(ctx context.Context) (*SystemdConn, error) {
	if systemdConn != nil {
		return systemdConn, nil
	}

	log := logger.FromContext(ctx)

	log.Info("Connecting to systemd")
	conn, err := systemd.NewConnection(dialBus)
	if err != nil {
		return nil, err
	}

	// separate connection for systemd inhibition since go-systemd doesn't support it
	dbusConn, err := dialBus()
	if err != nil {
		return nil, fmt.Errorf("failed to connect to dbus: %w", err)
	}

	systemdConn = &SystemdConn{
		conn:                     conn,
		login1conn:               dbusConn,
		login1obj:                dbusConn.Object("org.freedesktop.login1", "/org/freedesktop/login1"),
		prepareForShutdownSignal: make(chan *dbus.Signal, 1),
		shutdownCh:               make(chan bool),
		fd:                       -1,
	}
	return systemdConn, nil
}

// EnableShutdownInhibit blocks shutdown by using systemd inhibition lock,
// and registers a shutdown callback
func (s *SystemdConn) EnableShutdownInhibit(ctx context.Context, cb func(context.Context) error) error {
	if s.fd != -1 {
		return fmt.Errorf("shutdown inhibition already enabled")
	}

	log := logger.Log.WithName("systemd")
	log.Info("enabling shutdown inhibition")

	// List inhibitors
	var inhibitors [][]any
	if err := s.login1obj.CallWithContext(
		ctx,
		"org.freedesktop.login1.Manager.ListInhibitors",
		0,
	).Store(&inhibitors); err != nil {
		return fmt.Errorf("failed to list inhibitors: %w", err)
	}
	log.Info("existing inhibitors", "inhibitors", inhibitors)

	// create inhibitor
	if err := s.login1obj.CallWithContext(
		ctx,
		"org.freedesktop.login1.Manager.Inhibit",
		0,
		"sleep:shutdown",
		"kvm-node-agent",
		"Emergency evacuation of host node.",
		"delay",
	).Store(&s.fd); err != nil {
		// ignore error if not running in k8s, so we can debug remotely
		return fmt.Errorf("error storing file descriptor: %w", err)
	}

	log.Info("registering shutdown callback")
	go func() {
		for {
			select {
			case <-s.shutdownCh:
				log.Info("stopping shutdown callback goroutine")
				return
			case signal, ok := <-s.prepareForShutdownSignal:
				if !ok {
					log.Info("prepareForShutdownSignal channel closed")
					return
				}
				log.Info("received shutdown signal", "signal", signal)

				// execute the shutdown callback
				if err := cb(context.Background()); err != nil {
					log.Error(err, "failed to execute shutdown callback")
				}

				log.Info("releasing shutdown inhibition")
				// release the inhibition lock to continue shutdown
				if err := s.DisableShutdownInhibit(); err != nil {
					log.Error(err, "failed to release shutdown inhibition")
				}
				return
			}
		}
	}()

	// register signal handler
	if err := s.login1conn.AddMatchSignal(
		dbus.WithMatchInterface("org.freedesktop.login1.Manager"),
		dbus.WithMatchObjectPath("/org/freedesktop/login1"),
		dbus.WithMatchMember("PrepareForShutdown"),
	); err != nil {
		return fmt.Errorf("failed to add match signal: %w", err)
	}
	s.login1conn.Signal(s.prepareForShutdownSignal)

	return nil
}

// DisableShutdownInhibit releases the systemd inhibition lock
func (s *SystemdConn) DisableShutdownInhibit() error {
	log := logger.Log.WithName("systemd")
	log.Info("disabling shutdown inhibition")

	if s.fd == -1 {
		// nothing to do
		return nil
	}

	// remove signal handler
	s.login1conn.RemoveSignal(s.prepareForShutdownSignal)

	// stopping the shutdown callback goroutine
	s.shutdownCh <- true

	err := syscall.Close(s.fd)
	if err != nil {
		return fmt.Errorf("failed to close file descriptor: %w", err)
	}
	s.fd = -1
	return nil
}

func (s *SystemdConn) Close() {
	s.conn.Close()
	_ = s.login1conn.Close()
}

func (s *SystemdConn) IsConnected() bool {
	return s.conn.Connected() && s.login1conn.Connected()
}

func (s *SystemdConn) ListUnitsByNames(ctx context.Context, units []string) ([]systemd.UnitStatus, error) {
	return s.conn.ListUnitsByNamesContext(ctx, units)
}

func (s *SystemdConn) GetUnitByName(ctx context.Context, unit string) (systemd.UnitStatus, error) {
	units, err := s.ListUnitsByNames(ctx, []string{unit})
	if err != nil {
		return systemd.UnitStatus{}, err
	}
	if len(units) == 0 {
		return systemd.UnitStatus{}, nil
	}
	return units[0], nil
}

func (s *SystemdConn) StartUnit(ctx context.Context, unit string) (int, error) {

	return s.conn.StartUnitContext(ctx, unit, "replace", nil)
}

func (s *SystemdConn) ReloadUnit(ctx context.Context, unit string) (int, error) {
	return s.conn.ReloadUnitContext(ctx, unit, "replace", nil)
}

var ErrFailed = errors.New("update has failed")

// ReconcileSysUpdate orchestrates a systemd-sysupdate via the systemd-sysupdate@.service unit.
func (s *SystemdConn) ReconcileSysUpdate(ctx context.Context, hv *v1.Hypervisor) (bool, error) {
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

func (s *SystemdConn) Describe(ctx context.Context) (*Descriptor, error) {
	// Get descriptor
	var res []byte
	if err := s.login1conn.
		Object("org.freedesktop.hostname1", "/org/freedesktop/hostname1").
		CallWithContext(
			ctx,
			"org.freedesktop.hostname1.Describe",
			0,
		).Store(&res); err != nil {
		return nil, fmt.Errorf("failed to fetch hostname descriptor: %w", err)
	}

	// Parse descriptor
	var desc Descriptor
	if err := json.Unmarshal(res, &desc); err != nil {
		return nil, fmt.Errorf("failed to unmarshal hostname descriptor: %w", err)
	}

	return &desc, nil
}
