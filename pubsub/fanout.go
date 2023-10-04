// Copyright (c) 2021 6 River Systems
//
// Permission is hereby granted, free of charge, to any person obtaining a copy of
// this software and associated documentation files (the "Software"), to deal in
// the Software without restriction, including without limitation the rights to
// use, copy, modify, merge, publish, distribute, sublicense, and/or sell copies of
// the Software, and to permit persons to whom the Software is furnished to do so,
// subject to the following conditions:
//
// The above copyright notice and this permission notice shall be included in all
// copies or substantial portions of the Software.
//
// THE SOFTWARE IS PROVIDED "AS IS", WITHOUT WARRANTY OF ANY KIND, EXPRESS OR
// IMPLIED, INCLUDING BUT NOT LIMITED TO THE WARRANTIES OF MERCHANTABILITY, FITNESS
// FOR A PARTICULAR PURPOSE AND NONINFRINGEMENT. IN NO EVENT SHALL THE AUTHORS OR
// COPYRIGHT HOLDERS BE LIABLE FOR ANY CLAIM, DAMAGES OR OTHER LIABILITY, WHETHER
// IN AN ACTION OF CONTRACT, TORT OR OTHERWISE, ARISING FROM, OUT OF OR IN
// CONNECTION WITH THE SOFTWARE OR THE USE OR OTHER DEALINGS IN THE SOFTWARE.

package pubsub

import (
	"context"
	"errors"

	"golang.org/x/sync/errgroup"
)

type FanoutMessageHandler func(context.Context, Message) RetriableError

// Fanout creates a composite handler that will call each of the given handlers
// concurrently, and ack the message if all of them either succeed or return
// non-retriable errors, or else nack the message if any returns a retriable
// error.
//
// Fanout will panic if no handlers are given.
//
// Fanout will call all the handlers concurrently via an errgroup. Thus the
// context passed to each will be canceled if any other handler fails. It is up
// to the inner handlers to decide if they should also cancel in this case or
// not.
//
// The inner handlers _MUST NOT_ call `Ack()` or `Nack()` on the message!
//
// If an inner handler panics, it may cause messages to get stuck indefinitely.
func Fanout(handlers ...FanoutMessageHandler) MessageHandler {
	if len(handlers) == 0 {
		panic(errors.New("Must provide at least one handler in fanout"))
	}

	return func(ctx context.Context, m Message) {
		eg, egCtx := errgroup.WithContext(ctx)
		results := make([]RetriableError, len(handlers))
		for i, h := range handlers {
			i, h := i, h // don't close over loop vars
			eg.Go(func() error {
				err := h(egCtx, m)
				results[i] = err
				return err
			})
		}
		eg.Wait() // nolint:errcheck // checking errors via results array

		// TODO: logging

		anyErrors, anyRetriable := false, false
		for _, err := range results {
			if err != nil {
				anyErrors = true
				if err.IsRetriable() {
					anyRetriable = true
				}
			}
		}

		if anyErrors && !anyRetriable {
			// something went wrong but we can try again later
			m.Nack()
		} else {
			// nothing went wrong, or everything that did was non-retriable
			m.Ack()
		}
	}
}

// RouteBy uses a classifier function to chose exactly one handler to process a
// message. If no handler matches the classifier return, routing handler will
// return nil (no error).
func RouteBy(classifier func(Message) string, handlers map[string]FanoutMessageHandler) FanoutMessageHandler {
	return func(ctx context.Context, m Message) RetriableError {
		class := classifier(m)
		if handler := handlers[class]; handler != nil {
			return handler(ctx, m)
		} else {
			// TODO: log
			return nil
		}
	}
}
