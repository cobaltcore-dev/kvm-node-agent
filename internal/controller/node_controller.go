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

package controller

import (
	"context"
	"fmt"

	corev1 "k8s.io/api/core/v1"
	k8serrors "k8s.io/apimachinery/pkg/api/errors"
	metav1 "k8s.io/apimachinery/pkg/apis/meta/v1"
	"k8s.io/apimachinery/pkg/runtime"
	ctrl "sigs.k8s.io/controller-runtime"
	"sigs.k8s.io/controller-runtime/pkg/client"
	"sigs.k8s.io/controller-runtime/pkg/controller/controllerutil"
	logger "sigs.k8s.io/controller-runtime/pkg/log"

	kvmv1alpha1 "github.com/cobaltcode-dev/kvm-node-agent/api/v1alpha1"
	"github.com/cobaltcode-dev/kvm-node-agent/internal/sys"
)

// NodeReconciler reconciles a Node object
type NodeReconciler struct {
	client.Client
	Scheme                       *runtime.Scheme
	Reboot                       bool
	EvacuateOnReboot             bool
	CreateCertManagerCertificate bool
}

const LabelMetalNodeName = "kubernetes.metal.cloud.sap/name"

// +kubebuilder:rbac:groups="",resources=nodes,verbs=get;list;watch
// +kubebuilder:rbac:groups="",resources=nodes/status,verbs=get
// +kubebuilder:rbac:groups=kvm.cloud.sap,resources=hypervisors,verbs=get;list;watch;create;delete

func (r *NodeReconciler) Reconcile(ctx context.Context, req ctrl.Request) (ctrl.Result, error) {
	log := logger.FromContext(ctx, "controller", "node")

	if req.Name != sys.Hostname {
		panic(fmt.Sprintf("reconciling node %s, but I am running on %s", req.Name, sys.Hostname))
	}

	node := &corev1.Node{}
	if err := r.Get(ctx, req.NamespacedName, node); client.IgnoreNotFound(err) != nil {
		// ignore not found errors, could be deleted
		return ctrl.Result{}, err
	}

	metalNodeName := sys.Hostname
	if name, ok := node.Labels[LabelMetalNodeName]; ok {
		metalNodeName = name
	}

	hypervisor := &kvmv1alpha1.Hypervisor{
		ObjectMeta: metav1.ObjectMeta{
			Name:      node.Name,
			Namespace: sys.Namespace,
			Labels: map[string]string{
				corev1.LabelHostname: sys.Hostname,
				LabelMetalNodeName:   metalNodeName,
			},
		},
		Spec: kvmv1alpha1.HypervisorSpec{
			Reboot:                       r.Reboot,
			EvacuateOnReboot:             r.EvacuateOnReboot,
			CreateCertManagerCertificate: r.CreateCertManagerCertificate,
		},
	}

	// Ensure corresponding hypervisor exists
	log.Info("Reconcile", "name", req.Name, "namespace", req.Namespace)
	if err := r.Get(ctx, client.ObjectKeyFromObject(hypervisor), hypervisor); err != nil {
		if k8serrors.IsNotFound(err) {
			// attach ownerReference for cascading deletion
			if err = controllerutil.SetControllerReference(node, hypervisor, r.Scheme); err != nil {
				return ctrl.Result{}, fmt.Errorf("failed setting controller reference: %w", err)
			}

			log.Info("Creating new hypervisor", "name", node.Name)
			if err = r.Create(ctx, hypervisor); err != nil {
				return ctrl.Result{}, err
			}

			// Requeue to update status
			return ctrl.Result{}, nil
		}

		return ctrl.Result{}, err
	}

	if node.ObjectMeta.DeletionTimestamp != nil {
		// node is being deleted, cleanup hypervisor
		if err := r.Delete(ctx, hypervisor); client.IgnoreNotFound(err) != nil {
			return ctrl.Result{}, fmt.Errorf("failed cleanup up hypervisor: %w", err)
		}
	}

	return ctrl.Result{}, nil
}

// SetupWithManager sets up the controller with the Manager.
func (r *NodeReconciler) SetupWithManager(mgr ctrl.Manager) error {
	return ctrl.NewControllerManagedBy(mgr).
		For(&corev1.Node{}).
		Complete(r)
}
