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

package sessionmw

import (
	"net/http"
	"net/url"
	"reflect"
	"time"

	"github.com/alien-bunny/ab/lib/middleware"
	"github.com/alien-bunny/ab/lib/session"
	"github.com/alien-bunny/ab/lib/util"
	"github.com/alien-bunny/ab/middlewares/configmw"
	"github.com/alien-bunny/ab/middlewares/logmw"
)

const (
	MiddlewareDependencySession = "*sessionmw.SessionMiddleware"
	sessionComponent            = "session middleware"
	sessionContextKey           = "SESSION"
)

// GetSession returns the session from the http request context.
func GetSession(r *http.Request) session.Session {
	return r.Context().Value(sessionContextKey).(session.Session)
}

type Config struct {
	Key       string
	CookieURL string
}

var _ middleware.Middleware = &SessionMiddleware{}

type SessionMiddleware struct {
	prefix       string
	expiresAfter time.Duration
}

// New creates a session middleware.
//
// The prefix is an optional prefix for the cookie name. The cookie name after the prefix is "_SESSION".
// The key holds the secret key to sign and verify the cookies.
// The cookie URL determines the domain and the path parts of the HTTP cookie that will be set. It can be nil.
// If the cookie URL starts with https://, then the cookie will be forced to work only on HTTPS.
// The expiresAfter sets a duration for the cookies to expire.
func New(prefix string, expiresAfter time.Duration) *SessionMiddleware {
	return &SessionMiddleware{
		prefix:       prefix,
		expiresAfter: expiresAfter,
	}
}

func (s *SessionMiddleware) Wrap(next http.Handler) http.Handler {
	return http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		ci, err := configmw.GetConfig(r).Get("session")
		if err != nil {
			logmw.Error(r, sessionComponent, configmw.CategoryConfigNotFound).Log("error", err)
			http.Error(w, "session not configured", http.StatusInternalServerError)
			return
		}
		c := ci.(Config)
		key := session.MustParse(c.Key)

		cookieURL, err := url.Parse(c.CookieURL)
		if err != nil {
			logmw.Error(r, sessionComponent, "url parsing").Log("error", err)
			http.Error(w, "bad cookie url", http.StatusInternalServerError)
			return
		}

		sess, err := readCookieFromRequest(r, s.prefix, key)
		if err != nil {
			logmw.Warn(r, sessionComponent, logmw.CategoryFormatError).Log("sessioncookieread", err)
		}

		logmw.Debug(r, sessionComponent, logmw.CategoryTracing).Log("session", sess)

		r = util.SetContext(r, sessionContextKey, sess)

		srw := &sessionResponseWriter{
			ResponseWriterWrapper: util.ResponseWriterWrapper{ResponseWriter: w},
			key:          key,
			prefix:       s.prefix,
			r:            r,
			expiresAfter: s.expiresAfter,
			cookieURL:    cookieURL,
		}

		next.ServeHTTP(srw, r)
		srw.WriteHeader(http.StatusOK)
	})
}

func (s *SessionMiddleware) ConfigSchema() map[string]reflect.Type {
	return map[string]reflect.Type{
		"session": reflect.TypeOf(Config{}),
	}
}

func (s *SessionMiddleware) Dependencies() []string {
	return []string{
		logmw.MiddlewareDependencyLog,
		configmw.MiddlewareDependencyConfig,
	}
}

func sessionToCookie(s session.Session, key session.SecretKey, prefix string, cookieURL *url.URL, expiresAfter time.Duration) *http.Cookie {
	cookieValue := session.EncodeSession(s, key)

	c := &http.Cookie{
		Name:     prefix + "_SESSION",
		Value:    cookieValue,
		Path:     "/",
		HttpOnly: true,
		Expires:  time.Now().Add(expiresAfter),
	}

	if cookieURL != nil {
		c.Domain = cookieURL.Host
		c.Path = cookieURL.Path
		c.Secure = cookieURL.Scheme == "https"
	}

	return c
}

func readCookieFromRequest(r *http.Request, prefix string, key session.SecretKey) (session.Session, error) {
	sesscookie, err := r.Cookie(prefix + "_SESSION")
	if err != nil || len(sesscookie.Value) == 0 {
		if err == http.ErrNoCookie {
			err = nil
		}
		return make(session.Session), err
	}

	return session.DecodeSession(sesscookie.Value, key)
}

var _ http.Hijacker = &sessionResponseWriter{}
var _ http.Flusher = &sessionResponseWriter{}
var _ http.Pusher = &sessionResponseWriter{}

var _ http.ResponseWriter = &sessionResponseWriter{}

type sessionResponseWriter struct {
	util.ResponseWriterWrapper
	key          session.SecretKey
	prefix       string
	r            *http.Request
	expiresAfter time.Duration
	written      bool
	cookieURL    *url.URL
}

func (srw *sessionResponseWriter) Write(b []byte) (int, error) {
	if !srw.written {
		srw.WriteHeader(http.StatusOK)
	}

	return srw.ResponseWriterWrapper.Write(b)
}

func (srw *sessionResponseWriter) WriteHeader(code int) {
	if srw.written {
		return
	}

	sess := GetSession(srw.r)
	logmw.Debug(srw.r, sessionComponent, logmw.CategoryTracing).Log("sessionend", sess)
	cookie := sessionToCookie(sess, srw.key, srw.prefix, srw.cookieURL, srw.expiresAfter)
	logmw.Debug(srw.r, sessionComponent, logmw.CategoryTracing).Log("sessioncookie", cookie)
	http.SetCookie(srw.ResponseWriterWrapper.ResponseWriter, cookie)

	srw.ResponseWriterWrapper.WriteHeader(code)

	srw.written = true
}
