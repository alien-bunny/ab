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

package securitymw

import (
	"net/http"
	"strconv"
	"strings"
	"time"

	"github.com/alien-bunny/ab/lib/middleware"
)

const MiddlewareDependencyHSTS = "*securitymw.HSTSMiddleware"

var _ middleware.Middleware = &HSTSMiddleware{}

// HSTSMiddleware adds HTTP Strict Transport Security headers to the responses.
type HSTSMiddleware struct {
	MaxAge            time.Duration
	IncludeSubDomains bool

	middleware.NoDependencies
}

func (h *HSTSMiddleware) Wrap(next http.Handler) http.Handler {
	headerValue := h.String()
	return http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		if r.TLS != nil {
			w.Header().Set("Strict-Transport-Security", headerValue)
		}
		next.ServeHTTP(w, r)
	})
}

func (h *HSTSMiddleware) String() string {
	var directives []string

	if h.MaxAge > 0 {
		directives = append(directives, "max-age="+strconv.Itoa(int(h.MaxAge.Seconds())))
	}

	if h.IncludeSubDomains {
		directives = append(directives, "includeSubDomains")
	}

	if len(directives) == 0 {
		return ""
	}

	return strings.Join(directives, "; ")
}
