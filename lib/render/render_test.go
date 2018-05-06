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

package render_test

import (
	"bytes"
	"crypto/rand"
	"html/template"
	"io"
	"net/http"
	"net/http/httptest"

	"github.com/alien-bunny/ab/lib/hal"
	"github.com/alien-bunny/ab/lib/render"
	"github.com/alien-bunny/ab/lib/util"
	. "github.com/onsi/ginkgo"
	. "github.com/onsi/ginkgo/extensions/table"
	. "github.com/onsi/gomega"
)

type test struct {
	A int
	B string
}

type testEL struct {
	A int
	B string
}

func (t testEL) Links() map[string][]interface{} {
	return map[string][]interface{}{}
}

func (t testEL) Curies() []hal.HALCurie {
	return []hal.HALCurie{}
}

func create() (*render.Renderer, *httptest.ResponseRecorder, *http.Request) {
	r := render.NewRenderer()
	rr := httptest.NewRecorder()
	req, _ := http.NewRequest("GET", "/", nil)

	return r, rr, req
}

func wrapjson(j string) string {
	return render.JSONSecurityPrefix + j + "\n"
}

type byteReaderCloser struct {
	*bytes.Reader
	Closed bool
}

func (brc *byteReaderCloser) Close() error {
	brc.Closed = true
	return nil
}

var _ = Describe("Render", func() {
	t := test{5, "asdf"}
	jsont := wrapjson(`{"A":5,"B":"asdf"}`)
	xmlt := "<test>\n\t<A>5</A>\n\t<B>asdf</B>\n</test>"

	Describe("A render object with multiple offers", func() {
		r, rr, req := create()
		req.Header.Set("Accept", "application/json")
		r.XML(t, false).JSON(t).SetCode(http.StatusTeapot)
		r.Render(rr, req)

		It("should return the one with the requested content type", func() {
			Expect(string(rr.Body.Bytes())).To(Equal(jsont))
		})

		It("should output the status code set", func() {
			Expect(rr.Code).To(Equal(http.StatusTeapot))
		})
	})

	Describe("A render object with multiple offers and a request without an accept header", func() {
		r, rr, req := create()
		r.XML(t, true).JSON(t)
		r.Render(rr, req)

		It("should return the first offer", func() {
			Expect(rr.Code).To(Equal(http.StatusOK))
			Expect(string(rr.Body.Bytes())).To(Equal(xmlt))
		})
	})

	Describe("A render object with no offers", func() {
		r, rr, req := create()
		r.Render(rr, req)

		It("should output the no content status code", func() {
			Expect(rr.Code).To(Equal(http.StatusNoContent))
		})

		It("should have an empty body", func() {
			Expect(rr.Body.Bytes()).To(BeEmpty())
		})
	})

	Describe("A render object with no offers and a status code set", func() {
		r, rr, req := create()
		r.SetCode(http.StatusTeapot).Render(rr, req)

		It("should output the no content status code", func() {
			Expect(rr.Code).To(Equal(http.StatusTeapot))
		})

		It("should have an empty body", func() {
			Expect(rr.Body.Bytes()).To(BeEmpty())
		})
	})

	Describe("A render object with a text offer", func() {
		r, rr, req := create()
		r.Text(t.B).Render(rr, req)

		It("should match the offered text exactly", func() {
			Expect(rr.Code).To(Equal(http.StatusOK))
			Expect(string(rr.Body.Bytes())).To(Equal(t.B))
		})
	})

	Describe("A render object with a binary offer", func() {
		data := make([]byte, 4096)
		io.ReadFull(rand.Reader, data)
		ct := "misc/random"
		fn := "random.dat"
		brc := &byteReaderCloser{Reader: bytes.NewReader(data)}
		r, rr, req := create()
		r.Binary(ct, fn, brc)
		r.Render(rr, req)

		It("should close the stream", func() {
			Expect(brc.Closed).To(BeTrue())
		})

		It("should have the same content", func() {
			Expect(rr.Body.Bytes()).To(Equal(data))
		})

		It("should have the proper headers", func() {
			Expect(rr.Header().Get("Content-Type")).To(Equal(ct))
			Expect(rr.Header().Get("Content-Disposition")).To(Equal("attachment; filename=" + fn))
		})
	})

	Describe("A render object with a HAL offer", func() {
		Describe("that is an EndpointLinker", func() {
			tel := testEL{t.A, t.B}
			teljson := wrapjson(`{"A":5,"B":"asdf","_links":{"curies":[]}}`)
			r, rr, req := create()
			r.HALJSON(tel)
			r.Render(rr, req)

			It("should contain the _links attribute", func() {
				Expect(rr.Code).To(Equal(http.StatusOK))
				Expect(string(rr.Body.Bytes())).To(Equal(teljson))
			})
		})
		Describe("that is not an EndpointLinker", func() {
			r, rr, req := create()
			r.HALJSON(t)
			r.Render(rr, req)

			It("should not contain the _links attribute", func() {
				Expect(rr.Code).To(Equal(http.StatusOK))
				Expect(string(rr.Body.Bytes())).To(Equal(jsont))
			})
		})
	})

	Describe("A render object with an HTML offer", func() {
		tpl := template.Must(template.New("test").Parse(`<html><head><title>{{.B}}</title></head><body><p>{{.A}}</p></body></html>`))
		r, rr, req := create()
		r.HTML(tpl, t)
		r.Render(rr, req)

		It("should render the given template with all values", func() {
			Expect(rr.Code).To(Equal(http.StatusOK))
			Expect(rr.Header().Get("Content-Type")).To(Equal("text/html"))
			Expect(string(rr.Body.Bytes())).To(Equal(`<html><head><title>asdf</title></head><body><p>5</p></body></html>`))
		})
	})

	Describe("A render object with a CSV offer", func() {
		data := [][]string{
			{"a", "b", "@c"},
			{"=1", "-2", "+3"},
		}
		csv := "a,b,\"\t@c\"\n\"\t=1\",\"\t-2\",\"\t+3\"\n"

		Describe("that uses [][]string", func() {
			r, rr, req := create()
			r.CSV(data)
			r.Render(rr, req)

			It("should convert the data to CSV", func() {
				Expect(rr.Code).To(Equal(http.StatusOK))
				Expect(rr.Header().Get("Content-Type")).To(Equal("text/csv"))
				Expect(string(rr.Body.Bytes())).To(Equal(csv))
			})
		})

		Describe("that uses a channel", func() {
			ch := make(chan []string)
			go func() {
				for _, r := range data {
					ch <- r
				}
				close(ch)
			}()

			r, rr, req := create()
			r.CSVChannel(ch)
			r.Render(rr, req)

			It("should stream the data and convert it to CSV", func() {
				Expect(ch).To(BeClosed())
				Expect(rr.Code).To(Equal(http.StatusOK))
				Expect(rr.Header().Get("Content-Type")).To(Equal("text/csv"))
				Expect(string(rr.Body.Bytes())).To(Equal(csv))
			})
		})

		Describe("that uses a generator function", func() {
			row := 0
			genf := func(f http.Flusher) ([]string, error) {
				if row < len(data) {
					r := data[row]
					row++
					return r, nil
				}

				return []string{}, io.EOF
			}

			r, rr, req := create()
			r.CSVGenerator(genf)
			r.Render(rr, req)

			It("should stream the data and convert it to CSV", func() {
				Expect(row).To(Equal(len(data)))
				Expect(rr.Code).To(Equal(http.StatusOK))
				Expect(rr.Header().Get("Content-Type")).To(Equal("text/csv"))
				Expect(string(rr.Body.Bytes())).To(Equal(csv))
			})
		})
	})

	Describe("A render object with the rendered property set", func() {
		r, rr, req := create()
		r.SetRendered()
		r.Render(rr, req)

		It("should be marked rendered", func() {
			Expect(r.IsRendered()).To(BeTrue())
		})

		It("should not write anything to the ResponseWriter", func() {
			Expect(rr.Body.Bytes()).To(BeEmpty())
		})
	})

	Describe("a value that needs sanitization cannot be printed", func() {
		tpl := template.Must(template.New("test").Parse(`<html><head><title>title</title></head><body><p>{{.Secret}}</p></body></html>`))
		DescribeTable("",
			func(format func(*render.Renderer, interface{})) {
				r, rr, req := create()
				secret := util.RandomString(8)
				s := &secretive{
					Secret: secret,
				}
				format(r, s)
				r.Render(rr, req)

				Expect(rr.Code).To(Equal(http.StatusOK))
				Expect(string(rr.Body.Bytes())).NotTo(ContainSubstring(secret))
			},
			Entry("HALJSON", func(r *render.Renderer, v interface{}) { r.HALJSON(v) }),
			Entry("JSON", func(r *render.Renderer, v interface{}) { r.JSON(v) }),
			Entry("HTML", func(r *render.Renderer, v interface{}) { r.HTML(tpl, v) }),
			Entry("XML", func(r *render.Renderer, v interface{}) { r.XML(v, true) }),
			Entry("YAML", func(r *render.Renderer, v interface{}) { r.YAML(v) }),
			Entry("TOML", func(r *render.Renderer, v interface{}) { r.TOML(v) }),
		)
	})

})

type secretive struct {
	Secret string
}

func (s *secretive) Sanitize() {
	s.Secret = ""
}
