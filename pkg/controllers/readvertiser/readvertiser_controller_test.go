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
	"fmt"
	"time"

	"k8s.io/client-go/tools/record"

	. "github.com/gardener/aws-lb-readvertiser/pkg/controllers/readvertiser"
	fake_resolver "github.com/gardener/aws-lb-readvertiser/pkg/net/fake"
	. "github.com/onsi/ginkgo"
	. "github.com/onsi/gomega"
	corev1 "k8s.io/api/core/v1"
	metav1 "k8s.io/apimachinery/pkg/apis/meta/v1"
	"k8s.io/apimachinery/pkg/runtime"
	"k8s.io/apimachinery/pkg/runtime/schema"
	"k8s.io/apimachinery/pkg/types"
	"k8s.io/client-go/kubernetes/fake"
	endpointsclient "k8s.io/client-go/kubernetes/typed/core/v1"
	corev1lister "k8s.io/client-go/listers/core/v1"
	clientgotesting "k8s.io/client-go/testing"
	"sigs.k8s.io/controller-runtime/pkg/reconcile"
	logf "sigs.k8s.io/controller-runtime/pkg/runtime/log"
)

const (
	serviceName       = "service-test"
	serviceNamespace  = "service-namespace"
	serviceKey        = "service-namespace/service-test"
	endpointName      = "endpoint-test"
	endpointNamespace = "endpoint-namespace"
	endpointKey       = "endpoint-namespace/endpoint-test"
)

var _ = Describe("Reconcile", func() {

	var (
		fakeClient              *fake.Clientset
		serviceIndexer          *fakeIndexer
		endpointIndexer         *fakeIndexer
		fakeResolver            *fake_resolver.Resolver
		fakeRecorder            *record.FakeRecorder
		result                  reconcile.Result
		err                     error
		expectedEndpoint        *corev1.Endpoints
		expectedResyncHostnames time.Duration
	)

	AssertNoErrorNoRequeue := func() func() {
		return func() {
			Expect(err).ToNot(HaveOccurred())
			Expect(result).To(Equal(reconcile.Result{Requeue: false, RequeueAfter: 0}))
		}
	}
	AssertErrorNoRequeue := func() func() {
		return func() {
			Expect(err).To(HaveOccurred())
			Expect(result).To(Equal(reconcile.Result{Requeue: false, RequeueAfter: 0}))
		}
	}
	AssertNoClientActions := func() func() {
		return func() {
			Expect(fakeClient.Actions()).To(BeEmpty())
		}
	}
	AssertNoHostnameResolve := func() func() {
		return func() {
			Expect(fakeResolver.Lookups).To(BeEmpty())
		}
	}
	AssertKeyLookup := func() func() {
		return func() {
			Expect(serviceIndexer.requestedKeys).To(ConsistOf(serviceKey))
			Expect(endpointIndexer.requestedKeys).To(ConsistOf(endpointKey))
		}
	}
	AssertHostnameResolve := func() func() {
		return func() {
			Expect(fakeResolver.Lookups).To(ConsistOf("foo.com"))
		}
	}

	AssertNoSyncEvent := func() func() {
		return func() {
			Eventually(fakeRecorder.Events).ShouldNot(Receive())
		}
	}

	AssertSyncEvent := func() func() {
		return func() {
			var item string
			Eventually(fakeRecorder.Events).Should(Receive(&item))
			Expect(item).Should(Equal(`Normal Synced Endpoints "endpoint-namespace/endpoint-test" synced successfully`))
		}
	}

	AssertOnlyOneAction := func(verb string) func() {
		return func() {
			Expect(fakeClient.Actions()).To(HaveLen(1))
			action := fakeClient.Actions()[0]
			Expect(action.GetVerb()).To(Equal(verb))
			Expect(action.GetSubresource()).To(BeEmpty())
			Expect(action.GetResource()).To(Equal(schema.GroupVersionResource{Group: "", Version: "v1", Resource: "endpoints"}))
			// update and create are the same interface
			update, ok := action.(clientgotesting.UpdateAction)
			Expect(ok).To(BeTrue())
			createdEnpoint, ok := update.GetObject().(*corev1.Endpoints)
			Expect(ok).To(BeTrue())
			Expect(createdEnpoint).To(Equal(expectedEndpoint))
		}
	}

	BeforeEach(func() {

		fakeClient = &fake.Clientset{}
		serviceIndexer = &fakeIndexer{}
		endpointIndexer = &fakeIndexer{}
		fakeResolver = &fake_resolver.Resolver{}
		fakeRecorder = record.NewFakeRecorder(10)
		err = nil
		expectedEndpoint = nil
		expectedResyncHostnames = 1
	})

	JustBeforeEach(func() {
		reconciler := Controller{
			EndpointsClient: fakeClient.CoreV1().Endpoints(endpointNamespace),
			Service:         serviceName,
			Endpoint:        types.NamespacedName{Name: endpointName, Namespace: endpointNamespace},
			ServiceLister:   corev1lister.NewServiceLister(serviceIndexer).Services(serviceNamespace),
			EndpointLister:  corev1lister.NewEndpointsLister(endpointIndexer).Endpoints(endpointNamespace),
			Log:             logf.NullLogger{},
			Recorder:        fakeRecorder,
			Resolver:        fakeResolver,
			ResyncHostnames: expectedResyncHostnames,
		}
		result, err = reconciler.Reconcile(reconcile.Request{})
	})

	Describe("error when getting service", func() {
		AssertCheckOnlyServiceLister := func() func() {
			return func() {
				Expect(serviceIndexer.requestedKeys).To(ConsistOf(serviceKey))
				Expect(endpointIndexer.requestedKeys).To(BeEmpty())
			}
		}
		Context("which doesn't exist", func() {

			It("should succeed and not requeue", AssertNoErrorNoRequeue())
			It("should not do any actions", AssertNoClientActions())
			It("should not resolve hostnames", AssertNoHostnameResolve())
			It("should only check service lister", AssertCheckOnlyServiceLister())
		})

		Context("which is random", func() {
			BeforeEach(func() {
				serviceIndexer.getError = fmt.Errorf("some error")
			})

			It("should fail and not requeue", AssertErrorNoRequeue())
			It("should not resolve hostnames", AssertNoHostnameResolve())
			It("should only check service lister", AssertCheckOnlyServiceLister())
			It("should not do any actions", AssertNoClientActions())
			It("should not create sync event", AssertNoSyncEvent())
		})
	})

	Describe("error when listing endpoint", func() {
		BeforeEach(func() {
			serviceIndexer.obj = newTestService(corev1.LoadBalancerIngress{Hostname: "foo.com"})
		})

		Context("not found error", func() {

			Context("hostname resolution fails", func() {
				BeforeEach(func() {
					fakeResolver.Error = fmt.Errorf("some error")
				})

				It("should fail and not requeue", AssertErrorNoRequeue())
				It("should resolve hostnames", AssertHostnameResolve())
				It("should check for correct keys in listers", AssertKeyLookup())
				It("should not do any actions", AssertNoClientActions())
				It("should not create sync event", AssertNoSyncEvent())
			})

			Context("without any ips", func() {
				BeforeEach(func() {
					fakeResolver.Addrs = []string{}
				})

				It("should succeed and not requeue", AssertNoErrorNoRequeue())
				It("should resolve hostnames", AssertHostnameResolve())
				It("should check for correct keys in listers", AssertKeyLookup())
				It("should not do any actions", AssertNoClientActions())
				It("should not create sync event", AssertNoSyncEvent())
			})

			Describe("creation of endpoint", func() {
				BeforeEach(func() {
					expectedEndpoint = newTestEndpoint("1.1.1.1")
					fakeResolver.Addrs = []string{"1.1.1.1"}
				})
				Context("failure", func() {
					BeforeEach(func() {
						fakeClient.AddReactor("create", "endpoints", func(action clientgotesting.Action) (bool, runtime.Object, error) {
							return true, nil, fmt.Errorf("some error")
						})
					})

					It("should fail and not requeue", AssertErrorNoRequeue())
					It("should check for correct keys in listers", AssertKeyLookup())
					It("should resolve hostnames", AssertHostnameResolve())
					It("should try to create Endpoint only once", AssertOnlyOneAction("create"))
					It("should not create sync event", AssertNoSyncEvent())
				})

				Context("success", func() {
					It("should succeed and not requeue", AssertNoErrorNoRequeue())
					It("should check for correct keys in listers", AssertKeyLookup())
					It("should resolve hostnames", AssertHostnameResolve())
					It("should try to create Endpoint only once", AssertOnlyOneAction("create"))
					It("should create sync event", AssertSyncEvent())
				})
			})

		})

		Context("other error", func() {
			BeforeEach(func() {
				endpointIndexer.getError = fmt.Errorf("some error")
			})

			It("should fail and not requeue", AssertErrorNoRequeue())
			It("should check for correct keys in listers", AssertKeyLookup())
			It("should not resolve hostnames", AssertNoHostnameResolve())
			It("should not do any actions", AssertNoClientActions())
		})
	})

	Describe("service and endpoint are listed", func() {
		BeforeEach(func() {
			serviceIndexer.obj = newTestService(corev1.LoadBalancerIngress{Hostname: "foo.com"})
			endpointIndexer.obj = newTestEndpoint("1.1.1.1")
		})

		Context("when failing hostname resolution", func() {
			BeforeEach(func() {
				fakeResolver.Error = fmt.Errorf("some error")
			})

			It("should fail and not requeue", AssertErrorNoRequeue())
			It("should check for correct keys in listers", AssertKeyLookup())
			It("should resolve hostnames", AssertHostnameResolve())
			It("should not do any actions", AssertNoClientActions())
			It("should not create sync event", AssertNoSyncEvent())
		})

		Context("when no ips", func() {
			BeforeEach(func() {
				fakeResolver.Addrs = []string{}
			})

			It("should succeed and not requeue", AssertNoErrorNoRequeue())
			It("should resolve hostnames", AssertHostnameResolve())
			It("should check for correct keys in listers", AssertKeyLookup())
			It("should not do any actions", AssertNoClientActions())
			It("should not create sync event", AssertNoSyncEvent())
		})
		Context("when succeeding in hostname resolution", func() {
			Context("when endpoint hostnames are same", func() {
				BeforeEach(func() {
					fakeResolver.Addrs = []string{"1.1.1.1"}
				})

				It("should succeeds and requeue later", func() {
					Expect(err).ToNot(HaveOccurred())
					Expect(result).To(Equal(reconcile.Result{Requeue: false, RequeueAfter: expectedResyncHostnames}))
				})
				It("should check for correct keys in listers", AssertKeyLookup())
				It("should resolve hostnames", AssertHostnameResolve())
				It("should not do any actions", AssertNoClientActions())
				It("should not create sync event", AssertNoSyncEvent())
			})
			Context("when endpoint hostnames are different", func() {
				BeforeEach(func() {
					expectedEndpoint = newTestEndpoint("1.1.1.1", "2.2.2.2")
					fakeResolver.Addrs = []string{"1.1.1.1", "2.2.2.2"}
				})

				Context("when updating endpoint fails", func() {
					BeforeEach(func() {
						fakeClient.AddReactor("update", "endpoints", func(action clientgotesting.Action) (bool, runtime.Object, error) {
							return true, nil, fmt.Errorf("some error")
						})
					})
					It("should fail and not requeue", AssertErrorNoRequeue())
					It("should check for correct keys in listers", AssertKeyLookup())
					It("should resolve hostnames", AssertHostnameResolve())
					It("should try to update object Endpoint only once", AssertOnlyOneAction("update"))
					It("should not create sync event", AssertNoSyncEvent())
				})
				Context("when updating endpoint succeeds", func() {
					It("should succeed and not requeue", AssertNoErrorNoRequeue())
					It("should check for correct keys in listers", AssertKeyLookup())
					It("should resolve hostnames", AssertHostnameResolve())
					It("should try to update object Endpoint only once", AssertOnlyOneAction("update"))
					It("should create sync event", AssertSyncEvent())
				})

				Context("when updating endpoint succeeds with ports and hostnames", func() {
					BeforeEach(func() {
						// order should not mattter
						fakeResolver.Addrs = []string{"2.2.2.2", "1.1.1.1"}
						serviceIndexer.obj = &corev1.Service{
							ObjectMeta: metav1.ObjectMeta{Name: serviceName, Namespace: serviceNamespace},
							Spec: corev1.ServiceSpec{
								Ports: []corev1.ServicePort{
									corev1.ServicePort{Name: "https", Port: 8443, Protocol: corev1.ProtocolTCP},
									corev1.ServicePort{Name: "http", Port: 8080, Protocol: corev1.ProtocolUDP},
								},
							},
							Status: corev1.ServiceStatus{
								LoadBalancer: corev1.LoadBalancerStatus{
									Ingress: []corev1.LoadBalancerIngress{
										corev1.LoadBalancerIngress{Hostname: "foo.com"},
										corev1.LoadBalancerIngress{IP: "8.8.8.8"},
										corev1.LoadBalancerIngress{IP: "8.8.4.4"},
									},
								},
							},
						}

						expectedEndpoint = &corev1.Endpoints{
							ObjectMeta: metav1.ObjectMeta{Name: endpointName, Namespace: endpointNamespace},
							Subsets: []corev1.EndpointSubset{corev1.EndpointSubset{
								Addresses: []corev1.EndpointAddress{
									// Services should be ordered by address
									corev1.EndpointAddress{IP: "1.1.1.1"},
									corev1.EndpointAddress{IP: "2.2.2.2"},
									corev1.EndpointAddress{IP: "8.8.4.4"},
									corev1.EndpointAddress{IP: "8.8.8.8"},
								},
								Ports: []corev1.EndpointPort{
									corev1.EndpointPort{Name: "https", Port: 8443, Protocol: corev1.ProtocolTCP},
									corev1.EndpointPort{Name: "http", Port: 8080, Protocol: corev1.ProtocolUDP},
								},
							}},
						}
					})
					It("should succeed and not requeue", AssertNoErrorNoRequeue())
					It("should check for correct keys in listers", AssertKeyLookup())
					It("should resolve hostnames", AssertHostnameResolve())
					It("should try to update object Endpoint only once", AssertOnlyOneAction("update"))
					It("should create sync event", AssertSyncEvent())
				})
			})
		})
	})

})

var _ = Describe("NewReconciler", func() {
	var (
		controller      *Controller
		endpoint        types.NamespacedName
		endpointLister  corev1lister.EndpointsNamespaceLister
		fakeClient      endpointsclient.EndpointsInterface
		fakeResolver    *fake_resolver.Resolver
		resyncHostnames time.Duration
		serviceLister   corev1lister.ServiceNamespaceLister
		fakeRecorder    *record.FakeRecorder
	)
	JustBeforeEach(func() {
		controller = NewReconciler(fakeClient, fakeRecorder, serviceLister, endpointLister, serviceName, endpoint, resyncHostnames, fakeResolver).(*Controller)
	})
	BeforeEach(func() {
		client := fake.Clientset{}
		fakeClient = client.CoreV1().Endpoints(endpointNamespace)
		endpoint = types.NamespacedName{Name: endpointName, Namespace: endpointNamespace}
		serviceLister = corev1lister.NewServiceLister(nil).Services(serviceNamespace)
		endpointLister = corev1lister.NewEndpointsLister(nil).Endpoints(endpointNamespace)
		fakeResolver = &fake_resolver.Resolver{}
		fakeRecorder = record.NewFakeRecorder(10)
		resyncHostnames = 1
	})
	Context("when Controller fields are set", func() {
		It("should have correct endpoint", func() {
			Expect(controller.Endpoint).To(BeIdenticalTo(endpoint))
		})
		It("should have correct endpoint lister", func() {
			Expect(controller.EndpointLister).To(BeIdenticalTo(endpointLister))
		})
		It("should have correct shoot client", func() {
			Expect(controller.EndpointsClient).To(BeIdenticalTo(fakeClient))
		})
		It("should have correct resolver", func() {
			Expect(controller.Resolver).To(BeIdenticalTo(fakeResolver))
		})
		It("should have correct hostname resync duration", func() {
			Expect(controller.ResyncHostnames).To(BeIdenticalTo(resyncHostnames))
		})
		It("should have correct service lister", func() {
			Expect(controller.ServiceLister).To(BeIdenticalTo(serviceLister))
		})
		It("should have correct logger", func() {
			Expect(controller.Log).To(Equal(logf.Log.WithName("controller.readvertser")))
		})
		It("should have correct recorder", func() {
			Expect(controller.Recorder).To(BeIdenticalTo(fakeRecorder))
		})
	})
})
