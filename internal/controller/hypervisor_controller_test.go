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
	"errors"
	"time"

	kvmv1 "github.com/cobaltcore-dev/openstack-hypervisor-operator/api/v1"
	"github.com/coreos/go-systemd/v22/dbus"
	golibvirt "github.com/digitalocean/go-libvirt"
	. "github.com/onsi/ginkgo/v2"
	. "github.com/onsi/gomega"
	apierrors "k8s.io/apimachinery/pkg/api/errors"
	"k8s.io/apimachinery/pkg/api/resource"
	metav1 "k8s.io/apimachinery/pkg/apis/meta/v1"
	"k8s.io/apimachinery/pkg/types"
	ctrl "sigs.k8s.io/controller-runtime"
	"sigs.k8s.io/controller-runtime/pkg/event"
	"sigs.k8s.io/controller-runtime/pkg/reconcile"

	"github.com/cobaltcore-dev/kvm-node-agent/internal/libvirt"
	"github.com/cobaltcore-dev/kvm-node-agent/internal/sys"
	"github.com/cobaltcore-dev/kvm-node-agent/internal/systemd"
)

var _ = Describe("Hypervisor Controller", func() {
	Context("When testing Start method", func() {
		It("should successfully start and subscribe to libvirt events", func() {
			ctx := context.Background()
			eventCallbackCalled := false

			controllerReconciler := &HypervisorReconciler{
				Client: k8sClient,
				Scheme: k8sClient.Scheme(),
				Libvirt: &libvirt.InterfaceMock{
					ConnectFunc: func() error {
						return nil
					},
					WatchDomainChangesFunc: func(eventId golibvirt.DomainEventID, handlerId string, handler func(context.Context, any)) {
						eventCallbackCalled = true
						Expect(handlerId).To(Equal("reconcile-on-domain-lifecycle"))
					},
				},
				reconcileCh: make(chan event.GenericEvent, 1),
			}

			err := controllerReconciler.Start(ctx)
			Expect(err).NotTo(HaveOccurred())
			Expect(eventCallbackCalled).To(BeTrue())
		})

		It("should fail when libvirt connection fails", func() {
			ctx := context.Background()

			controllerReconciler := &HypervisorReconciler{
				Client: k8sClient,
				Scheme: k8sClient.Scheme(),
				Libvirt: &libvirt.InterfaceMock{
					ConnectFunc: func() error {
						return errors.New("connection failed")
					},
				},
				reconcileCh: make(chan event.GenericEvent, 1),
			}

			err := controllerReconciler.Start(ctx)
			Expect(err).To(HaveOccurred())
			Expect(err.Error()).To(ContainSubstring("connection failed"))
		})
	})

	Context("When testing triggerReconcile method", func() {
		It("should send an event to reconcile channel", func() {
			const testHostname = "test-host"
			const testNamespace = "test-namespace"

			// Override hostname and namespace for this test
			originalHostname := sys.Hostname
			originalNamespace := sys.Namespace
			sys.Hostname = testHostname
			sys.Namespace = testNamespace
			defer func() {
				sys.Hostname = originalHostname
				sys.Namespace = originalNamespace
			}()

			controllerReconciler := &HypervisorReconciler{
				Client:      k8sClient,
				Scheme:      k8sClient.Scheme(),
				reconcileCh: make(chan event.GenericEvent, 1),
			}

			// Trigger reconcile in a goroutine to avoid blocking
			go controllerReconciler.triggerReconcile()

			// Wait for the event with a timeout
			select {
			case evt := <-controllerReconciler.reconcileCh:
				Expect(evt.Object).NotTo(BeNil())
				hv, ok := evt.Object.(*kvmv1.Hypervisor)
				Expect(ok).To(BeTrue())
				Expect(hv.Name).To(Equal(testHostname))
				Expect(hv.Namespace).To(Equal(testNamespace))
				Expect(hv.Kind).To(Equal("Hypervisor"))
				Expect(hv.APIVersion).To(Equal("kvm.cloud.sap/v1"))
			case <-time.After(2 * time.Second):
				Fail("timeout waiting for reconcile event")
			}
		})
	})

	Context("When testing SetupWithManager method", func() {
		It("should successfully setup controller with manager", func() {
			// Create a test manager
			mgr, err := ctrl.NewManager(cfg, ctrl.Options{
				Scheme: k8sClient.Scheme(),
			})
			Expect(err).NotTo(HaveOccurred())

			controllerReconciler := &HypervisorReconciler{
				Client: k8sClient,
				Scheme: k8sClient.Scheme(),
				Systemd: &systemd.InterfaceMock{
					DescribeFunc: func(ctx context.Context) (*systemd.Descriptor, error) {
						return &systemd.Descriptor{
							OperatingSystemReleaseData: []string{
								"PRETTY_NAME=\"Garden Linux 1877.8\"",
								"GARDENLINUX_VERSION=1877.8",
							},
							KernelVersion:   "6.1.0",
							KernelRelease:   "6.1.0-gardenlinux",
							KernelName:      "Linux",
							HardwareVendor:  "Test Vendor",
							HardwareModel:   "Test Model",
							HardwareSerial:  "TEST123",
							FirmwareVersion: "1.0",
							FirmwareVendor:  "Test BIOS",
							FirmwareDate:    time.Now().UnixMicro(),
						}, nil
					},
				},
			}

			err = controllerReconciler.SetupWithManager(mgr)
			Expect(err).NotTo(HaveOccurred())
			Expect(controllerReconciler.reconcileCh).NotTo(BeNil())
			Expect(controllerReconciler.osDescriptor).NotTo(BeNil())
			Expect(controllerReconciler.osDescriptor.OperatingSystemReleaseData).To(HaveLen(2))
		})

		It("should fail when systemd Describe returns error", func() {
			// Create a test manager
			mgr, err := ctrl.NewManager(cfg, ctrl.Options{
				Scheme: k8sClient.Scheme(),
			})
			Expect(err).NotTo(HaveOccurred())

			controllerReconciler := &HypervisorReconciler{
				Client: k8sClient,
				Scheme: k8sClient.Scheme(),
				Systemd: &systemd.InterfaceMock{
					DescribeFunc: func(ctx context.Context) (*systemd.Descriptor, error) {
						return nil, errors.New("systemd describe failed")
					},
				},
			}

			err = controllerReconciler.SetupWithManager(mgr)
			Expect(err).To(HaveOccurred())
			Expect(err.Error()).To(ContainSubstring("unable to get Systemd hostname describe()"))
			Expect(err.Error()).To(ContainSubstring("systemd describe failed"))
		})
	})

	Context("When reconciling a resource", func() {
		const resourceName = "test-resource"

		ctx := context.Background()

		typeNamespacedName := types.NamespacedName{
			Name:      resourceName,
			Namespace: "default", // TODO(user):Modify as needed
		}
		hypervisor := &kvmv1.Hypervisor{}

		BeforeEach(func() {
			By("creating the custom resource for the Kind Hypervisor")
			err := k8sClient.Get(ctx, typeNamespacedName, hypervisor)
			if err != nil && apierrors.IsNotFound(err) {
				resource := &kvmv1.Hypervisor{
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
			resource := &kvmv1.Hypervisor{}
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
					ProcessFunc: func(hv kvmv1.Hypervisor) (kvmv1.Hypervisor, error) {
						hv.Status.Instances = []kvmv1.Instance{
							{
								ID:     "25e2ea06-f6be-4bac-856d-8c2d0bdbcdee",
								Name:   "test-instance",
								Active: false,
							},
						}
						hv.Status.LibVirtVersion = "10.9.0"
						hv.Status.NumInstances = 1
						hv.Status.Capabilities = kvmv1.Capabilities{
							HostCpuArch: "x86_64",
							HostCpus:    *resource.NewQuantity(4, resource.DecimalSI),
							HostMemory:  *resource.NewQuantity(8192, resource.DecimalSI),
						}
						hv.Status.DomainCapabilities = kvmv1.DomainCapabilities{
							Arch:              "x86_64",
							HypervisorType:    "kvm",
							SupportedCpuModes: []string{"mode/example", "mode/example/1"},
						}
						return hv, nil
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
				osDescriptor: &systemd.Descriptor{
					OperatingSystemReleaseData: []string{
						"PRETTY_NAME=\"Garden Linux 1877.8\"",
						"GARDENLINUX_VERSION=1877.8",
						"GARDENLINUX_COMMIT_ID_LONG=abcdef1234567890",
						"GARDENLINUX_FEATURES=_rescue,log,sap",
						"VARIANT_ID=metal-sci_usi-amd64",
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

			Expect(hypervisor.Status.Conditions).To(HaveLen(2))

			Expect(hypervisor.Status.Conditions[0].Type).To(Equal("test-unit"))
			Expect(hypervisor.Status.Conditions[0].Status).To(Equal(metav1.ConditionTrue))
			Expect(hypervisor.Status.Conditions[0].Reason).To(Equal("Running"))

			Expect(hypervisor.Status.Conditions[1].Type).To(Equal("LibVirtConnection"))
			Expect(hypervisor.Status.Conditions[1].Status).To(Equal(metav1.ConditionTrue))
			Expect(hypervisor.Status.Conditions[1].Reason).To(Equal("Connected"))

			Expect(hypervisor.Status.Capabilities.HostCpuArch).To(Equal("x86_64"))
			Expect(hypervisor.Status.Capabilities.HostCpus.AsDec().UnscaledBig()).
				To(Equal(resource.NewQuantity(4, resource.DecimalSI).AsDec().UnscaledBig()))
			Expect(hypervisor.Status.Capabilities.HostMemory.AsDec().UnscaledBig()).
				To(Equal(resource.NewQuantity(8192, resource.DecimalSI).AsDec().UnscaledBig()))
			Expect(hypervisor.Status.OperatingSystem.PrettyVersion).To(Equal("\"Garden Linux 1877.8\""))
			Expect(hypervisor.Status.OperatingSystem.Version).To(Equal("1877.8"))
			Expect(hypervisor.Status.OperatingSystem.GardenLinuxCommitID).To(Equal("abcdef1234567890"))
			Expect(hypervisor.Status.OperatingSystem.GardenLinuxFeatures).To(Equal([]string{"_rescue", "log", "sap"}))
			Expect(hypervisor.Status.OperatingSystem.VariantID).To(Equal("metal-sci_usi-amd64"))
		})
	})
})
