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

package dbmw

import (
	"database/sql"
	"net/http"
	"reflect"
	"sync"
	"time"

	"github.com/alien-bunny/ab/lib/config"
	"github.com/alien-bunny/ab/lib/db"
	"github.com/alien-bunny/ab/lib/errors"
	"github.com/alien-bunny/ab/lib/event"
	"github.com/alien-bunny/ab/lib/middleware"
	"github.com/alien-bunny/ab/lib/server"
	"github.com/alien-bunny/ab/lib/util"
	"github.com/alien-bunny/ab/middlewares/configmw"
)

const (
	MiddlewareDependencyDB = "*dbmw.Middleware"

	dbConnectionKey = "abdb"
)

// GetConnection returns DB from the request context.
func GetConnection(r *http.Request) db.DB {
	return r.Context().Value(dbConnectionKey).(db.DB)
}

func connect(connectString string, maxIdleConnections, maxOpenConnections int, connMaxLifetime time.Duration) *sql.DB {
	conn := db.RetryDBConn(connectString, 10)
	conn.SetMaxIdleConns(maxIdleConnections)
	conn.SetMaxOpenConns(maxOpenConnections)
	conn.SetConnMaxLifetime(connMaxLifetime)

	return conn
}

type DBConfig struct {
	ConnectionString string
	SchemaVersions   map[string]int
}

func (c DBConfig) SchemaVersion(name string) int {
	if found, ok := c.SchemaVersions[name]; ok {
		return found
	}

	return -1
}

var _ middleware.Middleware = &Middleware{}
var _ config.ConfigSchemaProvider = &Middleware{}
var _ event.Subscriber = &Middleware{}

type Middleware struct {
	MaxIdleConnections    int
	MaxOpenConnections    int
	ConnectionMaxLifetime time.Duration

	mtx         sync.Mutex
	connections map[string]*sql.DB
	server      *server.Server
}

func NewMiddleware(s *server.Server) *Middleware {
	return &Middleware{
		server: s,
	}
}

func (m *Middleware) ensureConnectionsMap() {
	if m.connections == nil {
		m.connections = make(map[string]*sql.DB)
	}
}

func (m *Middleware) Wrap(next http.Handler) http.Handler {
	m.ensureConnectionsMap()
	return http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		confInterface, err := configmw.GetConfig(r).Get("database")
		if err != nil {
			errors.Fail(http.StatusInternalServerError, err)
		}
		if confInterface == nil {
			errors.Fail(http.StatusServiceUnavailable, errors.New("database config not found"))
		}

		conf := confInterface.(DBConfig)
		conn := m.getConnection(conf.ConnectionString)
		r = util.SetContext(r, dbConnectionKey, conn)

		next.ServeHTTP(w, r)
	})
}

func (m *Middleware) getConnection(connStr string) *sql.DB {
	var conn *sql.DB

	m.mtx.Lock()
	// connect() below might panic, but this lock must be unlocked, else the whole server will freeze up
	defer m.mtx.Unlock()

	if conn = m.connections[connStr]; conn == nil {
		conn = connect(connStr, m.MaxIdleConnections, m.MaxOpenConnections, m.ConnectionMaxLifetime)
		m.connections[connStr] = conn
	}

	return conn
}

func (m *Middleware) Close() {
	m.mtx.Lock()
	defer m.mtx.Unlock()

	for _, conn := range m.connections {
		conn.Close()
	}

	// reset connection map
	m.connections = nil
	m.ensureConnectionsMap()
}

func (m *Middleware) Connections() int {
	m.mtx.Lock()
	count := len(m.connections)
	m.mtx.Unlock()

	return count
}

func (m *Middleware) Dependencies() []string {
	return []string{
		configmw.MiddlewareDependencyConfig,
	}
}

func (m *Middleware) ConfigSchema() map[string]reflect.Type {
	return map[string]reflect.Type{
		"database": reflect.TypeOf(DBConfig{}),
	}
}

func (m *Middleware) Handle(e event.Event) error {
	r := e.(requester).Request()
	confInterface, saver, err := configmw.GetWritableConfig(r).GetWritable("database")
	if err != nil {
		return err
	}
	if confInterface == nil {
		return errors.New("database config not found")
	}

	conf := confInterface.(DBConfig)
	if conf.SchemaVersions == nil {
		conf.SchemaVersions = make(map[string]int)
	}

	conn := GetConnection(r)

	for _, svc := range m.server.GetServices() {
		if p, ok := svc.(db.DBSchemaProvider); ok {
			name := p.Name()
			version := conf.SchemaVersion(name)
			newVersion, err := p.DBSchema().UpgradeFrom(version, conn)

			conf.SchemaVersions[name] = newVersion
			if errs := saver.Save(conf); errs != nil {
				return errs
			}

			if err != nil {
				return err
			}
		}
	}

	return nil
}

type requester interface {
	Request() *http.Request
}

var _ middleware.Middleware = &TransactionMiddleware{}

// TransactionMiddleware turns the DB connection in the context into a transaction.
//
// The transaction gets committed automatically, or rolled back if an error occours.
type TransactionMiddleware struct {
}

func Begin() *TransactionMiddleware {
	return &TransactionMiddleware{}
}

func (t *TransactionMiddleware) Wrap(next http.Handler) http.Handler {
	return http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		conn := GetConnection(r)
		var tx *sql.Tx
		var err error
		if dbconn, ok := conn.(*sql.DB); ok {
			tx, err = dbconn.Begin()
			if err != nil {
				errors.Fail(http.StatusInternalServerError, err)
			}
			defer tx.Rollback()
			r = util.SetContext(r, dbConnectionKey, tx)
		}

		next.ServeHTTP(w, r)

		if tx != nil {
			tx.Commit()
		}
	})
}

func (t *TransactionMiddleware) Dependencies() []string {
	return []string{MiddlewareDependencyDB}
}
