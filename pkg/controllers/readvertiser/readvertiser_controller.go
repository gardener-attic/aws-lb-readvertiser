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
	"fmt"
	"time"

	resolver "github.com/gardener/aws-lb-readvertiser/pkg/net"
	"github.com/go-logr/logr"
	corev1 "k8s.io/api/core/v1"
	"k8s.io/apimachinery/pkg/api/errors"
	metav1 "k8s.io/apimachinery/pkg/apis/meta/v1"
	"k8s.io/apimachinery/pkg/types"
	"k8s.io/apimachinery/pkg/util/sets"
	endpointsclient "k8s.io/client-go/kubernetes/typed/core/v1"
	listerv1 "k8s.io/client-go/listers/core/v1"
	"k8s.io/client-go/tools/record"
	"sigs.k8s.io/controller-runtime/pkg/reconcile"
	logf "sigs.k8s.io/controller-runtime/pkg/runtime/log"
)

const (
	// SuccessSynced is used as part of the Event 'reason' when a Foo is synced
	SuccessSynced = "Synced"
	// MessageResourceSynced is the message used for an Event fired when a Foo
	// is synced successfully
	MessageResourceSynced = "Endpoints %q synced successfully"
)

//Controller a controller for propagating newly monitored endpoints
type Controller struct {
	Endpoint        types.NamespacedName
	EndpointLister  listerv1.EndpointsNamespaceLister
	Log             logr.Logger
	Recorder        record.EventRecorder
	Resolver        resolver.Resolver
	ResyncHostnames time.Duration
	Service         string
	ServiceLister   listerv1.ServiceNamespaceLister
	EndpointsClient endpointsclient.EndpointsInterface
}

// NewReconciler creates a new isntance of Controller.
func NewReconciler(
	client endpointsclient.EndpointsInterface,
	recorder record.EventRecorder,
	sl listerv1.ServiceNamespaceLister,
	epl listerv1.EndpointsNamespaceLister,
	svc string,
	endpoint types.NamespacedName,
	rh time.Duration,
	rsvr resolver.Resolver) reconcile.Reconciler {
	return &Controller{
		Endpoint:        endpoint,
		EndpointLister:  epl,
		Log:             logf.Log.WithName("controller.readvertser"),
		Recorder:        recorder,
		Resolver:        rsvr,
		ResyncHostnames: rh,
		Service:         svc,
		ServiceLister:   sl,
		EndpointsClient: client,
	}
}

// Reconcile compares IP addresses in a LoadBalancer service and updates
// endpoint in the Shoot cluster.
func (c *Controller) Reconcile(request reconcile.Request) (reconcile.Result, error) {
	log := c.Log.WithValues("request", request)
	hasHostnames := false

	log.Info("reconciling")
	defer log.Info("reconile done")
	// controller-runtime's cache client is only limited to 1 cluster.
	service, err := c.ServiceLister.Get(c.Service)
	if err != nil {
		if errors.IsNotFound(err) {
			log.Info("does not exists, skipping", "service", c.Service)
			return c.result(hasHostnames, nil)
		}
		return c.result(hasHostnames, err)
	}

	endpoint, err := c.EndpointLister.Get(c.Endpoint.Name)
	if err != nil {
		if errors.IsNotFound(err) {
			ips, _, err := c.resolveIPs(service)
			if err != nil {
				return c.result(false, err)
			}
			if ips.Len() == 0 {
				log.Info("no IPs available. skipping")
				return c.result(false, err)
			}
			subsets := toEndPointSubsets(service, ips)
			newEndpoint := &corev1.Endpoints{
				ObjectMeta: metav1.ObjectMeta{
					Name:      c.Endpoint.Name,
					Namespace: c.Endpoint.Namespace,
				},
				Subsets: subsets,
			}
			log.Info("creating endpoint", "endpoint", *newEndpoint)
			_, err = c.EndpointsClient.Create(newEndpoint)
			if err == nil {
				msg := fmt.Sprintf(MessageResourceSynced, c.Endpoint)
				c.Recorder.Event(service, corev1.EventTypeNormal, SuccessSynced, msg)
			}
			return c.result(false, err)
		}
		return c.result(false, err)
	}

	ips, hasHostnames, err := c.resolveIPs(service)
	if err != nil {
		return c.result(false, err)
	}
	if ips.Len() == 0 {
		log.Info("no IPs available. skipping")
		return c.result(false, err)
	}

	if !ips.Equal(toIPs(endpoint)) {
		endpointCopy := endpoint.DeepCopy()
		endpointCopy.Subsets = toEndPointSubsets(service, ips)
		log.Info("updating endpoint", "endpoint", endpointCopy)
		_, err = c.EndpointsClient.Update(endpointCopy)
		if err == nil {
			msg := fmt.Sprintf(MessageResourceSynced, c.Endpoint)
			c.Recorder.Event(service, corev1.EventTypeNormal, SuccessSynced, msg)
		}
		// don't requeue for sync later - this is going to be handled bellow
		return c.result(false, err)
	}
	return c.result(hasHostnames, err)

}

func (c *Controller) result(hasHostames bool, err error) (reconcile.Result, error) {
	r := reconcile.Result{}
	if hasHostames && err == nil {
		r.RequeueAfter = c.ResyncHostnames
	}
	return r, err
}

func (c *Controller) resolveIPs(s *corev1.Service) (sets.String, bool, error) {
	hasHostnames := false
	ips := sets.NewString()
	for _, i := range s.Status.LoadBalancer.Ingress {
		if len(i.Hostname) > 0 {
			hasHostnames = true
			records, err := c.Resolver.LookupHost(i.Hostname)
			if err != nil {
				return ips, hasHostnames, err
			}
			ips.Insert(records...)
		}
		if len(i.IP) > 0 {
			ips.Insert(i.IP)
		}
	}
	return ips, hasHostnames, nil
}

func toEndPointSubsets(s *corev1.Service, ips sets.String) []corev1.EndpointSubset {
	ep := make([]corev1.EndpointPort, 0, len(s.Spec.Ports))
	epAddrs := make([]corev1.EndpointAddress, 0, ips.Len())
	for _, p := range s.Spec.Ports {
		ep = append(ep, corev1.EndpointPort{Name: p.Name, Port: p.Port, Protocol: p.Protocol})
	}
	for _, ip := range ips.List() {
		epAddrs = append(epAddrs, corev1.EndpointAddress{IP: ip})
	}
	return []corev1.EndpointSubset{corev1.EndpointSubset{Addresses: epAddrs, Ports: ep}}
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
