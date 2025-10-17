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
	"os"
	"path/filepath"

	kvmv1 "github.com/cobaltcore-dev/openstack-hypervisor-operator/api/v1"
	v1 "k8s.io/api/core/v1"
	"k8s.io/apimachinery/pkg/api/meta"
	metav1 "k8s.io/apimachinery/pkg/apis/meta/v1"
	"k8s.io/apimachinery/pkg/runtime"
	"k8s.io/apimachinery/pkg/types"
	"k8s.io/client-go/util/retry"
	ctrl "sigs.k8s.io/controller-runtime"
	"sigs.k8s.io/controller-runtime/pkg/client"
	"sigs.k8s.io/controller-runtime/pkg/handler"
	logger "sigs.k8s.io/controller-runtime/pkg/log"
	"sigs.k8s.io/controller-runtime/pkg/reconcile"

	"github.com/cobaltcore-dev/kvm-node-agent/internal/certificates"
	"github.com/cobaltcore-dev/kvm-node-agent/internal/sys"
	"github.com/cobaltcore-dev/kvm-node-agent/internal/systemd"
)

// SecretReconciler reconciles a Secret object
type SecretReconciler struct {
	client.Client
	Scheme  *runtime.Scheme
	Systemd systemd.Interface

	lastResourceVersion string
}

// +kubebuilder:rbac:groups="",resources=secrets,verbs=get;list;watch
// +kubebuilder:rbac:groups=kvm.cloud.sap,resources=hypervisors,verbs=get;list;watch
// +kubebuilder:rbac:groups=kvm.cloud.sap,resources=hypervisors/status,verbs=get;update;patch

// Reconcile is part of the main kubernetes reconciliation loop which aims to
// move the current state of the cluster closer to the desired state.
func (r *SecretReconciler) Reconcile(ctx context.Context, req ctrl.Request) (ctrl.Result, error) {
	log := logger.FromContext(ctx)

	// Fetch the Secret instance
	secret := &v1.Secret{}
	err := r.Get(ctx, req.NamespacedName, secret)
	if err != nil {
		return ctrl.Result{}, client.IgnoreNotFound(err)
	}

	// Fetch the Hypervisor instance
	hv := &kvmv1.Hypervisor{}
	if err = r.Get(ctx, types.NamespacedName{Name: sys.Hostname}, hv); err != nil {
		return ctrl.Result{}, err
	}

	if !hv.Spec.InstallCertificate {
		log.Info("Hypervisor does not require TLS certificate installation, skipping Secret reconciliation")
		return ctrl.Result{}, nil
	}

	if secret.ResourceVersion == r.lastResourceVersion {
		return ctrl.Result{}, nil
	}

	if err = r.setTLSStatusCondition(ctx, metav1.ConditionFalse,
		"Installing", "Installing TLS certificate from Secret"); err != nil {
		return ctrl.Result{}, err
	}

	if err = certificates.UpdateTLSCertificate(ctx, secret.Data); err != nil {
		// update conditions
		if err := r.setTLSStatusCondition(ctx, metav1.ConditionFalse,
			"FailedToUpdateTLSCertificate", fmt.Sprintf("Failed to update TLS certificate: %v", err)); err != nil {
			return ctrl.Result{}, err
		}
		return ctrl.Result{}, err
	}

	// Reload the libvirtd service
	if _, err = r.Systemd.StartUnit(ctx, "virt-admin-server-update-tls.service"); err != nil {
		if err := r.setTLSStatusCondition(ctx, metav1.ConditionFalse,
			"FailedToStartUpdateTLSService",
			fmt.Sprintf("Failed to start virt-admin-server-update-tls service: %v", err)); err != nil {
			return ctrl.Result{}, err
		}
		log.Error(err, "failed to start virt-admin-server-update-tls service")
		// Start the libvirtd service
		if _, err = r.Systemd.StartUnit(ctx, "libvirtd.service"); err != nil {
			return ctrl.Result{}, err
		}
	}

	// Save the last resource version to file system
	pki := os.Getenv("PKI_PATH")
	path := filepath.Join(pki, "CA", ".last_resource_version")
	if err = os.WriteFile(path, []byte(secret.ResourceVersion), 0600); err != nil {
		// not a failure condition, just log the error
		log.Error(err, "failed to write last resource version", "path", path)
	}
	r.lastResourceVersion = secret.ResourceVersion

	return ctrl.Result{}, r.setTLSStatusCondition(ctx, metav1.ConditionTrue, "Ready",
		"TLS certificate is ready and updated")
}

// SetupWithManager sets up the controller with the Manager.
func (r *SecretReconciler) SetupWithManager(mgr ctrl.Manager) error {
	// Load the last resource version from file system, so we can skip
	// processing if the resource version hasn't changed
	pki := os.Getenv("PKI_PATH")
	path := filepath.Join(pki, "CA", ".last_resource_version")
	if buf, err := os.ReadFile(path); err != nil {
		logger.Log.Info("No last resource version found for PKI secrets", "path", path)
	} else {
		r.lastResourceVersion = string(buf)
	}

	secretName, _ := certificates.GetSecretAndCertName(sys.Hostname)
	// Watch for changes to Secrets for this specific host
	evHandler := handler.EnqueueRequestsFromMapFunc(func(ctx context.Context, a client.Object) []reconcile.Request {
		secret := a.(*v1.Secret)
		if secret.Name == secretName {
			return []reconcile.Request{
				{NamespacedName: types.NamespacedName{Name: secretName, Namespace: secret.Namespace}},
			}
		}
		return nil
	})
	return ctrl.NewControllerManagedBy(mgr).
		Named("secret").
		Watches(&v1.Secret{}, evHandler).
		Complete(r)
}

func (r *SecretReconciler) setTLSStatusCondition(ctx context.Context, status metav1.ConditionStatus,
	reason, message string) error {

	log := logger.FromContext(ctx)
	hv := &kvmv1.Hypervisor{}

	return retry.RetryOnConflict(retry.DefaultRetry, func() error {
		if err := r.Get(ctx, types.NamespacedName{Name: sys.Hostname}, hv); err != nil {
			log.Error(err, "failed to get hypervisor for updating status condition")
			return err
		}

		meta.SetStatusCondition(&hv.Status.Conditions, metav1.Condition{
			Type:    "TLSCertificateInstalled",
			Status:  status,
			Reason:  reason,
			Message: message,
		})

		return r.Status().Update(ctx, hv)
	})
}
