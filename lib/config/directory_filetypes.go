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

package config

import (
	"encoding/json"
	"encoding/xml"
	"io"
	"io/ioutil"

	"github.com/pelletier/go-toml"
	"gopkg.in/yaml.v2"
)

var _ FileType = &JSON{}
var _ FileType = &YAML{}
var _ FileType = &TOML{}
var _ FileType = &XML{}

type JSON struct {
	Strict bool
	Prefix string
	Indent string
}

func (t *JSON) Extensions() []string {
	return []string{"json"}
}

func (t *JSON) Unmarshal(stream io.Reader, v interface{}) error {
	dec := json.NewDecoder(stream)
	if t.Strict {
		dec.DisallowUnknownFields()
	}
	return dec.Decode(v)
}

func (t *JSON) Marshal(stream io.Writer, v interface{}) error {
	enc := json.NewEncoder(stream)
	enc.SetIndent(t.Prefix, t.Indent)
	return enc.Encode(v)
}

type YAML struct {
	Strict bool
}

func (t *YAML) Extensions() []string {
	return []string{"yml", "yaml"}
}

func (t *YAML) Unmarshal(stream io.Reader, v interface{}) error {
	data, err := ioutil.ReadAll(stream)
	if err != nil {
		return err
	}

	if t.Strict {
		return yaml.UnmarshalStrict(data, v)
	} else {
		return yaml.Unmarshal(data, v)
	}
}

func (t *YAML) Marshal(stream io.Writer, v interface{}) error {
	data, err := yaml.Marshal(v)
	if err != nil {
		return err
	}

	_, err = stream.Write(data)
	return err
}

type TOML struct {
	ArraysWithOneElementPerLine bool
	QuoteMapKeys                bool
}

func (t *TOML) Extensions() []string {
	return []string{"toml"}
}

func (t *TOML) Unmarshal(stream io.Reader, v interface{}) error {
	return toml.NewDecoder(stream).Decode(v)
}

func (t *TOML) Marshal(stream io.Writer, v interface{}) error {
	enc := toml.NewEncoder(stream)
	enc.ArraysWithOneElementPerLine(t.ArraysWithOneElementPerLine)
	enc.QuoteMapKeys(t.QuoteMapKeys)
	return enc.Encode(v)
}

type XML struct {
	Strict        bool
	AutoClose     []string
	Entity        map[string]string
	CharsetReader func(charset string, input io.Reader) (io.Reader, error)
	DefaultSpace  string

	Prefix string
	Indent string
}

func (t *XML) Extensions() []string {
	return []string{"xml"}
}

func (t *XML) Unmarshal(stream io.Reader, v interface{}) error {
	dec := xml.NewDecoder(stream)
	dec.Strict = t.Strict
	dec.AutoClose = t.AutoClose
	dec.Entity = t.Entity
	dec.CharsetReader = t.CharsetReader
	dec.DefaultSpace = t.DefaultSpace
	return dec.Decode(v)
}

func (t *XML) Marshal(stream io.Writer, v interface{}) error {
	enc := xml.NewEncoder(stream)
	enc.Indent(t.Prefix, t.Indent)
	return enc.Encode(v)
}
