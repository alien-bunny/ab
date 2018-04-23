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
	"os"
	"strings"

	"github.com/alien-bunny/ab/lib/env"
)

var _ Provider = &EnvConfigProvider{}

type EnvConfigProvider struct {
	Prefix    string
	Separator string
	variables map[string]string
}

func NewEnvConfigProvider() *EnvConfigProvider {
	return &EnvConfigProvider{
		Prefix:    "",
		Separator: "_",
		variables: nil,
	}
}

func (e *EnvConfigProvider) maybeInitializeVariables() {
	if e.variables != nil {
		return
	}

	e.variables = make(map[string]string)
	for _, ev := range os.Environ() {
		parts := strings.SplitN(ev, "=", 2)
		e.variables[parts[0]] = parts[1]
	}
}

func (e *EnvConfigProvider) Reset() {
	e.variables = nil
}

func (e *EnvConfigProvider) prefixedKey(key string) string {
	key = strings.ToUpper(key)
	if e.Prefix == "" {
		return key
	}
	return e.Prefix + e.Separator + key
}

func (e *EnvConfigProvider) loader(key string) (string, bool) {
	val, found := e.variables[e.prefixedKey(key)]
	return val, found
}

func (e *EnvConfigProvider) Has(key string) bool {
	e.maybeInitializeVariables()
	key = e.prefixedKey(key)
	for k := range e.variables {
		if strings.HasPrefix(k, key) {
			return true
		}
	}

	return false
}

func (e *EnvConfigProvider) Unmarshal(key string, v interface{}) error {
	e.maybeInitializeVariables()
	u := env.NewUnmarshaler()
	u.Prefix = key
	u.Separator = e.Separator
	u.Loader = e.loader

	return u.Unmarshal(v)
}
