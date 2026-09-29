// Command hello serves hello.v1.HelloService with
// the platform's hardened defaults, the gRPC health service, reflection, and
// Prometheus metrics on a separate port. Generated once by vikrant; this file
// belongs to the service team.
package main

import (
	"context"
	"errors"
	"flag"
	"log/slog"
	"net"
	"net/http"
	"os"
	"os/signal"
	"syscall"
	"time"

	"github.com/prometheus/client_golang/prometheus"
	"github.com/prometheus/client_golang/prometheus/promhttp"
	"google.golang.org/grpc/health"
	healthpb "google.golang.org/grpc/health/grpc_health_v1"
	"google.golang.org/grpc/reflection"

	hellov1 "github.com/smallStepGiantLeap/hello/client/gen/hello/v1"
	"github.com/smallStepGiantLeap/hello/internal/handlers"
	"github.com/smallStepGiantLeap/hello/internal/platform/metrics"
	"github.com/smallStepGiantLeap/hello/internal/platform/server"
	"github.com/smallStepGiantLeap/hello/internal/platform/tracing"
)

func main() {
	var (
		grpcAddr        = flag.String("grpc-addr", ":50051", "gRPC listen address")
		metricsAddr     = flag.String("metrics-addr", ":9090", "Prometheus /metrics listen address")
		version         = flag.String("version", envOr("VERSION", "dev"), "version reported in vikrant_build_info")
		drainDelay      = flag.Duration("drain-delay", 5*time.Second, "after SIGTERM, keep serving while reporting NOT_SERVING")
		drainSpread     = flag.Duration("drain-spread", 10*time.Second, "end open streams at random points within this window")
		shutdownTimeout = flag.Duration("shutdown-timeout", 20*time.Second, "after GOAWAY, how long RPCs may run before Stop cuts them")
	)
	opts := server.Defaults()
	opts.RegisterFlags(flag.CommandLine)
	flag.Parse()

	log := slog.New(slog.NewJSONHandler(os.Stdout, nil))
	slog.SetDefault(log)

	flushTraces, err := tracing.Setup(context.Background(), "hello", *version)
	if err != nil {
		log.Error("tracing", "err", err)
		os.Exit(1)
	}

	reg := prometheus.NewRegistry()
	m := metrics.NewServer(reg, *version)
	drainer := server.NewDrainer(*drainSpread, m.Drained)
	srv := server.New(opts, m, drainer)

	hs := health.NewServer() // the overall status "" starts SERVING
	hs.SetServingStatus(server.LivenessService, healthpb.HealthCheckResponse_SERVING)
	readiness := []string{""}
	hellov1.RegisterHelloServiceServer(srv, handlers.NewHelloService())
	readiness = append(readiness, hellov1.HelloService_ServiceDesc.ServiceName)
	for _, k := range readiness[1:] {
		hs.SetServingStatus(k, healthpb.HealthCheckResponse_SERVING)
	}
	healthpb.RegisterHealthServer(srv, hs)
	reflection.Register(srv)

	lis, err := net.Listen("tcp", *grpcAddr)
	if err != nil {
		log.Error("listen", "addr", *grpcAddr, "err", err)
		os.Exit(1)
	}
	mux := http.NewServeMux()
	mux.Handle("/metrics", promhttp.HandlerFor(reg, promhttp.HandlerOpts{Registry: reg}))
	metricsSrv := &http.Server{Addr: *metricsAddr, Handler: mux, ReadHeaderTimeout: 5 * time.Second}
	go func() {
		if err := metricsSrv.ListenAndServe(); err != nil && !errors.Is(err, http.ErrServerClosed) {
			log.Error("metrics server", "err", err)
			os.Exit(1)
		}
	}()

	serveErr := make(chan error, 1)
	go func() { serveErr <- srv.Serve(lis) }()
	log.Info("serving", "grpc_addr", lis.Addr().String(), "metrics_addr", *metricsAddr, "version", *version)

	ctx, stop := signal.NotifyContext(context.Background(), syscall.SIGTERM, syscall.SIGINT)
	defer stop()
	select {
	case err := <-serveErr:
		log.Error("serve", "err", err)
		os.Exit(1)
	case <-ctx.Done():
	}

	server.Shutdown{
		GRPC:          srv,
		Health:        hs,
		ReadinessKeys: readiness,
		Drainer:       drainer,
		DrainDelay:    *drainDelay,
		Timeout:       *shutdownTimeout,
		InFlight:      m.InFlight,
		Log:           log,
	}.Run()
	_ = metricsSrv.Close()
	flushCtx, cancel := context.WithTimeout(context.Background(), 5*time.Second)
	defer cancel()
	_ = flushTraces(flushCtx)
}

func envOr(key, def string) string {
	if v := os.Getenv(key); v != "" {
		return v
	}
	return def
}
