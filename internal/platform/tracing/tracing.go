// Package tracing carries trace context through hello and, when an OTLP
// endpoint is configured, exports its spans. Generated once by vikrant; this
// file belongs to the service team.
//
// The mesh's sidecars record a span for every hop, but only the application
// knows which outbound call an inbound request caused. Propagation is how
// it says so: the server handler reads the caller's W3C traceparent, and
// every call made with the handler's context sends it on. Without that,
// each hop through this service starts a new trace.
package tracing

import (
	"context"
	"os"

	"go.opentelemetry.io/otel"
	"go.opentelemetry.io/otel/attribute"
	"go.opentelemetry.io/otel/exporters/otlp/otlptrace/otlptracegrpc"
	"go.opentelemetry.io/otel/propagation"
	"go.opentelemetry.io/otel/sdk/resource"
	sdktrace "go.opentelemetry.io/otel/sdk/trace"
)

// Setup installs the W3C propagator, always, and a batching OTLP exporter
// when OTEL_EXPORTER_OTLP_ENDPOINT (or ..._TRACES_ENDPOINT) is set; the
// exporter reads the other standard OTEL_* variables itself. Sampling
// follows the caller's decision, which in the mesh is the sidecar's.
// The returned function flushes buffered spans; call it on shutdown.
func Setup(ctx context.Context, service, version string) (shutdown func(context.Context) error, err error) {
	otel.SetTextMapPropagator(propagation.NewCompositeTextMapPropagator(propagation.TraceContext{}, propagation.Baggage{}))
	if os.Getenv("OTEL_EXPORTER_OTLP_ENDPOINT") == "" && os.Getenv("OTEL_EXPORTER_OTLP_TRACES_ENDPOINT") == "" {
		return func(context.Context) error { return nil }, nil
	}
	exp, err := otlptracegrpc.New(ctx)
	if err != nil {
		return nil, err
	}
	// Later detectors win: OTEL_SERVICE_NAME and OTEL_RESOURCE_ATTRIBUTES
	// override the defaults.
	res, err := resource.New(ctx,
		resource.WithAttributes(attribute.String("service.name", service), attribute.String("service.version", version)),
		resource.WithTelemetrySDK(),
		resource.WithFromEnv(),
	)
	if err != nil {
		return nil, err
	}
	tp := sdktrace.NewTracerProvider(sdktrace.WithBatcher(exp), sdktrace.WithResource(res))
	otel.SetTracerProvider(tp)
	return tp.Shutdown, nil
}
