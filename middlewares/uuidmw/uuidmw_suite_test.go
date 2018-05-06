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

package uuidmw_test

import (
	"crypto/rand"
	"net/http"
	"testing"

	"github.com/alien-bunny/ab/lib/abtest"
	"github.com/alien-bunny/ab/lib/config"
	"github.com/alien-bunny/ab/lib/event"
	"github.com/alien-bunny/ab/lib/server"
	"github.com/alien-bunny/ab/middlewares/rendermw"
	"github.com/alien-bunny/ab/middlewares/uuidmw"
	. "github.com/onsi/ginkgo"
	. "github.com/onsi/gomega"
)

var key = genKey()

var base, clientFactory = abtest.HopMock(func(conf *config.Store, s *server.Server, dispatcher *event.Dispatcher, base, schema string) (abtest.DataMockerFunc, error) {
	s.GetF("/test/:uuid", func(w http.ResponseWriter, r *http.Request) {
		rendermw.Render(r).Text(server.GetParams(r).ByName("uuid"))
	}, uuidmw.New(key, false, "uuid"))

	s.GetF("/notstrict", func(w http.ResponseWriter, r *http.Request) {
	}, uuidmw.New(key, false, "uuid"))

	s.GetF("/strict", func(w http.ResponseWriter, r *http.Request) {
	}, uuidmw.New(key, true, "uuid"))

	return nil, nil
})

func TestUUIDmw(t *testing.T) {
	RegisterFailHandler(Fail)
	RunSpecs(t, "UUID middleware Suite")
}

func genKey() []byte {
	k := make([]byte, 64)
	rand.Read(k)
	return k
}
