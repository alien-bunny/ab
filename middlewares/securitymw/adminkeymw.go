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

	"github.com/alien-bunny/ab/lib/errors"
	"github.com/alien-bunny/ab/middlewares/errormw"
)

type AdminKeyMiddleware string

func (key AdminKeyMiddleware) Wrap(next http.Handler) http.Handler {
	return http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		urlkey := r.URL.Query().Get("key")
		if urlkey != string(key) {
			errors.Fail(http.StatusForbidden, errors.New("invalid key"))
		}

		next.ServeHTTP(w, r)
	})
}

func (key AdminKeyMiddleware) Dependencies() []string {
	return []string{
		errormw.MiddlewareDependencyError,
	}
}
