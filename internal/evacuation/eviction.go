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

package evacuation

import (
	"context"
	"fmt"
	"time"

	kvmv1 "github.com/cobaltcore-dev/openstack-hypervisor-operator/api/v1"
	"k8s.io/apimachinery/pkg/apis/meta/v1/unstructured"
	"k8s.io/apimachinery/pkg/runtime/schema"
	"sigs.k8s.io/controller-runtime/pkg/client"
	logger "sigs.k8s.io/controller-runtime/pkg/log"

	"github.com/cobaltcore-dev/kvm-node-agent/internal/sys"
)

type EvictionController struct {
	client.Client
}

// EvictCurrentHost callback is allowed to block. It is called when the hypervisor is about to be rebooted.
// It should migrate all VMs away from the current host.
// It is able to block up to InhibitDelayMaxSec seconds to evict virtual machines.
// see `systemd-analyze cat-config systemd/logind.conf` for the current setting.
func (e *EvictionController) EvictCurrentHost(ctx context.Context) error {
	log := logger.FromContext(ctx)

	// Check for running VMs before creating the eviction custom resource
	var hypervisor kvmv1.Hypervisor
	if err := e.Get(ctx, client.ObjectKey{Namespace: sys.Namespace, Name: sys.Hostname}, &hypervisor); err != nil {
		return fmt.Errorf("could not get hypervisor: %w", err)
	}

	if hypervisor.Status.NumInstances == 0 {
		log.Info("EvictCurrentHost due shutdown: No running VMs found on current host, no eviction needed")
		return nil
	}

	u := &unstructured.Unstructured{}
	u.SetUnstructuredContent(map[string]any{
		"spec": map[string]any{
			"hypervisor": sys.Hostname,
			"reason":     "kvm-node-agent: emergency evacuation due to host reboot",
		},
	})
	u.SetGroupVersionKind(schema.GroupVersionKind{
		Group:   "kvm.cloud.sap",
		Kind:    "Eviction",
		Version: "v1",
	})
	u.SetName(sys.Hostname)
	// todo: namespace? cluster-wide?
	u.SetNamespace(sys.Namespace)

	// ... create the eviction custom resource
	if err := e.Create(ctx, u); client.IgnoreAlreadyExists(err) != nil {
		return err
	}

	log.Info("Eviction custom resource created for current host")

	for {
		if ctx.Err() != nil {
			return ctx.Err()
		}

		if err := e.Get(ctx, client.ObjectKeyFromObject(u), u); err != nil {
			return err
		}

		state, _, err := unstructured.NestedString(u.Object, "status", "evictionState")
		if err != nil {
			return err
		}

		log.WithValues("node", u.GetName(), "state", state).Info("Eviction progress")

		if state == "Succeeded" {
			return nil
		}

		time.Sleep(10 * time.Second)
	}
}
