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

package event_test

import (
	"github.com/alien-bunny/ab/lib/errors"
	"github.com/alien-bunny/ab/lib/event"
	. "github.com/onsi/ginkgo"
	. "github.com/onsi/gomega"
)

const eventName = "test"

var _ = Describe("Event", func() {
	It("should subscribe a simple subscriber", func() {
		d := event.NewDispatcher()
		err := d.Subscribe(eventName, &simpleSubscriber{})
		Expect(err).NotTo(HaveOccurred())
	})

	It("should dispatch an event", func() {
		d := event.NewDispatcher()
		var executed bool
		err := d.Subscribe(eventName, event.SubscriberFunc(func(e event.Event) error {
			executed = true
			return nil
		}))
		Expect(err).NotTo(HaveOccurred())

		errs := d.Dispatch(&simpleEvent{
			name:       eventName,
			strategy:   event.ErrorStrategyIgnore,
			shouldFail: false,
		})
		Expect(errs).To(BeEmpty())
		Expect(executed).To(BeTrue())
	})

	It("should only dispatch the event when the subscriber is subscribed to that event", func() {
		d := event.NewDispatcher()
		var executed bool
		err := d.Subscribe("asdf", event.SubscriberFunc(func(e event.Event) error {
			executed = true
			return nil
		}))
		Expect(err).NotTo(HaveOccurred())

		errs := d.Dispatch(&simpleEvent{
			name:       eventName,
			strategy:   event.ErrorStrategyIgnore,
			shouldFail: false,
		})
		Expect(errs).To(BeEmpty())
		Expect(executed).To(BeFalse())
	})

	It("should ignore errors when specified", func() {
		d := event.NewDispatcher()

		err := d.Subscribe(eventName, &simpleSubscriber{})
		Expect(err).NotTo(HaveOccurred())

		errs := d.Dispatch(&simpleEvent{
			name:       eventName,
			strategy:   event.ErrorStrategyIgnore,
			shouldFail: true,
		})
		Expect(errs).To(BeEmpty())
	})

	It("should stop at an error when specified", func() {
		d := event.NewDispatcher()

		err := d.Subscribe(eventName, &simpleSubscriber{})
		Expect(err).NotTo(HaveOccurred())

		var executed bool
		err = d.Subscribe(eventName, event.SubscriberFunc(func(e event.Event) error {
			executed = true
			return nil
		}))
		Expect(err).NotTo(HaveOccurred())

		errs := d.Dispatch(&simpleEvent{
			name:       eventName,
			strategy:   event.ErrorStrategyStop,
			shouldFail: true,
		})
		Expect(errs).To(HaveLen(1))
		Expect(executed).To(BeFalse())
	})

	It("should collect errors when specified", func() {
		d := event.NewDispatcher()

		err := d.Subscribe(eventName, &simpleSubscriber{})
		Expect(err).NotTo(HaveOccurred())

		err = d.Subscribe(eventName, &simpleSubscriber{})
		Expect(err).NotTo(HaveOccurred())

		errs := d.Dispatch(&simpleEvent{
			name:       eventName,
			strategy:   event.ErrorStrategyAggregate,
			shouldFail: true,
		})
		Expect(errs).To(HaveLen(2))
	})
})

var _ event.Event = &simpleEvent{}

type simpleEvent struct {
	name       string
	strategy   event.ErrorStrategy
	shouldFail bool
}

func (e *simpleEvent) Name() string {
	return e.name
}

func (e *simpleEvent) ErrorStrategy() event.ErrorStrategy {
	return e.strategy
}

var _ event.Subscriber = &simpleSubscriber{}

type simpleSubscriber struct{}

func (s *simpleSubscriber) Handle(e event.Event) error {
	se := e.(*simpleEvent)

	if se.shouldFail {
		return errors.New("event failed")
	}

	return nil
}
