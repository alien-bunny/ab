// Copyright 2018 Tamás Demeter-Haludka
//
// Licensed under the Apache License, Version 2.0 (the "License");
// you may not use this file except in compliance with the License.
// You may obtain a copy of the License at
//
//     http://www.apache.org/licenses/LICENSE-2.0
//
// Unless required by applicable law or agreed to in writing, software
// distributed under the License is distributed on an "AS IS" BASIS,
// WITHOUT WARRANTIES OR CONDITIONS OF ANY KIND, either express or implied.
// See the License for the specific language governing permissions and
// limitations under the License.

package securitymw

import (
	"net"
	"net/http"
	"strings"

	"github.com/alien-bunny/ab/lib/middleware"
)

const MIDDLEWARE_DEPENDENCY_RESTRICTADDRESS = "*securitymw.RestrictAddressMiddleware"

var _ middleware.Middleware = &RestrictAddressMiddleware{}

// RestrictAddressMiddleware restricts access based on the IP address of the client.
//
// Only IP addresses in the given CIDR address ranges will be allowed.
type RestrictAddressMiddleware struct {
	cidrnets []*net.IPNet

	middleware.NoDependencies
}

func NewRestrictAddressMiddleware(addresses ...string) *RestrictAddressMiddleware {
	cidrnets := make([]*net.IPNet, len(addresses))
	var err error
	for i, address := range addresses {
		_, cidrnets[i], err = net.ParseCIDR(address)
		if err != nil {
			panic(err)
		}
	}

	return &RestrictAddressMiddleware{
		cidrnets: cidrnets,
	}
}

func NewRestrictPrivateAddressMiddleware() *RestrictAddressMiddleware {
	return NewRestrictAddressMiddleware("10.255.255.255/8", "172.31.255.255/12", "192.168.255.255/16", "127.0.0.0/8")
}

func (m *RestrictAddressMiddleware) Wrap(next http.Handler) http.Handler {
	return http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		reqAddress := strings.Split(r.RemoteAddr, ":")[0]
		ip := net.ParseIP(reqAddress)
		for _, cidrnet := range m.cidrnets {
			if cidrnet.Contains(ip) {
				next.ServeHTTP(w, r)
				return
			}
		}

		http.Error(w, http.StatusText(http.StatusServiceUnavailable), http.StatusServiceUnavailable)
	})
}
