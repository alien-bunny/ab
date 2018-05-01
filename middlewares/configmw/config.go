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

package configmw

import (
	"net/http"
	"reflect"
	"strings"
	"sync"

	"github.com/alien-bunny/ab/lib/config"
	"github.com/alien-bunny/ab/lib/errors"
	"github.com/alien-bunny/ab/lib/middleware"
	"github.com/alien-bunny/ab/lib/util"
	"github.com/alien-bunny/ab/middlewares/logmw"
)

const (
	MiddlewareDependencyConfig = "*configmw.ConfigMiddleware"
	CategoryConfigNotFound     = "config not found"
	configKey                  = "abconfig"
	configWritableKey          = "abwritableconfig"
)

type NamespaceNegotiator interface {
	NegotiateNamespace(r *http.Request) (string, error)
}

func GetConfig(r *http.Request) config.Config {
	return r.Context().Value(configKey).(config.Config)
}

func GetWritableConfig(r *http.Request) config.WritableConfig {
	return r.Context().Value(configWritableKey).(config.WritableConfig)
}

var _ middleware.Middleware = &ConfigMiddleware{}

type ConfigMiddleware struct {
	configStore *config.Store
	negotiator  NamespaceNegotiator

	middleware.NoDependencies
}

func NewConfigMiddleware(store *config.Store, negotiator NamespaceNegotiator) *ConfigMiddleware {
	return &ConfigMiddleware{
		configStore: store,
		negotiator:  negotiator,
	}
}

func (c *ConfigMiddleware) Wrap(next http.Handler) http.Handler {
	return http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		namespace, err := c.negotiator.NegotiateNamespace(r)
		if err != nil {
			http.Error(w, err.Error(), http.StatusBadRequest)
			return
		}

		cfg := c.configStore.Get(namespace)
		if cfg == nil {
			http.Error(w, "host not found", http.StatusNotFound)
			return
		}
		r = util.SetContext(r, configWritableKey, c.configStore.GetWritable(namespace))
		r = util.SetContext(r, configKey, cfg)

		next.ServeHTTP(w, r)
	})
}

var _ NamespaceNegotiator = &HostNamespaceNegotiator{}

type HostNamespaceNegotiator struct {
	SkipPort bool
}

func (h *HostNamespaceNegotiator) NegotiateNamespace(r *http.Request) (string, error) {
	if h.SkipPort {
		return strings.Split(r.Host, ":")[0], nil
	}

	return r.Host, nil
}

func NewHostNamespaceNegotiator() *HostNamespaceNegotiator {
	return &HostNamespaceNegotiator{}
}

var _ NamespaceNegotiator = &HostMapNamespaceNegotiator{}

type HostMapNamespaceNegotiator struct {
	mtx     sync.RWMutex
	hostmap map[string]string
}

func NewHostMapNamespaceNegotiator() *HostMapNamespaceNegotiator {
	return &HostMapNamespaceNegotiator{
		hostmap: make(map[string]string),
	}
}

func (h *HostMapNamespaceNegotiator) Add(host, namespace string) {
	h.mtx.Lock()
	h.hostmap[host] = namespace
	h.mtx.Unlock()
}

func (h *HostMapNamespaceNegotiator) Remove(host string) {
	h.mtx.Lock()
	delete(h.hostmap, host)
	h.mtx.Unlock()
}

func (h *HostMapNamespaceNegotiator) NegotiateNamespace(r *http.Request) (string, error) {
	h.mtx.RLock()
	namespace, found := h.hostmap[r.Host]
	h.mtx.RUnlock()

	if !found {
		return "", errors.New("host not found")
	}

	return namespace, nil
}

var _ NamespaceNegotiator = &ChainedNamespaceNegotiator{}

type ChainedNamespaceNegotiator struct {
	negotiators []NamespaceNegotiator
}

func NewChainedNamespaceNegotiator(negotiators ...NamespaceNegotiator) *ChainedNamespaceNegotiator {
	return &ChainedNamespaceNegotiator{
		negotiators: negotiators,
	}
}

func (n *ChainedNamespaceNegotiator) AddNegotiator(negotiator NamespaceNegotiator) {
	n.negotiators = append(n.negotiators, negotiator)
}

func (n *ChainedNamespaceNegotiator) NegotiateNamespace(r *http.Request) (string, error) {
	for _, negotiator := range n.negotiators {
		if namespace, err := negotiator.NegotiateNamespace(r); err == nil {
			return namespace, nil
		}
	}

	return config.Default, errors.New("host not found")
}

type middlewareWrapper struct {
	key string
	t   reflect.Type
}

func (m *middlewareWrapper) ConfigSchema() map[string]reflect.Type {
	return map[string]reflect.Type{
		m.key: m.t,
	}
}

func (m *middlewareWrapper) Dependencies() []string {
	return append([]string{
		MiddlewareDependencyConfig,
		logmw.MiddlewareDependencyLog,
	}, reflect.New(m.t).Interface().(middleware.Middleware).Dependencies()...)
}

func (m *middlewareWrapper) Wrap(next http.Handler) http.Handler {
	return http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		mwi, err := GetConfig(r).Get(m.key)
		if err != nil {
			logmw.Info(r, "", CategoryConfigNotFound).Log("error", err, "key", m.key)
		}

		if mwi != nil {
			mw := mwi.(middleware.Middleware)
			mw.Wrap(next).ServeHTTP(w, r)
		} else {
			next.ServeHTTP(w, r)
		}
	})
}

func WrapMiddleware(key string, t reflect.Type) middleware.Middleware {
	return &middlewareWrapper{key: key, t: t}
}
