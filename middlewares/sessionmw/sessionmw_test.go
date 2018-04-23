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

package sessionmw_test

import (
	"encoding/hex"
	"net/http"
	"net/http/cookiejar"
	"net/http/httptest"
	"net/url"
	"time"

	"github.com/alien-bunny/ab/lib/abtest"
	"github.com/alien-bunny/ab/lib/middleware"
	"github.com/alien-bunny/ab/lib/util"
	"github.com/alien-bunny/ab/middlewares/logmw"
	"github.com/alien-bunny/ab/middlewares/sessionmw"
	. "github.com/onsi/ginkgo"
	. "github.com/onsi/gomega"
	"golang.org/x/net/publicsuffix"
)

var _ = Describe("Session Middleware", func() {
	logger, conf, cmw := abtest.SetupConfigMiddleware()

	smw := sessionmw.New("", time.Hour)
	conf.MaybeRegisterSchema(smw)
	_, saver, _ := conf.GetWritable("test").GetWritable("session")
	saver.Save(sessionmw.Config{
		Key: hex.EncodeToString(abtest.FakeKey),
	})

	stack := middleware.NewStack(nil)
	stack.Push(cmw)
	stack.Push(logmw.New(logger))
	stack.Push(smw)

	jar, _ := cookiejar.New(&cookiejar.Options{
		PublicSuffixList: publicsuffix.List,
	})

	It("should store and retrieve data", func() {
		var w *httptest.ResponseRecorder
		data := util.RandomString(16)

		By("storing data in session")
		w = request(stack, jar, func(w http.ResponseWriter, r *http.Request) {
			sess := sessionmw.GetSession(r)
			sess["data"] = data
		})
		Expect(w.Code).To(Equal(http.StatusOK))

		By("retrieving data from session")
		w = request(stack, jar, func(w http.ResponseWriter, r *http.Request) {
			sess := sessionmw.GetSession(r)
			w.Header().Set("Content-Type", "text/plain")
			w.Write([]byte(sess["data"]))
		})

		Expect(w.Code).To(Equal(http.StatusOK))
		body := string(w.Body.Bytes())
		Expect(body).To(Equal(data))
	})
})

func request(stack *middleware.Stack, jar *cookiejar.Jar, handler http.HandlerFunc) *httptest.ResponseRecorder {
	w := httptest.NewRecorder()

	r, reqerr := abtest.NewRequest("GET", "/", nil)
	Expect(reqerr).NotTo(HaveOccurred())

	u, uerr := url.Parse("http://test")
	Expect(uerr).NotTo(HaveOccurred())

	for _, cookie := range jar.Cookies(u) {
		r.AddCookie(cookie)
	}

	stack.Wrap(handler).ServeHTTP(w, r)

	jar.SetCookies(u, (&http.Response{Header: w.Header()}).Cookies())

	return w
}
