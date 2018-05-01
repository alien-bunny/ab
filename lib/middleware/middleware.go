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

package middleware

import (
	"net/http"
	"reflect"
)

type HandlerWrapper interface {
	Wrap(http.Handler) http.Handler
}

type HasMiddlewareDependencies interface {
	Dependencies() []string
}

type Middleware interface {
	HandlerWrapper
	HasMiddlewareDependencies
}

type Func func(http.Handler) http.Handler

func (mf Func) Wrap(next http.Handler) http.Handler {
	return mf(next)
}

func (mf Func) Dependencies() []string {
	return []string{}
}

type wrappedHandler struct {
	http.Handler
	deps []string
}

func (wh *wrappedHandler) ServeHTTP(w http.ResponseWriter, r *http.Request) {
	wh.Handler.ServeHTTP(w, r)
}

func (wh *wrappedHandler) Dependencies() []string {
	return wh.deps
}

func (wh *wrappedHandler) Unwrap() http.Handler {
	return wh.Handler
}

func WrapHandler(h http.Handler, dependencies ...string) http.Handler {
	return &wrappedHandler{
		Handler: h,
		deps:    dependencies,
	}
}

func WrapHandlerFunc(f func(http.ResponseWriter, *http.Request), dependencies ...string) http.Handler {
	return WrapHandler(http.HandlerFunc(f), dependencies...)
}

type Stack struct {
	middlewares []Middleware
	provided    map[string]struct{}
	parent      *Stack
}

func NewStack(parent *Stack) *Stack {
	return &Stack{
		middlewares: make([]Middleware, 0),
		provided:    make(map[string]struct{}),
		parent:      parent,
	}
}

func (ms *Stack) ValidateHandler(h http.Handler) error {
	if d, ok := h.(HasMiddlewareDependencies); ok {
		return ms.validate(d)
	}

	return nil
}

func (ms *Stack) validate(d HasMiddlewareDependencies) error {
	for _, dep := range d.Dependencies() {
		if !ms.isProvided(dep) {
			return DependencyError{
				NotFound: dep,
				Provided: ms.collectDependencies(),
			}
		}
	}

	return nil
}

func (ms *Stack) collectDependencies() []string {
	current := ms
	var deps []string

	for current != nil {
		for d := range current.provided {
			deps = append(deps, d)
		}

		current = current.parent
	}

	return deps
}

func (ms *Stack) isProvided(dep string) bool {
	if _, ok := ms.provided[dep]; ok {
		return true
	}

	if ms.parent != nil {
		return ms.parent.isProvided(dep)
	}

	return false
}

func (ms *Stack) provide(v Middleware) {
	name := reflect.TypeOf(v).String()
	ms.provided[name] = struct{}{}
}

func (ms *Stack) Push(m Middleware) error {
	if verr := ms.validate(m); verr != nil {
		return verr
	}

	ms.provide(m)
	ms.middlewares = append(ms.middlewares, m)

	return nil
}

func (ms *Stack) Shift(m Middleware) error {
	if verr := ms.validate(m); verr != nil {
		return verr
	}

	ms.provide(m)
	ms.middlewares = append([]Middleware{m}, ms.middlewares...)

	return nil
}

func (ms *Stack) Wrap(handler http.Handler) http.Handler {
	for i := len(ms.middlewares) - 1; i >= 0; i-- {
		handler = ms.middlewares[i].Wrap(handler)
	}

	return handler
}

type DependencyError struct {
	NotFound string
	Provided []string
}

func (de DependencyError) Error() string {
	return `dependency "` + de.NotFound + `" is not found`
}

type NoDependencies struct{}

func (n NoDependencies) Dependencies() []string {
	return []string{}
}
