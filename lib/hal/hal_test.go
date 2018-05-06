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

package hal_test

import (
	"encoding/json"

	"github.com/alien-bunny/ab/lib/hal"
	"github.com/alien-bunny/ab/services/resource"
	. "github.com/onsi/ginkgo"
	. "github.com/onsi/gomega"
)

var _ hal.EndpointLinker = &item{}

type item struct {
	A int    `json:"a"`
	B string `json:"b"`
	C uint   `json:"c,omitempty"`
}

func (i *item) Links() map[string][]interface{} {
	return map[string][]interface{}{
		"rel0": {"http://example.com", "http://asdf.example.com"},
		"rel1": {"http://xzcv.example.com"},
	}
}

func (i *item) Curies() []hal.HALCurie {
	return []hal.HALCurie{
		{
			Name:      "qwerty",
			Href:      "http://qwerty.example.com",
			Templated: false,
		},
		{
			Name:      "asdf",
			Href:      "http://asdf.example.com/{rel}",
			Templated: true,
		},
	}
}

type nonhalitem struct {
	A int
	B string
}

const (
	haljson        = `{"_links":{"curies":[{"name":"qwerty","href":"http://qwerty.example.com","templated":false},{"name":"asdf","href":"http://asdf.example.com/{rel}","templated":true}],"rel0":[{"href":"http://example.com"},{"href":"http://asdf.example.com"}],"rel1":[{"href":"http://xzcv.example.com"}]},"a":5,"b":"asdf"}`
	reslisthaljson = `{"items":[{"_links":{"curies":[{"name":"qwerty","href":"http://qwerty.example.com","templated":false},{"name":"asdf","href":"http://asdf.example.com/{rel}","templated":true}],"rel0":[{"href":"http://example.com"},{"href":"http://asdf.example.com"}],"rel1":[{"href":"http://xzcv.example.com"}]},"a":5,"b":"asdf","c":8},{"_links":{"curies":[{"name":"qwerty","href":"http://qwerty.example.com","templated":false},{"name":"asdf","href":"http://asdf.example.com/{rel}","templated":true}],"rel0":[{"href":"http://example.com"},{"href":"http://asdf.example.com"}],"rel1":[{"href":"http://xzcv.example.com"}]},"a":2,"b":"zxcvbn"},{"A":1,"B":"bar"},{"A":7,"B":"baz"}],"_links":{"curies":[{"name":"foo","href":"http://foo.example.com","templated":false},{"name":"bar","href":"http://bar.example.com","templated":false},{"name":"baz","href":"http://baz.example.com","templated":false}],"page next":[{"href":"/api/foo?page=6"}],"page previous":[{"href":"/api/foo?page=4"}],"rel test":[{"href":"http://test.example.com"}]}}`
)

var _ = Describe("Hal", func() {
	Describe("A simple struct embedded in a HAL wrapper", func() {
		i := &item{
			A: 5,
			B: "asdf",
		}

		w := hal.NewHalWrapper(i)
		It("should produce a flattened struct", func() {
			marshaled, merr := json.Marshal(w)
			Expect(merr).To(BeNil())
			Expect(string(marshaled)).To(Equal(haljson))
		})
	})

	Describe("A list of HAL and non-HAL resources in a ResourceList", func() {
		l := &resource.ResourceList{
			Items: []resource.Resource{
				&item{A: 5, B: "asdf", C: 8},
				&item{A: 2, B: "zxcvbn"},
				&nonhalitem{A: 1, B: "bar"},
				&nonhalitem{A: 7, B: "baz"},
			},
			Page:     5,
			PageSize: 4,
			BasePath: "/api/foo",
			Curies: []hal.HALCurie{
				{Name: "foo", Href: "http://foo.example.com"},
				{Name: "bar", Href: "http://bar.example.com"},
				{Name: "baz", Href: "http://baz.example.com"},
			},
			Rels: map[string][]interface{}{
				"rel test": {"http://test.example.com"},
			},
		}

		It("should have _links on all applicable places", func() {
			marshaled, merr := json.Marshal(l)
			Expect(merr).To(BeNil())
			Expect(string(marshaled)).To(Equal(reslisthaljson))
		})
	})
})
