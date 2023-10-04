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
	"fmt"
	"os"
	"sync"

	"github.com/prometheus/client_golang/prometheus"
)

const (
	DefaultProjectIdEnvVar = "PUBSUB_GCLOUD_PROJECT_ID"
	DefaultEmulatorHost    = "localhost:8802"
	EmulatorHostEnvVar     = "PUBSUB_EMULATOR_HOST"
)

func DefaultProjectId() string {
	envId := os.Getenv(DefaultProjectIdEnvVar)
	if envId == "" {
		envName := os.Getenv("NODE_ENV")
		switch envName {
		case "test", "acceptance":
			envId = "test"
		case "development":
			envId = "development"
		case "production":
			panic(fmt.Errorf("Can't guess pubsub project in production"))
		default:
			// assume development
			// TODO: detect production?
			envId = "development"
			// panic(fmt.Errorf("Unrecognized environment '%s'", envName))
		}
	}
	return envId
}

// TODO: replace this default client gunk with a proper DI system

var (
	defaultClient     Client
	defaultClientErr  error
	defaultClientOnce sync.Once
)

func DefaultClient(promNamespace string) (Client, error) {
	defaultClientOnce.Do(func() {
		defaultClient, defaultClientErr = NewClient(
			context.Background(),
			"",
			prometheus.DefaultRegisterer,
			promNamespace,
			nil,
		)
	})
	return defaultClient, defaultClientErr
}

func MustDefaultClient() Client {
	if defaultClientErr != nil {
		panic(defaultClientErr)
	}
	if defaultClient == nil {
		panic(fmt.Errorf("No default client initialized yet"))
	}
	return defaultClient
}
