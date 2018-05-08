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

package render

import (
	"encoding/csv"
	"encoding/json"
	"encoding/xml"
	"html/template"
	"io"
	"net/http"

	"github.com/alien-bunny/ab/lib"
	"github.com/alien-bunny/ab/lib/hal"
	"github.com/golang/gddo/httputil"
	"github.com/pelletier/go-toml"
	"gopkg.in/yaml.v2"
)

const JSONSecurityPrefix = ")]}',\n"

// JSONPrefix is a global switch for the ")]}',\n" JSON response prefix.
//
// This prefix increases security for browser-based applications, but requires extra support on the client side.
var JSONPrefix = true

// Renderer is a per-request struct for the Render API.
//
// The Render API handles content negotiation with the client. The server's preference is the order how the offers are added by either the AddOffer() low-level method or the JSON()/HTML()/Text() higher level methods.
//
// A quick example how to use the Render API:
//
//     func pageHandler(w http.ResponseWriter, r *http.Request) {
//         ...
//         ab.Render(r).
//             HTML(pageTemplate, data).
//             JSON(data)
//     }
//
// In this example, the server prefers rendering an HTML page / fragment, but it can render a JSON if that's the client's preference. The default is HTML, because that is the first offer.
type Renderer struct {
	handlers map[string]func(w http.ResponseWriter)
	offers   []string
	rendered bool
	Code     int // HTTP status code.
}

// NewRenderer creates a new Renderer.
func NewRenderer() *Renderer {
	return &Renderer{
		handlers: make(map[string]func(w http.ResponseWriter)),
		offers:   make([]string, 0),
		rendered: false,
		Code:     0,
	}
}

// SetCode sets the HTTP status code.
func (r *Renderer) SetCode(code int) *Renderer {
	r.Code = code
	return r
}

// AddOffer adds an offer for the content negotiation.
//
// See the Render() method for more information. The mediaType is the content type, the handler renders the data to the ResponseWriter.
// You probably want to use the JSON(), HTML(), Text() methods instead of this.
func (r *Renderer) AddOffer(mediaType string, handler func(w http.ResponseWriter)) *Renderer {
	r.offers = append(r.offers, mediaType)
	r.handlers[mediaType] = handler

	return r
}

// Binary adds a binary file offer for the Renderer struct.
//
// If reader is an io.ReadCloser, it will be closed automatically.
func (r *Renderer) Binary(mediaType, filename string, reader io.Reader) *Renderer {
	return r.AddOffer(mediaType, func(w http.ResponseWriter) {
		w.Header().Set("Content-Disposition", "attachment; filename="+filename)
		io.Copy(w, reader)
		if rc, ok := reader.(io.ReadCloser); ok {
			rc.Close()
		}
	})
}

func maybePrefix(w io.Writer) {
	if JSONPrefix {
		w.Write([]byte(JSONSecurityPrefix))
	}
}

// JSON adds a JSON offer to the Renderer struct.
func (r *Renderer) JSON(v interface{}) *Renderer {
	return r.AddOffer("application/json", func(w http.ResponseWriter) {
		maybePrefix(w)
		maybeSanitize(v)
		json.NewEncoder(w).Encode(v)
	})
}

// HALJSON adds a HAL+JSON offer to the Renderer struct.
func (r *Renderer) HALJSON(v interface{}) *Renderer {
	return r.AddOffer("application/hal+json", func(w http.ResponseWriter) {
		maybePrefix(w)
		maybeSanitize(v)
		enc := json.NewEncoder(w)
		if el, ok := v.(hal.EndpointLinker); ok {
			enc.Encode(hal.NewHalWrapper(el))
		} else {
			enc.Encode(v)
		}
	})
}

// HTML adds an HTML offer to the Renderer struct.
func (r *Renderer) HTML(t *template.Template, v interface{}) *Renderer {
	return r.AddOffer("text/html", func(w http.ResponseWriter) {
		maybeSanitize(v)
		if terr := t.Execute(w, v); terr != nil {
			panic(terr)
		}
	})
}

// Text adds a plain text offer to the Renderer struct.
func (r *Renderer) Text(t string) *Renderer {
	return r.AddOffer("text/plain", func(w http.ResponseWriter) {
		w.Write([]byte(t))
	})
}

// XML adds XML offer to the Renderer object.
//
// If pretty is set, the XML will be indented.
// Also text/xml content type header will be sent instead of application/xml.
func (r *Renderer) XML(v interface{}, pretty bool) *Renderer {
	mt := "application/xml"
	if pretty {
		mt = "text/xml"
	}

	return r.AddOffer(mt, func(w http.ResponseWriter) {
		maybeSanitize(v)
		e := xml.NewEncoder(w)
		if pretty {
			e.Indent("", "\t")
		}
		e.Encode(v)
	})
}

// YAML adds a YAML offer to the Renderer struct.
func (r *Renderer) YAML(v interface{}) *Renderer {
	return r.AddOffer("application/yaml", func(w http.ResponseWriter) {
		maybeSanitize(v)
		yaml.NewEncoder(w).Encode(v)
	})
}

// TOML adds a TOML offer to the Renderer struct.
func (r *Renderer) TOML(v interface{}) *Renderer {
	return r.AddOffer("application/toml", func(w http.ResponseWriter) {
		maybeSanitize(v)
		toml.NewEncoder(w).Encode(v)
	})
}

// CommonFormats adds HAL+JSON, JSON, XML, YAML and TOML offers to the renderer struct.
func (r *Renderer) CommonFormats(v interface{}) *Renderer {
	if l, ok := v.(hal.EndpointLinker); ok {
		r.HALJSON(l)
	}

	return r.JSON(v).YAML(v).TOML(v).XML(v, true)
}

// CSV adds a CSV offer for the Renderer object.
//
// Use this function for smaller CSV responses.
func (r *Renderer) CSV(records [][]string) *Renderer {
	return r.AddOffer("text/csv", func(w http.ResponseWriter) {
		for i := range records {
			for j := range records[i] {
				records[i][j] = maybePrefixCSVField(records[i][j])
			}
		}
		csv.NewWriter(w).WriteAll(records)
	})
}

// CSVChannel adds a CSV offer for the Renderer object.
//
// The records are streamed through a channel.
func (r *Renderer) CSVChannel(records <-chan []string) *Renderer {
	return r.AddOffer("text/csv", func(w http.ResponseWriter) {
		csvw := csv.NewWriter(w)
		for record := range records {
			for i := range record {
				record[i] = maybePrefixCSVField(record[i])
			}
			csvw.Write(record)
		}
		csvw.Flush()
	})
}

// CSVGenerator adds a CSV offer for the Renderer object.
//
// The records are generated with a generator function. If the function
// returns an error, the streaming to the output stops.
func (r *Renderer) CSVGenerator(recgen func(http.Flusher) ([]string, error)) *Renderer {
	return r.AddOffer("text/csv", func(w http.ResponseWriter) {
		csvw := csv.NewWriter(w)
		defer csvw.Flush()
		for {
			record, err := recgen(csvw)
			if err != nil {
				return
			}
			for i := range record {
				record[i] = maybePrefixCSVField(record[i])
			}
			csvw.Write(record)
		}
	})
}

// maybePrefixCSVField helps avoiding a CSV injection attack.
//
// When a field begins with =, -, +, or @, it is possible to coerce
// excel or google docs to execute code. To avoid this, the field
// gets prefixed with a tab character.
//
// See: http://georgemauer.net/2017/10/07/csv-injection.html
func maybePrefixCSVField(content string) string {
	if len(content) > 0 {
		if content[0] == '=' || content[0] == '-' || content[0] == '+' || content[0] == '@' {
			return "\t" + content
		}
	}

	return content
}

// Render renders the best offer to the ResponseWriter according to the client's content type preferences.
func (r *Renderer) Render(w http.ResponseWriter, req *http.Request) {
	if r.IsRendered() {
		return
	}

	defer func() {
		r.SetRendered()
	}()

	if len(r.offers) == 0 {
		if r.Code == 0 || r.Code == http.StatusOK {
			w.WriteHeader(http.StatusNoContent)
		} else {
			w.WriteHeader(r.Code)
		}
		return
	}

	ct := r.offers[0]
	if len(r.offers) > 1 {
		ct = httputil.NegotiateContentType(req, r.offers, ct)
	}

	w.Header().Add("Content-Type", ct)

	if r.Code > 0 {
		w.WriteHeader(r.Code)
	}

	r.handlers[ct](w)
}

// IsRendered checks if the renderer has written its content to an output.
func (r *Renderer) IsRendered() bool {
	return r.rendered
}

// SetRendered marks this Renderer as rendered.
//
// This means that even if Render() will be called, nothing will happen.
func (r *Renderer) SetRendered() {
	r.rendered = true
}

func maybeSanitize(v interface{}) {
	if sanitizer, ok := v.(lib.Sanitizer); ok {
		sanitizer.Sanitize()
	}
}
