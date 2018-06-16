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

package config_test

import (
	"errors"
	"testing"

	"github.com/alien-bunny/ab/lib/config"
	. "github.com/onsi/ginkgo"
	. "github.com/onsi/gomega"
)

func TestConfig(t *testing.T) {
	RegisterFailHandler(Fail)
	RunSpecs(t, "Config Suite")
}

var _ config.WritableProvider = &errorProvider{}

type errorProvider struct {
	OnRead  bool
	OnWrite bool
}

func (p *errorProvider) CanSave(key string) bool {
	return true
}

func (p *errorProvider) Save(key string, v interface{}) error {
	if p.OnWrite {
		return errors.New("")
	}

	return nil
}

func (p *errorProvider) Has(key string) bool {
	return true
}

func (p *errorProvider) Unmarshal(key string, v interface{}) error {
	if p.OnRead {
		return errors.New("")
	}

	return nil
}
