# Go Six

An experimental helper library for building idiomatic (to Go and 6RS) cloud
services.

## Developing

Updating generated code:

    go generate -x ./...

Build:

    go build -v ./...

Test:

    go test -race ./...

Or, using `make`:

    make # does everything

There are other `make` targets to run individual steps as well

## Using

See the 6RiverSystems/gosix-example repo for a simple app built using this
framework.
