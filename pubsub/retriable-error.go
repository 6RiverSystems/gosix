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
