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

package translation_test

import (
	"strconv"

	"github.com/alien-bunny/ab/lib/log"
	"github.com/alien-bunny/ab/lib/translation"
	"github.com/go-kit/kit/log/level"
	. "github.com/onsi/ginkgo"
	. "github.com/onsi/gomega"
	"golang.org/x/text/language"
)

func newTranslator() *translation.Translator {
	logger := log.DefaultDevLogger(level.AllowDebug())
	tr := translation.NewTranslator(logger)
	tr.DefaultPluralForms()
	tr.SetTranslations(language.Hungarian, map[string]string{
		"@count piece":              "@count darab",
		"@count pieces":             "@count darab",
		"#file saved successfully.": "#file sikeresen mentve.",
	})

	return tr
}

var _ = Describe("Smoke test", func() {
	tr := newTranslator()
	f := &translation.HTMLFormatter{}
	t := tr.Instance(language.Hungarian, f)
	p := tr.PluralInstance(language.Hungarian, f)

	It("should translate a simple case with a parameter", func() {
		Expect(t("#file saved successfully.", map[string]string{
			"#file": "/tmp/foobar.baz",
		})).To(Equal("<em>/tmp/foobar.baz</em> sikeresen mentve."))
	})

	It("should translate a simple plural string", func() {
		for i := 0; i < 10; i++ {
			Expect(p(i, "@count piece", "@count pieces", nil)).To(Equal(strconv.Itoa(i) + " darab"))
		}
	})
})

var _ = Describe("HTML escaping test", func() {
	tr := newTranslator()
	f := &translation.HTMLFormatter{}
	t := tr.Instance(language.English, f)

	It("should not escape raw HTML parameters", func() {
		Expect(t("!raw parameter", map[string]string{
			"!raw": "<script></script>",
		})).To(Equal("<script></script> parameter"))
	})

	It("should escape normal HTML parameters", func() {
		Expect(t("@normal parameter", map[string]string{
			"@normal": "<script></script>",
		})).To(Equal("&lt;script&gt;&lt;/script&gt; parameter"))
	})

	It("should escape empasized HTML parameters", func() {
		Expect(t("#emphasized parameter", map[string]string{
			"#emphasized": "<script></script>",
		})).To(Equal("<em>&lt;script&gt;&lt;/script&gt;</em> parameter"))
	})
})
