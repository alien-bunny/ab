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

package translationmw_test

import (
	"encoding/hex"
	"net/http"
	"net/url"
	"time"

	"github.com/alien-bunny/ab/lib/abtest"
	"github.com/alien-bunny/ab/lib/middleware"
	"github.com/alien-bunny/ab/middlewares/logmw"
	"github.com/alien-bunny/ab/middlewares/sessionmw"
	"github.com/alien-bunny/ab/middlewares/translationmw"
	. "github.com/onsi/ginkgo"
	. "github.com/onsi/gomega"
	"golang.org/x/text/language"
)

func newStack(requestManipulator func(r *http.Request), negotiators ...translationmw.LanguageNegotiator) *middleware.Stack {
	logger, conf, cmw := abtest.SetupConfigMiddleware()

	sessmw := sessionmw.New("", time.Hour)
	conf.MaybeRegisterSchema(sessmw)
	_, cfgsaver, _ := conf.GetWritable("test").GetWritable("session")
	cfgsaver.Save(sessionmw.Config{
		Key:       hex.EncodeToString(abtest.FakeKey),
		CookieURL: "https://test/",
	})

	trmw := translationmw.New(logger, []language.Tag{language.Hungarian, language.English}, negotiators...)

	stack := middleware.NewStack(nil)
	stack.Push(cmw)
	stack.Push(logmw.New(logger))
	stack.Push(sessmw)
	stack.Push(middleware.Func(func(next http.Handler) http.Handler {
		return http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
			if requestManipulator != nil {
				requestManipulator(r)
			}

			next.ServeHTTP(w, r)
		})
	}))
	stack.Push(trmw)

	return stack
}

func defaultStack(requestManipulator func(r *http.Request)) *middleware.Stack {
	return newStack(
		requestManipulator,
		translationmw.URLParamLanguage("lang"),
		translationmw.SessionLanguage{},
		translationmw.CookieLanguage("language"),
		translationmw.AcceptLanguage{},
		translationmw.StaticDefaultLanguage(language.Hungarian),
	)
}

var _ = Describe("URL parameter language", func() {
	stack := defaultStack(func(r *http.Request) {
		r.URL, _ = url.Parse(r.URL.String() + "?lang=en")
	})
	It("should return the language specified in the url", func() {
		abtest.TestMiddleware(stack, func(w http.ResponseWriter, r *http.Request) {
			Expect(translationmw.GetLanguage(r).String()).To(Equal("en"))
		})
	})
})

var _ = Describe("Session language", func() {
	stack := defaultStack(func(r *http.Request) {
		translationmw.SetSessionLanguage(r, language.English)
	})
	It("should return the language specified in the session", func() {
		abtest.TestMiddleware(stack, func(w http.ResponseWriter, r *http.Request) {
			Expect(translationmw.GetLanguage(r).String()).To(Equal("en"))
		})
	})
})

var _ = Describe("Cookie language", func() {
	stack := defaultStack(func(r *http.Request) {
		r.AddCookie(&http.Cookie{
			Name:  "language",
			Value: "en",
		})
	})
	It("should return the language specified in the cookie", func() {
		abtest.TestMiddleware(stack, func(w http.ResponseWriter, r *http.Request) {
			Expect(translationmw.GetLanguage(r).String()).To(Equal("en"))
		})
	})
})

var _ = Describe("Correct Accept-Language", func() {
	stack := defaultStack(func(r *http.Request) {
		r.Header.Set("Accept-Language", "en")
	})
	It("should return the language specified in the header", func() {
		abtest.TestMiddleware(stack, func(w http.ResponseWriter, r *http.Request) {
			Expect(translationmw.GetLanguage(r).String()).To(Equal("en"))
		})
	})
})

var _ = Describe("Incorrect Accept-Language", func() {
	stack := defaultStack(func(r *http.Request) {
		r.Header.Set("Accept-Language", "asdf")
	})
	It("should return the language specified in the header", func() {
		abtest.TestMiddleware(stack, func(w http.ResponseWriter, r *http.Request) {
			Expect(translationmw.GetLanguage(r).String()).To(Equal("hu"))
		})
	})
})

var _ = Describe("Default language", func() {
	stack := defaultStack(nil)
	It("should return the default langauge when no information is provided", func() {
		abtest.TestMiddleware(stack, func(w http.ResponseWriter, r *http.Request) {
			Expect(translationmw.GetLanguage(r).String()).To(Equal("hu"))
		})
	})
})

var _ = Describe("No negotiators", func() {
	stack := newStack(nil)
	It("should return english when no language negotiation happens", func() {
		abtest.TestMiddleware(stack, func(w http.ResponseWriter, r *http.Request) {
			Expect(translationmw.GetLanguage(r).String()).To(Equal("en"))
		})
	})
})
