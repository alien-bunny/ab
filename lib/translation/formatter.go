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
	"html/template"

	"github.com/fatih/color"
)

var (
	emphasizedTerminalColor = color.New(color.Bold)
)

type Formatter interface {
	FormatRaw(string) string
	FormatNormal(string) string
	FormatEmphasized(string) string
}

type SimpleRawFormatter struct{}

func (f SimpleRawFormatter) FormatRaw(s string) string {
	return s
}

var _ Formatter = &HTMLFormatter{}

type HTMLFormatter struct {
	SimpleRawFormatter
}

func (f *HTMLFormatter) FormatNormal(s string) string {
	return template.HTMLEscapeString(s)
}

func (f *HTMLFormatter) FormatEmphasized(s string) string {
	return "<em>" + f.FormatNormal(s) + "</em>"
}

var _ Formatter = &TerminalFormatter{}

type TerminalFormatter struct {
	SimpleRawFormatter
}

func (f *TerminalFormatter) FormatNormal(s string) string {
	return s // TODO filter
}

func (f *TerminalFormatter) FormatEmphasized(s string) string {
	return emphasizedTerminalColor.Sprint(f.FormatNormal(s))
}
