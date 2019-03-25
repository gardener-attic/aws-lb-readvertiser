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

package fake

import (
	"sync"

	net_resolver "github.com/gardener/aws-lb-readvertiser/pkg/net"
)

var _ net_resolver.Resolver = &Resolver{}

// Resolver fake dummy resolver
type Resolver struct {
	Addrs   []string
	Error   error
	Lookups []string
	mu      sync.Mutex
}

// LookupHost returns addresses from "Addrs" and error from "Error"
func (f *Resolver) LookupHost(host string) (addrs []string, err error) {
	f.mu.Lock()
	defer f.mu.Unlock()
	f.Lookups = append(f.Lookups, host)
	return f.Addrs, f.Error
}
