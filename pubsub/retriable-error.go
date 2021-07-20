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

import "errors"

type RetriableError interface {
	error
	IsRetriable() bool
}

func AsRetriable(err error, retriable bool) RetriableError {
	return &wrappedRetriableError{
		err:       err,
		retriable: retriable,
	}
}

type wrappedRetriableError struct {
	err       error
	retriable bool
}

var _ RetriableError = &wrappedRetriableError{}

func (e *wrappedRetriableError) IsRetriable() bool {
	return e.retriable
}

func (e *wrappedRetriableError) Error() string {
	return e.err.Error()
}

func (e *wrappedRetriableError) As(target interface{}) bool {
	return errors.As(e.err, target)
}

func (e *wrappedRetriableError) Is(target error) bool {
	return errors.Is(e.err, target)
}

func (e *wrappedRetriableError) Unwrap() error {
	return errors.Unwrap(e.err)
}
