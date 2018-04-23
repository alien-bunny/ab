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

package uuidmw

import (
	"net/http"

	"github.com/alien-bunny/ab/lib/errors"
	"github.com/alien-bunny/ab/lib/middleware"
	"github.com/alien-bunny/ab/lib/server"
	"github.com/alien-bunny/ab/lib/uuid"
	"github.com/alien-bunny/ab/middlewares/errormw"
)

var _ middleware.Middleware = &UUIDMiddleware{}

// UUIDMiddleware checks the validity of the UUIDs in URL parameters.
//
// Because it relies on the URL parameters to be present in the context, it can only be used as handler middlewares.
type UUIDMiddleware struct {
	parameters []string
	key        []byte
	strict     bool
}

// New creates a new UUIDMiddleware.
//
// If the strict parameter is false, then empty values will be allowed.
func New(key []byte, strict bool, parameters ...string) *UUIDMiddleware {
	return &UUIDMiddleware{
		parameters: parameters,
		key:        key,
		strict:     strict,
	}
}

func (m *UUIDMiddleware) Wrap(next http.Handler) http.Handler {
	return http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		for _, param := range m.parameters {
			p := server.GetParams(r).ByName(param)
			if p == "" {
				if m.strict {
					errors.Fail(http.StatusNotFound, nil)
				}
			} else if !uuid.ParseAndVerify(m.key, p) {
				errors.Fail(http.StatusNotFound, nil)
			}
		}

		next.ServeHTTP(w, r)
	})
}

func (m *UUIDMiddleware) Dependencies() []string {
	return []string{
		errormw.MiddlewareDependencyError,
	}
}
