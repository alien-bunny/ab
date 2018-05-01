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

package ab_test

import (
	"crypto/rand"
	"encoding/hex"
	"errors"
	"fmt"
	"io"
	"net/http"
	"os"
	"strconv"
	"testing"

	"github.com/alien-bunny/ab"
	"github.com/alien-bunny/ab/lib/abtest"
	"github.com/alien-bunny/ab/lib/collectionloader"
	"github.com/alien-bunny/ab/lib/config"
	"github.com/alien-bunny/ab/lib/db"
	"github.com/alien-bunny/ab/lib/server"
	"github.com/alien-bunny/ab/middlewares/dbmw"
	"github.com/alien-bunny/ab/middlewares/securitymw"
	. "github.com/onsi/ginkgo"
	. "github.com/onsi/gomega"
)

func TestAB(t *testing.T) {
	RegisterFailHandler(Fail)
	ab.RegisterSiteProvider("fixtures", func(conf map[string]string, readOnly bool) config.CollectionLoader {
		return collectionloader.NewDirectory("fixtures/sites", conf, readOnly)
	})
	RunSpecs(t, "AB Suite")
}

var _, clientFactory = abtest.HopMock(func(conf *config.Store, s *server.Server, base, schema string) (abtest.DataMockerFunc, error) {
	s.AddFile("/frontend", "fixtures/index.html")

	s.GetF("/csrf", func(w http.ResponseWriter, r *http.Request) {
		ab.Render(r).Text("CSRF SUCCESS")
	}, securitymw.NewCSRFGetMiddleware("token"))

	s.PostF("/csrf", func(w http.ResponseWriter, r *http.Request) {
		ab.Render(r).Text("CSRF SUCCESS")
	})

	s.GetF("/empty", func(w http.ResponseWriter, r *http.Request) {
	})

	s.GetF("/restricted", func(w http.ResponseWriter, r *http.Request) {
		ab.Render(r).Text("RESTRICTED")
	}, securitymw.NewRestrictAddressMiddleware("192.168.255.255/8"))

	s.GetF("/restrictedok", func(w http.ResponseWriter, r *http.Request) {
		ab.Render(r).Text("RestrictedOK")
	}, securitymw.NewRestrictAddressMiddleware("127.0.0.1/8"))

	s.PostF("/decode", func(w http.ResponseWriter, r *http.Request) {
		v := testDecode{}
		ab.MustDecode(r, &v)

		ab.Render(r).JSON(v)
	})

	s.GetF("/panic", func(w http.ResponseWriter, r *http.Request) {
		panic("oops")
	})

	s.GetF("/fail", func(w http.ResponseWriter, r *http.Request) {
		ab.MaybeFail(http.StatusTeapot, errors.New("oops"))
	})

	buf1k := make([]byte, 512)
	io.ReadFull(rand.Reader, buf1k)
	hex1k := hex.EncodeToString(buf1k)
	s.GetF("/1k", func(w http.ResponseWriter, r *http.Request) {
		ab.Render(r).Text(hex1k)
	})

	s.GetF("/binary", func(w http.ResponseWriter, r *http.Request) {
		file, err := os.Open("fixtures/binary.bin")
		ab.MaybeFail(http.StatusInternalServerError, err)
		ab.Render(r).Binary("application/octet-stream", "binary.bin", file)
	})

	svc := &testService{}
	s.RegisterService(svc)

	return nil, nil
})

var _ server.Service = &testService{}

type testService struct {
}

func (s *testService) Register(srv *server.Server) error {
	srv.GetF("/test", func(w http.ResponseWriter, r *http.Request) {
		rows, err := ab.GetDB(r).Query("SELECT * FROM test ORDER BY a")
		ab.MaybeFail(http.StatusInternalServerError, err)
		var ret []testDecode
		for rows.Next() {
			d := testDecode{}
			rows.Scan(&d.A, &d.B)
			ret = append(ret, d)
		}

		ab.Render(r).JSON(ret)
	})

	srv.GetF("/test/:id", func(w http.ResponseWriter, r *http.Request) {
		id := ab.GetParams(r).ByName("id")
		d := testDecode{}
		ab.MaybeFail(http.StatusInternalServerError, ab.GetDB(r).QueryRow("SELECT * FROM test WHERE a = $1", id).Scan(&d.A, &d.B))

		ab.Render(r).JSON(d)
	})

	srv.PostF("/test", func(w http.ResponseWriter, r *http.Request) {
		d := testDecode{}
		ab.MustDecode(r, &d)
		_, err := ab.GetDB(r).Exec("INSERT INTO test(b) VALUES($1)", d.B)
		ab.MaybeFail(http.StatusBadRequest, err)
		ab.Render(r).SetCode(http.StatusCreated)
	}, dbmw.Begin())

	srv.PutF("/test/:id", func(w http.ResponseWriter, r *http.Request) {
		id, err := strconv.Atoi(ab.GetParams(r).ByName("id"))
		ab.MaybeFail(http.StatusBadRequest, err)
		d := testDecode{}
		ab.MustDecode(r, &d)
		if d.A != id {
			ab.Fail(http.StatusBadRequest, fmt.Errorf("ids must match"))
		}

		res, err := ab.GetDB(r).Exec("UPDATE test SET b = $1 WHERE a = $2", d.B, d.A)
		ab.MaybeFail(http.StatusInternalServerError, err)
		aff, _ := res.RowsAffected()
		if aff == 0 {
			ab.Fail(http.StatusNotFound, nil)
		}
	}, dbmw.Begin())

	srv.DeleteF("/test/:id", func(w http.ResponseWriter, r *http.Request) {
		id := ab.GetParams(r).ByName("id")
		res, err := ab.GetDB(r).Exec("DELETE FROM test WHERE a = $1", id)
		ab.MaybeFail(http.StatusInternalServerError, err)
		aff, _ := res.RowsAffected()
		if aff == 0 {
			ab.Fail(http.StatusNotFound, nil)
		}
	}, dbmw.Begin())

	return nil
}

func (s *testService) SchemaInstalled(conn db.DB) bool {
	return db.TableExists(conn, "test")
}

func (s *testService) SchemaSQL() string {
	return "CREATE TABLE test(a serial NOT NULL PRIMARY KEY, b text NOT NULL);"
}

type testDecode struct {
	A int
	B string
}
