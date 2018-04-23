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

package configmw_test

import (
	"net/http"
	"reflect"

	"github.com/alien-bunny/ab/lib/abtest"
	"github.com/alien-bunny/ab/lib/middleware"
	"github.com/alien-bunny/ab/lib/util"
	"github.com/alien-bunny/ab/middlewares/configmw"
	"github.com/alien-bunny/ab/middlewares/logmw"
	. "github.com/onsi/ginkgo"
	. "github.com/onsi/gomega"
)

const customHeader = "X-Custom-Header"

var _ = Describe("Config middleware", func() {
	logger := abtest.GetLogger()
	conf := abtest.GetConfigForMiddleware(logger)

	It("should find the correct namespace based on host", func() {
		stack := middleware.NewStack(nil)
		stack.Push(configmw.NewConfigMiddleware(conf, configmw.NewHostNamespaceNegotiator()))

		abtest.TestMiddleware(stack, func(w http.ResponseWriter, r *http.Request) {
			cfg := configmw.GetWritableConfig(r)
			Expect(cfg).NotTo(BeNil())
		})
	})

	It("should find the correct namespace in a hostmap", func() {
		stack := middleware.NewStack(nil)
		negotiator := configmw.NewHostMapNamespaceNegotiator()
		negotiator.Add("testhost", "test")
		stack.Push(configmw.NewConfigMiddleware(conf, negotiator))

		abtest.TestMiddleware(stack, func(w http.ResponseWriter, r *http.Request) {
			cfg := configmw.GetConfig(r)
			Expect(cfg).NotTo(BeNil())
		})
	})

	It("should find the correct namespace with a chained negotiator", func() {
		stack := middleware.NewStack(nil)
		negotiator := configmw.NewHostMapNamespaceNegotiator()
		negotiator.Add("testhost", "notfound")
		stack.Push(configmw.NewConfigMiddleware(conf, configmw.NewChainedNamespaceNegotiator(
			negotiator,
			configmw.NewHostNamespaceNegotiator(),
		)))

		abtest.TestMiddleware(stack, func(w http.ResponseWriter, r *http.Request) {
			cfg := configmw.GetConfig(r)
			Expect(cfg).NotTo(BeNil())
		})
	})

})

var _ = Describe("Config-wrapped middleware", func() {
	logger := abtest.GetLogger()
	conf := abtest.GetConfigForMiddleware(logger)
	conf.RegisterSchema("testmw", reflect.TypeOf(testMiddleware{}))
	_, saver, _ := conf.GetWritable("test").GetWritable("testmw")
	value := util.RandomString(12)
	saver.Save(testMiddleware{
		customHeaderValue: value,
	})

	It("should unwrap the middleware properly", func() {
		stack := middleware.NewStack(nil)
		stack.Push(logmw.New(logger))
		stack.Push(configmw.NewConfigMiddleware(conf, configmw.NewHostNamespaceNegotiator()))
		stack.Push(configmw.WrapMiddleware("testmw", reflect.TypeOf(testMiddleware{})))

		abtest.TestMiddleware(stack, func(w http.ResponseWriter, r *http.Request) {
			Expect(w.Header().Get(customHeader)).To(Equal(value))
		})
	})
})

type testMiddleware struct {
	customHeaderValue string

	middleware.NoDependencies
}

func (m testMiddleware) Wrap(next http.Handler) http.Handler {
	return http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		w.Header().Set(customHeader, m.customHeaderValue)
		next.ServeHTTP(w, r)
	})
}
