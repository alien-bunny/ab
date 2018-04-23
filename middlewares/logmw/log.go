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

package logmw

import (
	"io"
	"net/http"

	"github.com/alien-bunny/ab/lib/log"
	"github.com/alien-bunny/ab/lib/middleware"
	"github.com/alien-bunny/ab/lib/util"
	"github.com/alien-bunny/ab/middlewares/requestmw"
	"github.com/fatih/color"
)

const (
	MiddlewareDependencyLog = "*logmw.LoggerMiddleware"
	categoryKey             = "category"
	componentKey            = "component"
	logKey                  = "ablog"
)

const (
	CategoryFormatError       = "format error"
	CategoryValidationFailure = "validation failure"
	CategoryTracing           = "tracing"
	CategoryInputError        = "input error"
)

var _ middleware.Middleware = &LoggerMiddleware{}

// LoggerMiddleware injects a logger and a per request log buffer into the request context.
type LoggerMiddleware struct {
	logger log.Logger

	middleware.NoDependencies
}

func New(logger log.Logger) *LoggerMiddleware {
	return &LoggerMiddleware{
		logger: logger,
	}
}

func (lm *LoggerMiddleware) Wrap(next http.Handler) http.Handler {
	return http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		l := lm.logger

		if reqid := requestmw.GetRequestID(r); reqid != "" {
			l = log.With(l, "requestid", reqidstr(reqid))
		}

		r = Update(r, l)

		next.ServeHTTP(w, r)
	})
}

type reqidstr string

var reqidcolor = color.New(color.FgRed)

func (s reqidstr) Format(w io.Writer, key, value interface{}) {
	reqidcolor.Fprint(w, value)
}

func Update(r *http.Request, logger log.Logger) *http.Request {
	return util.SetContext(r, logKey, logger)
}

func getLog(r *http.Request) log.Logger {
	return r.Context().Value(logKey).(log.Logger)
}

func addctx(l log.Logger, component, category interface{}) log.Logger {
	if component != nil {
		l = log.With(l, componentKey, component)
	}
	if category != nil {
		l = log.With(l, categoryKey, category)
	}

	return l
}

func Debug(r *http.Request, component, category interface{}) log.Logger {
	return addctx(log.Debug(getLog(r)), component, category)
}

func Info(r *http.Request, component, category interface{}) log.Logger {
	return addctx(log.Info(getLog(r)), component, category)
}

func Warn(r *http.Request, component, category interface{}) log.Logger {
	return addctx(log.Warn(getLog(r)), component, category)
}

func Error(r *http.Request, component, category interface{}) log.Logger {
	return addctx(log.Error(getLog(r)), component, category)
}
