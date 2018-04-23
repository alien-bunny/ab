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

package middleware_test

import (
	"net/http"
	"net/http/httptest"

	"github.com/alien-bunny/ab/lib/middleware"
	"github.com/alien-bunny/ab/lib/server"

	. "github.com/onsi/ginkgo"
	. "github.com/onsi/gomega"
)

type testMiddlewareA struct {
	middleware.NoDependencies
}

func (t *testMiddlewareA) Wrap(next http.Handler) http.Handler {
	return http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		next.ServeHTTP(w, r)
	})
}

type testMiddlewareB struct{}

func (t *testMiddlewareB) Wrap(next http.Handler) http.Handler {
	return http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		next.ServeHTTP(w, r)
	})
}

func (t *testMiddlewareB) Dependencies() []string {
	return []string{"*middleware_test.testMiddlewareA"}
}

type testHandler struct{}

func (t *testHandler) ServeHTTP(w http.ResponseWriter, r *http.Request) {}

var _ = Describe("Middleware", func() {
	Describe("A function wrapped in Func", func() {
		wrapped := false
		mf := middleware.Func(func(next http.Handler) http.Handler {
			wrapped = true
			return http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
				next.ServeHTTP(w, r)
			})
		})
		mf.Wrap(nil)

		It("should have empty dependencies", func() {
			Expect(mf.Dependencies()).To(BeEmpty())
		})

		It("should wrap a handler", func() {
			Expect(wrapped).To(BeTrue())
		})
	})

	Describe("A wrapped handler", func() {
		h := &testHandler{}
		wh := middleware.WrapHandler(h, "asdf")

		whd, hok := wh.(middleware.HasMiddlewareDependencies)
		uwh, uok := wh.(server.HandlerUnwrapper)

		It("should have dependencies", func() {
			Expect(hok).To(BeTrue())
			Expect(whd.Dependencies()).To(Equal([]string{"asdf"}))
		})

		It("should unwrap the same handler", func() {
			Expect(uok).To(BeTrue())
			Expect(uwh.Unwrap()).To(Equal(h))
		})

		served := false
		middleware.WrapHandlerFunc(func(w http.ResponseWriter, r *http.Request) {
			served = true
		}).ServeHTTP(nil, nil)

		It("should proxy ServeHTTP to the wrapped handler", func() {
			Expect(served).To(BeTrue())
		})
	})

	Describe("An empty Stack", func() {
		ms := middleware.NewStack(nil)

		It("should not allow pushing a middleware with a dependency", func() {
			merr := ms.Push(&testMiddlewareB{})
			Expect(merr).To(HaveOccurred())
		})

		It("should not allow shifting a middleware with a dependency", func() {
			merr := ms.Shift(&testMiddlewareB{})
			Expect(merr).To(HaveOccurred())
		})

		It("should not validate a handler with a dependency", func() {
			merr := ms.ValidateHandler(middleware.WrapHandlerFunc(nil, "asdf"))
			Expect(merr).To(HaveOccurred())
		})

		It("should validate a handler without any dependencies", func() {
			merr := ms.ValidateHandler(http.HandlerFunc(nil))
			Expect(merr).NotTo(HaveOccurred())
		})
	})

	Describe("A Stack", func() {
		ms := middleware.NewStack(nil)

		It("should accept a middleware", func() {
			merr := ms.Push(&testMiddlewareA{})
			Expect(merr).NotTo(HaveOccurred())
		})

		It("should allow pushing a middleware when its dependencies are met", func() {
			merr := ms.Push(&testMiddlewareB{})
			Expect(merr).NotTo(HaveOccurred())
		})

		It("should allow shifting a middleware when its dependencies are met", func() {
			merr := ms.Shift(&testMiddlewareB{})
			Expect(merr).NotTo(HaveOccurred())
		})

		It("should validate a handler when its dependencies are met", func() {
			merr := ms.ValidateHandler(middleware.WrapHandlerFunc(nil, "*middleware_test.testMiddlewareA"))
			Expect(merr).NotTo(HaveOccurred())
		})
	})

	Describe("A Stack with a parent", func() {
		pms := middleware.NewStack(nil)
		pms.Push(&testMiddlewareA{})
		ms := middleware.NewStack(pms)

		It("should allow pushing a middleware when its dependencies are met in the parent", func() {
			merr := ms.Push(&testMiddlewareB{})
			Expect(merr).NotTo(HaveOccurred())
		})

		It("should allow shifting a middleware when its dependencies are met in the parent", func() {
			merr := ms.Shift(&testMiddlewareB{})
			Expect(merr).NotTo(HaveOccurred())
		})

		It("should collect the dependencies from its parent as well", func() {
			merr := ms.ValidateHandler(middleware.WrapHandlerFunc(nil, "asdf"))
			Expect(merr).To(HaveOccurred())

			derr, ok := merr.(middleware.DependencyError)
			Expect(ok).To(BeTrue())

			Expect(derr.NotFound).To(Equal("asdf"))
			Expect(derr.Provided).To(ContainElement("*middleware_test.testMiddlewareA"))
		})
	})

	Describe("A Stack with middlewares", func() {
		rr := httptest.NewRecorder()
		mwFirst := byteMiddleware(1)
		mwSecond := byteMiddleware(2)
		mwThird := byteMiddleware(3)

		ms := middleware.NewStack(nil)

		ms.Push(mwSecond)
		ms.Push(mwThird)
		ms.Shift(mwFirst)

		handler := http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
			w.Write([]byte{4})
		})

		mwh := ms.Wrap(handler)

		mwh.ServeHTTP(rr, nil)

		It("should execute the middlewares in the correct order", func() {
			Expect(rr.Body.Bytes()).To(Equal([]byte{1, 2, 3, 4}))
		})
	})

	Describe("A DependencyError", func() {
		derr := middleware.DependencyError{
			NotFound: "asdf",
		}

		It("should contain the middleware name in its error message", func() {
			Expect(derr.Error()).To(ContainSubstring(derr.NotFound))
		})
	})
})

func byteMiddleware(b byte) middleware.Middleware {
	return middleware.Func(func(next http.Handler) http.Handler {
		return http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
			w.Write([]byte{b})
			next.ServeHTTP(w, r)
		})
	})
}
