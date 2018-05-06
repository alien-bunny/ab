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
	"net/http"

	"github.com/alien-bunny/ab/lib/event"
)

const (
	EventBeforeResourceList   = "before-resource-list"
	EventAfterResourceList    = "after-resource-list"
	EventBeforeResourcePost   = "before-resource-post"
	EventDuringResourcePost   = "during-resource-post"
	EventAfterResourcePost    = "after-resource-post"
	EventBeforeResourceGet    = "before-resource-get"
	EventAfterResourceGet     = "after-resource-get"
	EventBeforeResourcePut    = "before-resource-put"
	EventDuringResourcePut    = "during-resource-put"
	EventAfterResourcePut     = "after-resource-put"
	EventBeforeResourceDelete = "before-resource-delete"
	EventDuringResourceDelete = "during-resource-delete"
	EventAfterResourceDelete  = "after-resource-delete"
)

type resourceEventBase struct {
	r *http.Request
}

func (e resourceEventBase) ErrorStrategy() event.ErrorStrategy {
	return event.ErrorStrategyAggregate
}

func (e resourceEventBase) Request() *http.Request {
	return e.r
}

type BeforeResourceListEvent struct {
	resourceEventBase
}

func (e *BeforeResourceListEvent) Name() string {
	return EventBeforeResourceList
}

func NewBeforeResourceListEvent(r *http.Request) *BeforeResourceListEvent {
	return &BeforeResourceListEvent{
		resourceEventBase{r: r},
	}
}

type AfterResourceListEvent struct {
	resourceEventBase
	list *ResourceList
}

func (e *AfterResourceListEvent) Name() string {
	return EventAfterResourceList
}

func (e *AfterResourceListEvent) List() *ResourceList {
	return e.list
}

func NewAfterResourceListEvent(r *http.Request, list *ResourceList) *AfterResourceListEvent {
	return &AfterResourceListEvent{
		resourceEventBase: resourceEventBase{r: r},
		list:              list,
	}
}

type ResourceCRUDEvent struct {
	resourceEventBase
	resource  Resource
	eventName string
}

func (e *ResourceCRUDEvent) Name() string {
	return e.eventName
}

func (e *ResourceCRUDEvent) Resource() Resource {
	return e.resource
}

func NewResourceCRUDEvent(eventName string, r *http.Request, resource Resource) *ResourceCRUDEvent {
	return &ResourceCRUDEvent{
		resourceEventBase: resourceEventBase{r},
		resource:          resource,
		eventName:         eventName,
	}
}
