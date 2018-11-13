package controller

import (
	"errors"
	"fmt"

	corev1 "k8s.io/api/core/v1"
	"k8s.io/apimachinery/pkg/util/sets"
)

type port struct {
	name, protocol string
	port           int
}

// checks to see if the IPs behind the loadbalancers didn't change and that the current values of the endpoints are conformant with these records.
func checkEndpointIsStillValid(currentEndpointValues []string, elbFetchedRecords []string) bool {
	currentEndpoints := sets.NewString(currentEndpointValues...)
	return currentEndpoints.HasAll(elbFetchedRecords...)
}

// createEndpointSubset creates an endpoint subset from a set of IPs and currently with a constant port 443
func createEndpointSubsetObjectFromRecords(ips []string) (*corev1.EndpointSubset, error) {
	if len(ips) == 0 {
		return nil, errors.New("Empty list of IPs")
	}

	var endpointAddresses []corev1.EndpointAddress
	for _, ip := range ips {
		endpointAddresses = append(endpointAddresses, corev1.EndpointAddress{
			IP: ip,
		})
	}

	return &corev1.EndpointSubset{
		Addresses: endpointAddresses,
		Ports: []corev1.EndpointPort{
			{
				Name:     "https",
				Port:     443,
				Protocol: "TCP",
			},
		},
	}, nil
}

// fetchEndpointIPsFromAddresses returns the list of endpoint IPs from a slice of an EndpointAddress object
func fetchEndpointIPsFromAddresses(addresses []corev1.EndpointAddress) ([]string, error) {
	if len(addresses) == 0 {
		return nil, fmt.Errorf("empty endpoint addresses")
	}

	var ips []string
	for _, addr := range addresses {
		ips = append(ips, addr.IP)
	}
	return ips, nil
}
