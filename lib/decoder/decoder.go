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

package decoder

import (
	"encoding/csv"
	"encoding/json"
	"encoding/xml"
	"io"
	"net/http"
	"strings"

	"github.com/alien-bunny/ab/lib"
	"github.com/alien-bunny/ab/lib/errors"
	"github.com/pelletier/go-toml"
	"gopkg.in/yaml.v2"
)

var NoDecoderErr = errors.New("no decoder found for the request content type")

// Decoders are POST data decoders. The key is the content type, the value is a decoder that decodes the contents of the Reader into v.
var Decoders = map[string]func(body io.Reader, v interface{}) error{
	"application/json": JSONDecoder,
	"application/yaml": YAMLDecoder,
	"application/toml": TOMLDecoder,
	"application/xml":  XMLDecoder,
	"text/xml":         XMLDecoder,
	"text/csv":         CSVDecoder,
}

// JSONDecoder decodes the request body using the built-in JSON decoder into v.
func JSONDecoder(body io.Reader, v interface{}) error {
	return json.NewDecoder(body).Decode(v)
}

// YAMLDecoder decodes the request body using a YAML decoder into v.
func YAMLDecoder(body io.Reader, v interface{}) error {
	return yaml.NewDecoder(body).Decode(v)
}

// TOMLDecoder decodes the request body using a TOML decoder into v.
func TOMLDecoder(body io.Reader, v interface{}) error {
	return toml.NewDecoder(body).Decode(v)
}

// XMLDecoder decodes the request body using the built-in XML decoder into v.
func XMLDecoder(body io.Reader, v interface{}) error {
	return xml.NewDecoder(body).Decode(v)
}

// CSVDecoder decodes the request body using the built-in CSV reader into v.
//
// v must be *[][]string
func CSVDecoder(body io.Reader, v interface{}) error {
	if m, ok := v.(*[][]string); ok {
		var err error
		*m, err = csv.NewReader(body).ReadAll()
		return err
	}

	return errors.New("invalid data type for csv")
}

// Decode decodes a request body into v. After decoding, it closes the body.
//
// This function considers only the Content-Type header, and requires its presence. See the Decoders variable for more information.
func Decode(r *http.Request, v interface{}) error {
	ct := strings.Split(r.Header.Get("Content-Type"), ";")[0]

	if dec, ok := Decoders[ct]; ok {
		defer r.Body.Close()
		return dec(r.Body, v)
	}

	return NoDecoderErr
}

// MustDecode is the same as Decode(), but it panics instead of returning an error.
//
// When using the kit with the recommended settings, this method is recommended instead of Decode(), because the panic will get caught by the error handler middleware.
func MustDecode(r *http.Request, v interface{}) {
	err := Decode(r, v)
	if err == NoDecoderErr {
		errors.Fail(http.StatusUnsupportedMediaType, err)
	}
	if err != nil {
		errors.Fail(http.StatusBadRequest, err)
	}
	if validator, ok := v.(lib.Validator); ok {
		if err := validator.Validate(); err != nil {
			errors.Fail(http.StatusUnprocessableEntity, err)
		}
	}
}
