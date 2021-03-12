package grpc

import (
	"context"
	"errors"

	"google.golang.org/grpc/codes"
	"google.golang.org/grpc/status"
)

func AsStatusError(err error) error {
	if err == nil {
		return nil
	}
	if _, ok := err.(interface {
		GRPCStatus() *status.Status
	}); ok {
		// already a status error
		return err
	}
	code := codes.Unknown
	if errors.Is(err, context.Canceled) {
		code = codes.Canceled
	} else if errors.Is(err, context.DeadlineExceeded) {
		code = codes.DeadlineExceeded
	}
	return status.Error(code, err.Error())
}
