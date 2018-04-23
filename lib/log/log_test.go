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

package log_test

import (
	"bytes"
	"io"

	"github.com/alien-bunny/ab/lib/log"
	"github.com/alien-bunny/ab/lib/util"
	"github.com/go-kit/kit/log/level"
	. "github.com/onsi/ginkgo"
	. "github.com/onsi/gomega"
)

type customValue string

func (s customValue) Format(w io.Writer) {
	w.Write([]byte(s))
}

var _ = Describe("Development logger encoder", func() {
	buf := bytes.NewBuffer(nil)
	logger := log.NewDevLogger(buf, level.AllowAll())
	msg := util.RandomString(64)
	msgCustom := customValue(util.RandomString(64))
	err := logger.Log(
		"msg", msg,
		"msgCustom", msgCustom,
	)

	It("should only display the key if the value is not a LogValueFormatter", func() {
		Expect(err).NotTo(HaveOccurred())
		output := string(buf.Bytes())
		Expect(output).To(ContainSubstring("msg"))
		Expect(output).To(ContainSubstring(msg))
		Expect(output).NotTo(ContainSubstring("msgCustom"))
		Expect(output).To(ContainSubstring(string(msgCustom)))
	})
})

type logStruct struct {
	A int
	B string
}

const serialized = `struct="{A:5 B:asdf}" map="map[string]string{\"qwer\":\"zxcv\"}" array="[]int{1, 2, 3, 4, 5}"
`

var _ = Describe("Logfmt logger encoder", func() {
	buf := bytes.NewBuffer(nil)
	logger := log.NewProdLogger(buf, level.AllowAll())
	err := logger.Log(
		"struct", logStruct{5, "asdf"},
		"map", map[string]string{"qwer": "zxcv"},
		"array", []int{1, 2, 3, 4, 5},
	)

	It("should log arrays, maps and structs correctly", func() {
		Expect(err).NotTo(HaveOccurred())
		Expect(string(buf.Bytes())).To(Equal(serialized))
	})
})
