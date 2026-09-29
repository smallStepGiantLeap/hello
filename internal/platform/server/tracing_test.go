package server

import (
	"context"
	"net"
	"testing"

	"github.com/prometheus/client_golang/prometheus"
	"go.opentelemetry.io/otel/trace"
	"google.golang.org/grpc"
	"google.golang.org/grpc/metadata"
	"google.golang.org/protobuf/types/known/emptypb"

	"github.com/smallStepGiantLeap/hello/client"
	"github.com/smallStepGiantLeap/hello/internal/platform/metrics"
	"github.com/smallStepGiantLeap/hello/internal/platform/tracing"
)

// A call that arrives with a W3C traceparent makes its outbound calls, via
// the canonical client, in the same trace. Without this the mesh records
// every hop through the service as a separate trace.
func TestTraceContextPropagates(t *testing.T) {
	if _, err := tracing.Setup(context.Background(), "test", "test"); err != nil {
		t.Fatal(err)
	}
	seen := make(chan trace.TraceID, 1)
	downstream := startRelay(t, func(ctx context.Context) error {
		seen <- trace.SpanContextFromContext(ctx).TraceID()
		return nil
	})
	cc, err := client.Dial(downstream)
	if err != nil {
		t.Fatal(err)
	}
	t.Cleanup(func() { cc.Close() })
	upstream := startRelay(t, func(ctx context.Context) error {
		return cc.Invoke(ctx, "/test.Relay/Call", &emptypb.Empty{}, &emptypb.Empty{})
	})

	const traceID = "4bf92f3577b34da6a3ce929d0e0e4736"
	ctx := metadata.AppendToOutgoingContext(context.Background(), "traceparent", "00-"+traceID+"-00f067aa0ba902b7-01")
	if err := dial(t, upstream).Invoke(ctx, "/test.Relay/Call", &emptypb.Empty{}, &emptypb.Empty{}); err != nil {
		t.Fatal(err)
	}
	if got := (<-seen).String(); got != traceID {
		t.Fatalf("downstream saw trace %s, want %s", got, traceID)
	}
}

// startRelay serves test.Relay/Call on the platform's server, calling h.
func startRelay(t *testing.T, h func(context.Context) error) string {
	t.Helper()
	lis, err := net.Listen("tcp", "127.0.0.1:0")
	if err != nil {
		t.Fatal(err)
	}
	srv := New(Defaults(), metrics.NewServer(prometheus.NewRegistry(), "test"), NewDrainer(0, nil))
	call := grpc.MethodDesc{
		MethodName: "Call",
		Handler: func(s any, ctx context.Context, dec func(any) error, in grpc.UnaryServerInterceptor) (any, error) {
			req := new(emptypb.Empty)
			if err := dec(req); err != nil {
				return nil, err
			}
			handle := func(ctx context.Context, _ any) (any, error) { return new(emptypb.Empty), h(ctx) }
			return in(ctx, req, &grpc.UnaryServerInfo{Server: s, FullMethod: "/test.Relay/Call"}, handle)
		},
	}
	srv.RegisterService(&grpc.ServiceDesc{ServiceName: "test.Relay", HandlerType: (*any)(nil), Methods: []grpc.MethodDesc{call}}, struct{}{})
	go srv.Serve(lis)
	t.Cleanup(srv.Stop)
	return lis.Addr().String()
}
