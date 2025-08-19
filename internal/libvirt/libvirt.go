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
	"context"
	"fmt"
	"os"
	"sync"

	"github.com/digitalocean/go-libvirt"
	"github.com/digitalocean/go-libvirt/socket/dialers"
	"sigs.k8s.io/controller-runtime/pkg/client"
	"sigs.k8s.io/controller-runtime/pkg/log"

	"github.com/cobaltcode-dev/kvm-node-agent/api/v1alpha1"
)

type LibVirt struct {
	virt          *libvirt.Libvirt
	client        client.Client
	migrationJobs map[string]context.CancelFunc
	migrationLock sync.Mutex
	version       string
	domains       map[libvirt.ConnectListAllDomainsFlags][]libvirt.Domain
}

func NewLibVirt(k client.Client) *LibVirt {
	socketPath := os.Getenv("LIBVIRT_SOCKET")
	if socketPath == "" {
		socketPath = "/run/libvirt/libvirt-sock"
	}
	log.Log.Info("Using libvirt unix domain socket", "socket", socketPath)
	return &LibVirt{
		libvirt.NewWithDialer(dialers.NewLocal(dialers.WithSocket(socketPath))),
		k,
		make(map[string]context.CancelFunc),
		sync.Mutex{},
		"N/A",
		make(map[libvirt.ConnectListAllDomainsFlags][]libvirt.Domain, 2),
	}
}

func (l *LibVirt) Connect() error {
	// Check if already connected
	if l.virt.IsConnected() {
		return nil
	}

	var libVirtUri = libvirt.ConnectURI("ch:///system")
	if uri, present := os.LookupEnv("LIBVIRT_DEFAULT_URI"); present {
		libVirtUri = libvirt.ConnectURI(uri)
	}
	err := l.virt.ConnectToURI(libVirtUri)
	if err == nil {
		// Update the version
		if version, err := l.virt.ConnectGetVersion(); err != nil {
			log.Log.Error(err, "unable to fetch libvirt version")
		} else {
			major, minor, release := version/1000000, (version/1000)%1000, version%1000
			l.version = fmt.Sprintf("%d.%d.%d", major, minor, release)
		}

		// Run the migration listener in a goroutine
		ctx := log.IntoContext(context.Background(), log.Log.WithName("libvirt-migration-listener"))
		go l.runMigrationListener(ctx)

		// Periodic status thread
		ctx = log.IntoContext(context.Background(), log.Log.WithName("libvirt-status-thread"))
		go l.runStatusThread(ctx)
	}

	return err
}

func (l *LibVirt) GetVersion() string {
	return l.version
}

func (l *LibVirt) Close() error {
	return l.virt.Disconnect()
}

func (l *LibVirt) GetInstances() ([]v1alpha1.Instance, error) {
	var instances []v1alpha1.Instance

	flags := []libvirt.ConnectListAllDomainsFlags{libvirt.ConnectListDomainsActive, libvirt.ConnectListDomainsInactive}
	for _, flag := range flags {
		for _, domain := range l.domains[flag] {
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
	return l.domains[libvirt.ConnectListDomainsActive], nil
}

func (l *LibVirt) IsConnected() bool {
	return l.virt.IsConnected()
}

func (l *LibVirt) GetNumInstances() int {
	return len(l.domains[libvirt.ConnectListDomainsActive])
}
