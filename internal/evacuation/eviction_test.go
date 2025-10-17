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

	kvmv1 "github.com/cobaltcore-dev/openstack-hypervisor-operator/api/v1"
	. "github.com/onsi/ginkgo/v2"
	. "github.com/onsi/gomega"
	"k8s.io/apimachinery/pkg/api/errors"
	"k8s.io/apimachinery/pkg/api/meta"
	metav1 "k8s.io/apimachinery/pkg/apis/meta/v1"
	"k8s.io/apimachinery/pkg/apis/meta/v1/unstructured"
	"k8s.io/apimachinery/pkg/runtime/schema"
	"k8s.io/apimachinery/pkg/types"

	"github.com/cobaltcore-dev/kvm-node-agent/internal/sys"
)

var _ = Describe("Evacuation Callback", func() {
	Context("When trying evacuation on an empty host", func() {

		It("should not create an eviction", func() {
			// Test	logic
			const resourceName = "test-resource"
			const resourceNamespace = "default"

			ctx := context.Background()

			typeNamespacedName := types.NamespacedName{
				Name:      resourceName,
				Namespace: resourceNamespace, // TODO(user):Modify as needed
			}
			hypervisor := &kvmv1.Hypervisor{}

			By("creating the custom resource for the Kind Hypervisor")
			err := k8sClient.Get(ctx, typeNamespacedName, hypervisor)
			if err != nil && errors.IsNotFound(err) {
				resource := &kvmv1.Hypervisor{
					TypeMeta: metav1.TypeMeta{},
					ObjectMeta: metav1.ObjectMeta{
						Name:      resourceName,
						Namespace: resourceNamespace,
					},
					Status: kvmv1.HypervisorStatus{
						NumInstances: 1,
					},
				}
				Expect(k8sClient.Create(ctx, resource)).To(Succeed())
			}

			// override the hostname/namespace
			sys.Hostname = resourceName
			sys.Namespace = resourceNamespace

			controller := EvictionController{k8sClient}
			err = controller.EvictCurrentHost(context.Background())
			Expect(err).NotTo(HaveOccurred())

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

			err = k8sClient.Get(ctx, typeNamespacedName, u)
			Expect(err).To(HaveOccurred())
			// Expect the eviction resource to not be created
			Expect(meta.IsNoMatchError(err)).To(BeTrue())
		})
	})
})
