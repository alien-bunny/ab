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

package requestmw_test

import (
	"bytes"
	"net/http"
	"strconv"

	"github.com/alien-bunny/ab/lib/abtest"
	"github.com/alien-bunny/ab/lib/log"
	"github.com/alien-bunny/ab/lib/middleware"
	"github.com/alien-bunny/ab/middlewares/requestmw"
	"github.com/go-kit/kit/log/level"
	. "github.com/onsi/ginkgo"
	. "github.com/onsi/gomega"
)

var _ = Describe("RequestID Middleware", func() {
	stack := middleware.NewStack(nil)
	stack.Push(requestmw.NewRequestIDMiddleware())

	It("should generate a request id", func() {
		w := abtest.TestMiddleware(stack, func(w http.ResponseWriter, r *http.Request) {
			id := requestmw.GetRequestID(r)
			w.Header().Set("Content-Type", "text/plain")
			w.Write([]byte(id))
		})

		body := string(w.Body.Bytes())
		Expect(body).NotTo(BeZero())
	})
})

var _ = Describe("RequestLogger Middleware", func() {
	lw := bytes.NewBuffer(nil)
	logger := log.NewDevLogger(lw, level.AllowAll())
	stack := middleware.NewStack(nil)
	stack.Push(requestmw.NewRequestLoggerMiddleware(logger))

	It("should log the request", func() {
		abtest.TestMiddleware(stack, func(w http.ResponseWriter, r *http.Request) {
			w.WriteHeader(http.StatusTeapot)
		})

		logs := string(lw.Bytes())
		Expect(logs).To(ContainSubstring("GET"))
		Expect(logs).To(ContainSubstring("/"))
		Expect(logs).To(ContainSubstring(strconv.Itoa(http.StatusTeapot)))
	})
})
