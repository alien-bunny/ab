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

package decoder_test

import (
	"bytes"
	"net/http"
	"reflect"

	"github.com/alien-bunny/ab/lib/decoder"

	. "github.com/onsi/ginkgo"
	. "github.com/onsi/ginkgo/extensions/table"
	. "github.com/onsi/gomega"
)

type testData struct {
	A int
	B string
}

var _ = Describe("Decoder", func() {
	DescribeTable("Decoder example tables",
		func(encoded []byte, mimetype string, data interface{}) {
			req, rerr := http.NewRequest("GET", "/", bytes.NewReader(encoded))
			Expect(rerr).To(BeNil())
			req.Header.Set("Content-Type", mimetype)

			decoded := reflect.New(reflect.TypeOf(data).Elem()).Interface()
			decoder.MustDecode(req, decoded)
			Expect(decoded).To(Equal(data))
		},
		Entry("simple json", []byte(`{"A": 5, "B": "asdf"}`), "application/json", &testData{A: 5, B: "asdf"}),
		Entry("simple xml", []byte(`<testData><A>5</A><B>asdf</B></testData>`), "application/xml", &testData{A: 5, B: "asdf"}),
		Entry("simple csv", []byte("a,b,c,d\ne,f,g,h"), "text/csv", &[][]string{
			[]string{"a", "b", "c", "d"},
			[]string{"e", "f", "g", "h"},
		}),
	)

	It("Panics on invalid content type", func() {
		req, rerr := http.NewRequest("GET", "/", bytes.NewReader([]byte{}))
		Expect(rerr).To(BeNil())
		req.Header.Set("Content-Type", "misc/xxx-undefined")
		Expect(func() {
			data := map[string]string{}
			decoder.MustDecode(req, &data)
		}).To(Panic())
	})
})
