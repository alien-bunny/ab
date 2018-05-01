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

package certcache_test

import (
	"crypto/tls"
	"io/ioutil"
	"strconv"

	"github.com/alien-bunny/ab/lib/certcache"
	"github.com/alien-bunny/ab/lib/errors"
	"github.com/alien-bunny/ab/lib/log"
	"github.com/alien-bunny/ab/lib/util"
	. "github.com/onsi/ginkgo"
	. "github.com/onsi/gomega"
)

func init() {
	hostmap = make(map[string]host)

	for i := 0; i < 2; i++ {
		name := strconv.Itoa(i) + ".example.com"
		h := host{}
		h.cert, h.key = util.GenerateCertificate(name, "ACME Co")
		hostmap[name] = h
	}

	hostmap["error.example.com"] = host{}
}

type host struct {
	cert string
	key  string
}

var hostmap map[string]host

var _ = Describe("Certcache", func() {
	logger := log.NewDevLogger(ioutil.Discard)
	cc := certcache.New(logger, func(host string) (string, string, error) {
		h, exists := hostmap[host]
		if !exists {
			return "", "", errors.New("host not found")
		}

		return h.cert, h.key, nil
	})

	It("should find and load a certificate", func() {
		cert, err := cc.Get(&tls.ClientHelloInfo{
			ServerName: "0.example.com",
		})
		Expect(cert).NotTo(BeNil())
		Expect(err).NotTo(HaveOccurred())
	})

	It("should return an error when an invalid host is requested", func() {
		cert, err := cc.Get(&tls.ClientHelloInfo{
			ServerName: "xxx.example.com",
		})
		Expect(cert).To(BeNil())
		Expect(err).To(HaveOccurred())
	})

	It("should return an error when the certificate is invalid", func() {
		cert, err := cc.Get(&tls.ClientHelloInfo{
			ServerName: "error.example.com",
		})
		Expect(cert).To(BeNil())
		Expect(err).To(HaveOccurred())
	})

	It("should return the value from cache", func() {
		cert, err := cc.Get(&tls.ClientHelloInfo{
			ServerName: "1.example.com",
		})
		Expect(cert).NotTo(BeNil())
		Expect(err).NotTo(HaveOccurred())

		delete(hostmap, "1.example.com")

		cert, err = cc.Get(&tls.ClientHelloInfo{
			ServerName: "1.example.com",
		})
		Expect(cert).NotTo(BeNil())
		Expect(err).NotTo(HaveOccurred())

		cc.Clear()

		cert, err = cc.Get(&tls.ClientHelloInfo{
			ServerName: "1.example.com",
		})
		Expect(cert).To(BeNil())
		Expect(err).To(HaveOccurred())
	})
})
