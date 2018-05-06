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

package resource_test

import (
	"database/sql"
	"net/http"
	"testing"
	"time"

	"github.com/alien-bunny/ab"
	"github.com/alien-bunny/ab/lib/abtest"
	"github.com/alien-bunny/ab/lib/config"
	"github.com/alien-bunny/ab/lib/db"
	"github.com/alien-bunny/ab/lib/event"
	"github.com/alien-bunny/ab/lib/server"
	"github.com/alien-bunny/ab/lib/uuid"
	"github.com/alien-bunny/ab/services/resource"
	. "github.com/onsi/ginkgo"
	. "github.com/onsi/gomega"
)

func TestResource(t *testing.T) {
	RegisterFailHandler(Fail)
	RunSpecs(t, "Resource Suite")
}

var _, clientFactory = abtest.HopMock(func(conf *config.Store, s *server.Server, dispatcher *event.Dispatcher, base, schema string) (abtest.DataMockerFunc, error) {
	d := &testResourceControllerDelegate{}

	updatedSubscriber := event.SubscriberFunc(func(e event.Event) error {
		tr := e.(*resource.ResourceCRUDEvent).Resource().(*testResource)
		tr.Updated = time.Now()

		return nil
	})

	dispatcher.Subscribe(resource.EventBeforeResourcePost, updatedSubscriber)
	dispatcher.Subscribe(resource.EventBeforeResourcePut, updatedSubscriber)

	rc := resource.NewResourceController(dispatcher, d).
		List(d).
		Post(d).
		Get(d).
		Put(d).
		Delete(d)

	s.RegisterService(rc)

	return nil, nil
})

type testResource struct {
	UUID    uuid.UUID
	A       string
	B       int
	Updated time.Time
}

var _ resource.ResourceControllerDelegate = &testResourceControllerDelegate{}
var _ resource.ResourceListDelegate = &testResourceControllerDelegate{}
var _ resource.ResourcePostDelegate = &testResourceControllerDelegate{}
var _ resource.ResourceGetDelegate = &testResourceControllerDelegate{}
var _ resource.ResourcePutDelegate = &testResourceControllerDelegate{}
var _ resource.ResourceDeleteDelegate = &testResourceControllerDelegate{}

type testResourceControllerDelegate struct {
	conn db.DB
}

func (t *testResourceControllerDelegate) GetName() string {
	return "test"
}

func (t *testResourceControllerDelegate) GetTables() []string {
	return []string{"testresource"}
}

func (t *testResourceControllerDelegate) GetSchemaSQL() string {
	return `
		CREATE TABLE testresource(
			uuid uuid NOT NULL DEFAULT uuid_generate_v4() PRIMARY KEY,
			a text NOT NULL,
			b int NOT NULL,
			updated timestamp with time zone NOT NULL
		);
	`
}

func (t *testResourceControllerDelegate) SchemaInstalled(conn db.DB) bool {
	return true
}

func (t *testResourceControllerDelegate) List(r *http.Request, start, limit int) ([]resource.Resource, error) {
	conn := ab.GetDB(r)
	rows, rerr := conn.Query("SELECT uuid, a, b, updated FROM testresource ORDER BY updated DESC LIMIT $2 OFFSET $1", start, limit)
	if rerr != nil {
		return []resource.Resource{}, rerr
	}

	ret := make([]resource.Resource, 0, limit)

	defer rows.Close()

	for rows.Next() {
		tr := &testResource{}
		if serr := rows.Scan(&tr.UUID, &tr.A, &tr.B, &tr.Updated); serr != nil {
			return []resource.Resource{}, serr
		}
		ret = append(ret, tr)
	}

	return ret, nil
}

func (t *testResourceControllerDelegate) PageLength() int {
	return 6
}

func (t *testResourceControllerDelegate) Empty() resource.Resource {
	return &testResource{}
}

func (t *testResourceControllerDelegate) Validate(data resource.Resource, r *http.Request) {
}

func (t *testResourceControllerDelegate) Insert(data resource.Resource, r *http.Request) error {
	conn := ab.GetDB(r)
	tr := data.(*testResource)
	return conn.QueryRow(
		"INSERT INTO testresource(a, b, updated) VALUES($1, $2, $3) RETURNING uuid",
		tr.A,
		tr.B,
		tr.Updated,
	).Scan(&tr.UUID)
}

func (t *testResourceControllerDelegate) Load(id string, r *http.Request) (resource.Resource, error) {
	conn := ab.GetDB(r)
	tr := &testResource{}
	qerr := conn.QueryRow("SELECT uuid, a, b, updated FROM testresource WHERE uuid = $1", id).Scan(
		&tr.UUID, &tr.A, &tr.B, &tr.Updated,
	)
	if qerr != nil {
		if qerr == sql.ErrNoRows {
			return nil, nil
		}
		return nil, qerr
	}

	return tr, nil
}

func (t *testResourceControllerDelegate) GetID(data resource.Resource) string {
	return data.(*testResource).UUID.String()
}

func (t *testResourceControllerDelegate) Update(data resource.Resource, r *http.Request) error {
	conn := ab.GetDB(r)
	tr := data.(*testResource)
	_, eerr := conn.Exec("UPDATE testresource SET a = $2, b = $3, updated = $4 WHERE uuid = $1",
		tr.UUID, tr.A, tr.B, tr.Updated,
	)

	return eerr
}

func (t *testResourceControllerDelegate) Delete(data resource.Resource, r *http.Request) error {
	conn := ab.GetDB(r)
	tr := data.(*testResource)
	_, eerr := conn.Exec("DELETE FROM testresource WHERE uuid = $1", tr.UUID)
	return eerr
}
