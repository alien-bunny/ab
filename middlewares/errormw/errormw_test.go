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

package errormw_test

import (
	"net/http"

	"github.com/alien-bunny/ab/lib/abtest"
	"github.com/alien-bunny/ab/lib/errors"
	"github.com/alien-bunny/ab/lib/middleware"
	"github.com/alien-bunny/ab/lib/util"
	"github.com/alien-bunny/ab/middlewares/errormw"
	"github.com/alien-bunny/ab/middlewares/logmw"
	"github.com/alien-bunny/ab/middlewares/requestmw"
	"github.com/alien-bunny/ab/middlewares/translationmw"
	. "github.com/onsi/ginkgo"
	. "github.com/onsi/gomega"
	"golang.org/x/text/language"
)

var _ = Describe("Error Middleware", func() {
	logger, _, cmw := abtest.SetupConfigMiddleware()
	stack := middleware.NewStack(nil)
	stack.Push(requestmw.NewRequestIDMiddleware())
	stack.Push(logmw.New(logger))
	stack.Push(cmw)

	stack.Push(translationmw.New(logger, []language.Tag{language.English}))
	stack.Push(errormw.New(true))

	It("should recover a panic and reply with an internal server error", func() {
		w := abtest.TestMiddleware(stack, func(w http.ResponseWriter, r *http.Request) {
			panic("")
		})

		Expect(w.Code).To(Equal(http.StatusInternalServerError))
	})

	It("should display an error message", func() {
		msg := util.RandomString(16)
		vmsg := util.RandomString(32)
		w := abtest.TestMiddleware(stack, func(w http.ResponseWriter, r *http.Request) {
			errors.Fail(http.StatusNotFound, errors.NewError(msg, vmsg, nil))
		})

		Expect(w.Code).To(Equal(http.StatusNotFound))
		body := string(w.Body.Bytes())
		Expect(body).To(ContainSubstring(msg))
		Expect(body).NotTo(ContainSubstring(vmsg))
	})
})
