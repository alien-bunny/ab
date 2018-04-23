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

package requestmw

import (
	"crypto/rand"
	"encoding/hex"
	"net/http"

	"github.com/alien-bunny/ab/lib/middleware"
	"github.com/alien-bunny/ab/lib/util"
)

const (
	MiddlewareDependencyRequestID = "*requestmw.RequestIDMiddleware"
	reqIDKey                      = "abreqid"
)

var _ middleware.Middleware = &RequestIDMiddleware{}

// RequestIDMiddleware generates a request id for every request. Useful for logging.
type RequestIDMiddleware struct {
	middleware.NoDependencies
}

func NewRequestIDMiddleware() *RequestIDMiddleware {
	return &RequestIDMiddleware{}
}

func (rid *RequestIDMiddleware) Wrap(next http.Handler) http.Handler {
	return http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		reqid := generateRequestID()
		r = util.SetContext(r, reqIDKey, reqid)
		next.ServeHTTP(w, r)
	})
}

// GetRequestID returns the current request's request id.
func GetRequestID(r *http.Request) string {
	val := r.Context().Value(reqIDKey)
	if val == nil {
		return ""
	}
	return val.(string)
}

func generateRequestID() string {
	buf := make([]byte, 4)
	rand.Read(buf)
	return hex.EncodeToString(buf)
}
