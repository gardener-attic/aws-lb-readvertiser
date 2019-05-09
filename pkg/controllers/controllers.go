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

package controllers

import (
	"github.com/gardener/aws-lb-readvertiser/pkg/controllers/readvertiser"
	resolver "github.com/gardener/aws-lb-readvertiser/pkg/net"
	"github.com/gardener/aws-lb-readvertiser/pkg/options"
	"github.com/gardener/aws-lb-readvertiser/pkg/recorder"
	metav1 "k8s.io/apimachinery/pkg/apis/meta/v1"
	"k8s.io/apimachinery/pkg/fields"
	"k8s.io/apimachinery/pkg/types"
	"k8s.io/client-go/informers"
	"k8s.io/client-go/kubernetes"
	"sigs.k8s.io/controller-runtime/pkg/controller"
	"sigs.k8s.io/controller-runtime/pkg/handler"
	"sigs.k8s.io/controller-runtime/pkg/manager"
	"sigs.k8s.io/controller-runtime/pkg/reconcile"
	logf "sigs.k8s.io/controller-runtime/pkg/runtime/log"
	"sigs.k8s.io/controller-runtime/pkg/source"
)

// AddControllers adds all controllers to the controller manager
func AddControllers(mgr manager.Manager, ops *options.ReadvertiserOptions) error {
	log := logf.Log.WithName("entrypoint")
	shootClient := kubernetes.NewForConfigOrDie(ops.ShootKubeConfig)
	seedClient := kubernetes.NewForConfigOrDie(ops.SeedKubeConfig)

	seedEventClient := kubernetes.NewForConfigOrDie(ops.SeedKubeConfig)

	shootInfFactory := informers.NewSharedInformerFactoryWithOptions(
		shootClient,
		ops.ControllerResyncPeriod,
		informers.WithNamespace(ops.EndpointNamespace),
		informers.WithTweakListOptions(func(lo *metav1.ListOptions) {
			lo.FieldSelector = fields.OneTermEqualSelector("metadata.name", ops.EndpointName).String()
			lo.Limit = 1
		}))

	seedInfFactory := informers.NewSharedInformerFactoryWithOptions(
		seedClient,
		ops.ControllerResyncPeriod,
		informers.WithNamespace(ops.ServiceNamespace),
		informers.WithTweakListOptions(func(lo *metav1.ListOptions) {
			lo.FieldSelector = fields.OneTermEqualSelector("metadata.name", ops.ServiceName).String()
			lo.Limit = 1
		}))

	serviceInformer := seedInfFactory.Core().V1().Services()
	endpointsInformer := shootInfFactory.Core().V1().Endpoints()

	err := mgr.Add(manager.RunnableFunc(func(s <-chan struct{}) error {
		log.Info("Starting informer factories")
		shootInfFactory.Start(s)
		seedInfFactory.Start(s)
		<-s
		log.Info("Informer factories stopped")
		return nil
	}))

	if err != nil {
		log.Error(err, "unable to sync caches")
		return err
	}

	endpointNamespacedName := types.NamespacedName{Name: ops.EndpointName, Namespace: ops.EndpointNamespace}
	c, err := controller.New("readvertiser-controller", mgr, controller.Options{
		Reconciler: readvertiser.NewReconciler(
			shootClient.CoreV1().Endpoints(ops.EndpointNamespace),
			recorder.NewProvider(seedEventClient.CoreV1().Events(""), log).GetEventRecorderFor("readvertiser-controller"),
			serviceInformer.Lister().Services(ops.ServiceNamespace),
			endpointsInformer.Lister().Endpoints(ops.EndpointNamespace),
			ops.ServiceName,
			endpointNamespacedName,
			ops.HostnameRefreshPeriod,
			resolver.Default,
		),
		MaxConcurrentReconciles: 1,
	})
	if err != nil {
		log.Error(err, "unable to create controller")
		return err
	}

	if err := c.Watch(
		&source.Informer{Informer: serviceInformer.Informer()},
		&handler.EnqueueRequestsFromMapFunc{
			ToRequests: handler.ToRequestsFunc(
				func(a handler.MapObject) []reconcile.Request {
					return []reconcile.Request{{NamespacedName: endpointNamespacedName}}
				}),
		}); err != nil {
		log.Error(err, "unable to create watch for services")
		return err
	}

	if err := c.Watch(
		&source.Informer{Informer: endpointsInformer.Informer()},
		&handler.EnqueueRequestForObject{}); err != nil {
		log.Error(err, "unable to create watch for endpoints")
		return err
	}

	return nil
}
