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

package env_test

import (
	. "github.com/onsi/ginkgo"
	. "github.com/onsi/ginkgo/extensions/table"
	. "github.com/onsi/gomega"

	"os"
	"reflect"

	"github.com/alien-bunny/ab/lib/env"
)

type data struct {
	A int
	B string
	C bool
	D uint
	E float64
}

type simpleData struct {
	A int
	B bool
}

type invalidData struct {
	f func()
}

var _ = Describe("Env", func() {
	entries := []TableEntry{
		Entry("basic data", map[string]string{
			"FOO_A": "-2",
			"FOO_B": "asdf",
			"FOO_C": "true",
			"FOO_D": "5",
			"FOO_E": "-1.2",
		}, &data{-2, "asdf", true, 5, -1.2}, "FOO"),
		Entry("simple data", map[string]string{
			"A": "5",
			"B": "false",
		}, &simpleData{5, false}, ""),
	}

	BeforeEach(func() {
		os.Clearenv()
	})

	DescribeTable("unmarshaling environment variables",
		func(e map[string]string, expected interface{}, prefix string) {
			for k, v := range e {
				os.Setenv(k, v)
			}

			v := reflect.New(reflect.Indirect(reflect.ValueOf(expected)).Type()).Interface()
			u := env.NewUnmarshaler()
			u.Prefix = prefix
			u.Strict = true
			err := u.Unmarshal(v)
			Expect(err).NotTo(HaveOccurred())
			Expect(v).To(Equal(expected))
		},
		entries...,
	)

	It("should fail when a non-pointer is given", func() {
		u := env.NewUnmarshaler()
		d := simpleData{}
		err := u.Unmarshal(d)
		Expect(err).To(HaveOccurred())
		Expect(err.Error()).To(Equal("env: Unmarshal(non-pointer env_test.simpleData)"))
	})

	It("should fail when a nil is given", func() {
		u := env.NewUnmarshaler()
		err := u.Unmarshal(nil)
		Expect(err).To(HaveOccurred())
		Expect(err.Error()).To(Equal("env: Unmarshal(nil)"))
	})

	It("should fail when an invalid type is given", func() {
		u := env.NewUnmarshaler()
		u.Strict = true
		d := &invalidData{}
		err := u.Unmarshal(d)
		Expect(err).To(HaveOccurred())
		Expect(err.Error()).To(Equal("env: Unmarshal(func())"))
	})
})
