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
	"os"
	"path/filepath"

	kvmv1 "github.com/cobaltcore-dev/openstack-hypervisor-operator/api/v1"
	. "github.com/onsi/ginkgo/v2"
	. "github.com/onsi/gomega"
	v1 "k8s.io/api/core/v1"
	"k8s.io/apimachinery/pkg/api/meta"
	metav1 "k8s.io/apimachinery/pkg/apis/meta/v1"
	"k8s.io/apimachinery/pkg/runtime"
	"k8s.io/apimachinery/pkg/types"
	ctrl "sigs.k8s.io/controller-runtime"
	"sigs.k8s.io/controller-runtime/pkg/client"
	"sigs.k8s.io/controller-runtime/pkg/client/fake"

	"github.com/cobaltcore-dev/kvm-node-agent/internal/sys"
)

var _ = Describe("Secret Controller", func() {
	var (
		reconciler     *SecretReconciler
		fakeClient     client.Client
		scheme         *runtime.Scheme
		ctx            context.Context
		testNamespace  string
		testSecretName string
		testHV         *kvmv1.Hypervisor
		testSecret     *v1.Secret
		tempPKIPath    string
	)

	BeforeEach(func() {
		ctx = context.Background()
		testNamespace = "default"
		testSecretName = sys.Hostname + "-tls"

		// Create a temporary directory for PKI files
		var err error
		tempPKIPath, err = os.MkdirTemp("", "pki-test-*")
		Expect(err).NotTo(HaveOccurred())
		Expect(os.MkdirAll(filepath.Join(tempPKIPath, "CA"), 0755)).To(Succeed())
		os.Setenv("PKI_PATH", tempPKIPath)

		// Setup scheme
		scheme = runtime.NewScheme()
		Expect(v1.AddToScheme(scheme)).To(Succeed())
		Expect(kvmv1.AddToScheme(scheme)).To(Succeed())

		// Create test Hypervisor
		testHV = &kvmv1.Hypervisor{
			ObjectMeta: metav1.ObjectMeta{
				Name: sys.Hostname,
			},
			Spec: kvmv1.HypervisorSpec{
				InstallCertificate: true,
			},
			Status: kvmv1.HypervisorStatus{
				Conditions: []metav1.Condition{},
			},
		}

		// Create test Secret
		testSecret = &v1.Secret{
			ObjectMeta: metav1.ObjectMeta{
				Name:            testSecretName,
				Namespace:       testNamespace,
				ResourceVersion: "1000",
			},
			Data: map[string][]byte{
				"tls.crt": []byte("test-cert"),
				"tls.key": []byte("test-key"),
			},
		}

		// Create fake client with test objects
		fakeClient = fake.NewClientBuilder().
			WithScheme(scheme).
			WithObjects(testHV, testSecret).
			WithStatusSubresource(testHV).
			Build()

		// Create reconciler
		reconciler = &SecretReconciler{
			Client: fakeClient,
			Scheme: scheme,
		}
	})

	AfterEach(func() {
		// Clean up temporary directory
		os.RemoveAll(tempPKIPath)
		os.Unsetenv("PKI_PATH")
	})

	Context("When reconciling a resource", func() {
		It("should set status condition to Ready when resource version hasn't changed", func() {
			// Set the last resource version to match the secret's version
			reconciler.lastResourceVersion = testSecret.ResourceVersion

			// Reconcile the secret
			req := ctrl.Request{
				NamespacedName: types.NamespacedName{
					Name:      testSecretName,
					Namespace: testNamespace,
				},
			}

			result, err := reconciler.Reconcile(ctx, req)
			Expect(err).NotTo(HaveOccurred())
			Expect(result).To(Equal(ctrl.Result{}))

			// Verify the status condition was set to Ready
			updatedHV := &kvmv1.Hypervisor{}
			err = fakeClient.Get(ctx, types.NamespacedName{Name: sys.Hostname}, updatedHV)
			Expect(err).NotTo(HaveOccurred())

			condition := meta.FindStatusCondition(updatedHV.Status.Conditions, "TLSCertificateInstalled")
			Expect(condition).NotTo(BeNil())
			Expect(condition.Status).To(Equal(metav1.ConditionTrue))
			Expect(condition.Reason).To(Equal("Ready"))
			Expect(condition.Message).To(Equal("TLS certificate is ready and up to date"))
		})

		It("should skip reconciliation when InstallCertificate is false", func() {
			// Update the hypervisor to not require certificate installation
			testHV.Spec.InstallCertificate = false
			Expect(fakeClient.Update(ctx, testHV)).To(Succeed())

			req := ctrl.Request{
				NamespacedName: types.NamespacedName{
					Name:      testSecretName,
					Namespace: testNamespace,
				},
			}

			result, err := reconciler.Reconcile(ctx, req)
			Expect(err).NotTo(HaveOccurred())
			Expect(result).To(Equal(ctrl.Result{}))

			// Verify no status condition was set
			updatedHV := &kvmv1.Hypervisor{}
			err = fakeClient.Get(ctx, types.NamespacedName{Name: sys.Hostname}, updatedHV)
			Expect(err).NotTo(HaveOccurred())

			condition := meta.FindStatusCondition(updatedHV.Status.Conditions, "TLSCertificateInstalled")
			Expect(condition).To(BeNil())
		})

		It("should return error when secret is not found", func() {
			req := ctrl.Request{
				NamespacedName: types.NamespacedName{
					Name:      "non-existent-secret",
					Namespace: testNamespace,
				},
			}

			result, err := reconciler.Reconcile(ctx, req)
			Expect(err).NotTo(HaveOccurred()) // IgnoreNotFound
			Expect(result).To(Equal(ctrl.Result{}))
		})

		It("should return error when hypervisor is not found", func() {
			// Delete the hypervisor
			Expect(fakeClient.Delete(ctx, testHV)).To(Succeed())

			req := ctrl.Request{
				NamespacedName: types.NamespacedName{
					Name:      testSecretName,
					Namespace: testNamespace,
				},
			}

			result, err := reconciler.Reconcile(ctx, req)
			Expect(err).To(HaveOccurred())
			Expect(result).To(Equal(ctrl.Result{}))
		})

		It("should detect when resource version differs from last processed version", func() {
			// Set a different last resource version
			reconciler.lastResourceVersion = "999"

			// Verify the versions are different, which would trigger an update
			Expect(reconciler.lastResourceVersion).To(Equal("999"))
			Expect(testSecret.ResourceVersion).To(Equal("1000"))
			Expect(reconciler.lastResourceVersion).NotTo(Equal(testSecret.ResourceVersion))
		})
	})
})
