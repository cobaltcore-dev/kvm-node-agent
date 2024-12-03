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
	"strings"

	"github.com/coreos/go-systemd/v22/dbus"
	. "github.com/onsi/ginkgo/v2"
	. "github.com/onsi/gomega"
	"k8s.io/apimachinery/pkg/api/errors"
	"k8s.io/apimachinery/pkg/types"
	"sigs.k8s.io/controller-runtime/pkg/reconcile"

	metav1 "k8s.io/apimachinery/pkg/apis/meta/v1"

	"github.com/cobaltcode-dev/kvm-node-agent/api/v1alpha1"
	kvmv1alpha1 "github.com/cobaltcode-dev/kvm-node-agent/api/v1alpha1"
	"github.com/cobaltcode-dev/kvm-node-agent/internal/libvirt"
	"github.com/cobaltcode-dev/kvm-node-agent/internal/sys"
	"github.com/cobaltcode-dev/kvm-node-agent/internal/systemd"
)

var _ = Describe("Hypervisor Controller", func() {
	Context("When reconciling a resource", func() {
		const resourceName = "test-resource"
		const operatingSystemVersion = "0.0.9"
		const updateSystemVersion = "0.1.0"

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
					Spec: kvmv1alpha1.HypervisorSpec{
						OperatingSystemVersion: operatingSystemVersion,
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
		It("should successfully reconcile the resource - same version", func() {
			By("Reconciling the created resource")
			controllerReconciler := &HypervisorReconciler{
				OperatingSystemVersion: operatingSystemVersion,
				Client:                 k8sClient,
				Scheme:                 k8sClient.Scheme(),
				libvirt: &libvirt.InterfaceMock{
					CloseFunc: func() error {
						return nil
					},
					ConnectFunc: func() error {
						return nil
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
				},
				systemd: &systemd.InterfaceMock{
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
			Expect(hypervisor.Status.Conditions[0].Type).To(Equal("Ready"))
			Expect(hypervisor.Status.Conditions[0].Status).To(Equal(metav1.ConditionTrue))
			Expect(hypervisor.Status.Conditions[0].Reason).To(Equal("Reconciled"))

			Expect(hypervisor.Status.Conditions[1].Type).To(Equal("LibVirtConnection"))
			Expect(hypervisor.Status.Conditions[1].Status).To(Equal(metav1.ConditionTrue))
			Expect(hypervisor.Status.Conditions[1].Reason).To(Equal("Connected"))

			Expect(hypervisor.Status.Conditions[2].Type).To(Equal("test-unit"))
			Expect(hypervisor.Status.Conditions[2].Status).To(Equal(metav1.ConditionTrue))
			Expect(hypervisor.Status.Conditions[2].Reason).To(Equal("Running"))
		})
		It("should successfully reconcile the resource - different version", func() {
			By("Reconciling the created resource")
			controllerReconciler := &HypervisorReconciler{
				OperatingSystemVersion: updateSystemVersion,
				Client:                 k8sClient,
				Scheme:                 k8sClient.Scheme(),
				libvirt: &libvirt.InterfaceMock{
					CloseFunc: func() error {
						return nil
					},
					ConnectFunc: func() error {
						return nil
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
				},
				systemd: &systemd.InterfaceMock{
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
			Expect(hypervisor.Status.Conditions[0].Type).To(Equal("Ready"))
			Expect(hypervisor.Status.Conditions[0].Status).To(Equal(metav1.ConditionFalse))
			Expect(hypervisor.Status.Conditions[0].Reason).To(Equal("Reconciling"))

			Expect(hypervisor.Status.Conditions[1].Type).To(Equal("LibVirtConnection"))
			Expect(hypervisor.Status.Conditions[1].Status).To(Equal(metav1.ConditionTrue))
			Expect(hypervisor.Status.Conditions[1].Reason).To(Equal("Connected"))

			Expect(hypervisor.Status.Conditions[2].Type).To(Equal("test-unit"))
			Expect(hypervisor.Status.Conditions[2].Status).To(Equal(metav1.ConditionTrue))
			Expect(hypervisor.Status.Conditions[2].Reason).To(Equal("Running"))

			_, err = controllerReconciler.Reconcile(ctx, reconcile.Request{
				NamespacedName: typeNamespacedName,
			})
			Expect(err).NotTo(HaveOccurred())
			err = k8sClient.Get(ctx, typeNamespacedName, hypervisor)
			Expect(err).NotTo(HaveOccurred())
			Expect(hypervisor.Status.Instances).To(HaveLen(1))
			Expect(hypervisor.Status.Instances[0].ID).To(Equal("25e2ea06-f6be-4bac-856d-8c2d0bdbcdee"))

			Expect(hypervisor.Status.Conditions).To(HaveLen(3))
			Expect(hypervisor.Status.Conditions[0].Type).To(Equal("Ready"))
			Expect(hypervisor.Status.Conditions[0].Status).To(Equal(metav1.ConditionTrue))
			Expect(hypervisor.Status.Conditions[0].Reason).To(Equal("Reconciled"))

			Expect(hypervisor.Status.Conditions[1].Type).To(Equal("LibVirtConnection"))
			Expect(hypervisor.Status.Conditions[1].Status).To(Equal(metav1.ConditionTrue))
			Expect(hypervisor.Status.Conditions[1].Reason).To(Equal("Connected"))

			Expect(hypervisor.Status.Conditions[2].Type).To(Equal("test-unit"))
			Expect(hypervisor.Status.Conditions[2].Status).To(Equal(metav1.ConditionTrue))
			Expect(hypervisor.Status.Conditions[2].Reason).To(Equal("Running"))

		})

		It("should failed to reconcile the resource with systemd issue", func() {
			By("Reconciling the created resource")
			controllerReconciler := &HypervisorReconciler{
				Client: k8sClient,
				Scheme: k8sClient.Scheme(),
				libvirt: &libvirt.InterfaceMock{
					CloseFunc: func() error {
						return nil
					},
					ConnectFunc: func() error {
						return nil
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
				},
				systemd: &systemd.InterfaceMock{
					CloseFunc: func() {},
					IsConnectedFunc: func() bool {
						return true
					},
					ListUnitsByNamesFunc: func(ctx context.Context, units []string) ([]dbus.UnitStatus, error) {
						return nil, fmt.Errorf("issue listing units by name")
					},
				},
			}

			_, err := controllerReconciler.Reconcile(ctx, reconcile.Request{
				NamespacedName: typeNamespacedName,
			})
			Expect(err).To(HaveOccurred())

			Expect(strings.Split(err.Error(), ":")[0]).To(Equal("unable to list units"))
		})

		It("should failed to reconcile the resource with libvirt issue", func() {
			By("Reconciling the created resource")
			controllerReconciler := &HypervisorReconciler{
				Client: k8sClient,
				Scheme: k8sClient.Scheme(),
				libvirt: &libvirt.InterfaceMock{
					CloseFunc: func() error {
						return nil
					},
					ConnectFunc: func() error {
						return fmt.Errorf("failed to connect")
					},
					IsConnectedFunc: func() bool {
						return false
					},
				},
			}

			_, err := controllerReconciler.Reconcile(ctx, reconcile.Request{
				NamespacedName: typeNamespacedName,
			})
			Expect(err).To(HaveOccurred())
		})
	})
})

var _ = Describe("Hypervisor Controller - testing update", func() {
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
			resource := &kvmv1alpha1.Hypervisor{}
			err := k8sClient.Get(ctx, typeNamespacedName, hypervisor)
			if err != nil && errors.IsNotFound(err) {
				resource = &kvmv1alpha1.Hypervisor{
					ObjectMeta: metav1.ObjectMeta{
						Name:      resourceName,
						Namespace: "default",
					},
					Spec: kvmv1alpha1.HypervisorSpec{
						OperatingSystemVersion: "0.0.9",
					},
					// TODO(user): Specify other spec details if needed.
				}
				Expect(k8sClient.Create(ctx, resource)).To(Succeed())
			}

			By("updating")
			err = k8sClient.Status().Update(ctx, resource)
			Expect(err).NotTo(HaveOccurred())
			err = k8sClient.Update(ctx, resource)
			Expect(err).NotTo(HaveOccurred())

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
		It("should failed to reconcile the resource with ReconcileSysUpdate issue", func() {
			By("Reconciling the created resource")
			controllerReconciler := &HypervisorReconciler{
				OperatingSystemVersion: "0.1",
				Client:                 k8sClient,
				Scheme:                 k8sClient.Scheme(),
				libvirt: &libvirt.InterfaceMock{
					CloseFunc: func() error {
						return nil
					},
					ConnectFunc: func() error {
						return nil
					},
					GetInstancesFunc: func() ([]kvmv1alpha1.Instance, error) {
						return []kvmv1alpha1.Instance{
							{
								ID:     "25e2ea06-f6be-4bac-856d-8c2d0bdbcdee",
								Name:   "test-instance",
								Active: true,
							},
						}, nil
					},
					IsConnectedFunc: func() bool {
						return true
					},
					GetVersionFunc: func() (string, error) {
						return "0.1", nil
					},
				},
				systemd: &systemd.InterfaceMock{
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
							{
								Name:        "test-unit2",
								Description: "test-unit2-description",
								ActiveState: "active",
							},
						}, nil
					},
					ReconcileSysUpdateFunc: func(ctx context.Context, hv *v1alpha1.Hypervisor) (bool, error) {
						return false, fmt.Errorf("issue reconcile sys update func")
					},
				},
			}

			_, err := controllerReconciler.Reconcile(ctx, reconcile.Request{
				NamespacedName: typeNamespacedName,
			})
			Expect(err).NotTo(HaveOccurred())
			err = k8sClient.Get(ctx, typeNamespacedName, hypervisor)
			Expect(err).NotTo(HaveOccurred())
			err = k8sClient.Status().Update(ctx, hypervisor)
			Expect(err).NotTo(HaveOccurred())
			err = k8sClient.Update(ctx, hypervisor)
			Expect(err).NotTo(HaveOccurred())
		})
	})
})
