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

package resource_test

import (
	"encoding/json"
	"net/http"
	"time"

	"github.com/alien-bunny/ab/lib/abtest"
	"github.com/alien-bunny/ab/lib/hal"
	"github.com/alien-bunny/ab/services/resource"
	. "github.com/onsi/ginkgo"
	. "github.com/onsi/gomega"
	. "github.com/onsi/gomega/gstruct"
)

const emptyListResponse = "[]\n"

var _ = Describe("Resource", func() {
	It("should go through the crud steps", func() {
		client := clientFactory()

		By("listing an empty endpoint")
		client.Request("GET", "/api/test", nil, nil, func(resp *http.Response) {
			Expect(client.ReadBody(resp, true)).To(Equal(emptyListResponse))
		}, http.StatusOK)

		By("creating a resource")
		res := &testResource{
			A: "asdf",
			B: 5,
		}
		client.Request("POST", "/api/test", client.JSONBuffer(res), nil, func(resp *http.Response) {
			res = loadResource(client, resp, res)
		}, http.StatusCreated)

		By("retrieving a resource")
		client.Request("GET", "/api/test/"+res.UUID.String(), nil, nil, func(resp *http.Response) {
			loadResource(client, resp, res)
		}, http.StatusOK)

		By("updating a resource")
		res.A += "zxcvbn"
		client.Request("PUT", "/api/test/"+res.UUID.String(), client.JSONBuffer(res), nil, func(resp *http.Response) {
			loadResource(client, resp, res)
		}, http.StatusOK)

		By("retrieving a resource")
		client.Request("GET", "/api/test/"+res.UUID.String(), nil, nil, func(resp *http.Response) {
			loadResource(client, resp, res)
		}, http.StatusOK)

		By("deleting a resource")
		client.Request("DELETE", "/api/test/"+res.UUID.String(), nil, nil, nil, http.StatusNoContent)

		By("attempting to retrieve a resource")
		client.Request("GET", "/api/test/"+res.UUID.String(), nil, nil, nil, http.StatusNotFound)

	})
})

func loadResource(client *abtest.TestClient, resp *http.Response, res *testResource) *testResource {
	loadedRes := &testResource{}
	client.AssertJSON(resp, loadedRes, PointTo(MatchAllFields(Fields{
		"UUID":    Not(BeZero()),
		"A":       Equal(res.A),
		"B":       Equal(res.B),
		"Updated": BeTemporally("~", time.Now(), time.Second),
	})))

	return loadedRes
}

var _ = Describe("Resource List", func() {
	const expected = `{"items":["0","1","2","3","4","5","6","7","8","9"],"_links":{"asdf":[{"href":"foo"},{"href":"bar"},{"href":"baz"}],"curies":[{"name":"test","href":"http://example.com","templated":false}],"page next":[{"href":"/resource-list?page=3"}],"page previous":[{"href":"/resource-list?page=1"}]}}`
	rl := resource.ResourceList{
		Items:    []resource.Resource{"0", "1", "2", "3", "4", "5", "6", "7", "8", "9"},
		Page:     2,
		PageSize: 10,
		BasePath: "/resource-list",
		Curies: []hal.HALCurie{
			{Name: "test", Href: "http://example.com", Templated: false},
		},
		Rels: map[string][]interface{}{
			"asdf": []interface{}{"foo", "bar", "baz"},
		},
	}

	It("should serialize properly with HAL", func() {
		marshaled, merr := json.Marshal(rl)
		Expect(merr).NotTo(HaveOccurred())
		Expect(string(marshaled)).To(Equal(expected))
	})
})
