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

package rendermw

import (
	"net/http"

	"github.com/alien-bunny/ab/lib/middleware"
	"github.com/alien-bunny/ab/lib/render"
	"github.com/alien-bunny/ab/lib/util"
)

const (
	MiddlewareDependencyRender = "*rendermw.RendererMiddleware"

	renderKey = "abrender"
)

var _ middleware.Middleware = &RendererMiddleware{}

// RendererMiddleware is the middleware for the Render API.
//
// This middleware is automatically added with PetBunny.
//
// This changes the behavior of the ResponseWriter in the following middlewares and the page handler. The ResponseWriter's WriteHeader() method will not write the headers, just sets the Code attribute of the Renderer struct in the page context. This hack is necessary, because else a middleware could write the headers before the Renderer. Given the default configuration, the session middleware comes after the RendererMiddleware (so the session middleware has a chance to set its session cookie), and the session middleware always calls WriteHeader(). See the rendererResponseWriter.WriteHeader() method's documentation for more details.
type RendererMiddleware struct {
	middleware.NoDependencies
}

func New() *RendererMiddleware {
	return &RendererMiddleware{}
}

func (m *RendererMiddleware) Wrap(next http.Handler) http.Handler {
	return http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		renderer := render.NewRenderer()
		r = util.SetContext(r, renderKey, renderer)
		next.ServeHTTP(&rendererResponseWriter{
			ResponseWriterWrapper: util.ResponseWriterWrapper{w},
			Renderer:              renderer,
		}, r)
		renderer.Render(w, r)
	})
}

// Render gets the Renderer struct from the request context.
func Render(r *http.Request) *render.Renderer {
	return r.Context().Value(renderKey).(*render.Renderer)
}

var _ http.Hijacker = &rendererResponseWriter{}
var _ http.Flusher = &rendererResponseWriter{}
var _ http.Pusher = &rendererResponseWriter{}

type rendererResponseWriter struct {
	util.ResponseWriterWrapper
	*render.Renderer
}

func (r *rendererResponseWriter) Write(b []byte) (int, error) {
	if !r.Renderer.IsRendered() {
		r.ResponseWriter.WriteHeader(r.Renderer.Code)
		r.Renderer.SetRendered()
	}
	return r.ResponseWriter.Write(b)
}

// WriteHeader overwrites the WriteHeader function of the http.ResponseWriter interface.
//
// The reason why this method does not write the headers is that it allows the Renderer
// middleware to output the response code along with the HTTP headers.
// Without this hack, middlewares could output the headers before the
// Renderer would. With the default settings, the session middleware always
// calls WriteHeader(), prohibiting the Renderer to work properly.
//
// However this method overwrites the Renderer's status code if the code is not set or the new code is not 200 or 0.
func (r *rendererResponseWriter) WriteHeader(code int) {
	if r.Renderer.Code == 0 || (code != http.StatusOK && code != 0) {
		r.Renderer.SetCode(code)
	}
}
