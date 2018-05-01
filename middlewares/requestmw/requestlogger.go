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
	"fmt"
	"io"
	"net/http"
	"time"

	"github.com/alien-bunny/ab/lib/middleware"
	"github.com/alien-bunny/ab/lib/util"
	"github.com/fatih/color"
	kitlog "github.com/go-kit/kit/log"
)

const MiddlewareDependencyRequestlogger = "*requestmw.RequestLoggerMiddleware"

var _ middleware.Middleware = &RequestLoggerMiddleware{}

// RequestLoggerMiddleware logs request data (method, length, path).
type RequestLoggerMiddleware struct {
	logger kitlog.Logger
	middleware.NoDependencies
}

var (
	http1xxColor = color.New(color.FgBlack, color.BgWhite)
	http2xxColor = color.New(color.FgWhite, color.BgGreen)
	http3xxColor = color.New(color.FgWhite, color.BgBlue)
	http4xxColor = color.New(color.FgWhite, color.BgYellow)
	http5xxColor = color.New(color.FgWhite, color.BgRed)
	methodColor  = color.New(color.FgCyan)
	pathColor    = color.New(color.FgBlue)
	reqidColor   = color.New(color.FgRed)
	startColor   = color.New(color.Faint)
	timeColor    = color.New(color.Bold)
)

type http1xx string

func (s http1xx) Format(w io.Writer) {
	http1xxColor.Fprint(w, s)
}

type http2xx string

func (s http2xx) Format(w io.Writer) {
	http2xxColor.Fprint(w, s)
}

type http3xx string

func (s http3xx) Format(w io.Writer) {
	http3xxColor.Fprint(w, s)
}

type http4xx string

func (s http4xx) Format(w io.Writer) {
	http4xxColor.Fprint(w, s)
}

type http5xx string

func (s http5xx) Format(w io.Writer) {
	http5xxColor.Fprint(w, s)
}

type method string

func (s method) Format(w io.Writer) {
	methodColor.Fprint(w, s)
}

type path string

func (s path) Format(w io.Writer) {
	pathColor.Fprint(w, s)
}

type reqid string

func (s reqid) Format(w io.Writer) {
	reqidColor.Fprint(w, s)
}

type reqstart string

func (s reqstart) Format(w io.Writer) {
	startColor.Fprint(w, s)
}

type reqtime string

func (s reqtime) Format(w io.Writer) {
	timeColor.Fprint(w, s)
}

func NewRequestLoggerMiddleware(logger kitlog.Logger) *RequestLoggerMiddleware {
	return &RequestLoggerMiddleware{
		logger: logger,
	}
}

func (rl *RequestLoggerMiddleware) Wrap(next http.Handler) http.Handler {
	return http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		var start, end int64
		starttime := time.Now()
		start = starttime.UnixNano()

		reqpath := r.URL.Path
		reqhost := r.Host
		protocol := "http"
		if r.TLS != nil {
			protocol = "https"
		}

		l := rl.logger

		if requestid := GetRequestID(r); requestid != "" {
			l = kitlog.With(l, "requestid", reqid(requestid))
		}

		rw := &requestLoggerResponseWriter{
			ResponseWriterWrapper: util.ResponseWriterWrapper{w},
			code: http.StatusOK,
		}

		next.ServeHTTP(rw, r)

		end = time.Now().UnixNano()
		duration := end - start
		durationTime := durationTime(duration)
		code := httpCode(rw.GetCode())

		l.Log(
			"httpmethod", method(r.Method),
			"httpreq", path(protocol+"://"+reqhost+reqpath),
			"httpcode", code,
			"start", reqstart(starttime.Format("2006/01/02 15:04:05")),
			"duration", reqtime(durationTime),
		)
	})
}

func durationTime(duration int64) string {
	if duration >= 1000000000 {
		return fmt.Sprintf("%.2fs", float64(duration)/1000000000)
	} else if duration >= 1000000 {
		return fmt.Sprintf("%.2fms", float64(duration)/1000000)
	} else if duration >= 1000 {
		return fmt.Sprintf("%.2fµs", float64(duration)/1000)
	}

	return fmt.Sprintf("%dns", duration)
}

func httpCode(httpCode int) interface{} {
	codeStr := fmt.Sprintf("%d", httpCode)
	if httpCode >= 100 && httpCode <= 199 {
		return http1xx(codeStr)
	} else if httpCode >= 200 && httpCode <= 299 {
		return http2xx(codeStr)
	} else if httpCode >= 300 && httpCode <= 399 {
		return http3xx(codeStr)
	} else if httpCode >= 400 && httpCode <= 499 {
		return http4xx(codeStr)
	} else if httpCode >= 500 && httpCode <= 599 {
		return http5xx(codeStr)
	}

	return codeStr
}

var _ http.Hijacker = &requestLoggerResponseWriter{}
var _ http.Flusher = &requestLoggerResponseWriter{}
var _ http.Pusher = &requestLoggerResponseWriter{}

var _ http.ResponseWriter = &requestLoggerResponseWriter{}

type requestLoggerResponseWriter struct {
	util.ResponseWriterWrapper
	code int
}

func (rw *requestLoggerResponseWriter) WriteHeader(code int) {
	rw.code = code
	rw.ResponseWriterWrapper.WriteHeader(code)
}

func (rw *requestLoggerResponseWriter) GetCode() int {
	return rw.code
}
