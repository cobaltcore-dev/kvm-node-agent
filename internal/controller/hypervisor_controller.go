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

	"k8s.io/apimachinery/pkg/api/meta"
	metav1 "k8s.io/apimachinery/pkg/apis/meta/v1"
	"k8s.io/apimachinery/pkg/runtime"
	ctrl "sigs.k8s.io/controller-runtime"
	"sigs.k8s.io/controller-runtime/pkg/client"
	logger "sigs.k8s.io/controller-runtime/pkg/log"

	kvmv1alpha1 "github.com/cobaltcode-dev/kvm-node-agent/api/v1alpha1"
	"github.com/cobaltcode-dev/kvm-node-agent/internal/libvirt"
	"github.com/cobaltcode-dev/kvm-node-agent/internal/sys"
	"github.com/cobaltcode-dev/kvm-node-agent/internal/systemd"
)

// HypervisorReconciler reconciles a Hypervisor object
type HypervisorReconciler struct {
	client.Client
	Scheme         *runtime.Scheme
	libvirt        libvirt.Interface
	systemd        systemd.Interface
	libvirtVersion string
}

const (
	OSUpdateType = "OperatingSystemUpdate"
	LibVirtType  = "LibVirtConnection"
)

// +kubebuilder:rbac:groups=kvm.cloud.sap,resources=hypervisors,verbs=get;list;watch;create;update;patch;delete
// +kubebuilder:rbac:groups=kvm.cloud.sap,resources=hypervisors/status,verbs=get;update;patch
// +kubebuilder:rbac:groups=kvm.cloud.sap,resources=hypervisors/finalizers,verbs=update

func (r *HypervisorReconciler) Reconcile(ctx context.Context, req ctrl.Request) (ctrl.Result, error) {
	log := logger.FromContext(ctx, "controller", "hypervisor")

	// only reconcile the node I am running on
	if req.Name != sys.Hostname {
		// only reconcile the node I am running on
		return ctrl.Result{}, nil
	}
	log.Info("Reconcile", "name", req.Name, "namespace", req.Namespace)

	var hypervisor kvmv1alpha1.Hypervisor
	if err := r.Get(ctx, req.NamespacedName, &hypervisor); err != nil {
		// ignore not found errors, could be deleted
		return ctrl.Result{}, client.IgnoreNotFound(err)
	}

	meta.SetStatusCondition(&hypervisor.Status.Conditions, metav1.Condition{
		Type:    "Ready",
		Status:  metav1.ConditionFalse,
		Message: "Reconciling",
		Reason:  "Reconciling",
	})

	if osImage := hypervisor.Spec.OperatingSystemImage; osImage != nil {
		log.Info("OperatingSystemImage is set",
			"image", hypervisor.Spec.OperatingSystemImage.URL,
			"force", hypervisor.Spec.OperatingSystemImage.Force)

		meta.SetStatusCondition(&hypervisor.Status.Conditions, metav1.Condition{
			Type:    OSUpdateType,
			Status:  metav1.ConditionTrue,
			Message: "image set to " + hypervisor.Spec.OperatingSystemImage.URL,
			Reason:  "ImageSet",
		})
		/* #TODO: run priviledged operating system update
		success, err := update.UpdateOs(ctx, osImage.URL, osImage.Force)
		if err != nil {
			// todo: use EnqueueRequestsFromMapFunc for callback
			log.Error(err, "unable to update operating system")
			meta.SetStatusCondition(&hypervisor.Status.Conditions, metav1.Condition{
				Type:    OSUpdateType,
				Status:  metav1.ConditionFalse,
				Message: err.Error(),
				Reason:  "UpdateFailed",
			})
			return ctrl.Result{}, err
		}

		if success {
			log.Info("Operating system updated successfully")
		}
		*/
	}

	// Update libvirt status
	if !r.libvirt.IsConnected() {
		meta.SetStatusCondition(&hypervisor.Status.Conditions, metav1.Condition{
			Type:    LibVirtType,
			Status:  metav1.ConditionFalse,
			Message: "libvirt not connected",
			Reason:  "NotConnected",
		})
	}

	// Try (re)connect to libvirt
	if !r.libvirt.IsConnected() {
		// Connect to libvirt
		if err := r.libvirt.Connect(); err != nil {
			log.Error(err, "unable to connect to libvirt system bus")
			meta.SetStatusCondition(&hypervisor.Status.Conditions, metav1.Condition{
				Type:    LibVirtType,
				Status:  metav1.ConditionFalse,
				Message: err.Error(),
				Reason:  "ConnectFailed",
			})
		} else {
			if r.libvirtVersion, err = r.libvirt.GetVersion(); err != nil {
				log.Error(err, "unable to fetch libvirt version")
			}
		}
	}

	if r.libvirt.IsConnected() {
		hypervisor.Status.Version = r.libvirtVersion
		meta.SetStatusCondition(&hypervisor.Status.Conditions, metav1.Condition{
			Type:   LibVirtType,
			Status: metav1.ConditionTrue,
			Reason: "Connected",
		})

		// Update hypervisor instances
		var err error
		hypervisor.Status.Instances, err = r.libvirt.GetInstances()
		if err != nil {
			log.Error(err, "unable to list instances")
			return ctrl.Result{}, err
		}
		hypervisor.Status.NumInstances = len(hypervisor.Status.Instances)
	}

	if r.systemd.IsConnected() {
		var unitNames = []string{"libvirtd.service", "openvswitch-switch.service"}
		units, err := r.systemd.ListUnitsByNames(ctx, unitNames)
		if err != nil {
			log.Error(err, "unable to list units")
			return ctrl.Result{}, err
		}

		var unitReasonsMap = map[string]string{
			"active":   "Running",
			"inactive": "Stopped",
		}
		var unitStatusesMap = map[string]metav1.ConditionStatus{
			"active":   metav1.ConditionTrue,
			"inactive": metav1.ConditionFalse,
		}

		for _, unit := range units {
			reason := unitReasonsMap[unit.ActiveState]
			status := unitStatusesMap[unit.ActiveState]
			meta.SetStatusCondition(&hypervisor.Status.Conditions, metav1.Condition{
				Type:    unit.Name,
				Status:  status,
				Reason:  reason,
				Message: fmt.Sprintf("%s: %s, %s", unit.Name, unit.ActiveState, unit.LoadState),
			})
		}
	}

	meta.SetStatusCondition(&hypervisor.Status.Conditions, metav1.Condition{
		Type:   "Ready",
		Status: metav1.ConditionTrue,
		Reason: "Reconciled",
	})

	if err := r.Status().Update(ctx, &hypervisor); err != nil {
		log.Error(err, "unable to update hypervisor status")
		return ctrl.Result{}, err
	}
	return ctrl.Result{}, nil
}

// SetupWithManager sets up the controller with the Manager.
func (r *HypervisorReconciler) SetupWithManager(mgr ctrl.Manager) error {
	r.libvirt = libvirt.NewLibVirt()

	var err error
	r.systemd, err = systemd.NewSystemd(context.Background())
	if err != nil {
		return fmt.Errorf("unable to connect to systemd: %w", err)
	}

	return ctrl.NewControllerManagedBy(mgr).
		For(&kvmv1alpha1.Hypervisor{}).
		Complete(r)
}
