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

// TableExists checks if a table exists in the database.
func TableExists(db DB, table string) bool {
	var found bool
	err := db.QueryRow("SELECT EXISTS(SELECT 1 FROM pg_catalog.pg_class c JOIN pg_catalog.pg_namespace n ON n.oid = c.relnamespace WHERE c.relname = $1 AND c.relkind = 'r');", table).Scan(&found)
	if err != nil {
		panic(err)
	}

	return found
}

// ConstraintExists checks if a constraint exists in the database
func ConstraintExists(db DB, constraint string) bool {
	var found bool
	err := db.QueryRow("SELECT EXISTS(SELECT 1 FROM pg_catalog.pg_constraint WHERE conname = $1)", constraint).Scan(&found)
	if err != nil {
		panic(err)
	}

	return found
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
