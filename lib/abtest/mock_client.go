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
	"net/http/httptest"
	"net/url"

	"golang.org/x/net/publicsuffix"
)

var _ TestClientDelegate = &MockDelegate{}

type MockDelegate struct {
	base    string
	handler http.Handler
	jar     *cookiejar.Jar
}

func NewMockDelegate(base string, handler http.Handler) *MockDelegate {
	jar, _ := cookiejar.New(&cookiejar.Options{
		PublicSuffixList: publicsuffix.List,
	})
	return &MockDelegate{
		base:    base,
		handler: handler,
		jar:     jar,
	}
}

func (c *MockDelegate) Do(r *http.Request) (*http.Response, error) {
	w := httptest.NewRecorder()
	c.handler.ServeHTTP(w, r)
	resp := w.Result()

	c.jar.SetCookies(r.URL, resp.Cookies())

	return resp, nil
}

func (c *MockDelegate) NewRequest(method, target string, body io.Reader) *http.Request {
	req := httptest.NewRequest(method, target, body)

	u, _ := url.Parse(target)

	for _, cookie := range c.jar.Cookies(u) {
		req.AddCookie(cookie)
	}

	return req
}

func NewMockTestClient(base string, handler http.Handler) *TestClient {
	return &TestClient{
		Delegate: NewMockDelegate(base, handler),
		base:     base,
	}
}

func NewMockTestClientWithToken(base string, handler http.Handler) *TestClient {
	c := NewMockTestClient(base, handler)
	c.GetToken()
	return c
}
