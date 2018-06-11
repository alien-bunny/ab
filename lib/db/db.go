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

package db

import (
	"database/sql"
	"fmt"
	"net"
	"time"

	"github.com/alien-bunny/ab/lib/errors"
	"github.com/lib/pq"
)

type DBSchemaProvider interface {
	Name() string
	DBSchema() SchemaGenerations
}

type Schema func(conn DB) error

func DefineSchemaGenerations(gens ...Schema) SchemaGenerations {
	return SchemaGenerations(gens)
}

type SchemaGenerations []Schema

func (g SchemaGenerations) UpgradeFrom(last int, conn DB) (int, error) {
	for next := last + 1; next < len(g); next++ {
		if err := g[next](conn); err != nil {
			return next - 1, err
		}
	}

	return len(g) - 1, nil
}

// DB is an abstraction over *sql.DB and *sql.Tx
type DB interface {
	Exec(string, ...interface{}) (sql.Result, error)
	Query(string, ...interface{}) (*sql.Rows, error)
	QueryRow(string, ...interface{}) *sql.Row
	Prepare(string) (*sql.Stmt, error)
}

func ConnectToDB(connectString string) (*sql.DB, error) {
	return sql.Open("postgres", connectString)
}

func RetryDBConn(connectString string, tries uint) *sql.DB {
	conn, err := ConnectToDB(connectString)
	if err != nil {
		if operr, ok := err.(*net.OpError); ok && operr.Op == "dial" && tries > 0 {
			<-time.After(time.Second)
			return RetryDBConn(connectString, tries-1)
		}
		panic(err)
	}

	return conn
}

// ConvertDBError converts an error with conv if that error is *pq.Error.
//
// Useful when processing database errors (e.g. constraint violations), so the user can get a nice error message.
func ConvertDBError(err error, conv func(*pq.Error) errors.Error) error {
	if err == nil {
		return nil
	}

	if perr, ok := err.(*pq.Error); ok {
		return conv(perr)
	}

	return err
}

// ConstraintErrorConverter converts a constraint violation error into a user-friendly message.
func ConstraintErrorConverter(msgMap map[string]string) func(*pq.Error) errors.Error {
	return func(err *pq.Error) errors.Error {
		if msg, ok := msgMap[err.Constraint]; ok {
			return errors.Wrap(err, msg, nil)
		}

		return errors.NewError(err.Message, err.Detail, nil)
	}
}

// DBErrorToVerboseString is a helper function that converts a *pq.Error into a detailed string.
func DBErrorToVerboseString(err *pq.Error) string {
	return fmt.Sprintf(`
	Severity         %s
	Code             %s
	Message          %s
	Detail           %s
	Hint             %s
	Position         %s
	InternalPosition %s
	InternalQuery    %s
	Where            %s
	Schema           %s
	Table            %s
	Column           %s
	DataTypeName     %s
	Constraint       %s
	File             %s
	Line             %s
	Routine          %s
`,
		err.Severity,
		err.Code,
		err.Message,
		err.Detail,
		err.Hint,
		err.Position,
		err.InternalPosition,
		err.InternalQuery,
		err.Where,
		err.Schema,
		err.Table,
		err.Column,
		err.DataTypeName,
		err.Constraint,
		err.File,
		err.Line,
		err.Routine,
	)
}
