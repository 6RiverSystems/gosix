package grpc

import (
	"context"
	"testing"

	"google.golang.org/grpc"

	"go.6river.tech/gosix/faults"
)

func unaryNoOp(context.Context, interface{}) (interface{}, error) {
	return nil, nil
}

func Benchmark_UnaryFaultInjection_empty_miss(b *testing.B) {
	ctx := context.Background()
	req := struct{ Data string }{
		Data: "xyzzy",
	}
	info := &grpc.UnaryServerInfo{
		FullMethod: "/foo.bar.baz.Bat/Quux",
	}
	set := faults.NewSet(b.Name())
	injector := UnaryFaultInjector(set)
	b.ResetTimer()
	for n := 0; n < b.N; n++ {
		if ret, err := injector(ctx, req, info, unaryNoOp); err != nil {
			b.Fatal("check of empty set returned error")
		} else if ret != nil {
			b.Fatal("impossible return from unary no-op")
		}
	}
}
