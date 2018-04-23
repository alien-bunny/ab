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

package abtest

import (
	"encoding/hex"
	"io"
	"io/ioutil"
	"net/http"
	"net/http/httptest"
	"os"
	"reflect"
	"strings"
	"time"

	"github.com/alien-bunny/ab"
	"github.com/alien-bunny/ab/lib/config"
	"github.com/alien-bunny/ab/lib/db"
	"github.com/alien-bunny/ab/lib/errors"
	"github.com/alien-bunny/ab/lib/log"
	"github.com/alien-bunny/ab/lib/middleware"
	"github.com/alien-bunny/ab/lib/server"
	"github.com/alien-bunny/ab/lib/util"
	"github.com/alien-bunny/ab/middlewares/configmw"
	"github.com/alien-bunny/ab/middlewares/dbmw"
	"github.com/alien-bunny/ab/middlewares/sessionmw"
	. "github.com/onsi/ginkgo"
	. "github.com/onsi/gomega"
	"golang.org/x/text/language"
)

const (
	schemaMiddlewareKey = "abtestschemamw"
)

var (
	FakeKey      = []byte{0, 1, 2, 3, 4, 5, 6, 7, 8, 9, 0, 1, 2, 3, 4, 5, 6, 7, 8, 9, 0, 1, 2, 3, 4, 5, 6, 7, 8, 9, 1, 2}
	FakeAdminKey = "00000000000000000000000000000000"
	LoggerWriter = ioutil.Discard

	cleanupRegistered = false
	removeDirectories []string
	schemas           []string
)

func init() {
	if os.Getenv("VERBOSE") == "1" {
		LoggerWriter = os.Stdout
	}
}

type DataMockerFunc func(db.DB) error

type SetupFunc func(conf *config.Store, s *server.Server, base, schema string) (DataMockerFunc, error)

type SchemaInfo interface {
	SetSearchPath(db.DB)
	GetSchemaName() string
}

var _ SchemaInfo = &SchemaMiddleware{}

type SchemaMiddleware struct {
	name string
}

func (m *SchemaMiddleware) Dependencies() []string {
	return []string{
		dbmw.MiddlewareDependencyDB,
	}
}

func (m *SchemaMiddleware) Wrap(next http.Handler) http.Handler {
	return http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		r = util.SetContext(r, schemaMiddlewareKey, m)
		m.SetSearchPath(dbmw.GetConnection(r))
		next.ServeHTTP(w, r)
	})
}

func (m *SchemaMiddleware) SetSearchPath(conn db.DB) {
	setSearchPath(conn, m.name)
}

func (m *SchemaMiddleware) GetSchemaName() string {
	return m.name
}

func NewSchemaMiddleware() *SchemaMiddleware {
	name := "test_" + strings.ToLower(util.RandomString(16))
	schemas = append(schemas, name)
	registerCleanup()

	conn := Connect("")
	if _, err := conn.Exec("CREATE SCHEMA " + name); err != nil {
		panic(err)
	}

	return &SchemaMiddleware{
		name: name,
	}
}

func GetSchemaMiddleware(r *http.Request) SchemaInfo {
	return r.Context().Value(schemaMiddlewareKey).(SchemaInfo)
}

func GetLogger() log.Logger {
	return log.NewDevLogger(LoggerWriter)
}

func GetConfig(logger log.Logger, host string) *config.Store {
	conf := config.NewStore(logger)
	conf.RegisterSchema("config", reflect.TypeOf(ab.Config{}))
	conf.RegisterSchema("site", reflect.TypeOf(ab.Site{}))

	defaultCollection := config.NewCollection()
	mp := config.NewMemoryConfigProvider()
	mp.Save("config", serverConfig())
	defaultCollection.AddProviders(mp)
	conf.AddCollection(config.Default, defaultCollection)

	conf.AddCollectionLoaders(config.CollectionLoaderFunc(func(name string) (*config.Collection, error) {
		if name == host {
			siteCollection := config.NewCollection()
			siteCollection.SetTemporary(true)

			smp := config.NewMemoryConfigProvider()
			smp.Save("site", siteConfig())
			smp.Save("database", dbConfig())
			smp.Save("session", sessionConfig())

			siteCollection.AddProviders(smp)

			return siteCollection, nil
		}

		return nil, errors.New("site not found")
	}))

	return conf
}

func GetConfigForMiddleware(logger log.Logger) *config.Store {
	return GetConfig(logger, "test")
}

func SetupConfigMiddleware() (log.Logger, *config.Store, *configmw.ConfigMiddleware) {
	logger := GetLogger()
	conf := GetConfigForMiddleware(logger)
	return logger, conf, configmw.NewConfigMiddleware(conf, configmw.NewHostNamespaceNegotiator())
}

func TestMiddleware(stack *middleware.Stack, handler http.HandlerFunc) *httptest.ResponseRecorder {
	w := httptest.NewRecorder()
	r, reqerr := NewRequest("GET", "/", nil)
	Expect(reqerr).NotTo(HaveOccurred())

	stack.Wrap(handler).ServeHTTP(w, r)

	return w
}

func NewRequest(method, url string, body io.Reader) (*http.Request, error) {
	r, err := http.NewRequest(method, url, body)
	if err != nil {
		return nil, err
	}

	r.Header.Set("Host", "test")
	r.Host = "test"

	return r, nil
}

func serverConfig() ab.Config {
	c := ab.Config{
		AdminKey:    FakeAdminKey,
		Root:        true,
		Gzip:        true,
		CryptSecret: hex.EncodeToString(FakeKey),
	}

	c.HostMap = map[string]string{
		"testhost": "test",
	}

	c.DB.MaxIdleConn = 1
	c.DB.MaxOpenConn = 1
	c.DB.ConnectionMaxLifetime = 120

	c.Cookie.Prefix = "AB_TEST_"

	c.Directories.Assets = "./fixtures/"

	c.Log.Access = true
	c.Log.DisplayErrors = true

	return c
}

func siteConfig() ab.Site {
	s := ab.Site{
		SupportedLanguages: []string{language.English.String()},
	}

	s.Directories.Public, _ = ioutil.TempDir("", "abtest_public_")
	s.Directories.Private, _ = ioutil.TempDir("", "abtest_private_")

	removeDirectories = append(removeDirectories, s.Directories.Public, s.Directories.Private)
	registerCleanup()

	return s
}

func dbConfig() dbmw.DBConfig {
	return dbmw.DBConfig{
		ConnectionString: os.Getenv("AB_TEST_DB"),
	}
}

func sessionConfig() sessionmw.Config {
	return sessionmw.Config{
		Key:       hex.EncodeToString(FakeKey),
		CookieURL: "/",
	}
}

func registerCleanup() {
	if cleanupRegistered {
		return
	}

	AfterSuite(func() {
		for _, dir := range removeDirectories {
			os.RemoveAll(dir)
		}

		conn := Connect("")
		for _, schema := range schemas {
			conn.Exec("DROP SCHEMA " + schema + " CASCADE;")
		}
	})

	cleanupRegistered = true
}

// Connect connects to the test database.
//
// If schema is specified, then it sets the search_path. If you want to leave the search_path on its default value,
// pass an empty string as the schema.
func Connect(schema string) db.DB {
	connstr := os.Getenv("AB_TEST_DB")
	conn, err := db.ConnectToDB(connstr)
	conn.SetConnMaxLifetime(120 * time.Second)
	conn.SetMaxIdleConns(1)
	conn.SetMaxOpenConns(1)
	if err != nil {
		panic(err)
	}
	if schema != "" {
		setSearchPath(conn, schema)
	}

	return conn
}

func setSearchPath(conn db.DB, schema string) {
	if _, err := conn.Exec("SET search_path = " + schema + ", public;"); err != nil {
		panic(err)
	}
}
