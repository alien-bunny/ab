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

package server_test

import (
	"net/http"

	"github.com/alien-bunny/ab/lib/abtest"
	"github.com/alien-bunny/ab/lib/util"
	. "github.com/onsi/ginkgo"
	. "github.com/onsi/gomega"
)

var _ = Describe("Server", func() {
	baseurl := "http://" + addr
	c := abtest.NewHTTPTestClient(baseurl)

	It("should be able to retrieve data from context", func() {
		c.Request("GET", "/context", nil, nil, func(resp *http.Response) {
			body := c.ReadBody(resp, false)
			Expect(body).To(Equal("true"))
		}, http.StatusOK)

		c.Request("GET", "/contextChanged", nil, nil, func(resp *http.Response) {
			body := c.ReadBody(resp, false)
			Expect(body).To(Equal("false"))
		}, http.StatusOK)
	})

	It("should be able to retrieve a parameter", func() {
		random := util.RandomString(16)
		c.Request("GET", "/echo/"+random, nil, nil, func(resp *http.Response) {
			body := c.ReadBody(resp, false)
			Expect(body).To(Equal(random))
		}, http.StatusOK)
	})

})
