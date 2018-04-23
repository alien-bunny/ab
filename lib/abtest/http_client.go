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
	"io"
	"net/http"
	"net/http/cookiejar"

	"golang.org/x/net/publicsuffix"
)

type HTTPClientDelegate struct {
	*http.Client
}

func (d *HTTPClientDelegate) NewRequest(method, target string, body io.Reader) *http.Request {
	req, _ := http.NewRequest(method, target, body)
	return req
}

// NewHTTPTestClient initializes TestClient with *http.Client.
//
// The wrapped http.Client will get an empty cookie jar.
func NewHTTPTestClient(base string) *TestClient {
	c := &http.Client{}
	c.Jar, _ = cookiejar.New(&cookiejar.Options{
		PublicSuffixList: publicsuffix.List,
	})

	return &TestClient{
		Delegate: &HTTPClientDelegate{
			Client: c,
		},
		base: base,
	}
}

// NewHTTPTestClientWithToken initializes a TestClient and retrieves a CSRF token.
func NewHTTPTestClientWithToken(base string) *TestClient {
	c := NewHTTPTestClient(base)
	c.GetToken()
	return c
}
