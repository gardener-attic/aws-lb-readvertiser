// Copyright (c) 2020 SAP SE or an SAP affiliate company. All rights reserved. This file is licensed under the Apache Software License, v. 2 except as noted otherwise in the LICENSE file
//
// Licensed under the Apache License, Version 2.0 (the "License");
// you may not use this file except in compliance with the License.
// You may obtain a copy of the License at
//
//      http://www.apache.org/licenses/LICENSE-2.0
//
// Unless required by applicable law or agreed to in writing, software
// distributed under the License is distributed on an "AS IS" BASIS,
// WITHOUT WARRANTIES OR CONDITIONS OF ANY KIND, either express or implied.
// See the License for the specific language governing permissions and
// limitations under the License.

package controller

import (
	"context"
	"time"

	. "github.com/onsi/ginkgo"
	. "github.com/onsi/gomega"
	corev1 "k8s.io/api/core/v1"
	metav1 "k8s.io/apimachinery/pkg/apis/meta/v1"
	k8sinformers "k8s.io/client-go/informers"
	"k8s.io/client-go/kubernetes/fake"
)

var _ = Describe("#applyTwoWayEndpointMergePatch", func() {
	var (
		fakeClient               = fake.NewSimpleClientset()
		sharedK8sInformerFactory = k8sinformers.NewSharedInformerFactory(fakeClient, time.Duration(time.Hour))
		endpointsInformer        = sharedK8sInformerFactory.Core().V1().Endpoints()

		oldIP  = "1.2.3.4"
		newIP  = "4.3.2.1"
		epName = "fakeName"
	)

	It("should apply the new ips and delete the old ips from the endpoint object", func() {
		epMeta := metav1.ObjectMeta{}
		epMeta.Name = epName
		epMeta.Namespace = metav1.NamespaceDefault
		oldEndpoints := &corev1.Endpoints{
			ObjectMeta: epMeta,
			Subsets: []corev1.EndpointSubset{
				{
					Addresses: []corev1.EndpointAddress{
						{
							IP: oldIP,
						},
					},
					Ports: []corev1.EndpointPort{
						{
							Name:     "https",
							Port:     443,
							Protocol: "TCP",
						},
					},
				},
			},
		}

		_, err := fakeClient.CoreV1().Endpoints(metav1.NamespaceDefault).Create(context.TODO(), oldEndpoints, metav1.CreateOptions{})
		Expect(err).To(BeNil())

		controller := NewAWSLBEndpointsController(fakeClient, endpointsInformer, "elbHostname", "endpointName")
		_, err = controller.applyTwoWayEndpointMergePatch(context.TODO(), oldEndpoints, []string{newIP})
		Expect(err).To(BeNil())

		expected := oldEndpoints.DeepCopy()
		expected.Subsets[0].Addresses[0].IP = newIP
		actual, err := fakeClient.CoreV1().Endpoints(metav1.NamespaceDefault).Get(context.TODO(), epName, metav1.GetOptions{})
		Expect(err).To(BeNil())
		Expect(*actual).To(Equal(*expected))
	})
})
