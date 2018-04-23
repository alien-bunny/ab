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

package errormw

import (
	"fmt"
	"html/template"
	"net/http"
	"runtime"
	"strings"

	"github.com/alien-bunny/ab/lib/errors"
	"github.com/alien-bunny/ab/lib/middleware"
	"github.com/alien-bunny/ab/lib/render"
	"github.com/alien-bunny/ab/middlewares/logmw"
	"github.com/alien-bunny/ab/middlewares/requestmw"
	"github.com/alien-bunny/ab/middlewares/translationmw"
)

const (
	MiddlewareDependencyError = "*errormw.ErrorHandlerMiddleware"
	error_component           = "error middleware"
)

// Color codes for HTML error pages
var (
	OtherForegroundColor   = "fdf6e3"
	WarningForegroundColor = "fdf6e3"
	ErrorForegroundColor   = "fdf6e3"
	OtherBackgroundColor   = "268bd2"
	WarningBackgroundColor = "b58900"
	ErrorBackgroundColor   = "dc322f"
)

var _ middleware.Middleware = &ErrorHandlerMiddleware{}

// ErrorHandlerMiddleware injects an ErrorHandler into the request context, and then recovers if the ErrorHandler paniced.
//
// This middleware is automatically added to the Server with PetBunny.
type ErrorHandlerMiddleware struct {
	displayErrors bool
}

func New(displayErrors bool) *ErrorHandlerMiddleware {
	return &ErrorHandlerMiddleware{
		displayErrors: displayErrors,
	}
}

func (e *ErrorHandlerMiddleware) Wrap(next http.Handler) http.Handler {
	return http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		defer func() {
			rec := recover()
			if rec == nil {
				return
			}

			stackTrace := make([]byte, 8192)
			runtime.Stack(stackTrace, false)

			p, ok := rec.(errors.Panic)
			if !ok {
				err, ok := rec.(error)
				if !ok {
					err = errors.New(fmt.Sprint(rec))
				}
				p = errors.Panic{
					Code: http.StatusInternalServerError,
					Err:  err,
				}
			}

			p.DisplayErrors = e.displayErrors
			p.StackTrace = strings.TrimRight(string(stackTrace), "\x00")

			renderPanic(p, w, r)
		}()

		next.ServeHTTP(w, r)
	})
}

func (e *ErrorHandlerMiddleware) Dependencies() []string {
	return []string{logmw.MiddlewareDependencyLog, translationmw.MiddlewareDependencyTranslation}
}

func renderPanic(p errors.Panic, w http.ResponseWriter, r *http.Request) {
	t := translationmw.GetTranslate(r)
	rd := render.NewRenderer().SetCode(p.Code)

	pageData := NewErrorPageData(p.Code, r)

	if p.DisplayErrors && p.Err != nil {
		pageData.Message = p.Error()
	} else {
		if ue := p.UserError(t); ue != "" {
			pageData.Message = ue
		} else {
			pageData.Message = t(http.StatusText(p.Code), nil)
		}
	}

	logs := ""
	if p.DisplayErrors {
		logs = p.StackTrace
	}

	pageData.Logs = logs

	if p.Err != nil {
		logmw.Info(r, error_component, nil).Log("error", p.Err)
		logmw.Debug(r, error_component, logmw.CategoryTracing).Log("stacktrace", p.StackTrace)
	}

	jsonMap := map[string]string{"message": pageData.Message}
	text := pageData.Message
	if pageData.RequestID != "" {
		jsonMap["requestid"] = pageData.RequestID
		text += "\n\nRequestID: " + pageData.RequestID
	}
	if p.DisplayErrors {
		jsonMap["logs"] = logs
		text += "\n\n" + logs
	}

	rd.
		HTML(ErrorPage, pageData).
		JSON(jsonMap).
		XML(jsonMap, false).
		Text(text)

	rd.Render(w, r)
}

func decideColor(code int, other, warn, err string) string {
	if code >= 500 && code <= 599 {
		return err
	}
	if code >= 400 && code <= 499 {
		return warn
	}
	return other
}

// ErrorPageData contains data for the ErrorPage template.
type ErrorPageData struct {
	BackgroundColor string
	ForegroundColor string
	Code            int
	Message         string
	Logs            string
	RequestID       string
}

func NewErrorPageData(code int, r *http.Request) ErrorPageData {
	return ErrorPageData{
		BackgroundColor: decideColor(code, OtherBackgroundColor, WarningBackgroundColor, ErrorBackgroundColor),
		ForegroundColor: decideColor(code, OtherForegroundColor, WarningForegroundColor, ErrorForegroundColor),
		Code:            code,
		Message:         "",
		RequestID:       requestmw.GetRequestID(r),
	}
}

// ErrorPage is the default HTML template for the standard HTML error page.
var ErrorPage = template.Must(template.New("ErrorPage").Parse(`<!DOCTYPE HTML>
<html>
<head>
	<meta http-equiv="X-UA-Compatible" content="IE=edge,chrome=1" />
	<meta charset="utf8" />
	<title>Error</title>
	<style type="text/css">
		body {
			background-color: #{{.BackgroundColor}};
			color: #{{.ForegroundColor}};
		}
	</style>
</head>
	<body>
		<h1>HTTP Error {{.Code}}</h1>
		<p>{{.Message}}</p>
		<hr/>
		{{if .RequestID}}<p> Request ID: {{.RequestID}} </p>
		<hr/>{{end}}
		<pre>{{.Logs}}</pre>
	</body>
</html>
`))
