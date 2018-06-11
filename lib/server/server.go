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

package server

import (
	"crypto/rand"
	"crypto/rsa"
	"crypto/tls"
	"io/ioutil"
	stdlog "log"
	"net/http"
	"os"
	"path/filepath"

	"github.com/alien-bunny/ab/lib/config"
	"github.com/alien-bunny/ab/lib/errors"
	"github.com/alien-bunny/ab/lib/log"
	"github.com/alien-bunny/ab/lib/middleware"
	"github.com/alien-bunny/ab/lib/util"
	"github.com/julienschmidt/httprouter"
	"golang.org/x/crypto/acme"
	"golang.org/x/crypto/acme/autocert"
)

const paramKey = "abparam"

// Service is a collection of endpoints that logically belong together or operate on the same part of the database schema.
type Service interface {
	// Name returns the name of this service instance.
	Name() string
	// Register the Service endpoints
	Register(*Server) error
}

type ServiceName string

func (n ServiceName) Name() string {
	return string(n)
}

// Server is the main server struct.
type Server struct {
	Router          *httprouter.Router
	config          *config.Store
	master          bool
	middlewareStack *middleware.Stack
	Logger          log.Logger
	TLSConfig       *tls.Config
	HTTPServer      *http.Server
	services        []Service
}

// NewServer creates a new server with a database connection.
func NewServer(config *config.Store, logger log.Logger) *Server {
	s := &Server{
		Router:          httprouter.New(),
		config:          config,
		middlewareStack: middleware.NewStack(nil),
		Logger:          logger,
	}
	s.Router.RedirectTrailingSlash = true
	s.Router.RedirectFixedPath = true
	s.Router.HandleMethodNotAllowed = true
	s.Router.HandleOPTIONS = true

	return s
}

// IsMaster tells if this server is in master mode.
//
// See SetMaster()
func (s *Server) IsMaster() bool {
	return s.master
}

// SetMaster enables master mode on this server.
//
// If a server is in master mode, it will execute
// SQL updates, and other potentially destructive operations.
// If you run a cluster, it is important that you either only run one
// master, or you deploy one master first, and the rest when all
// changes are finished.
func (s *Server) SetMaster() {
	s.master = true
}

// Use adds middlewares to the top of the middleware stack.
func (s *Server) Use(m middleware.Middleware) {
	if merr := s.middlewareStack.Push(m); merr != nil {
		panic(merr)
	}
	s.config.MaybeRegisterSchema(m)
}

// UseF adds a middleware function to the top of the middleware stack.
func (s *Server) UseF(m func(http.Handler) http.Handler) {
	s.Use(middleware.Func(m))
}

// UseTop adds middlewares to the bottom of the middleware stack.
func (s *Server) UseTop(m middleware.Middleware) {
	if merr := s.middlewareStack.Shift(m); merr != nil {
		panic(merr)
	}
	s.config.MaybeRegisterSchema(m)
}

// UseTopF adds a middleware function to the bottom of the middleware stack.
func (s *Server) UseTopF(m func(http.Handler) http.Handler) {
	s.UseTop(middleware.Func(m))
}

// UseHandler uses a http.Handler as a middleware.
func (s *Server) UseHandler(h http.Handler) {
	s.UseF(func(next http.Handler) http.Handler {
		return http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
			h.ServeHTTP(w, r)
			next.ServeHTTP(w, r)
		})
	})
}

// Handler creates a http.Handler from the server (using the middlewares and the router).
func (s *Server) Handler() http.Handler {
	return s.middlewareStack.Wrap(s.Router)
}

// Handle adds a handler to the router.
//
// The middleware list will be applied to this handler only.
func (s *Server) Handle(method, path string, handler http.Handler, middlewares ...middleware.Middleware) {
	ms := s.middlewareStack
	h := handler

	// callstack cleanup
	if hu, ok := h.(HandlerUnwrapper); ok {
		h = hu.Unwrap()
	}

	if len(middlewares) > 0 {
		ms = middleware.NewStack(s.middlewareStack)
		for _, m := range middlewares {
			if merr := ms.Push(m); merr != nil {
				panic(merr)
			}
		}

		h = ms.Wrap(h)
	}

	if verr := ms.ValidateHandler(handler); verr != nil {
		panic(verr)
	}

	s.config.MaybeRegisterSchema(handler)

	s.Router.Handle(method, path, httprouter.Handle(func(w http.ResponseWriter, r *http.Request, p httprouter.Params) {
		r = util.SetContext(r, paramKey, p)
		h.ServeHTTP(w, r)
	}))
}

// Head adds a HEAD handler to the router.
func (s *Server) Head(path string, handler http.Handler, middlewares ...middleware.Middleware) {
	s.Handle("HEAD", path, handler, middlewares...)
}

// Get adds a GET handler to the router.
func (s *Server) Get(path string, handler http.Handler, middlewares ...middleware.Middleware) {
	s.Handle("GET", path, handler, middlewares...)
}

// Post adds a POST handler to the router.
func (s *Server) Post(path string, handler http.Handler, middlewares ...middleware.Middleware) {
	s.Handle("POST", path, handler, middlewares...)
}

// Put adds a PUT handler to the router.
func (s *Server) Put(path string, handler http.Handler, middlewares ...middleware.Middleware) {
	s.Handle("PUT", path, handler, middlewares...)
}

// Delete adds a DELETE handler to the router.
func (s *Server) Delete(path string, handler http.Handler, middlewares ...middleware.Middleware) {
	s.Handle("DELETE", path, handler, middlewares...)
}

// Patch adds a PATCH handler to the router.
func (s *Server) Patch(path string, handler http.Handler, middlewares ...middleware.Middleware) {
	s.Handle("PATCH", path, handler, middlewares...)
}

// Options adds an OPTIONS handler to the router.
func (s *Server) Options(path string, handler http.Handler, middlewares ...middleware.Middleware) {
	s.Handle("OPTIONS", path, handler, middlewares...)
}

// HeadF adds a HEAD HandlerFunc to the router.
func (s *Server) HeadF(path string, handler http.HandlerFunc, middlewares ...middleware.Middleware) {
	s.Handle("HEAD", path, handler, middlewares...)
}

// GetF adds a GET HandlerFunc to the router.
func (s *Server) GetF(path string, handler http.HandlerFunc, middlewares ...middleware.Middleware) {
	s.Handle("GET", path, handler, middlewares...)
}

// PostF adds a POST HandlerFunc to the router.
func (s *Server) PostF(path string, handler http.HandlerFunc, middlewares ...middleware.Middleware) {
	s.Handle("POST", path, handler, middlewares...)
}

// PutF adds a PUT HandlerFunc to the router.
func (s *Server) PutF(path string, handler http.HandlerFunc, middlewares ...middleware.Middleware) {
	s.Handle("PUT", path, handler, middlewares...)
}

// DeleteF adds a DELETE HandlerFunc to the router.
func (s *Server) DeleteF(path string, handler http.HandlerFunc, middlewares ...middleware.Middleware) {
	s.Handle("DELETE", path, handler, middlewares...)
}

// PatchF adds a PATCH HandlerFunc to the router.
func (s *Server) PatchF(path string, handler http.HandlerFunc, middlewares ...middleware.Middleware) {
	s.Handle("PATCH", path, handler, middlewares...)
}

// OptionsF adds an OPTIONS HandlerFunc to the router.
func (s *Server) OptionsF(path string, handler http.HandlerFunc, middlewares ...middleware.Middleware) {
	s.Handle("OPTIONS", path, handler, middlewares...)
}

// GetParams returns the path parameter values from the request.
func GetParams(r *http.Request) httprouter.Params {
	return r.Context().Value(paramKey).(httprouter.Params)
}

// AddStaticLocalDir adds a local directory to the router.
func (s *Server) AddStaticLocalDir(prefix, path string) *Server {
	s.Router.ServeFiles(prefix+"/*filepath", http.Dir(path))

	return s
}

func (s *Server) AddDynamicLocalDir(prefix string, getPath func(*http.Request) http.FileSystem) *Server {
	s.GetF(prefix+"/*filepath", func(w http.ResponseWriter, r *http.Request) {
		fileSystem := getPath(r)
		if fileSystem == nil {
			http.Error(w, "file system not found", http.StatusNotFound)
			return
		}

		fileServer := http.FileServer(fileSystem)
		r.URL.Path = GetParams(r).ByName("filepath")
		fileServer.ServeHTTP(w, r)
	})

	return s
}

// AddFile adds a local file to the router.
func (s *Server) AddFile(path, file string) *Server {
	s.GetF(path, func(w http.ResponseWriter, r *http.Request) {
		http.ServeFile(w, r, file)
	})

	return s
}

// RegisterService adds a service on the server.
//
// See the Service interface for more information.
func (s *Server) RegisterService(svc Service) {
	if svc.Name() == "" {
		panic("empty service name")
	}

	s.services = append(s.services, svc)
	s.config.MaybeRegisterSchema(svc)
	svc.Register(s)
}

func (s *Server) GetServices() []Service {
	return s.services[:]
}

// StartHTTPS starts the server.
func (s *Server) StartHTTPS(addr, certFile, keyFile string) error {
	return s.startServer(addr, certFile, keyFile, false)
}

func (s *Server) startServer(addr, certFile, keyFile string, forceHTTP bool) error {
	s.HTTPServer = &http.Server{
		Addr:      addr,
		Handler:   s.Handler(),
		TLSConfig: s.TLSConfig,
	}

	s.Logger.Log("serveraddr", addr)

	s.HTTPServer.ErrorLog = stdlog.New(log.NewStdlibAdapter(s.Logger), "", stdlog.LstdFlags)

	var err error
	if !forceHTTP && ((certFile != "" && keyFile != "") || s.HTTPServer.TLSConfig != nil) {
		err = s.HTTPServer.ListenAndServeTLS(certFile, keyFile)
	} else {
		err = s.HTTPServer.ListenAndServe()
	}

	return err
}

// StartHTTP starts the server.
func (s *Server) StartHTTP(addr string) error {
	return s.startServer(addr, "", "", true)
}

// EnableAutocert adds autocert to the server.
//
// caDirEndpoint points to a ca directory endpoint. cacheDir defaults to "private/autocert-cache"
// when empty. If you use the recommended app layout, leave it empty. hostWhitelist is a list of
// hosts where the SSL certificate is valid. Supply at least one domain, else it won't work.
func (s *Server) EnableAutocert(caDirEndpoint, cacheDir string, hostPolicy autocert.HostPolicy) error {
	if cacheDir == "" {
		cacheDir = "private/autocert-cache"
	}

	var key *rsa.PrivateKey
	path := filepath.Join(cacheDir, ".server.key")
	if _, err := os.Stat(path); err == nil {
		content, err := ioutil.ReadFile(path)
		if err != nil {
			return err
		}
		key = util.UnmarshalPrivateKey(content)
		if key == nil {
			return errors.New("failed to load server private key")
		}
	} else {
		key, _ = rsa.GenerateKey(rand.Reader, 2048)
		marshalled := util.MarshalPrivateKey(key)
		if err := ioutil.WriteFile(path, marshalled, 0600); err != nil {
			return err
		}
	}

	client := &acme.Client{
		Key:          key,
		DirectoryURL: caDirEndpoint,
	}

	m := autocert.Manager{
		Client:     client,
		Prompt:     autocert.AcceptTOS,
		HostPolicy: hostPolicy,
		Cache:      autocert.DirCache(cacheDir),
	}

	if s.TLSConfig == nil {
		s.TLSConfig = &tls.Config{}
	}

	s.TLSConfig.GetCertificate = m.GetCertificate

	return nil
}

// EnableLetsEncrypt adds autocert to the server with LetsEncrypt CA dir.
//
// See the documentation of EnableAutocert for cacheDir and hostPolicy.
func (s *Server) EnableLetsEncrypt(cacheDir string, hostPolicy autocert.HostPolicy) error {
	return s.EnableAutocert(acme.LetsEncryptURL, cacheDir, hostPolicy)
}

type HandlerUnwrapper interface {
	Unwrap() http.Handler
}
