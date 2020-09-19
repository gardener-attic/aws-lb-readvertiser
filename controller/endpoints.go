// SPDX-FileCopyrightText: 2020 SAP SE or an SAP affiliate company and Gardener contributors
//
// SPDX-License-Identifier: Apache-2.0

package controller

import (
	"context"
	"encoding/json"
	"fmt"
	"net"
	"time"

	"k8s.io/apimachinery/pkg/api/errors"
	"k8s.io/apimachinery/pkg/types"
	"k8s.io/apimachinery/pkg/util/strategicpatch"

	log "github.com/sirupsen/logrus"
	corev1 "k8s.io/api/core/v1"
	metav1 "k8s.io/apimachinery/pkg/apis/meta/v1"
	"k8s.io/apimachinery/pkg/util/runtime"
	informercorev1 "k8s.io/client-go/informers/core/v1"
	"k8s.io/client-go/kubernetes"
	gettercorev1 "k8s.io/client-go/kubernetes/typed/core/v1"
	listercorev1 "k8s.io/client-go/listers/core/v1"
	"k8s.io/client-go/tools/cache"
)

const endpointName = "kubernetes"

//AWSLBReadvertiserController a controller for propagating newly monitored endpoints
type AWSLBReadvertiserController struct {
	client              kubernetes.Interface
	endpointsGetter     gettercorev1.EndpointsGetter
	endpointsLister     listercorev1.EndpointsLister
	endpointsListerSync cache.InformerSynced

	elbHostName, endpointName string
}

// NewAWSLBEndpointsController initialize endpoints Informer
func NewAWSLBEndpointsController(client kubernetes.Interface, endpointsInformer informercorev1.EndpointsInformer, elbHostName, endpointName string) *AWSLBReadvertiserController {
	awsLBReadvertiserController := &AWSLBReadvertiserController{
		client:              client,
		endpointsGetter:     client.CoreV1(),
		endpointsLister:     endpointsInformer.Lister(),
		endpointsListerSync: endpointsInformer.Informer().HasSynced,

		elbHostName:  elbHostName,
		endpointName: endpointName,
	}

	return awsLBReadvertiserController
}

func (c *AWSLBReadvertiserController) createTwoWayEndpointMergePatch(endpoint *corev1.Endpoints, dnsRecords []string) (*corev1.EndpointSubset, error) {
	endpointCopy := endpoint.DeepCopy()

	endpoints, err := createEndpointSubsetObjectFromRecords(dnsRecords)
	if err != nil {
		return nil, fmt.Errorf("Failed to update endpoint")
	}

	// Set Subset to new endpoint IPs
	endpointCopy.Subsets = []corev1.EndpointSubset{*endpoints}

	// start the update process with Kubernetes
	oldEndpoint, err := json.Marshal(endpoint)
	if err != nil {
		return nil, fmt.Errorf("Failed to marshal old endpoint")
	}

	newEndPoint, err := json.Marshal(endpointCopy)
	if err != nil {
		return nil, fmt.Errorf("failed to marshal new endpoint")
	}

	patchBytes, err := strategicpatch.CreateTwoWayMergePatch(oldEndpoint, newEndPoint, corev1.Endpoints{})
	if err != nil {
		return nil, fmt.Errorf("failed to patch bytes")
	}

	_, err = c.client.CoreV1().Endpoints(metav1.NamespaceDefault).Patch(endpoint.Name, types.StrategicMergePatchType, patchBytes)
	if err != nil {
		return nil, fmt.Errorf("failed to update endpoint with new value: %s", err.Error())
	}
	return endpoints, nil
}

//Run the AWSLBReconciler
func (c *AWSLBReadvertiserController) Run(ctx context.Context, refreshTicker *time.Ticker) {
	defer func() {
		runtime.HandleCrash()
	}()

	log.Info("waiting for cache sync")
	if !cache.WaitForCacheSync(ctx.Done(), c.endpointsListerSync) {
		log.Print("timed out waiting for cache sync")
		return
	}
	log.Info("Caches are synced")

	log.Info("Watching AWS ELB records for changes...!!")
	for {
		select {

		case <-refreshTicker.C:
			var (
				endpointIPs []string
				dnsRecords  []string
			)

			// lookup Elastic Loadbalancer DNS name
			dnsRecords, err := net.LookupHost(c.elbHostName)
			if err != nil {
				log.Errorf("%s warning: could not resolve the DNS name of the elb: %v\n", time.Now(), err)
				break
			}
			log.Printf("DNS lookup results are: %s", dnsRecords)

			endpoint, err := c.endpointsLister.Endpoints(metav1.NamespaceDefault).Get(endpointName)
			createEndpoint := func() error {
				endpointSubset, err := createEndpointSubsetObjectFromRecords(dnsRecords)
				if err != nil {
					return fmt.Errorf("%s warning: could not resolve the DNS name of the elb: %v", time.Now(), err)
				}

				endpoint, err = c.client.CoreV1().Endpoints(metav1.NamespaceDefault).Create(&corev1.Endpoints{
					ObjectMeta: metav1.ObjectMeta{
						Name: endpointName,
					},
					Subsets: []corev1.EndpointSubset{*endpointSubset},
				})
				if err != nil {
					return fmt.Errorf("%s warning: could not create the kubernetes endpoint : %v", time.Now(), err)
				}

				return nil
			}

			if err != nil {
				// Check if the endpoint is there and create it if its not
				if errors.IsNotFound(err) {
					log.Infof("The default %q endpoint was not found, creating it now", "kubernetes")
					if err := createEndpoint(); err != nil {
						log.Error(err)
						break
					}
				} else {
					log.Errorf("%s error: could not get endpoint, an error occurred: %v\n", time.Now(), err)
					break
				}
			}

			// handle the case where endpoint exists but has no subsets
			if len(endpoint.Subsets) == 0 {
				log.Infof("Found empty %s endpoint, adding correct LB IPs", endpointName)
				endpoints, err := c.createTwoWayEndpointMergePatch(endpoint, dnsRecords)
				if err != nil {
					log.Error(err)
					break
				}
				newEndpointAddresses, err := fetchEndpointIPsFromAddresses(endpoints.Addresses)
				if err != nil {
					log.Error("Endpoint subset has empty IPs")
				}
				log.Infof("New endpoint IPs are %q, ELB IPs are %s", newEndpointAddresses, dnsRecords)
			} else {
				// Endpoint exists but with the possiblity of outdated IPs
				endpointIPs, err = fetchEndpointIPsFromAddresses(endpoint.Subsets[0].Addresses)
				if err != nil {
					log.Error(err)
					break
				}
				log.Infof("Kubernetes Endpoint IPs : %q", endpointIPs)

				// Check validity of endpoint and change respectively
				if checkEndpointIsStillValid(endpointIPs, dnsRecords) {
					log.Info("Nothing to be done")
					break
				}

				log.Info("ELB records changed, reconciling cluster endpoint to match")

				endpoints, err := c.createTwoWayEndpointMergePatch(endpoint, dnsRecords)
				if err != nil {
					log.Error(err)
					break
				}

				newEndpointAddresses, err := fetchEndpointIPsFromAddresses(endpoints.Addresses)
				if err != nil {
					log.Error("Endpoint subset has empty IPs")
				}
				log.Infof("Old endpoint IPs are %q, new endpoint IPs are %q, ELB IPs are %s", endpointIPs, newEndpointAddresses, dnsRecords)
			}

		case <-ctx.Done():
			refreshTicker.Stop()
			return
		}
	}
}
