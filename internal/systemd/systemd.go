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

//go:generate moq -out systemd_mock.go . Interface

package systemd

import (
	"context"

	"github.com/coreos/go-systemd/v22/dbus"
)

type Interface interface {
	// Close closes the connection to the systemd D-Bus API.
	Close()

	// IsConnected returns true if the connection to the systemd D-Bus API is open.
	IsConnected() bool

	// ListUnitsByNames returns the status of the units with the given names.
	ListUnitsByNames(ctx context.Context, units []string) ([]dbus.UnitStatus, error)
}

type SystemdConn struct {
	conn *dbus.Conn
}

func NewSystemd(ctx context.Context) (*SystemdConn, error) {
	conn, err := dbus.NewWithContext(ctx)
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
	return s.conn.ListUnitsByNamesContext(context.Background(), units)
}
