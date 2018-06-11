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
	"github.com/alien-bunny/ab/lib/errors"
	"github.com/alien-bunny/ab/middlewares/dbmw"
	"github.com/lib/pq"
	. "github.com/onsi/ginkgo"
	. "github.com/onsi/gomega"
)

const errmsg = `
	Severity         ERROR
	Code             42P07
	Message          relation "migrationtest" already exists
	Detail           
	Hint             
	Position         
	InternalPosition 
	InternalQuery    
	Where            
	Schema           
	Table            
	Column           
	DataTypeName     
	Constraint       
	File             heap.c
	Line             1067
	Routine          heap_create_with_catalog
`

var _ = Describe("DB", func() {
	logger := abtest.GetLogger()
	conf := abtest.GetConfig(logger, "db")
	conf.MaybeRegisterSchema(dbmw.NewMiddleware(nil))
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
			smw := abtest.NewSchemaMiddleware()
			conn = abtest.Connect(smw.GetSchemaName())
			Expect(conn).NotTo(BeNil())
		})

		AfterEach(func() {
			Expect(conn.(*sql.DB).Close()).To(BeNil())
		})

		It("should be able to migrate simple instructions", func() {
			gens := db.DefineSchemaGenerations(
				func(conn db.DB) error {
					_, err := conn.Exec(`
						CREATE TABLE libdbtest(
							uuid uuid NOT NULL DEFAULT uuid_generate_v4(),
							CONSTRAINT libdbtest_pkey PRIMARY KEY (uuid)
						);
					`)
					return err
				},
				func(conn db.DB) error {
					_, err := conn.Exec(`
						INSERT INTO libdbtest(uuid) VALUES(uuid_generate_v4());
					`)
					return err
				},
				func(conn db.DB) error {
					cnt := 0
					err := conn.QueryRow(`SELECT COUNT(*) FROM libdbtest;`).Scan(&cnt)
					Expect(err).NotTo(HaveOccurred())
					Expect(cnt).To(Equal(1))
					return nil
				},
				func(conn db.DB) error {
					_, err := conn.Exec(`DELETE FROM libdbtest;`)
					return err
				},
			)

			version, err := gens.UpgradeFrom(-1, conn)
			Expect(err).NotTo(HaveOccurred())
			Expect(version).To(Equal(len(gens) - 1))
		})

		It("should handle migration errors", func() {
			gens := db.DefineSchemaGenerations(
				func(conn db.DB) error {
					_, err := conn.Exec(`
						CREATE TABLE migrationtest(
							uuid uuid NOT NULL DEFAULT uuid_generate_v4(),
							CONSTRAINT migrationtest_pkey PRIMARY KEY(uuid)
						);
					`)
					return err
				},
				func(conn db.DB) error {
					_, err := conn.Exec(`
						CREATE TABLE migrationtest();
					`)
					return err
				},
				func(conn db.DB) error {
					_, err := conn.Exec(`
						DROP TABLE migrationtest();
					`)
					return err
				},
			)

			version, err := gens.UpgradeFrom(-1, conn)
			Expect(err).To(HaveOccurred())
			Expect(db.DBErrorToVerboseString(err.(*pq.Error))).To(Equal(errmsg))
			Expect(version).To(Equal(0))
			Expect(db.ConvertDBError(err, db.ConstraintErrorConverter(map[string]string{
				"": "asdf",
			})).(errors.Error).UserError(nil)).To(Equal("asdf"))
			err.(*pq.Error).Constraint = "aaa"
			Expect(db.ConvertDBError(err, db.ConstraintErrorConverter(map[string]string{
				"": "asdf",
			})).(errors.Error).UserError(nil)).To(BeZero())

			_, err = conn.Exec(`DELETE FROM migrationtest;`)
			Expect(err).NotTo(HaveOccurred())
		})
	})
})

var _ = Describe("Error converter", func() {
	It("should return the error when it is not a db error", func() {
		err := errors.New("asdf")
		Expect(db.ConvertDBError(err, nil)).To(Equal(err))
	})
})
