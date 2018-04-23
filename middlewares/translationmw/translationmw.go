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

package translationmw

import (
	"net/http"
	"reflect"

	"github.com/alien-bunny/ab/lib/config"
	"github.com/alien-bunny/ab/lib/log"
	"github.com/alien-bunny/ab/lib/middleware"
	"github.com/alien-bunny/ab/lib/translation"
	"github.com/alien-bunny/ab/lib/util"
	"github.com/alien-bunny/ab/middlewares/configmw"
	"github.com/alien-bunny/ab/middlewares/logmw"
	"github.com/alien-bunny/ab/middlewares/sessionmw"
	"golang.org/x/text/language"
)

const (
	MiddlewareDependencyTranslation = "*translationmw.TranslationMiddleware"
	translationKey                  = "abtranslation"
	pluralKey                       = "abtranslationplural"
	languageKey                     = "language"
	dynamicDefaultLanguageKey       = "defaultLanguage"
)

var _ middleware.Middleware = &TranslationMiddleware{}

type TranslationMiddleware struct {
	translator  *translation.Translator
	negotiators []LanguageNegotiator
	Formatter   translation.Formatter
	matcher     language.Matcher
	Filter      func(*http.Request, language.Tag) bool
}

func (m *TranslationMiddleware) ConfigSchema() map[string]reflect.Type {
	schema := make(map[string]reflect.Type)

	for _, negotiator := range m.negotiators {
		if p, ok := negotiator.(config.ConfigSchemaProvider); ok {
			for n, t := range p.ConfigSchema() {
				schema[n] = t
			}
		}
	}

	return schema
}

func (m *TranslationMiddleware) Dependencies() []string {
	return []string{
		configmw.MiddlewareDependencyConfig,
		logmw.MiddlewareDependencyLog,
	}
}

func New(logger log.Logger, supportedLanguages []language.Tag, negotiators ...LanguageNegotiator) *TranslationMiddleware {
	tr := translation.NewTranslator(logger)
	tr.DefaultPluralForms()
	return &TranslationMiddleware{
		translator:  tr,
		negotiators: negotiators,
		Formatter:   &translation.HTMLFormatter{},
		matcher:     language.NewMatcher(supportedLanguages),
	}
}

func (m *TranslationMiddleware) negotiateLanguage(r *http.Request) language.Tag {
	for _, negotiator := range m.negotiators {
		if lang := negotiator.NegotiateLanguage(r); len(lang) > 0 {
			tag, _, confidence := m.matcher.Match(lang...)
			if confidence != language.No && (m.Filter == nil || m.Filter(r, tag)) {
				return tag
			}
		}
	}

	return language.English
}

func (m *TranslationMiddleware) Wrap(next http.Handler) http.Handler {
	return http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		lang := m.negotiateLanguage(r)
		r = util.SetContext(r, translationKey, m.translator.Instance(lang, m.Formatter))
		r = util.SetContext(r, pluralKey, m.translator.PluralInstance(lang, m.Formatter))
		r = util.SetContext(r, languageKey, lang)
		w.Header().Set("Content-Language", lang.String())

		next.ServeHTTP(w, r)
	})
}

func GetTranslate(r *http.Request) func(message string, params map[string]string) string {
	return r.Context().Value(translationKey).(func(string, map[string]string) string)
}

func GetPluralTranslate(r *http.Request) func(count int, singular, plural string, params map[string]string) string {
	return r.Context().Value(pluralKey).(func(int, string, string, map[string]string) string)
}

func GetLanguage(r *http.Request) language.Tag {
	return r.Context().Value(languageKey).(language.Tag)
}

type LanguageNegotiator interface {
	NegotiateLanguage(r *http.Request) []language.Tag
}

type AcceptLanguage struct{}

func (a AcceptLanguage) NegotiateLanguage(r *http.Request) []language.Tag {
	header := r.Header.Get("Accept-Language")
	if header == "" {
		return []language.Tag{}
	}

	tags, _, err := language.ParseAcceptLanguage(header)
	if err != nil {
		return []language.Tag{}
	}

	return tags
}

type SessionLanguage struct{}

func (l SessionLanguage) NegotiateLanguage(r *http.Request) []language.Tag {
	sess := sessionmw.GetSession(r)

	if lang, found := sess[languageKey]; found {
		return []language.Tag{language.Make(lang)}
	}

	return []language.Tag{}
}

func SetSessionLanguage(r *http.Request, lang language.Tag) {
	sess := sessionmw.GetSession(r)
	if sess != nil {
		sess[languageKey] = lang.String()
	}
}

type CookieLanguage string

func (name CookieLanguage) NegotiateLanguage(r *http.Request) []language.Tag {
	c, err := r.Cookie(string(name))
	if err != nil || c == nil {
		return []language.Tag{}
	}

	return []language.Tag{language.Make(c.Value)}
}

type URLParamLanguage string

func (p URLParamLanguage) NegotiateLanguage(r *http.Request) []language.Tag {
	if val := r.URL.Query().Get(string(p)); val != "" {
		return []language.Tag{language.Make(val)}
	}

	return []language.Tag{}
}

type StaticDefaultLanguage language.Tag

func (lang StaticDefaultLanguage) NegotiateLanguage(r *http.Request) []language.Tag {
	return []language.Tag{language.Tag(lang)}
}

type DefaultLanguage struct {
	Language string
}

type DynamicDefaultLanguage struct{}

func (lang DynamicDefaultLanguage) ConfigSchema() map[string]reflect.Type {
	return map[string]reflect.Type{
		dynamicDefaultLanguageKey: reflect.TypeOf(DefaultLanguage{}),
	}
}

func (lang DynamicDefaultLanguage) NegotiateLanguage(r *http.Request) []language.Tag {
	l, err := configmw.GetConfig(r).Get(dynamicDefaultLanguageKey)
	if err != nil {
		logmw.Warn(r, "dynamic default language", configmw.CategoryConfigNotFound).Log("error", err)
		return []language.Tag{}
	}
	if l == nil {
		return []language.Tag{language.English}
	}

	return []language.Tag{language.MustParse(l.(DefaultLanguage).Language)}
}
