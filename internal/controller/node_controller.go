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

package controller

import (
	"context"
	"fmt"

	v1 "k8s.io/api/core/v1"
	metav1 "k8s.io/apimachinery/pkg/apis/meta/v1"
	"k8s.io/apimachinery/pkg/runtime"
	"k8s.io/apimachinery/pkg/types"
	ctrl "sigs.k8s.io/controller-runtime"
	"sigs.k8s.io/controller-runtime/pkg/client"
	logger "sigs.k8s.io/controller-runtime/pkg/log"

	kvmv1alpha1 "github.com/cobaltcode-dev/kvm-node-agent/api/v1alpha1"
	"github.com/cobaltcode-dev/kvm-node-agent/internal/sys"
)

// NodeReconciler reconciles a Node object
type NodeReconciler struct {
	client.Client
	Scheme *runtime.Scheme
}

// +kubebuilder:rbac:groups="",resources=nodes,verbs=get;list;watch
// +kubebuilder:rbac:groups="",resources=nodes/status,verbs=get

func (r *NodeReconciler) getNode(ctx context.Context) (*v1.Node, error) {
	// Fetch the Node we're current running on
	var nodes v1.NodeList
	err := r.List(ctx, &nodes, client.MatchingLabels{v1.LabelHostname: sys.Hostname})
	if client.IgnoreNotFound(err) != nil {
		return nil, fmt.Errorf("failed fetching nodes: %w", err)
	}

	switch len(nodes.Items) {
	case 0:
		return nil, nil
	case 1:
		return &nodes.Items[0], nil
	default:
		return nil, fmt.Errorf("found more than one node with label %s=%s", v1.LabelHostname, sys.Hostname)
	}

}

func (r *NodeReconciler) Reconcile(ctx context.Context, req ctrl.Request) (ctrl.Result, error) {
	log := logger.FromContext(ctx, "controller", "node")

	if req.Name != sys.Hostname {
		// only reconcile the node I am running on
		return ctrl.Result{}, nil
	}

	namespace := req.Namespace
	if namespace == "" {
		namespace = "monsoon3"
	}

	node, err := r.getNode(ctx)
	if err != nil {
		return ctrl.Result{}, fmt.Errorf("failed fetching node: %w", err)
	}
	if node == nil {
		return ctrl.Result{}, nil
	}
	// Todo: check I am really an hypervisor?

	// Ensure corresponding hypervisor exists
	log.Info("Reconcile", "name", req.Name, "namespace", req.Namespace)
	var hypervisors kvmv1alpha1.HypervisorList
	if err = r.List(ctx, &hypervisors, client.MatchingLabels{v1.LabelHostname: sys.Hostname}); err != nil {
		return ctrl.Result{}, fmt.Errorf("failed fetching hypervisors: %w", err)
	}

	if len(hypervisors.Items) == 0 {
		// create hypervisor
		if err = r.Create(ctx, &kvmv1alpha1.Hypervisor{
			ObjectMeta: metav1.ObjectMeta{
				Name:      node.Name,
				Namespace: namespace,
				Labels:    map[string]string{v1.LabelHostname: sys.Hostname},
			},
			Spec: kvmv1alpha1.HypervisorSpec{},
			Status: kvmv1alpha1.HypervisorStatus{
				Node:    types.NodeName(node.Name),
				Version: sys.GetVersion(),
			},
		}); err != nil {
			return ctrl.Result{}, fmt.Errorf("failed creating hypervisor: %w", err)
		}
	}
	return ctrl.Result{}, nil
}

// SetupWithManager sets up the controller with the Manager.
func (r *NodeReconciler) SetupWithManager(mgr ctrl.Manager) error {
	return ctrl.NewControllerManagedBy(mgr).
		For(&v1.Node{}).
		Complete(r)
}
