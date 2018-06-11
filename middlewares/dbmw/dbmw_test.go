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

package dbmw_test

import (
	"net/http"
	"time"

	"github.com/alien-bunny/ab/lib/abtest"
	"github.com/alien-bunny/ab/lib/db"
	"github.com/alien-bunny/ab/lib/middleware"
	"github.com/alien-bunny/ab/lib/util"
	"github.com/alien-bunny/ab/middlewares/dbmw"
	. "github.com/onsi/ginkgo"
	. "github.com/onsi/gomega"
)

var _ = Describe("DB Middleware", func() {
	smw := abtest.NewSchemaMiddleware()

	_, conf, cmw := abtest.SetupConfigMiddleware()
	mw := dbmw.NewMiddleware(nil)
	mw.ConnectionMaxLifetime = 120 * time.Second
	mw.MaxOpenConnections = 1
	mw.MaxIdleConnections = 1
	tx := dbmw.Begin()

	stack := middleware.NewStack(nil)
	stack.Push(cmw)
	stack.Push(mw)
	stack.Push(smw)

	txStack := middleware.NewStack(nil)
	txStack.Push(cmw)
	txStack.Push(mw)
	txStack.Push(tx)
	txStack.Push(smw)

	conf.MaybeRegisterSchema(mw)

	It("should return the connection object in a handler and install the table", func() {
		abtest.TestMiddleware(stack, func(w http.ResponseWriter, r *http.Request) {
			conn := dbmw.GetConnection(r)
			Expect(conn).NotTo(BeNil())

			assertSearchPath(smw, conn)
			_, err := conn.Exec(`
				CREATE TABLE test(
					uuid uuid NOT NULL DEFAULT uuid_generate_v4(),
					data text NOT NULL,
					CONSTRAINT test_pkey PRIMARY KEY (uuid)
				);
			`)
			Expect(err).NotTo(HaveOccurred())
		})
	})

	It("should execute a query in a transaction", func() {
		text := util.RandomString(32)
		abtest.TestMiddleware(txStack, func(w http.ResponseWriter, r *http.Request) {
			conn := dbmw.GetConnection(r)
			Expect(conn).NotTo(BeNil())

			assertSearchPath(smw, conn)
			_, ierr := conn.Exec(`INSERT INTO test(data) VALUES($1)`, text)
			Expect(ierr).NotTo(HaveOccurred())
		})

		abtest.TestMiddleware(stack, func(w http.ResponseWriter, r *http.Request) {
			conn := dbmw.GetConnection(r)
			Expect(conn).NotTo(BeNil())

			assertSearchPath(smw, conn)

			var res int
			qerr := conn.QueryRow("SELECT COUNT(*) FROM test WHERE data = $1", text).Scan(&res)
			Expect(qerr).NotTo(HaveOccurred())
			Expect(res).To(Equal(1))
		})
	})

	It("should roll back the transaction when an error occours", func() {
		text := util.RandomString(32)
		Expect(func() {
			abtest.TestMiddleware(txStack, func(w http.ResponseWriter, r *http.Request) {
				conn := dbmw.GetConnection(r)
				Expect(conn).NotTo(BeNil())
				assertSearchPath(smw, conn)
				_, ierr := conn.Exec(`INSERT INTO test(data) VALUES($1)`, text)
				Expect(ierr).NotTo(HaveOccurred())
				panic("")
			})
		}).To(Panic())

		abtest.TestMiddleware(stack, func(w http.ResponseWriter, r *http.Request) {
			conn := dbmw.GetConnection(r)
			Expect(conn).NotTo(BeNil())

			assertSearchPath(smw, conn)

			var res int
			qerr := conn.QueryRow("SELECT COUNT(*) FROM test WHERE data = $1", text).Scan(&res)
			Expect(qerr).NotTo(HaveOccurred())
			Expect(res).To(Equal(0))
		})
	})

})

func assertSearchPath(smw *abtest.SchemaMiddleware, conn db.DB) {
	var path string
	err := conn.QueryRow("SHOW search_path").Scan(&path)
	Expect(err).NotTo(HaveOccurred())
	Expect(path).To(Equal(smw.GetSchemaName() + ", public"))
}
