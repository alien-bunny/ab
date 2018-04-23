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

package ab_test

import (
	"bytes"
	"io/ioutil"
	"net/http"
	"os"

	. "github.com/onsi/ginkgo"
	. "github.com/onsi/gomega"
)

var _ = Describe("Service CRUD", func() {
	It("should return all saved content", func() {
		c := clientFactory()
		data := []testDecode{
			{1, "a"},
			{2, "b"},
			{3, "c"},
			{4, "d"},
			{5, "e"},
		}

		By("saving all content")
		for _, d := range data {
			c.Request("POST", "/test", c.JSONBuffer(d), nil, nil, http.StatusCreated)
		}

		By("returning all content")
		c.Request("GET", "/test", nil, nil, func(resp *http.Response) {
			c.AssertJSON(resp, &[]testDecode{}, Equal(&data))
		}, http.StatusOK)
	})
})

var _ = Describe("Frontend path", func() {
	It("should return an index.html file", func() {
		_, perr := os.Stat("fixtures/index.html")
		Expect(perr).NotTo(HaveOccurred())

		c := clientFactory()

		c.Request("GET", "/frontend", nil, func(req *http.Request) {
			req.Header.Set("Accept", "text/html")
		}, func(resp *http.Response) {
			c.AssertFile(resp, "fixtures/index.html")
		}, http.StatusOK)
	})
})

var _ = Describe("Decoder", func() {
	It("should decode data from JSON endpoint", func() {
		c := clientFactory()

		By("failing on invalid data")
		buf := bytes.NewBuffer(nil)
		buf.WriteString("[<>?<<><]]]}}}}")
		c.Request("POST", "/decode", buf, nil, nil, http.StatusBadRequest)

		By("failing on invalid content type")
		c.Request("POST", "/decode", nil, func(req *http.Request) {
			req.Header.Set("Content-Type", "xxx/invalid")
		}, nil, http.StatusUnsupportedMediaType)

		By("returning the POST data")
		data := testDecode{
			A: 65536,
			B: "asdf",
		}
		c.Request("POST", "/decode", c.JSONBuffer(data), nil, func(resp *http.Response) {
			c.AssertJSON(resp, &testDecode{}, Equal(&data))
		}, http.StatusOK)
	})
})

var _ = Describe("Error", func() {
	It("should handle panic", func() {
		c := clientFactory()
		c.Request("GET", "/panic", nil, nil, nil, http.StatusInternalServerError)
	})

	It("should handle a failing endpoint", func() {
		c := clientFactory()
		c.Request("GET", "/fail", nil, nil, nil, http.StatusTeapot)
	})
})

var _ = Describe("Binary", func() {
	It("should retrieve the exact same data", func() {
		c := clientFactory()
		c.Request("GET", "/binary", nil, nil, func(resp *http.Response) {
			c.AssertFile(resp, "fixtures/binary.bin")
		}, http.StatusOK)
	})
})

var _ = Describe("Empty endpoint", func() {
	It("should return http.StatusNoContent", func() {
		c := clientFactory()
		c.Request("GET", "/empty", nil, nil, nil, http.StatusNoContent)
	})
})

var _ = Describe("Benchmarks", func() {
	Measure("empty endpoint", func(b Benchmarker) {
		c := clientFactory()
		runtime := b.Time("runtime", func() {
			c.Request("GET", "/empty", nil, nil, nil, http.StatusNoContent)
		})

		Expect(runtime.Seconds()).To(BeNumerically("<", 1))
	}, 1000)

	Measure("1k endpoint", func(b Benchmarker) {
		c := clientFactory()
		runtime := b.Time("runtime", func() {
			c.Request("GET", "/1k", nil, nil, func(resp *http.Response) {
				ioutil.ReadAll(resp.Body)
				resp.Body.Close()
			}, http.StatusOK)
		})

		Expect(runtime.Seconds()).To(BeNumerically("<", 2))
	}, 1000)
})
