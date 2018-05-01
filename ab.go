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
	"context"
	"crypto/tls"
	"encoding/hex"
	"net/http"
	"os"
	"os/signal"
	"path/filepath"
	"reflect"
	"strconv"
	"strings"
	"syscall"
	"time"

	"github.com/NYTimes/gziphandler"
	"github.com/alien-bunny/ab/lib/certcache"
	"github.com/alien-bunny/ab/lib/collectionloader"
	"github.com/alien-bunny/ab/lib/config"
	"github.com/alien-bunny/ab/lib/errors"
	"github.com/alien-bunny/ab/lib/log"
	"github.com/alien-bunny/ab/lib/middleware"
	"github.com/alien-bunny/ab/lib/server"
	"github.com/alien-bunny/ab/middlewares/configmw"
	"github.com/alien-bunny/ab/middlewares/cryptmw"
	"github.com/alien-bunny/ab/middlewares/dbmw"
	"github.com/alien-bunny/ab/middlewares/errormw"
	"github.com/alien-bunny/ab/middlewares/logmw"
	"github.com/alien-bunny/ab/middlewares/rendermw"
	"github.com/alien-bunny/ab/middlewares/requestmw"
	"github.com/alien-bunny/ab/middlewares/securitymw"
	"github.com/alien-bunny/ab/middlewares/sessionmw"
	"github.com/alien-bunny/ab/middlewares/translationmw"
	"golang.org/x/crypto/acme/autocert"
	"golang.org/x/text/language"
)

const (
	// VERSION is the version of the framework.
	VERSION = "dev"
)

func init() {
	RegisterSiteProvider("directory", func(conf map[string]string, readOnly bool) config.CollectionLoader {
		return collectionloader.NewDirectory("./sites", conf, readOnly)
	})
}

type SiteProvider func(conf map[string]string, readOnly bool) config.CollectionLoader

var siteProviders = make(map[string]SiteProvider)

func RegisterSiteProvider(name string, provider SiteProvider) {
	siteProviders[name] = provider
}

func GetSiteProvider(name string) SiteProvider {
	return siteProviders[name]
}

// Hop sets up a server with the recommended settings.
//
// The configure function runs after the server is set up with middlewares. This is the place where endpoints and
// services should be registered.
//
// The logger parameter is optional. If nil is passed, then a dev logger will be created, logging to os.Stdout.
//
// The basedir parameter is in which directory the server config is. An empty value will default to ".".
//
// The returned channel with either return an error very soon, or it will wait until SIGKILL/SIGTERM is received. The
// channel is not read-only, so it can be closed. Sending something to the channel, or closing it will stop the server.
// The idiomatic way to stop the server is to close the channel.
func Hop(configure func(conf *config.Store, s *server.Server) error, logger log.Logger, basedir string) chan error {
	ret := make(chan error)

	if basedir == "" {
		basedir = "."
	}

	go func() {
		if logger == nil {
			logger = log.NewDevLogger(os.Stdout)
		}
		conf := setupConfig(logger, basedir)

		s, err := Pet(conf, config.Default, logger)
		if err != nil {
			logger.Log("pet", err)
			ret <- err
			return
		}

		if err := configure(conf, s); err != nil {
			logger.Log("server configuration", err)
			ret <- err
			return
		}

		serverConfig, err := getConfig(conf, config.Default, logger)
		if err != nil {
			ret <- err
			return
		}

		if serverConfig.Config.Provider == "" {
			serverConfig.Config.Provider = "directory"
		}

		if err = setupSites(conf, serverConfig); err != nil {
			ret <- err
			return
		}

		setupHTTPS(conf, logger, serverConfig, s)

		stopch := make(chan os.Signal)
		signal.Notify(stopch, os.Interrupt, syscall.SIGKILL, syscall.SIGTERM)

		addr := serverConfig.Host + ":" + serverConfig.Port
		go func() {
			if err := s.StartHTTPS(addr, "", ""); err != nil && err != http.ErrServerClosed {
				logger.Log("startserver", err)
				ret <- err
			}
		}()

		// Wait for either the program to get a signal or a cancellation.
		select {
		case <-stopch:
			// Close the channel, so the caller waiting can exit.
			defer close(ret)
		case <-ret:
		}

		logger.Log("graceful", "received sigint")
		ctx, cancel := context.WithTimeout(context.Background(), time.Duration(serverConfig.Timeout)*time.Second)
		if err := s.HTTPServer.Shutdown(ctx); err != nil {
			logger.Log("graceful", "shutting down", "error", err)
			cancel()
		} else {
			logger.Log("graceful", "stopped")
		}
	}()

	return ret
}

func setupConfig(logger log.Logger, basedir string) *config.Store {
	conf := config.NewStore(logger)
	conf.RegisterSchema("config", reflect.TypeOf(Config{}))
	defaultCollection := config.NewCollection()
	directoryConfigProvider := config.NewDirectoryConfigProvider(basedir, true)
	directoryConfigProvider.RegisterFiletype(&config.JSON{})
	directoryConfigProvider.RegisterFiletype(&config.YAML{})
	directoryConfigProvider.RegisterFiletype(&config.TOML{})
	directoryConfigProvider.RegisterFiletype(&config.XML{})
	defaultCollection.AddProviders(
		config.NewEnvConfigProvider(),
		directoryConfigProvider,
	)
	conf.AddCollection(config.Default, defaultCollection)

	return conf
}

func setupSites(conf *config.Store, serverConfig Config) error {
	var loader config.CollectionLoader
	if provider := GetSiteProvider(serverConfig.Config.Provider); provider != nil {
		if loader = provider(serverConfig.Config.Config, serverConfig.Config.ReadOnly); loader != nil {
			conf.AddCollectionLoaders(loader)
		} else {
			return errors.New("failed to initialize site config loader")
		}
	} else {
		return errors.New("site config provider not found")
	}

	return nil
}

func setupHTTPS(conf *config.Store, logger log.Logger, serverConfig Config, s *server.Server) {
	if serverConfig.HTTPS.LetsEncrypt {
		s.EnableLetsEncrypt("", hostPolicy(conf, logger))
	} else if serverConfig.HTTPS.Autocert != "" {
		s.EnableAutocert(serverConfig.HTTPS.Autocert, "", hostPolicy(conf, logger))
	} else if serverConfig.HTTPS.Site {
		s.TLSConfig = &tls.Config{}
		cc := certcache.New(logger, func(serverName string) (string, string, error) {
			siteConfigInterface, err := conf.Get(serverName).Get("site")
			if err != nil {
				return "", "", err
			}
			if siteConfigInterface == nil {
				return "", "", errors.New("config not found")
			}

			siteConfig := siteConfigInterface.(Site)

			return siteConfig.TLS.Certificate, siteConfig.TLS.Key, nil
		})
		s.TLSConfig.GetCertificate = cc.Get
		s.SubscribeCacheClear(cc.Clear)
	}

	if s.TLSConfig != nil {
		s.TLSConfig.PreferServerCipherSuites = true
		s.TLSConfig.CurvePreferences = []tls.CurveID{
			tls.CurveP256,
			tls.X25519,
		}
		s.TLSConfig.MinVersion = tls.VersionTLS12
		s.TLSConfig.CipherSuites = []uint16{
			tls.TLS_ECDHE_ECDSA_WITH_AES_256_GCM_SHA384,
			tls.TLS_ECDHE_RSA_WITH_AES_256_GCM_SHA384,
			tls.TLS_ECDHE_ECDSA_WITH_CHACHA20_POLY1305,
			tls.TLS_ECDHE_RSA_WITH_CHACHA20_POLY1305,
			tls.TLS_ECDHE_ECDSA_WITH_AES_128_GCM_SHA256,
			tls.TLS_ECDHE_RSA_WITH_AES_128_GCM_SHA256,
		}
	}
}

func hostPolicy(conf *config.Store, logger log.Logger) autocert.HostPolicy {
	return func(ctx context.Context, host string) error {
		if conf.Get(host) == nil {
			log.Warn(logger).Log("certificate host not found", host)
			return errors.New("site not found")
		}

		return nil
	}
}

type Config struct {
	AdminKey string
	Config   struct {
		Provider string
		Config   map[string]string
		ReadOnly bool
	}
	Cookie struct {
		Prefix       string
		ExpiresAfter string
	}
	DB struct {
		MaxIdleConn           int
		MaxOpenConn           int
		ConnectionMaxLifetime int64
	}
	Directories struct {
		Assets string
	}
	Log struct {
		Access        bool
		DisplayErrors bool
	}
	Root                 bool
	Gzip                 bool
	DisableMaster        bool
	CryptSecret          string
	Host                 string
	Port                 string
	NamespaceNegotiation struct {
		HostMap  map[string]string
		SkipPort bool
	}
	HTTPS struct {
		LetsEncrypt bool
		Autocert    string
		Site        bool
	}
	Timeout  int
	Language struct {
		Default   string
		Supported string
	}
}

type Site struct {
	SupportedLanguages []string
	Directories        struct {
		Public  string
		Private string
	}
	TLS struct {
		Certificate string
		Key         string
	}
}

func Pet(conf *config.Store, serverNamespace string, logger log.Logger) (*server.Server, error) {
	conf.RegisterSchema("config", reflect.TypeOf(Config{}))
	conf.RegisterSchema("site", reflect.TypeOf(Site{}))

	serverConfig, err := getConfig(conf, serverNamespace, logger)
	if err != nil {
		return nil, err
	}

	s := server.NewServer(conf, logger)
	s.Router.NotFound = simpleErrorPage(http.StatusNotFound)
	s.Router.MethodNotAllowed = simpleErrorPage(http.StatusMethodNotAllowed)

	s.SubscribeCacheClear(conf.ClearAllCaches)

	if !serverConfig.DisableMaster {
		s.SetMaster()
	}

	if hostname, _ := os.Hostname(); hostname != "" {
		s.Logger = log.With(s.Logger, "hostname", hostname)
	}

	s.UseHandler(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		w.Header().Set("X-Powered-By", "Alien-Bunny "+VERSION)
	}))

	middlewareFactories := []func(Config) (middleware.Middleware, error){
		setupRequestIDMiddleware,
		setupAccessLogMiddleware(s),
		setupGzipMiddleware,
		setupConfigMiddleware(conf),
		setupLogMiddleware(s),
		setupHSTSMiddleware,
		setupCookieMiddleware,
		setupLanguageMiddleware(s),
		setupErrorMiddleware,
		setupRenderMiddleware,
		setupCSRFMiddleware,
		setupDBMiddleware,
		setupCryptMiddleware,
	}

	for _, mf := range middlewareFactories {
		mw, err := mf(serverConfig)

		if err != nil {
			return nil, err
		}

		if mw != nil {
			s.Use(mw)
		}
	}

	s.GetF("/api/token", func(w http.ResponseWriter, r *http.Request) {
		token := securitymw.GetCSRFToken(r)

		Render(r).
			JSON(map[string]string{"token": token}).
			Text(token)
	})

	if serverConfig.Directories.Assets != "-" {
		if serverConfig.Directories.Assets == "" {
			serverConfig.Directories.Assets = "assets"
		}

		s.AddStaticLocalDir("/assets", serverConfig.Directories.Assets)

		if serverConfig.Root {
			s.AddFile("/", filepath.Join(serverConfig.Directories.Assets, "index.html"))
		}
	}

	s.AddDynamicLocalDir("/public", func(r *http.Request) http.FileSystem {
		s, err := configmw.GetConfig(r).Get("site")
		if err != nil {
			logmw.Warn(r, "public directory", configmw.CategoryConfigNotFound).Log("error", err)
			return nil
		}

		d := s.(Site).Directories.Public
		if d == "" {
			return nil
		}

		return http.Dir(d)
	})

	maybeSetupAdmin(s, serverConfig.AdminKey)

	return s, nil
}

func setupRequestIDMiddleware(serverConfig Config) (middleware.Middleware, error) {
	return requestmw.NewRequestIDMiddleware(), nil
}

func setupAccessLogMiddleware(s *server.Server) func(serverConfig Config) (middleware.Middleware, error) {
	return func(serverConfig Config) (middleware.Middleware, error) {
		if serverConfig.Log.Access {
			return requestmw.NewRequestLoggerMiddleware(s.Logger), nil
		}

		return nil, nil
	}
}

func setupGzipMiddleware(serverConfig Config) (middleware.Middleware, error) {
	if serverConfig.Gzip {
		handler, err := gziphandler.GzipHandlerWithOpts(gziphandler.CompressionLevel(9))
		if err != nil {
			return nil, err
		}
		return middleware.Func(handler), nil
	}

	return nil, nil
}

func setupConfigMiddleware(conf *config.Store) func(serverConfig Config) (middleware.Middleware, error) {
	return func(serverConfig Config) (middleware.Middleware, error) {
		cn := configmw.NewChainedNamespaceNegotiator()
		if len(serverConfig.NamespaceNegotiation.HostMap) > 0 {
			hostMapNamespaceNegotiator := configmw.NewHostMapNamespaceNegotiator()
			for host, namespace := range serverConfig.NamespaceNegotiation.HostMap {
				hostMapNamespaceNegotiator.Add(host, namespace)
			}
			cn.AddNegotiator(hostMapNamespaceNegotiator)
		}
		hostNamespaceNegotiator := configmw.NewHostNamespaceNegotiator()
		hostNamespaceNegotiator.SkipPort = serverConfig.NamespaceNegotiation.SkipPort
		cn.AddNegotiator(hostNamespaceNegotiator)
		return configmw.NewConfigMiddleware(conf, cn), nil
	}
}

func setupLogMiddleware(s *server.Server) func(serverConfig Config) (middleware.Middleware, error) {
	return func(serverConfig Config) (middleware.Middleware, error) {
		return logmw.New(s.Logger), nil
	}
}

func setupHSTSMiddleware(serverConfig Config) (middleware.Middleware, error) {
	return configmw.WrapMiddleware("hsts", reflect.TypeOf(securitymw.HSTSMiddleware{})), nil
}

func setupCookieMiddleware(serverConfig Config) (middleware.Middleware, error) {
	var expiresAfter time.Duration
	if serverConfig.Cookie.ExpiresAfter == "" {
		expiresAfter = time.Hour * 24 * 365
	} else {
		var err error
		expiresAfter, err = time.ParseDuration(serverConfig.Cookie.ExpiresAfter)
		if err != nil {
			return nil, err
		}
	}

	return sessionmw.New(serverConfig.Cookie.Prefix, expiresAfter), nil
}

func setupLanguageMiddleware(s *server.Server) func(serverConfig Config) (middleware.Middleware, error) {
	return func(serverConfig Config) (middleware.Middleware, error) {
		lang := language.English
		if serverConfig.Language.Default != "" {
			lang = language.MustParse(serverConfig.Language.Default)
		}
		tmw := translationmw.New(
			s.Logger,
			append(parseSupportedLanguages(serverConfig.Language.Supported), lang),
			translationmw.URLParamLanguage("lang"),
			translationmw.SessionLanguage{},
			translationmw.CookieLanguage(serverConfig.Cookie.Prefix+"_LANGUAGE"),
			translationmw.AcceptLanguage{},
			translationmw.DynamicDefaultLanguage{},
			translationmw.StaticDefaultLanguage(lang),
		)
		tmw.Filter = func(r *http.Request, tag language.Tag) bool {
			s, err := configmw.GetConfig(r).Get("site")
			if err != nil {
				logmw.Warn(r, "language filter", configmw.CategoryConfigNotFound).Log("error", err)
				return true
			}
			supported := s.(Site).SupportedLanguages
			sl := make([]language.Tag, len(supported))
			for i, s := range supported {
				sl[i] = language.MustParse(s)
			}

			if len(sl) == 0 {
				return true
			}

			ts := tag.String()
			for _, l := range sl {
				if l.String() == ts {
					return true
				}
			}

			return false
		}

		return tmw, nil
	}
}

func setupErrorMiddleware(serverConfig Config) (middleware.Middleware, error) {
	return errormw.New(serverConfig.Log.DisplayErrors), nil
}

func setupRenderMiddleware(serverConfig Config) (middleware.Middleware, error) {
	return rendermw.New(), nil
}

func setupCSRFMiddleware(serverConfig Config) (middleware.Middleware, error) {
	return securitymw.NewCSRFMiddleware(), nil
}

func setupDBMiddleware(serverConfig Config) (middleware.Middleware, error) {
	dbMiddleware := dbmw.NewMiddleware()
	dbMiddleware.MaxIdleConnections = serverConfig.DB.MaxIdleConn
	dbMiddleware.MaxOpenConnections = serverConfig.DB.MaxOpenConn
	dbMiddleware.ConnectionMaxLifetime = time.Duration(serverConfig.DB.ConnectionMaxLifetime) * time.Second

	return dbMiddleware, nil
}

func setupCryptMiddleware(serverConfig Config) (middleware.Middleware, error) {
	if serverConfig.CryptSecret == "" {
		return nil, errors.New("empty crypt secret")
	}

	cryptSecret, err := hex.DecodeString(serverConfig.CryptSecret)
	if err != nil {
		return nil, err
	}

	cmw, err := cryptmw.NewCryptMiddleware(cryptSecret)
	if err != nil {
		return nil, err
	}

	return cmw, nil
}

func maybeSetupAdmin(s *server.Server, adminKey string) {
	if adminKey != "" {
		keymw := securitymw.AdminKeyMiddleware(adminKey)

		if s.IsMaster() {
			s.GetF("/install", func(w http.ResponseWriter, r *http.Request) {
				conn := dbmw.GetConnection(r)
				s.InstallServices(conn)
			}, keymw)
		}

		s.GetF("/cache-clear", func(w http.ResponseWriter, r *http.Request) {
			s.ClearCaches()
		}, keymw)
	}
}

func getConfig(conf *config.Store, namespace string, logger log.Logger) (Config, error) {
	serverConfigInterface, err := conf.Get(namespace).Get("config")
	if err != nil {
		logger.Log("configuration load", err)
		return Config{}, err
	}
	if serverConfigInterface == nil {
		err := errors.New("server config not found")
		logger.Log("configuration load", err.Error())
		return Config{}, err
	}
	return serverConfigInterface.(Config), nil
}

func parseSupportedLanguages(supported string) []language.Tag {
	if supported == "" {
		return []language.Tag{}
	}

	var sl []language.Tag
	for _, l := range strings.Split(supported, ",") {
		l = strings.TrimSpace(l)

		sl = append(sl, language.MustParse(l))
	}

	return sl
}

var defaultDeps = []string{
	cryptmw.MiddlewareDependencyCrypt,
	requestmw.MiddlewareDependencyRequestID,
	logmw.MiddlewareDependencyLog,
	translationmw.MiddlewareDependencyTranslation,
	errormw.MiddlewareDependencyError,
	rendermw.MiddlewareDependencyRender,
	sessionmw.MiddlewareDependencySession,
	securitymw.MiddlewareDependencyCSRF,
	dbmw.MiddlewareDependencyDB,
}

type DefaultDependencies struct{}

func (d DefaultDependencies) Dependencies() []string {
	return defaultDeps
}

// WrapHandler adds all middlewares from PetBunny as a dependency to the given handler.
func WrapHandler(h http.Handler, extradeps ...string) http.Handler {
	return middleware.WrapHandler(h, append(defaultDeps, extradeps...)...)
}

// WrapHandlerFunc wraps a handler func with WrapHandler.
func WrapHandlerFunc(f func(http.ResponseWriter, *http.Request), extradeps ...string) http.Handler {
	return WrapHandler(http.HandlerFunc(f), extradeps...)
}

func simpleErrorPage(code int) http.Handler {
	return http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		pd := errormw.NewErrorPageData(code, r)
		Render(r).SetCode(code).HTML(errormw.ErrorPage, pd)
	})
}

// Pager is a function that implements pagination for listing endpoints.
//
// It extracts the "page" query from the url, and returns the offset to that given page.
// The parameter limit specifies the number of elements on a given page.
func Pager(r *http.Request, limit int) int {
	start := 0

	if page := r.URL.Query().Get("page"); page != "" {
		pagenum, err := strconv.Atoi(page)
		MaybeFail(http.StatusBadRequest, err)
		start = (pagenum - 1) * limit
	}

	return start
}

// RedirectHTTPSServer sets up and starts a http server that redirects all requests to https.
func RedirectHTTPSServer(logger log.Logger, addr string) error {
	return (&http.Server{
		Addr:         addr,
		ReadTimeout:  4 * time.Second,
		WriteTimeout: 4 * time.Second,
		IdleTimeout:  128 * time.Second,
		Handler: http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
			w.Header().Set("Connection", "close")

			newUrl := "https://" + r.Host + r.URL.String()

			log.Debug(logger).Log(
				"component", "redirect server",
				"from", r.URL.String(),
				"to", newUrl,
			)

			http.Redirect(w, r, newUrl, http.StatusMovedPermanently)
		}),
	}).ListenAndServe()
}
