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

package translation

import (
	"strconv"
	"strings"
	"sync"

	"github.com/go-kit/kit/log"
	"golang.org/x/text/language"
)

const (
	// ETX (end of text)
	DELIMITER = "\x03"
)

type dictionary map[string]string

type Translator struct {
	mtx          sync.RWMutex
	dictionaries map[language.Tag]dictionary

	pluralMtx   sync.RWMutex
	pluralforms map[language.Tag]map[int]int

	logger log.Logger
}

func NewTranslator(logger log.Logger) *Translator {
	return &Translator{
		dictionaries: make(map[language.Tag]dictionary),
		pluralforms:  make(map[language.Tag]map[int]int),
		logger:       logger,
	}
}

func (t *Translator) SetTranslations(lang language.Tag, translations map[string]string) {
	t.mtx.Lock()
	defer t.mtx.Unlock()

	if _, ok := t.dictionaries[lang]; !ok {
		t.dictionaries[lang] = make(dictionary)
	}

	for message, translation := range translations {
		t.dictionaries[lang][message] = translation
	}
}

func (t *Translator) SetPluralForms(lang language.Tag, plurals map[int]int) {
	t.pluralMtx.Lock()
	t.pluralforms[lang] = plurals
	t.pluralMtx.Unlock()
}

func (t *Translator) DefaultPluralForms() {
	for lang, data := range pluralForms {
		t.SetPluralForms(lang, data)
	}
}

func (t *Translator) translateMessage(lang language.Tag, message string) string {
	t.mtx.RLock()
	translatedMessage := t.dictionaries[lang][message]
	if translatedMessage == "" {
		base, conf := lang.Base()
		if conf == language.Exact {
			baseLang, _ := language.Compose(base)
			translatedMessage = t.dictionaries[baseLang][message]
		}
	}
	t.mtx.RUnlock()

	if translatedMessage == "" {
		t.logger.Log("untranslated", message)
		return message
	}
	return translatedMessage
}

func (t *Translator) formatParameters(formatter Formatter, translatedMessage string, params map[string]string) string {
	for k, v := range params {
		switch k[0] {
		case '!':
			translatedMessage = strings.Replace(translatedMessage, k, formatter.FormatRaw(v), -1)
		case '@':
			translatedMessage = strings.Replace(translatedMessage, k, formatter.FormatNormal(v), -1)
		case '#':
			translatedMessage = strings.Replace(translatedMessage, k, formatter.FormatEmphasized(v), -1)
		default:
			panic("invalid parameter: " + k)
		}
	}

	return translatedMessage
}

func (t *Translator) Translate(lang language.Tag, formatter Formatter, message string, params map[string]string) string {
	if lang != language.English {
		message = t.translateMessage(lang, message)
	}

	if params != nil {
		return t.formatParameters(formatter, message, params)
	}

	return message
}

func (t *Translator) FormatPlural(lang language.Tag, formatter Formatter, count int, singular, plural string, params map[string]string) string {
	if params == nil {
		params = make(map[string]string)
	}
	params["@count"] = strconv.Itoa(count)

	t.pluralMtx.RLock()
	index := t.pluralforms[lang][count]
	t.pluralMtx.RUnlock()

	if index == 1 { // singular
		return t.Translate(lang, formatter, singular, params)
	}

	pluralForm := t.Translate(lang, formatter, plural, params)
	translations := strings.Split(pluralForm, DELIMITER)
	if len(translations) > index {
		return translations[index]
	}

	return translations[0]
}

func (t *Translator) Instance(lang language.Tag, formatter Formatter) func(message string, params map[string]string) string {
	return func(message string, params map[string]string) string {
		return t.Translate(lang, formatter, message, params)
	}
}

func (t *Translator) PluralInstance(lang language.Tag, formatter Formatter) func(count int, singular string, plural string, params map[string]string) string {
	return func(count int, singular, plural string, params map[string]string) string {
		return t.FormatPlural(lang, formatter, count, singular, plural, params)
	}
}
