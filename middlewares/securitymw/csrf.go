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
	"crypto/rand"
	"encoding/hex"
	"net/http"

	"github.com/alien-bunny/ab/lib/errors"
	"github.com/alien-bunny/ab/lib/middleware"
	"github.com/alien-bunny/ab/middlewares/logmw"
	"github.com/alien-bunny/ab/middlewares/sessionmw"
)

const (
	MiddlewareDependencyCSRF    = "*securitymw.CSRFMiddleware"
	MiddlewareDependencyCSRFGet = "*securitymw.CSRFGetMiddleware"
	csrfComponent               = "csrf middleware"
	csrfGetComponent            = "csrf get middleware"
)

var _ middleware.Middleware = &CSRFMiddleware{}
var _ middleware.Middleware = &CSRFGetMiddleware{}

// CSRFMiddleware enforces the correct X-CSRF-Token header on all POST, PUT, DELETE, PATCH requests.
//
// To obtain a token, use CSRFTokenHandler on a path.
type CSRFMiddleware struct{}

func NewCSRFMiddleware() *CSRFMiddleware {
	return &CSRFMiddleware{}
}

func (c *CSRFMiddleware) Wrap(next http.Handler) http.Handler {
	return http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		if r.Method == "POST" || r.Method == "PUT" || r.Method == "DELETE" || r.Method == "PATCH" {
			s := sessionmw.GetSession(r)
			token := s["_csrf"]

			userToken := r.Header.Get("X-CSRF-Token")

			if userToken == "" || userToken != token {
				logmw.Debug(r, csrfComponent, logmw.CategoryValidationFailure).Log(
					"usertoken", userToken,
					"token", token,
				)
				errors.Fail(http.StatusForbidden, errors.New("CSRF token validation failed"))
			}
		}

		next.ServeHTTP(w, r)
	})
}

func (c *CSRFMiddleware) Dependencies() []string {
	return []string{logmw.MiddlewareDependencyLog, sessionmw.MiddlewareDependencySession}
}

// CSRFGetMiddleware checks the CSRF token in the urlParam URL parameter.
//
// This is useful if you want CSRF protection in a GET request. For example, this middleware is used on the auth service's login/logout endpoints.
// Adding this to the server is discouraged. The middlware should be used only on the individual handlers.
type CSRFGetMiddleware struct {
	urlParam string
}

func NewCSRFGetMiddleware(urlParam string) *CSRFGetMiddleware {
	return &CSRFGetMiddleware{
		urlParam: urlParam,
	}
}

func (c *CSRFGetMiddleware) Wrap(next http.Handler) http.Handler {
	return http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		s := sessionmw.GetSession(r)
		token := s["_csrf"]

		userToken := r.URL.Query().Get(c.urlParam)

		if userToken == "" || userToken != token {
			logmw.Debug(r, csrfGetComponent, logmw.CategoryValidationFailure).Log(
				"csrfusertoken", userToken,
				"csrftoken", token,
			)
			errors.Fail(http.StatusForbidden, errors.New("CSRF token validation failed"))
		}

		next.ServeHTTP(w, r)
	})
}

func (c *CSRFGetMiddleware) Dependencies() []string {
	return []string{logmw.MiddlewareDependencyLog, sessionmw.MiddlewareDependencySession}
}

// GetCSRFToken returns the CSRF token for the current session.
//
// If the token is not exists, the function generates one and places it inside the session.
func GetCSRFToken(r *http.Request) string {
	s := sessionmw.GetSession(r)
	token := s["_csrf"]

	if token == "" {
		rawToken := make([]byte, 32)
		if _, err := rand.Read(rawToken); err != nil {
			panic(err)
		}
		token = hex.EncodeToString(rawToken)
		s["_csrf"] = token
	}

	return token
}
