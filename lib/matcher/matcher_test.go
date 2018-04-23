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

package matcher_test

import (
	"github.com/alien-bunny/ab/lib/matcher"
	. "github.com/onsi/ginkgo"
	. "github.com/onsi/gomega"
)

var _ = Describe("Matcher", func() {
	It("should match all test cases", func() {
		m := matcher.NewMatcher(".")
		Expect(m.Get("foo.bar")).To(BeNil())

		value := "asdf"
		m.Set("item.*", value)
		Expect(m.Get("item.baz")).To(Equal(value))

		value = "qwer"
		m.Set("item.*.bar", value)
		Expect(m.Get("item.foo.bar")).To(Equal(value))

		value = "zxcv"
		m.Set("item.*.*.baz", value)
		Expect(m.Get("item.foo.baz.baz")).To(Equal(value))
	})
})
