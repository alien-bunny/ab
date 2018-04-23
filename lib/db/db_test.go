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

package db_test

import (
	"database/sql"

	"github.com/alien-bunny/ab/lib/abtest"
	"github.com/alien-bunny/ab/lib/db"
	"github.com/alien-bunny/ab/middlewares/dbmw"
	. "github.com/onsi/ginkgo"
	. "github.com/onsi/gomega"
)

var _ = Describe("DB", func() {
	logger := abtest.GetLogger()
	conf := abtest.GetConfig(logger, "db")
	conf.MaybeRegisterSchema(dbmw.NewMiddleware())
	dbconfig, err := conf.Get("db").Get("database")
	connstr := dbconfig.(dbmw.DBConfig).ConnectionString

	Specify("the connect string is set", func() {
		Expect(err).NotTo(HaveOccurred())
		Expect(connstr).NotTo(BeZero())
	})

	Describe("Connecting to the database and doing a few simple operations", func() {
		var conn db.DB

		BeforeEach(func() {
			By("Connecting to the database")
			var err error
			conn, err = db.ConnectToDB(connstr)
			Expect(err).To(BeNil())
			Expect(conn).NotTo(BeNil())
		})

		AfterEach(func() {
			Expect(conn.(*sql.DB).Close()).To(BeNil())
		})

		It("should not detect a table that does not exists", func() {
			exists := db.TableExists(conn, "zxcvbn")
			Expect(exists).To(BeFalse())
		})

		It("should detect existing tables and constraints", func() {
			By("creating a table with a primary key")
			_, err := conn.Exec(`
				CREATE TABLE libdbtest(
					uuid uuid NOT NULL DEFAULT uuid_generate_v4(),
					CONSTRAINT libdbtest_pkey PRIMARY KEY (uuid)
				);
			`)
			Expect(err).To(BeNil())
			defer conn.Exec("DROP TABLE libdbtest;")

			By("checking that the table exists")
			exists := db.TableExists(conn, "libdbtest")
			Expect(exists).To(BeTrue())

			By("checking that the constraint exists")
			exists = db.ConstraintExists(conn, "libdbtest_pkey")
			Expect(exists).To(BeTrue())
		})
	})
})
