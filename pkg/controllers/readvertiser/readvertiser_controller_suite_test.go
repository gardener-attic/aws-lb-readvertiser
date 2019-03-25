// Copyright (c) 2019 SAP SE or an SAP affiliate company. All rights reserved. This file is licensed under the Apache Software License, v. 2 except as noted otherwise in the LICENSE file
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

package readvertiser_test

import (
	"sync"
	"testing"

	corev1 "k8s.io/api/core/v1"
	metav1 "k8s.io/apimachinery/pkg/apis/meta/v1"
	"k8s.io/client-go/tools/cache"

	. "github.com/onsi/ginkgo"
	. "github.com/onsi/gomega"
)

func TestLbReadvertiser(t *testing.T) {
	RegisterFailHandler(Fail)
	RunSpecs(t, "LbReadvertiser Suite")
}

type fakeIndexer struct {
	cache.Indexer
	getError      error
	lock          sync.Mutex
	requestedKeys []string
	obj           interface{}
}

func (f *fakeIndexer) GetByKey(key string) (interface{}, bool, error) {
	f.lock.Lock()
	defer f.lock.Unlock()
	f.requestedKeys = append(f.requestedKeys, key)

	return f.obj, f.obj != nil, f.getError
}

func newTestService(ing ...corev1.LoadBalancerIngress) *corev1.Service {
	return &corev1.Service{
		ObjectMeta: metav1.ObjectMeta{Name: serviceName, Namespace: serviceNamespace},
		Status:     corev1.ServiceStatus{LoadBalancer: corev1.LoadBalancerStatus{Ingress: ing}}}
}

func newTestEndpoint(ips ...string) *corev1.Endpoints {
	addresses := []corev1.EndpointAddress{}
	for _, ip := range ips {
		addresses = append(addresses, corev1.EndpointAddress{IP: ip})
	}
	return &corev1.Endpoints{
		ObjectMeta: metav1.ObjectMeta{Name: endpointName, Namespace: endpointNamespace},
		Subsets:    []corev1.EndpointSubset{{Addresses: addresses, Ports: []corev1.EndpointPort{}}},
	}
}
