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

package config

import (
	"reflect"
	"sync"

	"github.com/alien-bunny/ab/lib/errors"
	"github.com/alien-bunny/ab/lib/log"
	"github.com/alien-bunny/ab/lib/matcher"
	"github.com/imdario/mergo"
)

const (
	Default = ""
)

type ConfigSchemaProvider interface {
	ConfigSchema() map[string]reflect.Type
}

type Config interface {
	Get(key string) (interface{}, error)
}

type WritableConfig interface {
	Config
	GetWritable(key string) (interface{}, Saver, error)
}

type Provider interface {
	Has(key string) bool
	Unmarshal(key string, v interface{}) error
}

type WritableProvider interface {
	Provider
	CanSave(key string) bool
	Save(key string, v interface{}) error
}

type Saver interface {
	Save(v interface{}) error
}

var _ Saver = saverFunc(nil)

type saverFunc func(v interface{}) error

func (f saverFunc) Save(v interface{}) error {
	return f(v)
}

type CollectionLoader interface {
	Load(name string) (*Collection, error)
}

var _ CollectionLoader = CollectionLoaderFunc(nil)

type CollectionLoaderFunc func(name string) (*Collection, error)

func (f CollectionLoaderFunc) Load(name string) (*Collection, error) {
	return f(name)
}

type Store struct {
	mtx               sync.RWMutex
	namespaces        map[string]*Collection
	schemas           *matcher.Matcher
	collectionLoaders []CollectionLoader
	logger            log.Logger
}

func NewStore(logger log.Logger) *Store {
	return &Store{
		namespaces: make(map[string]*Collection),
		schemas:    matcher.NewMatcher("."),
		logger:     logger,
	}
}

func (s *Store) AddCollection(namespace string, collection *Collection) {
	s.mtx.Lock()
	s.namespaces[namespace] = collection
	s.mtx.Unlock()
}

func (s *Store) AddCollectionLoaders(cl ...CollectionLoader) {
	s.collectionLoaders = append(s.collectionLoaders, cl...)
}

func (s *Store) MaybeRegisterSchema(v interface{}) {
	if csp, ok := v.(ConfigSchemaProvider); ok {
		for name, t := range csp.ConfigSchema() {
			s.RegisterSchema(name, t)
		}
	}
}

func (s *Store) RegisterSchema(name string, schema reflect.Type) {
	s.mtx.Lock()
	defer s.mtx.Unlock()

	if s.schemas.Get(name) != nil && s.schemas.Get(name) != schema {
		panic("schema " + name + " is already registered")
	}

	s.schemas.Set(name, schema)
}

func (s *Store) ClearAllCaches() {
	s.mtx.Lock()
	defer s.mtx.Unlock()

	for _, c := range s.namespaces {
		c.ClearCache()
	}
}

func (s *Store) ensureNamespace(namespace string) *Collection {
	s.mtx.RLock()
	collection, exists := s.namespaces[namespace]
	s.mtx.RUnlock()
	if exists {
		return collection
	}

	for _, loader := range s.collectionLoaders {
		collection, err := loader.Load(namespace)
		if err != nil {
			log.Warn(s.logger).Log("namespace load error", err)
		}

		if collection != nil {
			s.mtx.Lock()
			s.namespaces[namespace] = collection
			s.mtx.Unlock()
			return collection
		}
	}

	return nil
}

func (s *Store) getInstance(namespace string, readonly bool) WritableConfig {
	if s.ensureNamespace(namespace) == nil {
		return nil
	}
	return &instance{
		namespace: namespace,
		parent:    s,
		readonly:  readonly,
	}
}

func (s *Store) Get(namespace string) Config {
	return s.getInstance(namespace, true)
}

func (s *Store) GetWritable(namespace string) WritableConfig {
	return s.getInstance(namespace, false)
}

func (s *Store) RemoveTemporary() {
	s.mtx.Lock()
	for namespace, data := range s.namespaces {
		if data.temporary {
			delete(s.namespaces, namespace)
		}
	}
	s.mtx.Unlock()
}

func (s *Store) get(namespace, key string) (interface{}, error) {
	collection := s.ensureNamespace(namespace)
	if collection == nil {
		return nil, CollectionNotFoundError{namespace}
	}

	var val interface{}
	var err error

	s.mtx.RLock()
	if returnType := s.schemas.Get(key); returnType != nil {
		val, err = collection.get(key, returnType.(reflect.Type))
	} else {
		err = errors.NewError("schema not found", "", nil)
	}
	s.mtx.RUnlock()

	return val, err
}

func (s *Store) set(namespace, key string, v interface{}) error {
	collection := s.ensureNamespace(namespace)
	if collection == nil {
		return CollectionNotFoundError{namespace}
	}

	s.mtx.RLock()
	defer s.mtx.RUnlock()
	if returnType := s.schemas.Get(key); returnType != nil {
		if reflect.TypeOf(v) != returnType.(reflect.Type) {
			return errors.New("invalid type")
		}
	} else {
		return errors.New("unknown type")
	}

	return collection.set(key, v)
}

type Collection struct {
	mtx       sync.RWMutex
	cache     map[string]interface{}
	providers []Provider
	temporary bool
}

func NewCollection() *Collection {
	c := &Collection{}
	c.ClearCache()
	return c
}

func (c *Collection) get(key string, returnType reflect.Type) (interface{}, error) {
	val, found := c.getFromCache(key)
	if found {
		return val, nil
	}

	val, err := c.find(key, returnType)

	if err != nil {
		return nil, err
	}

	c.putToCache(key, val)

	return val, nil
}

func (c *Collection) find(key string, returnType reflect.Type) (interface{}, error) {
	var ptr reflect.Value
	merge := false

	for _, provider := range c.providers {
		if provider.Has(key) {
			currentPtr := reflect.New(returnType)
			if err := provider.Unmarshal(key, currentPtr.Interface()); err != nil {
				return nil, err
			}
			if !merge {
				ptr = currentPtr
				merge = true
			} else if err := mergo.Merge(ptr.Interface(), reflect.Indirect(currentPtr).Interface()); err != nil {
				return nil, err
			}
		}
	}

	if ptr.IsValid() {
		return reflect.Indirect(ptr).Interface(), nil
	}

	return nil, nil
}

func (c *Collection) set(key string, v interface{}) error {
	var err error
	var saved bool
	c.mtx.Lock()
	for _, provider := range c.providers {
		if wp, ok := provider.(WritableProvider); ok && wp.CanSave(key) {
			err = wp.Save(key, v)
			saved = true
			break
		}
	}
	c.mtx.Unlock()

	if !saved {
		return errors.New("failed to save config")
	}

	if err == nil {
		c.putToCache(key, v)
	}

	return nil
}

func (c *Collection) getFromCache(key string) (interface{}, bool) {
	c.mtx.RLock()
	val, exists := c.cache[key]
	c.mtx.RUnlock()

	return val, exists
}

func (c *Collection) putToCache(key string, v interface{}) {
	c.mtx.Lock()
	c.cache[key] = v
	c.mtx.Unlock()
}

func (c *Collection) ClearCache() {
	c.mtx.Lock()
	c.cache = make(map[string]interface{})
	c.mtx.Unlock()
}

func (c *Collection) SetTemporary(temporary bool) {
	c.temporary = temporary
}

func (c *Collection) AddProviders(providers ...Provider) {
	c.providers = append(c.providers, providers...)
	c.ClearCache()
}

type instance struct {
	namespace string
	parent    *Store
	readonly  bool
}

func (i *instance) Get(key string) (interface{}, error) {
	return i.parent.get(i.namespace, key)
}

func (i *instance) GetWritable(key string) (interface{}, Saver, error) {
	if i.readonly {
		return nil, nil, errors.NewError("readonly instance cannot be used as writable", "", nil)
	}

	val, err := i.Get(key)
	if err != nil {
		return nil, nil, err
	}

	return val, saverFunc(func(v interface{}) error {
		return i.parent.set(i.namespace, key, v)
	}), nil
}

var _ error = CollectionNotFoundError{}

type CollectionNotFoundError struct {
	Name string
}

func (e CollectionNotFoundError) Error() string {
	return "collection not found: " + e.Name
}
