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

package readvertiser

import (
	corev1 "k8s.io/api/core/v1"
	"k8s.io/apimachinery/pkg/util/sets"
)

func toEndPointSubsets(s *corev1.Service, ips sets.String) []corev1.EndpointSubset {
	endpointPorts := make([]corev1.EndpointPort, 0, len(s.Spec.Ports))
	endpointAddresses := make([]corev1.EndpointAddress, 0, ips.Len())
	for _, p := range s.Spec.Ports {
		endpointPorts = append(endpointPorts, corev1.EndpointPort{Name: p.Name, Port: p.Port, Protocol: p.Protocol})
	}
	for _, ip := range ips.List() {
		endpointAddresses = append(endpointAddresses, corev1.EndpointAddress{IP: ip})
	}
	return []corev1.EndpointSubset{{Addresses: endpointAddresses, Ports: endpointPorts}}
}

func toIPs(e *corev1.Endpoints) sets.String {
	ip := sets.String{}
	for _, subset := range e.Subsets {
		for _, addr := range subset.Addresses {
			ip.Insert(addr.IP)
		}
	}
	return ip
}
