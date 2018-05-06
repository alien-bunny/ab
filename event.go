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

package ab

import (
	"net/http"

	"github.com/alien-bunny/ab/lib/event"
)

// CacheClearEvent fires when some cache should be cleared.
type CacheClearEvent struct{}

// Name of the event. Always returns EventCacheClear.
func (e *CacheClearEvent) Name() string {
	return EventCacheClear
}

// ErrorStrategy of the event. Always returns event.ErrorStrategyAggregate.
func (e *CacheClearEvent) ErrorStrategy() event.ErrorStrategy {
	return event.ErrorStrategyAggregate
}

// InstallEvent fires after the site is installed.
type InstallEvent struct {
	r *http.Request
}

// NewInstallEvent constructs an InstallEvent.
func NewInstallEvent(r *http.Request) *InstallEvent {
	return &InstallEvent{
		r: r,
	}
}

// Request returns the current request.
func (e *InstallEvent) Request() *http.Request {
	return e.r
}

// Name of the event. Always returns EventInstall.
func (e *InstallEvent) Name() string {
	return EventInstall
}

// Name of the event. Always returns event.ErrorStrategyStop.
func (e *InstallEvent) ErrorStrategy() event.ErrorStrategy {
	return event.ErrorStrategyStop
}
