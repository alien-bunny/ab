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

package ab

import (
	"crypto/tls"
	"path/filepath"
	"sync"

	"github.com/alien-bunny/ab/lib/config"
	"github.com/alien-bunny/ab/lib/errors"
	"github.com/alien-bunny/ab/lib/log"
)

type certCache struct {
	conf     *config.Store
	mtx      sync.RWMutex
	cache    map[string]*tls.Certificate
	logger   log.Logger
	certFile string
	keyFile  string
}

func newCertCache(conf *config.Store, logger log.Logger, certFile, keyFile string) *certCache {
	c := &certCache{
		conf:     conf,
		logger:   logger,
		certFile: certFile,
		keyFile:  keyFile,
	}
	c.clear()

	return c
}

func (c *certCache) clear() {
	c.mtx.Lock()
	c.cache = make(map[string]*tls.Certificate)
	c.mtx.Unlock()
}

func (c *certCache) get(hello *tls.ClientHelloInfo) (*tls.Certificate, error) {
	var cert *tls.Certificate
	var err error

	if cert = c.getCached(hello); cert != nil {
		return cert, nil
	}

	cert, err = c.load(hello)
	if err != nil {
		return nil, err
	}

	c.mtx.Lock()
	c.cache[hello.ServerName] = cert
	c.mtx.Unlock()

	return cert, nil
}

func (c *certCache) getCached(hello *tls.ClientHelloInfo) *tls.Certificate {
	c.mtx.RLock()
	cert := c.cache[hello.ServerName]
	c.mtx.RUnlock()

	return cert
}

func (c *certCache) load(hello *tls.ClientHelloInfo) (*tls.Certificate, error) {
	cfg := c.conf.Get(hello.ServerName)
	if cfg == nil {
		return nil, errors.New("namespace not found")
	}

	s, err := cfg.Get("site")
	if err != nil {
		return nil, err
	}

	dir := s.(Site).Directories.TLSCertDir
	cert, err := tls.LoadX509KeyPair(filepath.Join(dir, c.certFile), filepath.Join(dir, c.keyFile))
	if err != nil {
		return nil, err
	}

	return &cert, nil
}
