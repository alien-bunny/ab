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

package event

// ErrorStrategy tells the dispatcher what to do when an error happens.
type ErrorStrategy int

const (
	// ErrorStrategyIgnore ignores the error. No error will be reported from Dispatch().
	ErrorStrategyIgnore ErrorStrategy = iota
	// ErrorStrategyStop stops at the first error.
	ErrorStrategyStop
	// ErrorStrategyAggregate collects the errors, but keeps processing the subscribers.
	ErrorStrategyAggregate
)

// Event represents the events that will be dispatched to the subscribers.
type Event interface {
	// Name returns the machine name of the event.
	// This function should be idempotent.
	Name() string
	// ErrorStrategy tells the dispatcher how to behave when an error happens.
	// This function should be idempotent.
	ErrorStrategy() ErrorStrategy
}

// Subscriber subscribes to an event.
type Subscriber interface {
	Handle(e Event) error
}

// SubscriberFunc is a simple subscriber that is a function.
type SubscriberFunc func(e Event) error

// Handle handles an event.
func (f SubscriberFunc) Handle(e Event) error {
	return f(e)
}

// Action is the simpliest event handler.
//
// It won't receive the event value, and it can't report an error.
type Action func()

// Handle handles an event.
func (a Action) Handle(e Event) error {
	a()
	return nil
}

// Dispatcher dispatches an event to its subscribers.
type Dispatcher struct {
	subscribers map[string][]Subscriber
}

// NewDispatcher creates a Dispatcher.
func NewDispatcher() *Dispatcher {
	return &Dispatcher{
		subscribers: make(map[string][]Subscriber),
	}
}

// Dispatch dispatches an event to the relevant subscribers.
func (d *Dispatcher) Dispatch(e Event) []error {
	var errors []error

	strategy := e.ErrorStrategy()

	for _, subscriber := range d.subscribers[e.Name()] {
		if err := subscriber.Handle(e); err != nil {
			switch strategy {
			case ErrorStrategyIgnore:
				continue
			case ErrorStrategyStop:
				return []error{err}
			case ErrorStrategyAggregate:
				errors = append(errors, err)
			}
		}
	}

	return errors
}

// Subscribe subscribes a subscriber to an event type.
func (d *Dispatcher) Subscribe(name string, s Subscriber) error {
	d.subscribers[name] = append(d.subscribers[name], s)

	return nil
}
