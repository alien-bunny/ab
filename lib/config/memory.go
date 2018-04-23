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
	"github.com/alien-bunny/ab/lib/errors"
	"github.com/imdario/mergo"
)

var _ WritableProvider = &MemoryConfigProvider{}

type MemoryConfigProvider struct {
	store map[string]interface{}
}

func NewMemoryConfigProvider() *MemoryConfigProvider {
	m := &MemoryConfigProvider{}
	m.Reset()
	return m
}

func (m *MemoryConfigProvider) Reset() {
	m.store = make(map[string]interface{})
}

func (m *MemoryConfigProvider) CanSave(key string) bool {
	return true
}

func (m *MemoryConfigProvider) Save(key string, v interface{}) error {
	m.store[key] = v
	return nil
}

func (m *MemoryConfigProvider) Has(key string) bool {
	_, found := m.store[key]
	return found
}

func (m *MemoryConfigProvider) Unmarshal(key string, v interface{}) error {
	val, found := m.store[key]
	if found {
		return mergo.Merge(v, val)
	}

	return errors.New("value not found")
}
