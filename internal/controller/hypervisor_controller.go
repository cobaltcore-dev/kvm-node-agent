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
	"errors"
	"fmt"
	"os"
	"strings"
	"time"

	"k8s.io/apimachinery/pkg/api/meta"
	metav1 "k8s.io/apimachinery/pkg/apis/meta/v1"
	"k8s.io/apimachinery/pkg/runtime"
	ctrl "sigs.k8s.io/controller-runtime"
	"sigs.k8s.io/controller-runtime/pkg/client"
	logger "sigs.k8s.io/controller-runtime/pkg/log"

	kvmv1alpha1 "github.com/cobaltcode-dev/kvm-node-agent/api/v1alpha1"
	"github.com/cobaltcode-dev/kvm-node-agent/internal/emulator"
	"github.com/cobaltcode-dev/kvm-node-agent/internal/evacuation"
	"github.com/cobaltcode-dev/kvm-node-agent/internal/libvirt"
	"github.com/cobaltcode-dev/kvm-node-agent/internal/sys"
	"github.com/cobaltcode-dev/kvm-node-agent/internal/systemd"
)

// HypervisorReconciler reconciles a Hypervisor object
type HypervisorReconciler struct {
	client.Client
	Scheme           *runtime.Scheme
	libvirt          libvirt.Interface
	systemd          systemd.Interface
	osDescriptor     *systemd.Descriptor
	evacuateOnReboot bool
}

const (
	OSUpdateType = "OperatingSystemUpdate"
	LibVirtType  = "LibVirtConnection"
)

// +kubebuilder:rbac:groups=kvm.cloud.sap,resources=hypervisors,verbs=get;list;watch;create;update;patch;delete
// +kubebuilder:rbac:groups=kvm.cloud.sap,resources=hypervisors/status,verbs=get;update;patch
// +kubebuilder:rbac:groups=kvm.cloud.sap,resources=hypervisors/finalizers,verbs=update
// +kubebuilder:rbac:groups=kvm.cloud.sap,resources=evictions,verbs=get;create
// +kubebuilder:rbac:groups=kvm.cloud.sap,resources=migrations,verbs=get;list;watch;create
// +kubebuilder:rbac:groups=kvm.cloud.sap,resources=migrations/status,verbs=get;update;patch

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

	if hypervisor.Spec.EvacuateOnReboot != r.evacuateOnReboot {
		if hypervisor.Spec.EvacuateOnReboot {
			e := &evacuation.EvictionController{Client: r.Client}
			if err := r.systemd.EnableShutdownInhibit(ctx, e.EvictCurrentHost); err != nil {
				return ctrl.Result{}, err
			}
		} else {
			if err := r.systemd.DisableShutdownInhibit(); err != nil {
				return ctrl.Result{}, err
			}
		}
		r.evacuateOnReboot = hypervisor.Spec.EvacuateOnReboot
	}

	meta.SetStatusCondition(&hypervisor.Status.Conditions, metav1.Condition{
		Type:    "Ready",
		Status:  metav1.ConditionFalse,
		Message: "Reconciling",
		Reason:  "Reconciling",
	})

	// ====================================================================================================
	// libvirt
	// ====================================================================================================

	// Try (re)connect to libvirt
	if err := r.libvirt.Connect(); err != nil {
		log.Error(err, "unable to connect to libvirt system bus")
		meta.SetStatusCondition(&hypervisor.Status.Conditions, metav1.Condition{
			Type:    LibVirtType,
			Status:  metav1.ConditionFalse,
			Message: err.Error(),
			Reason:  "ConnectFailed",
		})
	} else {
		hypervisor.Status.LibVirtVersion = r.libvirt.GetVersion()
		meta.SetStatusCondition(&hypervisor.Status.Conditions, metav1.Condition{
			Type:   LibVirtType,
			Status: metav1.ConditionTrue,
			Reason: "Connected",
		})

		// Update hypervisor instances
		hypervisor.Status.Instances, err = r.libvirt.GetInstances()
		if err != nil {
			log.Error(err, "unable to list instances")
			return ctrl.Result{}, err
		}
		hypervisor.Status.NumInstances = len(hypervisor.Status.Instances)
	}

	// ====================================================================================================
	// systemd
	// ====================================================================================================

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

		if r.osDescriptor != nil && hypervisor.Status.OperatingSystem.Version == "" {
			for _, line := range r.osDescriptor.OperatingSystemReleaseData {
				switch strings.Split(line, "=")[0] {
				case "PRETTY_NAME":
					hypervisor.Status.OperatingSystem.PrettyVersion = strings.Split(line, "=")[1]
				case "GARDENLINUX_VERSION":
					hypervisor.Status.OperatingSystem.Version = strings.Split(line, "=")[1]
				}
			}
			hypervisor.Status.OperatingSystem.KernelVersion = r.osDescriptor.KernelVersion
			hypervisor.Status.OperatingSystem.KernelRelease = r.osDescriptor.KernelRelease
			hypervisor.Status.OperatingSystem.KernelName = r.osDescriptor.KernelName
			hypervisor.Status.OperatingSystem.HardwareVendor = r.osDescriptor.HardwareVendor
			hypervisor.Status.OperatingSystem.HardwareModel = r.osDescriptor.HardwareModel
			hypervisor.Status.OperatingSystem.HardwareSerial = r.osDescriptor.HardwareSerial
			hypervisor.Status.OperatingSystem.FirmwareVersion = r.osDescriptor.FirmwareVersion
			hypervisor.Status.OperatingSystem.FirmwareVendor = r.osDescriptor.FirmwareVendor
			hypervisor.Status.OperatingSystem.FirmwareDate = metav1.NewTime(time.UnixMicro(r.osDescriptor.FirmwareDate))
		}
	}

	// Reconcile operating system update
	if hypervisor.Spec.OperatingSystemVersion != "" &&
		// only update if the version is different to current running version
		hypervisor.Spec.OperatingSystemVersion != hypervisor.Status.OperatingSystem.Version &&
		// only update if the version is different to the installed version
		hypervisor.Spec.OperatingSystemVersion != hypervisor.Status.Update.Installed {

		if hypervisor.Status.Update.Retry == 0 {
			// we reached the retry limit, unset the version to stop the update
			// failed message of past retries is still available in the conditions

			// reset retry count
			hypervisor.Status.Update.Retry = 3
			if err := r.Status().Update(ctx, &hypervisor); err != nil {
				log.Error(err, "unable to update hypervisor status spec")
				return ctrl.Result{}, err
			}
			hypervisor.Spec.OperatingSystemVersion = ""
			if err := r.Update(ctx, &hypervisor); err != nil {
				log.Error(err, "unable to update hypervisor spec")
				return ctrl.Result{}, err
			}

			// Todo: include some timeout?
			return ctrl.Result{}, nil
		}

		// Reconcile operating system update
		running, err := r.systemd.ReconcileSysUpdate(ctx, &hypervisor)

		// failed
		if err != nil {
			meta.SetStatusCondition(&hypervisor.Status.Conditions, metav1.Condition{
				Type:    OSUpdateType,
				Status:  metav1.ConditionFalse,
				Reason:  "Stopped",
				Message: err.Error(),
			})

			if !errors.Is(err, systemd.ErrFailed) {
				log.Error(err, "error while reconcile operating system update")
			}

			// decrease retry count
			hypervisor.Status.Update.Retry--
		}

		// started
		if !hypervisor.Status.Update.InProgress && running {
			meta.SetStatusCondition(&hypervisor.Status.Conditions, metav1.Condition{
				Type:   OSUpdateType,
				Status: metav1.ConditionTrue,
				Reason: "Running",
				Message: fmt.Sprintf("Operating system update to %s is running",
					hypervisor.Spec.OperatingSystemVersion),
			})
		}

		// finished
		if !running && err == nil {
			meta.SetStatusCondition(&hypervisor.Status.Conditions, metav1.Condition{
				Type:   OSUpdateType,
				Status: metav1.ConditionTrue,
				Reason: "Completed",
				Message: fmt.Sprintf("Operating system update %s is installed",
					hypervisor.Spec.OperatingSystemVersion),
			})
			hypervisor.Status.Update.Installed = hypervisor.Spec.OperatingSystemVersion
		}
		hypervisor.Status.Update.InProgress = running
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
	return ctrl.Result{RequeueAfter: 1 * time.Minute}, nil
}

// SetupWithManager sets up the controller with the Manager.
func (r *HypervisorReconciler) SetupWithManager(mgr ctrl.Manager) error {
	ctx := context.Background()
	log := logger.Log.WithName("HypervisorReconciler")
	emulate := os.Getenv("EMULATE")

	var err error
	if emulate != "" {
		r.libvirt = emulator.NewLibVirtEmulator(ctx)
		r.systemd = emulator.NewSystemdEmulator(ctx)
	} else {
		r.libvirt = libvirt.NewLibVirt(mgr.GetClient())
		r.systemd, err = systemd.NewSystemd(ctx)
		if err != nil {
			return fmt.Errorf("unable to connect to systemd: %w", err)
		}
	}

	// Initialize libvirt connection
	if err := r.libvirt.Connect(); err != nil {
		log.Error(err, "unable to connect to libvirt system bus, reconnecting on reconcillation")
	}

	r.osDescriptor, err = r.systemd.Describe(ctx)
	if err != nil {
		return fmt.Errorf("unable to get systemd hostname describe(): %w", err)
	}

	return ctrl.NewControllerManagedBy(mgr).
		For(&kvmv1alpha1.Hypervisor{}).
		Complete(r)
}
