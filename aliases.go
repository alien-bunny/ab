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

package ab

import (
	"net/http"

	kitlog "github.com/go-kit/kit/log"
	"github.com/julienschmidt/httprouter"

	"github.com/alien-bunny/ab/lib/db"
	"github.com/alien-bunny/ab/lib/decoder"
	"github.com/alien-bunny/ab/lib/errors"
	"github.com/alien-bunny/ab/lib/render"
	"github.com/alien-bunny/ab/lib/server"
	"github.com/alien-bunny/ab/lib/session"
	"github.com/alien-bunny/ab/middlewares/dbmw"
	"github.com/alien-bunny/ab/middlewares/logmw"
	"github.com/alien-bunny/ab/middlewares/rendermw"
	"github.com/alien-bunny/ab/middlewares/sessionmw"
	"github.com/alien-bunny/ab/middlewares/translationmw"
)

func LogDebug(r *http.Request, component, category interface{}) kitlog.Logger {
	return logmw.Debug(r, component, category)
}

func LogInfo(r *http.Request, component, category interface{}) kitlog.Logger {
	return logmw.Info(r, component, category)
}

func LogWarn(r *http.Request, component, category interface{}) kitlog.Logger {
	return logmw.Warn(r, component, category)
}

func LogError(r *http.Request, component, category interface{}) kitlog.Logger {
	return logmw.Error(r, component, category)
}

// GetDB returns the database connection or transaction for the current request.
func GetDB(r *http.Request) db.DB {
	return dbmw.GetConnection(r)
}

// Fail stops the current request from executing.
func Fail(code int, ferr error) {
	errors.Fail(code, ferr)
}

// MaybeFail calls Fail if the error is not nil and not excluded.
func MaybeFail(code int, ferr error, excludedErrors ...error) {
	if ferr == nil {
		return
	}

	for _, e := range excludedErrors {
		if e == ferr {
			return
		}
	}

	errors.Fail(code, ferr)
}

// Render returns the renderer for the current request.
func Render(r *http.Request) *render.Renderer {
	return rendermw.Render(r)
}

// GetSession returns the current session for the request.
func GetSession(r *http.Request) session.Session {
	return sessionmw.GetSession(r)
}

// GetParams returns the path parameter values from the request.
func GetParams(r *http.Request) httprouter.Params {
	return server.GetParams(r)
}

// MustDecode decodes the the request body into v.
func MustDecode(r *http.Request, v interface{}) {
	decoder.MustDecode(r, v)
}

func GetTranslate(r *http.Request) func(message string, params map[string]string) string {
	return translationmw.GetTranslate(r)
}

func GetPluralTranslate(r *http.Request) func(count int, singular, plural string, params map[string]string) string {
	return translationmw.GetPluralTranslate(r)
}
