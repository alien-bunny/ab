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

package abtest

import (
	"net/http"
	"time"

	"github.com/alien-bunny/ab"
	"github.com/alien-bunny/ab/lib/config"
	"github.com/alien-bunny/ab/lib/server"
	"github.com/alien-bunny/ab/lib/util"
)

func Hop(setup SetupFunc) (string, func() *TestClient) {
	base, mock := NewTestServer().Start(setup)
	installSite(NewHTTPTestClient(base))
	mock()
	return base, func() *TestClient {
		return NewHTTPTestClientWithToken(base)
	}
}

func HopMock(setup SetupFunc) (string, func() *TestClient) {
	s := NewTestServer()
	srv, mock := s.Setup(setup)
	handler := srv.Handler()
	base := "http://" + s.Addr
	installSite(NewMockTestClient(base, handler))
	mock()

	return base, func() *TestClient {
		return NewMockTestClientWithToken(base, handler)
	}
}

// TestServer is a temporary Server for integration tests.
type TestServer struct {
	Addr string
}

func NewTestServer() *TestServer {
	return &TestServer{
		Addr: util.TestServerAddress(),
	}
}

// Setup sets up a mock server.
//
// Returns a server and a mocker function. Run the mocker function after installing the schemas.
func (s *TestServer) Setup(setup SetupFunc) (*server.Server, func()) {
	logger := GetLogger()
	conf := GetConfig(logger, s.Addr)
	srv, err := ab.Pet(conf, config.Default, logger)
	if err != nil {
		panic(err)
	}

	smw := NewSchemaMiddleware()
	srv.Use(smw)

	var mock DataMockerFunc
	if setup != nil {
		mock, err = setup(conf, srv, "http://"+s.Addr, smw.GetSchemaName())
		if err != nil {
			panic(err)
		}
	}

	return srv, func() {
		if mock != nil {
			conn := Connect(smw.GetSchemaName())
			if err := mock(conn); err != nil {
				panic(err)
			}
		}
	}
}

// Start starts a Server with test-optimized settings.
func (s *TestServer) Start(setup SetupFunc) (string, func()) {
	srv, mock := s.Setup(setup)
	go srv.StartHTTP(s.Addr)
	<-time.After(time.Second)
	return "http://" + s.Addr, mock
}

func installSite(c *TestClient) {
	c.panic = true
	c.Request("GET", "/install?key="+FakeAdminKey, nil, nil, nil, http.StatusNoContent)
}
