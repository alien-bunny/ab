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

package certcache

import (
	"crypto/tls"
	"sync"

	"github.com/alien-bunny/ab/lib/log"
)

type CertCache struct {
	mtx    sync.RWMutex
	cache  map[string]*tls.Certificate
	logger log.Logger
	loader func(string) (string, string, error)
}

func New(logger log.Logger, loader func(string) (string, string, error)) *CertCache {
	c := &CertCache{
		logger: logger,
		loader: loader,
	}
	c.Clear()

	return c
}

func (c *CertCache) Clear() {
	c.mtx.Lock()
	c.cache = make(map[string]*tls.Certificate)
	c.mtx.Unlock()
	log.Debug(c.logger).Log("cache clear", "certcache")
}

func (c *CertCache) Get(hello *tls.ClientHelloInfo) (*tls.Certificate, error) {
	var cert *tls.Certificate
	var err error

	if cert = c.getCached(hello); cert != nil {
		log.Debug(c.logger).Log("certificate found", hello.ServerName, "cache", "true")
		return cert, nil
	}

	cert, err = c.load(hello)
	if err != nil {
		return nil, err
	}

	c.mtx.Lock()
	c.cache[hello.ServerName] = cert
	c.mtx.Unlock()
	log.Debug(c.logger).Log("certificate found", hello.ServerName, "cache", "false")

	return cert, nil
}

func (c *CertCache) getCached(hello *tls.ClientHelloInfo) *tls.Certificate {
	c.mtx.RLock()
	cert := c.cache[hello.ServerName]
	c.mtx.RUnlock()

	return cert
}

func (c *CertCache) load(hello *tls.ClientHelloInfo) (*tls.Certificate, error) {
	cert, key, err := c.loader(hello.ServerName)
	if err != nil {
		return nil, err
	}

	kp, err := tls.X509KeyPair([]byte(cert), []byte(key))
	if err != nil {
		return nil, err
	}

	return &kp, nil
}
