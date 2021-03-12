package pubsub

import (
	"context"
	"os"
	"sync"

	"github.com/pkg/errors"
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
			panic(errors.Errorf("Can't guess pubsub project in production"))
		default:
			// assume development
			// TODO: detect production?
			envId = "development"
			// panic(errors.Errorf("Unrecognized environment '%s'", envName))
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
		panic(errors.Errorf("No default client initialized yet"))
	}
	return defaultClient
}
