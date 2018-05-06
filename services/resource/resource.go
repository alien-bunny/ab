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

package resource

import (
	"encoding/json"
	"fmt"
	"net/http"

	"github.com/alien-bunny/ab"
	"github.com/alien-bunny/ab/lib"
	"github.com/alien-bunny/ab/lib/db"
	"github.com/alien-bunny/ab/lib/errors"
	"github.com/alien-bunny/ab/lib/event"
	"github.com/alien-bunny/ab/lib/hal"
	"github.com/alien-bunny/ab/lib/middleware"
	"github.com/alien-bunny/ab/lib/render"
	"github.com/alien-bunny/ab/lib/server"
	"github.com/alien-bunny/ab/middlewares/dbmw"
	"github.com/lib/pq"
)

var ErrNoEndpoints = errors.New("no endpoints are enabled for this resource")

// Resource labels data for CRUD operation through API endpoints.
type Resource interface {
}

// ResourceList is an extended list of resources.
type ResourceList struct {
	Items    []Resource               `json:"items"`
	Page     int                      `json:"-"`
	PageSize int                      `json:"-"`
	BasePath string                   `json:"-"`
	Curies   []hal.HALCurie           `json:"-"`
	Rels     map[string][]interface{} `json:"-"`
}

func (rl *ResourceList) Sanitize() {
	for _, item := range rl.Items {
		if sanitizer, ok := item.(lib.Sanitizer); ok {
			sanitizer.Sanitize()
		}
	}
}

type resourceListHALJSON struct {
	Items []interface{}          `json:"items"`
	Links map[string]interface{} `json:"_links"`
}

func (rl *ResourceList) MarshalJSON() ([]byte, error) {
	items := make([]interface{}, len(rl.Items))
	for i, item := range rl.Items {
		if el, ok := item.(hal.EndpointLinker); ok {
			items[i] = hal.NewHalWrapper(el)
		} else {
			items[i] = item
		}
	}

	return json.Marshal(resourceListHALJSON{
		Items: items,
		Links: hal.CreateHALLinkList(rl.links(), rl.Curies),
	})
}

func (rl *ResourceList) links() map[string][]interface{} {
	if rl.Page > 1 {
		rl.Rels["page previous"] = append(rl.Rels["page previous"], fmt.Sprintf("%s?page=%d", rl.BasePath, rl.Page-1))
	}
	if len(rl.Items) == rl.PageSize {
		rl.Rels["page next"] = append(rl.Rels["page next"], fmt.Sprintf("%s?page=%d", rl.BasePath, rl.Page+1))
	}

	return rl.Rels
}

// ResourceListDelegate helps a ResourceController to list resources.
type ResourceListDelegate interface {
	List(r *http.Request, start, limit int) ([]Resource, error)
	PageLength() int
}

// ResourcePostDelegate helps a ResourceController to handle POST for a resource.
type ResourcePostDelegate interface {
	Empty() Resource
	Validate(data Resource, r *http.Request)
	Insert(data Resource, r *http.Request) error
}

// ResourceGetDelegate helps a ResourceController to handle GET for a resource.
type ResourceGetDelegate interface {
	Load(id string, r *http.Request) (Resource, error)
}

// ResourcePutDelegate helps a ResourceController to handle PUT for a resource.
type ResourcePutDelegate interface {
	Empty() Resource
	Load(id string, r *http.Request) (Resource, error)
	GetID(Resource) string
	Validate(data Resource, r *http.Request)
	Update(data Resource, r *http.Request) error
}

// ResourceDeleteDelegate helps a ResourceController to handle DELETE for a resource.
type ResourceDeleteDelegate interface {
	Load(id string, r *http.Request) (Resource, error)
	Delete(data Resource, r *http.Request) error
}

// ResourcePathOverrider can be implemented by a ResourceDelegate to change the path pattern for the given resource operation.
type ResourcePathOverrider interface {
	OverridePath(string) string
}

// ResourceFormatter formats resources for the HTTP response.
type ResourceFormatter interface {
	FormatSingle(Resource, *render.Renderer)
	FormatMulti(*ResourceList, *render.Renderer)
}

// ResourceControllerDelegate customizes a ResourceController.
type ResourceControllerDelegate interface {
	// GetName returns machine name of the resource
	GetName() string
	// GetTables returns a list of tables that this resource uses.
	// These will be automatically checked on service install.
	GetTables() []string
	// GetSchemaSQL returns the full schema for the resource.
	GetSchemaSQL() string
	// SchemaInstalled can be used to make extra checks to
	// ensure that the complete schema is installed.
	SchemaInstalled(db.DB) bool
}

var _ server.Service = &ResourceController{}

// ResourceController represents a CRUD service.
type ResourceController struct {
	ResourceFormatter
	dispatcher     *event.Dispatcher
	delegate       ResourceControllerDelegate
	errorConverter func(err *pq.Error) errors.Error

	listDelegate    ResourceListDelegate
	listMiddlewares []middleware.Middleware

	postDelegate    ResourcePostDelegate
	postMiddlewares []middleware.Middleware

	getDelegate    ResourceGetDelegate
	getMiddlewares []middleware.Middleware

	putDelegate    ResourcePutDelegate
	putMiddlewares []middleware.Middleware

	deleteDelegate    ResourceDeleteDelegate
	deleteMiddlewares []middleware.Middleware

	ExtraEndpoints func(s *server.Server) error
}

// NewResourceController creates a ResourceController with a given delegate and sensible defaults.
func NewResourceController(dispatcher *event.Dispatcher, delegate ResourceControllerDelegate) *ResourceController {
	return &ResourceController{
		ResourceFormatter: &DefaultResourceFormatter{},
		dispatcher:        dispatcher,
		delegate:          delegate,
		postMiddlewares:   []middleware.Middleware{dbmw.Begin()},
		putMiddlewares:    []middleware.Middleware{dbmw.Begin()},
		deleteMiddlewares: []middleware.Middleware{dbmw.Begin()},
		errorConverter: func(err *pq.Error) errors.Error {
			return errors.NewError(err.Message, err.Detail, nil)
		},
	}
}

// GetName returns the name of this ResourceController.
func (res *ResourceController) GetName() string {
	return res.delegate.GetName()
}

// List enables the listing endpoint.
func (res *ResourceController) List(d ResourceListDelegate, middlewares ...middleware.Middleware) *ResourceController {
	res.listDelegate = d
	res.listMiddlewares = middlewares

	return res
}

// Post enables the POST endpoint.
func (res *ResourceController) Post(d ResourcePostDelegate, middlewares ...middleware.Middleware) *ResourceController {
	res.postDelegate = d
	res.postMiddlewares = middlewares

	return res
}

// Get enables the GET endpoint.
func (res *ResourceController) Get(d ResourceGetDelegate, middlewares ...middleware.Middleware) *ResourceController {
	res.getDelegate = d
	res.getMiddlewares = middlewares

	return res
}

// Put enables the PUT endpoint.
func (res *ResourceController) Put(d ResourcePutDelegate, middlewares ...middleware.Middleware) *ResourceController {
	res.putDelegate = d
	res.putMiddlewares = middlewares

	return res
}

// Delete enables the DELETE endpoint.
func (res *ResourceController) Delete(d ResourceDeleteDelegate, middlewares ...middleware.Middleware) *ResourceController {
	res.deleteDelegate = d
	res.deleteMiddlewares = middlewares

	return res
}

func (res *ResourceController) convertError(err error) error {
	return db.ConvertDBError(err, res.errorConverter)
}

func (res *ResourceController) listHandler(w http.ResponseWriter, r *http.Request) {
	limit := res.listDelegate.PageLength()
	start := ab.Pager(r, limit)

	errs := res.dispatcher.Dispatch(NewBeforeResourceListEvent(r))
	ab.MaybeFail(http.StatusInternalServerError, errors.NewMultiError(errs))

	list, err := res.listDelegate.List(r, start, limit)
	ab.MaybeFail(http.StatusInternalServerError, res.convertError(err))
	reslist := &ResourceList{
		Items:    list,
		PageSize: limit,
		Page:     start / limit,
		BasePath: "/api/" + res.delegate.GetName(),
	}

	errs = res.dispatcher.Dispatch(NewAfterResourceListEvent(r, reslist))
	ab.MaybeFail(http.StatusInternalServerError, errors.NewMultiError(errs))

	res.ResourceFormatter.FormatMulti(reslist, ab.Render(r))
}

func (res *ResourceController) postHandler(w http.ResponseWriter, r *http.Request) {
	d := res.postDelegate.Empty()
	ab.MustDecode(r, d)

	errs := res.dispatcher.Dispatch(NewResourceCRUDEvent(EventBeforeResourcePost, r, d))
	ab.MaybeFail(http.StatusInternalServerError, errors.NewMultiError(errs))

	res.postDelegate.Validate(d, r)

	if v, ok := d.(lib.Validator); ok {
		err := v.Validate()
		ab.MaybeFail(http.StatusBadRequest, err)
	}

	errs = res.dispatcher.Dispatch(NewResourceCRUDEvent(EventDuringResourcePost, r, d))
	ab.MaybeFail(http.StatusInternalServerError, errors.NewMultiError(errs))

	err := res.postDelegate.Insert(d, r)
	ab.MaybeFail(http.StatusInternalServerError, res.convertError(err))

	errs = res.dispatcher.Dispatch(NewResourceCRUDEvent(EventAfterResourcePost, r, d))
	ab.MaybeFail(http.StatusInternalServerError, errors.NewMultiError(errs))

	res.ResourceFormatter.FormatSingle(d, ab.Render(r).SetCode(http.StatusCreated))
}

func (res *ResourceController) getHandler(w http.ResponseWriter, r *http.Request) {
	id := server.GetParams(r).ByName("id")

	errs := res.dispatcher.Dispatch(NewResourceCRUDEvent(EventBeforeResourceGet, r, nil))
	ab.MaybeFail(http.StatusInternalServerError, errors.NewMultiError(errs))

	d, err := res.getDelegate.Load(id, r)
	ab.MaybeFail(http.StatusInternalServerError, res.convertError(err))
	if d == nil {
		ab.Fail(http.StatusNotFound, nil)
	}

	errs = res.dispatcher.Dispatch(NewResourceCRUDEvent(EventAfterResourceGet, r, d))
	ab.MaybeFail(http.StatusInternalServerError, errors.NewMultiError(errs))

	res.ResourceFormatter.FormatSingle(d, ab.Render(r))
}

func (res *ResourceController) putHandler(w http.ResponseWriter, r *http.Request) {
	id := server.GetParams(r).ByName("id")

	d := res.putDelegate.Empty()
	ab.MustDecode(r, d)

	errs := res.dispatcher.Dispatch(NewResourceCRUDEvent(EventBeforeResourcePut, r, d))
	ab.MaybeFail(http.StatusInternalServerError, errors.NewMultiError(errs))

	if res.putDelegate.GetID(d) != id {
		ab.Fail(http.StatusBadRequest, nil)
	}

	res.putDelegate.Validate(d, r)

	if v, ok := d.(lib.Validator); ok {
		err := v.Validate()
		ab.MaybeFail(http.StatusBadRequest, err)
	}

	errs = res.dispatcher.Dispatch(NewResourceCRUDEvent(EventDuringResourcePut, r, d))
	ab.MaybeFail(http.StatusInternalServerError, errors.NewMultiError(errs))

	err := res.putDelegate.Update(d, r)
	ab.MaybeFail(http.StatusInternalServerError, res.convertError(err))

	errs = res.dispatcher.Dispatch(NewResourceCRUDEvent(EventAfterResourcePut, r, d))
	ab.MaybeFail(http.StatusInternalServerError, errors.NewMultiError(errs))

	res.ResourceFormatter.FormatSingle(d, ab.Render(r))
}

func (res *ResourceController) deleteHandler(w http.ResponseWriter, r *http.Request) {
	id := server.GetParams(r).ByName("id")

	errs := res.dispatcher.Dispatch(NewResourceCRUDEvent(EventBeforeResourceDelete, r, nil))
	ab.MaybeFail(http.StatusInternalServerError, errors.NewMultiError(errs))

	d, err := res.deleteDelegate.Load(id, r)
	ab.MaybeFail(http.StatusInternalServerError, res.convertError(err))
	if d == nil {
		ab.Fail(http.StatusNotFound, nil)
	}

	errs = res.dispatcher.Dispatch(NewResourceCRUDEvent(EventDuringResourceDelete, r, d))
	ab.MaybeFail(http.StatusInternalServerError, errors.NewMultiError(errs))

	err = res.deleteDelegate.Delete(d, r)
	ab.MaybeFail(http.StatusInternalServerError, res.convertError(err))

	errs = res.dispatcher.Dispatch(NewResourceCRUDEvent(EventAfterResourceDelete, r, d))
	ab.MaybeFail(http.StatusInternalServerError, errors.NewMultiError(errs))
}

func (res *ResourceController) Register(srv *server.Server) error {
	if res.listDelegate == nil && res.postDelegate == nil && res.getDelegate == nil && res.putDelegate == nil && res.deleteDelegate == nil && res.ExtraEndpoints == nil {
		return ErrNoEndpoints
	}

	base := "/api/" + res.delegate.GetName()
	id := base + "/:id"

	if res.listDelegate != nil {
		path := base
		if po, ok := res.listDelegate.(ResourcePathOverrider); ok {
			path = po.OverridePath(path)
		}
		srv.Get(path, ab.WrapHandlerFunc(res.listHandler), res.listMiddlewares...)
	}

	if res.postDelegate != nil {
		path := base
		if po, ok := res.postDelegate.(ResourcePathOverrider); ok {
			path = po.OverridePath(path)
		}
		srv.Post(path, ab.WrapHandlerFunc(res.postHandler), res.postMiddlewares...)
	}

	if res.getDelegate != nil {
		path := id
		if po, ok := res.getDelegate.(ResourcePathOverrider); ok {
			path = po.OverridePath(path)
		}
		srv.Get(path, ab.WrapHandlerFunc(res.getHandler), res.getMiddlewares...)
	}

	if res.putDelegate != nil {
		path := id
		if po, ok := res.putDelegate.(ResourcePathOverrider); ok {
			path = po.OverridePath(path)
		}
		srv.Put(path, ab.WrapHandlerFunc(res.putHandler), res.putMiddlewares...)
	}

	if res.deleteDelegate != nil {
		path := id
		if po, ok := res.deleteDelegate.(ResourcePathOverrider); ok {
			path = po.OverridePath(path)
		}
		srv.Delete(path, ab.WrapHandlerFunc(res.deleteHandler), res.deleteMiddlewares...)
	}

	if res.ExtraEndpoints != nil {
		return res.ExtraEndpoints(srv)
	}
	return nil
}

func (res *ResourceController) SchemaInstalled(conn db.DB) bool {
	installed := true

	for _, table := range res.delegate.GetTables() {
		installed = installed && db.TableExists(conn, table)
	}

	return installed && res.delegate.SchemaInstalled(conn)
}

func (res *ResourceController) SchemaSQL() string {
	return res.delegate.GetSchemaSQL()
}
