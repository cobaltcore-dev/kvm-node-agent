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

	"github.com/coreos/go-systemd/v22/dbus"
	golibvirt "github.com/digitalocean/go-libvirt"
	. "github.com/onsi/ginkgo/v2"
	. "github.com/onsi/gomega"
	"k8s.io/apimachinery/pkg/api/errors"
	"k8s.io/apimachinery/pkg/api/resource"
	"k8s.io/apimachinery/pkg/types"
	"sigs.k8s.io/controller-runtime/pkg/reconcile"

	metav1 "k8s.io/apimachinery/pkg/apis/meta/v1"

	kvmv1alpha1 "github.com/cobaltcore-dev/kvm-node-agent/api/v1alpha1"
	"github.com/cobaltcore-dev/kvm-node-agent/internal/libvirt"
	"github.com/cobaltcore-dev/kvm-node-agent/internal/sys"
	"github.com/cobaltcore-dev/kvm-node-agent/internal/systemd"
)

var _ = Describe("Hypervisor Controller", func() {
	Context("When reconciling a resource", func() {
		const resourceName = "test-resource"

		ctx := context.Background()

		typeNamespacedName := types.NamespacedName{
			Name:      resourceName,
			Namespace: "default", // TODO(user):Modify as needed
		}
		hypervisor := &kvmv1alpha1.Hypervisor{}

		BeforeEach(func() {
			By("creating the custom resource for the Kind Hypervisor")
			err := k8sClient.Get(ctx, typeNamespacedName, hypervisor)
			if err != nil && errors.IsNotFound(err) {
				resource := &kvmv1alpha1.Hypervisor{
					ObjectMeta: metav1.ObjectMeta{
						Name:      resourceName,
						Namespace: "default",
					},
					// TODO(user): Specify other spec details if needed.
				}
				Expect(k8sClient.Create(ctx, resource)).To(Succeed())
			}

			// override the hostname
			sys.Hostname = resourceName
		})

		AfterEach(func() {
			// TODO(user): Cleanup logic after each test, like removing the resource instance.
			resource := &kvmv1alpha1.Hypervisor{}
			err := k8sClient.Get(ctx, typeNamespacedName, resource)
			Expect(err).NotTo(HaveOccurred())

			By("Cleanup the specific resource instance Hypervisor")
			Expect(k8sClient.Delete(ctx, resource)).To(Succeed())
		})
		It("should successfully reconcile the resource", func() {
			By("Reconciling the created resource")
			controllerReconciler := &HypervisorReconciler{
				Client: k8sClient,
				Scheme: k8sClient.Scheme(),
				Libvirt: &libvirt.InterfaceMock{
					CloseFunc: func() error {
						return nil
					},
					ConnectFunc: func() error {
						return nil
					},
					GetDomainsActiveFunc: func() ([]golibvirt.Domain, error) {
						return []golibvirt.Domain{}, nil
					},
					GetInstancesFunc: func() ([]kvmv1alpha1.Instance, error) {
						return []kvmv1alpha1.Instance{
							{
								ID:     "25e2ea06-f6be-4bac-856d-8c2d0bdbcdee",
								Name:   "test-instance",
								Active: false,
							},
						}, nil
					},
					IsConnectedFunc: func() bool {
						return true
					},
					GetVersionFunc: func() string {
						return "10.9.0"
					},
					GetNumInstancesFunc: func() int {
						return 1
					},
					GetCapabilitiesFunc: func() (kvmv1alpha1.CapabilitiesStatus, error) {
						return kvmv1alpha1.CapabilitiesStatus{
							HostCpuArch: "x86_64",
							HostCpus:    *resource.NewQuantity(4, resource.DecimalSI),
							HostMemory:  *resource.NewQuantity(8192, resource.DecimalSI),
						}, nil
					},
				},
				Systemd: &systemd.InterfaceMock{
					CloseFunc: func() {},
					IsConnectedFunc: func() bool {
						return true
					},
					ListUnitsByNamesFunc: func(ctx context.Context, units []string) ([]dbus.UnitStatus, error) {
						return []dbus.UnitStatus{
							{
								Name:        "test-unit",
								Description: "test-unit-description",
								ActiveState: "active",
							},
						}, nil
					},
					DescribeFunc: func(ctx context.Context) (*systemd.Descriptor, error) {
						return nil, nil
					},
				},
			}

			_, err := controllerReconciler.Reconcile(ctx, reconcile.Request{
				NamespacedName: typeNamespacedName,
			})
			Expect(err).NotTo(HaveOccurred())
			By("Checking the status of the reconciled resource")
			err = k8sClient.Get(ctx, typeNamespacedName, hypervisor)
			Expect(err).NotTo(HaveOccurred())
			Expect(hypervisor.Status.Instances).To(HaveLen(1))
			Expect(hypervisor.Status.Instances[0].ID).To(Equal("25e2ea06-f6be-4bac-856d-8c2d0bdbcdee"))

			Expect(hypervisor.Status.Conditions).To(HaveLen(3))

			Expect(hypervisor.Status.Conditions[0].Type).To(Equal("LibVirtConnection"))
			Expect(hypervisor.Status.Conditions[0].Status).To(Equal(metav1.ConditionTrue))
			Expect(hypervisor.Status.Conditions[0].Reason).To(Equal("Connected"))

			Expect(hypervisor.Status.Conditions[1].Type).To(Equal("CapabilitiesClientConnection"))
			Expect(hypervisor.Status.Conditions[1].Status).To(Equal(metav1.ConditionTrue))
			Expect(hypervisor.Status.Conditions[1].Reason).To(Equal("CapabilitiesClientGetSucceeded"))

			Expect(hypervisor.Status.Conditions[2].Type).To(Equal("test-unit"))
			Expect(hypervisor.Status.Conditions[2].Status).To(Equal(metav1.ConditionTrue))
			Expect(hypervisor.Status.Conditions[2].Reason).To(Equal("Running"))

			Expect(hypervisor.Status.Capabilities.HostCpuArch).To(Equal("x86_64"))
			Expect(hypervisor.Status.Capabilities.HostCpus.AsDec().UnscaledBig()).
				To(Equal(resource.NewQuantity(4, resource.DecimalSI).AsDec().UnscaledBig()))
			Expect(hypervisor.Status.Capabilities.HostMemory.AsDec().UnscaledBig()).
				To(Equal(resource.NewQuantity(8192, resource.DecimalSI).AsDec().UnscaledBig()))
		})
	})
})
