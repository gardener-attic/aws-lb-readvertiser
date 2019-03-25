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

package main

import (
	"flag"
	"os"

	"github.com/gardener/aws-lb-readvertiser/pkg/controllers"
	"github.com/gardener/aws-lb-readvertiser/pkg/options"
	"sigs.k8s.io/controller-runtime/pkg/manager"
	logf "sigs.k8s.io/controller-runtime/pkg/runtime/log"
	"sigs.k8s.io/controller-runtime/pkg/runtime/signals"
)

func main() {
	logf.SetLogger(logf.ZapLogger(false))
	log := logf.Log.WithName("entrypoint")

	ops := &options.ReadvertiserOptions{}
	if err := ops.Parse(); err != nil {
		log.Error(err, "unable to set up options")
		os.Exit(1)
	}

	flag.VisitAll(func(f *flag.Flag) {
		log.Info("flags", f.Name, f.Value.String())
	})

	// Create a new Cmd to provide shared dependencies and start components
	log.Info("setting up manager")
	mgr, err := manager.New(ops.ShootKubeConfig, manager.Options{
		MetricsBindAddress:      ops.MetricsAddr,
		LeaderElection:          true,
		LeaderElectionNamespace: "default",
		LeaderElectionID:        "readvertiser",
	})
	if err != nil {
		log.Error(err, "unable to set up overall controller manager")
		os.Exit(1)
	}

	if err := controllers.AddControllers(mgr, ops); err != nil {
		os.Exit(1)
	}

	log.Info("Starting manager.")
	if err := mgr.Start(signals.SetupSignalHandler()); err != nil {
		log.Error(err, "unable to run the manager")
		os.Exit(1)
	}
}
