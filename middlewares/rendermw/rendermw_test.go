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

package rendermw_test

import (
	"net/http"

	"github.com/alien-bunny/ab/lib/abtest"
	"github.com/alien-bunny/ab/lib/middleware"
	"github.com/alien-bunny/ab/lib/util"
	"github.com/alien-bunny/ab/middlewares/rendermw"
	. "github.com/onsi/ginkgo"
	. "github.com/onsi/gomega"
)

var _ = Describe("Render Middleware", func() {
	stack := middleware.NewStack(nil)
	stack.Push(rendermw.New())

	It("should render content", func() {
		msg := util.RandomString(64)
		w := abtest.TestMiddleware(stack, func(w http.ResponseWriter, r *http.Request) {
			rendermw.Render(r).Text(msg)
		})
		body := string(w.Body.String())
		Expect(body).To(Equal(msg))
	})

	It("should be compatible with http.ResponseWriter", func() {
		msg := util.RandomString(32)
		w := abtest.TestMiddleware(stack, func(w http.ResponseWriter, r *http.Request) {
			w.Header().Set("Content-Type", "text/plain")
			w.WriteHeader(http.StatusTeapot)
			w.Write([]byte(msg))
		})
		Expect(w.Code).To(Equal(http.StatusTeapot))
		body := string(w.Body.Bytes())
		Expect(body).To(Equal(msg))
	})
})
