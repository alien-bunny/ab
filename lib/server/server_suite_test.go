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

package server_test

import (
	"net/http"
	"testing"

	"github.com/alien-bunny/ab/lib/abtest"
	"github.com/alien-bunny/ab/lib/middleware"
	"github.com/alien-bunny/ab/lib/server"
	"github.com/alien-bunny/ab/lib/util"
	. "github.com/onsi/ginkgo"
	. "github.com/onsi/gomega"
)

func TestServer(t *testing.T) {
	RegisterFailHandler(Fail)
	setupServer()
	RunSpecs(t, "Server Suite")
}

var addr = util.TestServerAddress()

func setupServer() {
	logger := abtest.GetLogger()
	s := server.NewServer(nil, logger)
	s.SetMaster()
	s.UseF(testMiddleware)
	s.GetF("/context", contextHandler)
	s.GetF("/contextChanged", contextHandler, middleware.Func(testMiddlewareChanged))
	s.GetF("/echo/:param", echoParam)

	go func() {
		if err := s.StartHTTP(addr); err != nil {
			panic(err)
		}
	}()
}

const testmwkey = "test"

func testMiddleware(next http.Handler) http.Handler {
	return http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		r = util.SetContext(r, testmwkey, true)
		next.ServeHTTP(w, r)
	})
}

func testMiddlewareChanged(next http.Handler) http.Handler {
	return http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		r = util.SetContext(r, testmwkey, false)
		next.ServeHTTP(w, r)
	})
}

func contextHandler(w http.ResponseWriter, r *http.Request) {
	v := r.Context().Value(testmwkey)
	if v == nil {
		http.Error(w, "", http.StatusNotFound)
		return
	}

	vb, ok := v.(bool)
	if !ok {
		http.Error(w, "", http.StatusInternalServerError)
		return
	}

	w.Header().Set("Content-Type", "text/plain")

	if vb {
		w.Write([]byte("true"))
	} else {
		w.Write([]byte("false"))
	}
}

func echoParam(w http.ResponseWriter, r *http.Request) {
	p := server.GetParams(r).ByName("param")
	w.Write([]byte(p))
}
