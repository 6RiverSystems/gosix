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

package pubsubmiddleware

import (
	"bytes"
	"context"
	"encoding/json"
	"reflect"

	"github.com/getkin/kin-openapi/openapi3"

	"go.6river.tech/gosix/logging"
	"go.6river.tech/gosix/pubsub"
)

type ErrorHandler func(context.Context, pubsub.Message, error) (ackMessage bool)
type FanoutErrorHandler func(context.Context, pubsub.Message, error) pubsub.RetriableError

// TODO: the ParsedMessageHandler types won't play nice with oapi-codegen, as it
// can't deserialize from map[string]interface{}

type ParsedMessageHandler func(context.Context, pubsub.Message, interface{})
type ParsedFanoutMessageHandler func(context.Context, pubsub.Message, interface{}) pubsub.RetriableError

// WithValidation wraps a message handler with OAS validation. Messages are
// assumed to be JSON. The inner messageHandler is required, but errorHandler
// may be nil, in which case all messages that fail validation will be logged
// and Ack()'d. Callers are encouraged to provide a customized handler so that
// the component name associated with the logging is identifiable.
func WithValidation(
	schema *openapi3.Schema,
	messageHandler ParsedMessageHandler,
	errorHandler ErrorHandler,
) pubsub.MessageHandler {
	if errorHandler == nil {
		errorHandler = DefaultErrorHandler(schema, logging.GetLogger(DefaultValidationLoggingComponent))
	}
	errorAction := func(ctx context.Context, msg pubsub.Message, err error) {
		if errorHandler(ctx, msg, err) {
			msg.Ack()
		} else {
			msg.Nack()
		}
	}
	// compare to implementation of openapi3filter.ValidateRequestBody
	return func(ctx context.Context, msg pubsub.Message) {
		reader := bytes.NewReader(msg.RealMessage().Data)
		decoder := json.NewDecoder(reader)
		var value interface{}
		if err := decoder.Decode(&value); err != nil {
			errorAction(ctx, msg, err)
			return
		}

		if err := schema.VisitJSON(value, openapi3.VisitAsRequest(), openapi3.MultiErrors()); err != nil {
			errorAction(ctx, msg, err)
			return
		}

		messageHandler(ctx, msg, value)
	}
}

// WithFanoutValidation wraps a fanout message handler with OAS validation.
// Messages are assumed to be JSON. The inner messageHandler is required, but
// errorHandler may be nil, in which case all messages that fail validation will
// be logged and the error re-classified as a non-retriable error. Callers are
// encouraged to provide a customized handler so that the component name
// associated with the logging is identifiable.
func WithFanoutValidation(
	schema *openapi3.Schema,
	messageHandler ParsedFanoutMessageHandler,
	errorHandler FanoutErrorHandler,
) pubsub.FanoutMessageHandler {
	if errorHandler == nil {
		errorHandler = DefaultFanoutErrorHandler(schema, logging.GetLogger(DefaultValidationLoggingComponent))
	}
	return func(ctx context.Context, msg pubsub.Message) pubsub.RetriableError {
		reader := bytes.NewReader(msg.RealMessage().Data)
		decoder := json.NewDecoder(reader)
		var value interface{}
		if err := decoder.Decode(&value); err != nil {
			return errorHandler(ctx, msg, err)
		}

		if err := schema.VisitJSON(value, openapi3.VisitAsRequest(), openapi3.MultiErrors()); err != nil {
			return errorHandler(ctx, msg, err)
		}

		return messageHandler(ctx, msg, value)
	}
}

// RouteBy uses a classifier function to chose exactly one handler to process a
// message. If no handler matches the classifier return, routing handler will
// return nil (no error). This is the same idea as pubsub.RouteBy, but for the
// parsed variant of handlers.
func RouteBy(
	classifier func(pubsub.Message, interface{}) string,
	handlers map[string]ParsedFanoutMessageHandler,
) ParsedFanoutMessageHandler {
	return func(ctx context.Context, m pubsub.Message, value interface{}) pubsub.RetriableError {
		class := classifier(m, value)
		if handler := handlers[class]; handler != nil {
			return handler(ctx, m, value)
		} else {
			// TODO: log
			return nil
		}
	}
}

// ValueParser is the type of the parser argument to WithParser. Functions of
// this type should return a replacement parsed value for message handlers given
// the input message and value, or an error if parsing fails.
type ValueParser func(context.Context, pubsub.Message, interface{}) (interface{}, error)

// WithParser wraps a handler with a message parser, replacing the interface{}
// value passed to the inner handler with that produced by the parser. This
// might be used, for example to replace the generic map[string]interface{} from
// JSON parsing done for validation with a strongly typed struct specific to the
// message. Any error from parser is always interpreted as non-retriable, even
// if an error implementing RetriableError is returned!
func WithParser(
	parser ValueParser,
	handler ParsedFanoutMessageHandler,
) ParsedFanoutMessageHandler {
	return func(ctx context.Context, m pubsub.Message, value interface{}) pubsub.RetriableError {
		parsed, err := parser(ctx, m, value)
		if err != nil {
			// treat all parsing errors as non-retriable
			return pubsub.AsRetriable(err, false)
		}
		return handler(ctx, m, parsed)
	}
}

// ParserFor generates a reflect-driven parser function for the type of the
// given sample value, which must be a pointer type.
func ParserFor(
	value interface{},
) ValueParser {
	t := reflect.ValueOf(value).Type().Elem()
	return func(_ context.Context, m pubsub.Message, _ interface{}) (interface{}, error) {
		value := reflect.New(t).Interface()
		err := json.Unmarshal(m.RealMessage().Data, value)
		return value, err
	}
}

func DefaultErrorHandler(schema *openapi3.Schema, logger *logging.Logger) ErrorHandler {
	return func(_ context.Context, msg pubsub.Message, err error) (ackMessage bool) {
		logger.Info().
			Str("messageID", msg.RealMessage().ID).
			Err(err).
			Msg("PubSub message failed schema validation, dropping")
		return true
	}
}

func DefaultFanoutErrorHandler(schema *openapi3.Schema, logger *logging.Logger) FanoutErrorHandler {
	return func(_ context.Context, msg pubsub.Message, err error) pubsub.RetriableError {
		logger.Info().
			Str("messageID", msg.RealMessage().ID).
			Err(err).
			Msg("PubSub message failed schema validation, dropping")

		return pubsub.AsRetriable(err, false)
	}
}

const DefaultValidationLoggingComponent = "pubsub/validation"
