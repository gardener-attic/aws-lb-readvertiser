// Copyright 2017 The Gardener Authors.
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
	"fmt"
	"net"
	"os"
	"strings"
	"time"

	metav1 "k8s.io/apimachinery/pkg/apis/meta/v1"
	"k8s.io/client-go/kubernetes"
	"k8s.io/client-go/rest"
)

func main() {
	name := flag.String("name", "kube-apiserver", "name of deployment")
	elb := flag.String("elb", "", "dns name of elb")
	flag.Parse()

	// checks
	if len(*elb) == 0 {
		panic("--elb is not set")
	}
	namespace, exists := os.LookupEnv("NAMESPACE")
	if !exists {
		panic("NAMESPACE env variable not set")
	}

	// creates the in-cluster config
	config, err := rest.InClusterConfig()
	if err != nil {
		panic(err.Error())
	}
	// creates the clientset
	clientset, err := kubernetes.NewForConfig(config)
	if err != nil {
		panic(err.Error())
	}

	for {
		var containerIndex int
		var cmdIndex int
		var needUpdate = true
		// lookup elb dns name
		nsRecords, err := net.LookupHost(*elb)
		if err != nil {
			fmt.Printf("%s warning: could not resolve the dns name of the elb: %v\n", time.Now(), err)
		} else {
			// get deployment
			deployment, err := clientset.ExtensionsV1beta1().Deployments(namespace).Get(*name, metav1.GetOptions{})
			if err != nil {
				fmt.Printf("%s error: could not get deployment manifest of apiserver, but an error occurred: %v\n", time.Now(), err)
			}
			for i, container := range deployment.Spec.Template.Spec.Containers {
				if container.Name == "kube-apiserver" {
					containerIndex = i
					for j, cmd := range container.Command {
						split := strings.Split(cmd, "=")
						if split[0] == "--advertise-address" {
							cmdIndex = j
							ip := split[1]
							for _, record := range nsRecords {
								if ip == record {
									needUpdate = false
								}
							}
						}
					}
				}
			}

			// update deployment if needed
			if needUpdate {
				newIP := nsRecords[0]
				fmt.Printf("%s need to update the advertise-address, use %s", time.Now(), newIP)
				deployment.
					Spec.
					Template.
					Spec.
					Containers[containerIndex].
					Command[cmdIndex] = fmt.Sprintf("--advertise-address=%s", newIP)
				fmt.Printf("%s Sending deployment to apiserver: %v", time.Now(), deployment)
				_, err := clientset.ExtensionsV1beta1().Deployments(namespace).Update(deployment)
				if err != nil {
					fmt.Printf("%s error: wanted to update the deployment, but an error occurred: %v\n", time.Now(), err)
				} else {
					fmt.Println(time.Now(), "\nsent manifest to apiserver successfully")
				}
			}
		}
		time.Sleep(5 * time.Second)
	}
}
