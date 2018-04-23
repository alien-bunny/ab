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

package securitymw_test

import (
	"crypto/tls"
	"net/http"
	"net/http/httptest"
	"time"

	"golang.org/x/text/language"

	"github.com/alien-bunny/ab/lib/abtest"
	"github.com/alien-bunny/ab/lib/middleware"
	"github.com/alien-bunny/ab/lib/util"
	"github.com/alien-bunny/ab/middlewares/errormw"
	"github.com/alien-bunny/ab/middlewares/logmw"
	"github.com/alien-bunny/ab/middlewares/securitymw"
	"github.com/alien-bunny/ab/middlewares/sessionmw"
	"github.com/alien-bunny/ab/middlewares/translationmw"
	. "github.com/onsi/ginkgo"
	. "github.com/onsi/ginkgo/extensions/table"
	. "github.com/onsi/gomega"
)

var _ = Describe("CSRF Middlewares", func() {
	logger, conf, cmw := abtest.SetupConfigMiddleware()

	smw := sessionmw.New("", time.Hour)
	conf.MaybeRegisterSchema(smw)

	stack := middleware.NewStack(nil)
	stack.Push(cmw)
	stack.Push(logmw.New(logger))
	stack.Push(smw)
	stack.Push(translationmw.New(logger, []language.Tag{language.English}))
	stack.Push(errormw.New(true))
	stack.Push(securitymw.NewCSRFMiddleware())

	stackGet := middleware.NewStack(nil)
	stackGet.Push(cmw)
	stackGet.Push(logmw.New(logger))
	stackGet.Push(smw)
	stackGet.Push(translationmw.New(logger, []language.Tag{language.English}))
	stackGet.Push(errormw.New(true))
	stackGet.Push(securitymw.NewCSRFMiddleware())
	stackGet.Push(securitymw.NewCSRFGetMiddleware("token"))

	It("should reject requests with an invalid csrf token", func() {
		w := httptest.NewRecorder()
		r, reqerr := abtest.NewRequest("POST", "/", nil)
		Expect(reqerr).NotTo(HaveOccurred())
		msg := util.RandomString(64)
		stack.Wrap(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
			w.Header().Set("Content-Type", "text/plain")
			w.Write([]byte(msg))
		})).ServeHTTP(w, r)

		Expect(w.Code).To(Equal(http.StatusForbidden))
		body := string(w.Body.Bytes())
		Expect(body).NotTo(ContainSubstring(msg))
	})

	It("should generate a token", func() {
		w := abtest.TestMiddleware(stack, func(w http.ResponseWriter, r *http.Request) {
			token := securitymw.GetCSRFToken(r)
			w.Header().Set("Content-Type", "text/plain")
			w.Write([]byte(token))
		})

		Expect(w.Code).To(Equal(http.StatusOK))
		body := string(w.Body.Bytes())
		Expect(body).NotTo(BeZero())
	})

	It("should reject GET requests with an invalid csrf token in the url", func() {
		msg := util.RandomString(64)
		w := abtest.TestMiddleware(stackGet, func(w http.ResponseWriter, r *http.Request) {
			token := securitymw.GetCSRFToken(r)
			Expect(token).NotTo(BeZero())
			w.Header().Set("Content-Type", "text/plain")
			w.Write([]byte(msg))
		})

		Expect(w.Code).To(Equal(http.StatusForbidden))
		body := string(w.Body.Bytes())
		Expect(body).NotTo(ContainSubstring(msg))
	})

})

var _ = Describe("HSTS Middleware", func() {
	stack := middleware.NewStack(nil)
	stack.Push(&securitymw.HSTSMiddleware{
		MaxAge:            time.Hour,
		IncludeSubDomains: true,
	})

	It("should add the header to the response", func() {
		w := httptest.NewRecorder()
		r, reqerr := abtest.NewRequest("GET", "/", nil)
		Expect(reqerr).NotTo(HaveOccurred())
		r.TLS = &tls.ConnectionState{}
		stack.Wrap(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
			w.WriteHeader(http.StatusNoContent)
		})).ServeHTTP(w, r)

		header := w.Header().Get("Strict-Transport-Security")
		Expect(header).To(Equal("max-age=3600; includeSubDomains"))
	})
})

var _ = Describe("RestrictAddress Middleware", func() {
	stack := middleware.NewStack(nil)
	stack.Push(securitymw.NewRestrictPrivateAddressMiddleware())

	DescribeTable("the ip address blocking",
		func(ip string, code int) {
			Expect(testRestrictAddress(stack, ip)).To(Equal(code))
		},
		Entry("192.168.1.1", "192.168.1.1", http.StatusOK),
		Entry("8.8.4.4", "8.8.4.4", http.StatusServiceUnavailable),
	)
})

func testRestrictAddress(stack *middleware.Stack, ip string) int {
	w := httptest.NewRecorder()
	r, reqerr := abtest.NewRequest("GET", "/", nil)
	Expect(reqerr).NotTo(HaveOccurred())
	r.RemoteAddr = ip + ":12345"
	stack.Wrap(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
	})).ServeHTTP(w, r)

	return w.Code
}
