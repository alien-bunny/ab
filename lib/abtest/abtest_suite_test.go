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

package abtest_test

import (
	"net/http"
	"testing"

	"github.com/alien-bunny/ab/lib/abtest"
	"github.com/alien-bunny/ab/lib/config"
	"github.com/alien-bunny/ab/lib/server"
	"github.com/alien-bunny/ab/middlewares/rendermw"
	. "github.com/onsi/ginkgo"
	. "github.com/onsi/gomega"
)

type testdata struct {
	A int
	B string
}

var posttest = testdata{
	A: 5,
	B: "asdf",
}

var configServer = func(conf *config.Store, s *server.Server, base, schema string) (abtest.DataMockerFunc, error) {
	s.PostF("/api/posttest", func(w http.ResponseWriter, r *http.Request) {
		rendermw.Render(r).JSON(posttest)
	})

	return nil, nil
}

var _, clientFactory = abtest.Hop(configServer)

var _, mockClientFactory = abtest.HopMock(configServer)

func TestTesting(t *testing.T) {
	RegisterFailHandler(Fail)
	RunSpecs(t, "Testing Suite")
}
