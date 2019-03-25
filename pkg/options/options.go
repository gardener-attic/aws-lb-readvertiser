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

package options

import (
	"flag"
	"fmt"
	"os"
	"time"

	"sigs.k8s.io/controller-runtime/pkg/client/config"

	"k8s.io/client-go/rest"
	"k8s.io/client-go/tools/clientcmd"
)

// ReadvertiserOptions are the options for the Readvertiser
type ReadvertiserOptions struct {
	ShootKubeConfig        *rest.Config
	SeedKubeConfig         *rest.Config
	HostnameRefreshPeriod  time.Duration
	ControllerResyncPeriod time.Duration
	MetricsAddr            string
	ServiceName            string
	ServiceNamespace       string
	EndpointName           string
	EndpointNamespace      string
}

const (
	userAgent = "lb-readvertiser"
)

// Parse flags.
func (r *ReadvertiserOptions) Parse() error {

	var shootKubeConfig string
	flag.StringVar(&r.MetricsAddr, "metrics-addr", ":8080", "The address the metric endpoint binds to.")
	flag.StringVar(&shootKubeConfig, "shoot-kubeconfig", "", "Paths to a shoot's kubeconfig. Required.")
	flag.StringVar(&r.ServiceName, "service-name", "kube-apiserver", "Name of the service of type Loabalancer")
	flag.StringVar(&r.ServiceNamespace, "service-namespace", "", "Namespace of the service of type Loabalancer")
	flag.StringVar(&r.EndpointName, "endpoint-name", "kubernetes", "TEST ONLY - name of the endpoint to reconcile")
	flag.StringVar(&r.EndpointNamespace, "endpoint-namespace", "default", "TEST ONLY - namespace of the endpoint to reconcile")
	flag.DurationVar(&r.HostnameRefreshPeriod, "hostname-refresh-period", time.Second*30, "The period at which the Loadbalancer's hostnames are resynced")
	flag.DurationVar(&r.ControllerResyncPeriod, "resync-period", time.Minute*30, "The period at which the controller sync with the cache will happen")

	flag.Parse()

	if len(shootKubeConfig) > 0 {
		conf, err := clientcmd.BuildConfigFromFlags("", shootKubeConfig)
		if err != nil {
			return fmt.Errorf("unable to set up seed client config: %v", err)
		}
		r.ShootKubeConfig = conf
	} else {
		return fmt.Errorf("shoot-kubeconfig is required")
	}
	r.ShootKubeConfig.UserAgent = userAgent

	if len(r.ServiceName) == 0 {
		return fmt.Errorf("service-name is required")
	}

	// This normally comes from K8S's DownwardAPI
	if len(os.Getenv("SERVICE_NAMESPACE")) > 0 {
		r.ServiceNamespace = os.Getenv("SERVICE_NAMESPACE")
	}

	if len(r.ServiceNamespace) == 0 {
		return fmt.Errorf("service-namespace is required")
	}

	if len(r.EndpointName) == 0 {
		return fmt.Errorf("endpoint-name is required")
	}

	if len(r.EndpointNamespace) == 0 {
		return fmt.Errorf("endpoint-namespace is required")
	}

	if r.HostnameRefreshPeriod <= 0 {
		return fmt.Errorf(`hostname-refresh-period should be greater or equal to "0"`)
	}

	if r.ControllerResyncPeriod < 0 {
		return fmt.Errorf(`resync-period should be greater than "0"`)
	}

	cfg, err := config.GetConfig()
	if err != nil {
		return fmt.Errorf("unable to set up seed client config: %v", err)
	}
	r.SeedKubeConfig = cfg
	r.SeedKubeConfig.UserAgent = userAgent

	return nil
}
