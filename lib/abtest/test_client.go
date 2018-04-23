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
	"bytes"
	"encoding/json"
	"fmt"
	"io"
	"io/ioutil"
	"net/http"

	. "github.com/onsi/gomega"
	"github.com/onsi/gomega/types"
)

type TestClientDelegate interface {
	Do(*http.Request) (*http.Response, error)
	NewRequest(method, target string, body io.Reader) *http.Request
}

// TestClient is a wrapper on the top of http.Client for integration tests.
type TestClient struct {
	Delegate TestClientDelegate
	Token    string
	panic    bool
	base     string
}

// Request sends a request to a TestServer.
//
// The method and endpoint parameters are mandatory. The body can be nil if the request does not have a body. The prcessReq function can modify the request, but it can be nil. The processResp function can deal with the response, but it can be nil as well. The statusCode parameter is the expected status code.
func (tc *TestClient) Request(method, endpoint string, body io.Reader, processReq func(*http.Request), processResp func(*http.Response), statusCode int) {
	req := tc.Delegate.NewRequest(method, tc.base+endpoint, body)
	req.Header.Set("Accept", "application/json")
	if body != nil {
		req.Header.Set("Content-Type", "application/json")
	}
	if tc.Token != "" {
		req.Header.Set("X-CSRF-Token", tc.Token)
	}
	if processReq != nil {
		processReq(req)
	}

	resp, err := tc.Delegate.Do(req)
	if tc.panic {
		if err != nil {
			panic(err)
		}
	} else {
		Expect(err).To(BeNil())
	}
	defer resp.Body.Close()

	ended := false
	defer func() {
		if !ended {
			fmt.Println("")
			fmt.Printf("%s %s\n", method, endpoint)
			fmt.Println("")
			if resp.Header.Get("Content-Type") == "application/json" {
				errorData := make(map[string]string)
				tc.ConsumePrefix(resp)
				json.NewDecoder(resp.Body).Decode(&errorData)
				resp.Body.Close()
				fmt.Printf("RequestID: %s\nMessage: %s\nLogs:\n%s\n", errorData["requestid"], errorData["message"], errorData["logs"])
			} else {
				fmt.Println(tc.ReadBody(resp, false))
			}
			fmt.Println("")
		}
	}()

	if tc.panic {
		if statusCode != resp.StatusCode {
			panic(fmt.Sprintf("status codes don't match! expected %d got %d", statusCode, resp.StatusCode))
		}
	} else {
		Expect(statusCode).To(Equal(resp.StatusCode))
	}
	if processResp != nil {
		processResp(resp)
	}

	ended = true
}

// JSONBuffer creates an in-memory buffer of a serialized JSON value.
func (tc *TestClient) JSONBuffer(v interface{}) io.Reader {
	buf := bytes.NewBuffer(nil)
	Expect(json.NewEncoder(buf).Encode(v)).To(BeNil())
	return buf
}

// AssertJSON decodes the JSON body of the response into v, and matches it with matcher.
//
// Example:
//
//		c.Request("GET", "/api/endpoint", nil, nil, func(r *http.Response) {
//			data := &dataType{}
//			c.AssertJSON(resp, data, PointTo(MatchAllFields(Fields{
//				"Fields": Not(BeZero()),
//			})))
//		}, http.StatusOK)
func (tc *TestClient) AssertJSON(resp *http.Response, v interface{}, matcher types.GomegaMatcher) {
	tc.ConsumePrefix(resp)
	Expect(json.NewDecoder(resp.Body).Decode(v)).To(BeNil())
	Expect(v).To(matcher)
}

// AssertFile asserts that the response body is equal to a file.
func (tc *TestClient) AssertFile(resp *http.Response, path string) {
	Expect(resp.StatusCode).To(Equal(http.StatusOK))

	body, err := ioutil.ReadAll(resp.Body)
	Expect(err).To(BeNil())
	defer resp.Body.Close()

	file, err := ioutil.ReadFile(path)
	Expect(err).To(BeNil())

	Expect(body).To(Equal(file))
}

// GetToken retrieves the token from the TestServer for TestClient.
func (tc *TestClient) GetToken() {
	tc.Request("GET", "/api/token", nil, func(req *http.Request) {
		req.Header.Set("Accept", "text/plain")
	}, func(resp *http.Response) {
		token := tc.ReadBody(resp, false)
		Expect(token).NotTo(BeZero())

		tc.Token = token
	}, http.StatusOK)
}

// ConsumePrefix consumes the JSONPrefix from the response body.
func (tc *TestClient) ConsumePrefix(r *http.Response) bool {
	prefix := make([]byte, 6)
	_, err := io.ReadFull(r.Body, prefix)
	if tc.panic {
		if err != nil {
			panic(err)
		}
	} else {
		Expect(err).To(BeNil())
	}
	return string(prefix) == ")]}',\n"
}

// ReadBody reads the response body into a string.
func (tc *TestClient) ReadBody(r *http.Response, JSONPrefix bool) string {
	if JSONPrefix {
		Expect(tc.ConsumePrefix(r)).To(BeTrue())
	}

	b, err := ioutil.ReadAll(r.Body)
	if tc.panic {
		if err != nil {
			panic(err)
		}
	} else {
		Expect(err).To(BeNil())
	}

	return string(b)
}
