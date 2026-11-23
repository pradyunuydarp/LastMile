// Package chassis is LastMile's "microservice chassis" (Richardson,
// Microservices Patterns, ch. 11): the cross-cutting plumbing every service
// needs, written once instead of copy-pasted into eight main.go files.
//
// A service built on the chassis gets:
//   - structured JSON logs tagged with the service name
//   - the standard gRPC health service (grpc.health.v1), which Kubernetes
//     liveness/readiness probes call directly
//   - server reflection, so `grpcurl -plaintext host:port list` just works
//   - keepalive.MaxConnectionAge, which makes clients reconnect periodically
//     and re-resolve DNS. Without it, a client that connected while there was
//     one Matching pod keeps talking to that one pod forever, even after you
//     scale to five.
//   - graceful shutdown on SIGTERM: report NOT_SERVING, give load balancers a
//     moment to notice, drain in-flight RPCs, then exit.
package chassis

import (
	"context"
	"errors"
	"log/slog"
	"net"
	"os"
	"os/signal"
	"syscall"
	"time"

	"google.golang.org/grpc"
	"google.golang.org/grpc/health"
	healthpb "google.golang.org/grpc/health/grpc_health_v1"
	"google.golang.org/grpc/keepalive"
	"google.golang.org/grpc/reflection"

	"lastmile/internal/pkg/logging"
)

// Service is a gRPC server plus the chassis concerns around it.
type Service struct {
	Name   string
	Logger *slog.Logger
	GRPC   *grpc.Server
	Health *health.Server
}

// New builds a Service. Register your handlers on s.GRPC, then call Run.
func New(name string) *Service {
	logger := logging.New(name).With("instance", Instance())

	srv := grpc.NewServer(
		grpc.KeepaliveParams(keepalive.ServerParameters{
			// Force clients to reconnect (and re-resolve DNS) every ~30s so
			// newly scaled replicas start receiving traffic quickly.
			MaxConnectionAge:      envDuration("GRPC_MAX_CONNECTION_AGE", 30*time.Second),
			MaxConnectionAgeGrace: 10 * time.Second,
		}),
		grpc.ChainUnaryInterceptor(unaryLogger(logger)),
	)

	hs := health.NewServer()
	healthpb.RegisterHealthServer(srv, hs)
	reflection.Register(srv)

	return &Service{Name: name, Logger: logger, GRPC: srv, Health: hs}
}

// Run listens on addr and serves until SIGINT/SIGTERM, then shuts down
// gracefully. It returns nil after a clean shutdown.
func (s *Service) Run(addr string) error {
	lis, err := net.Listen("tcp", addr)
	if err != nil {
		return err
	}

	// Mark every registered service (and the "" overall status) as SERVING.
	for name := range s.GRPC.GetServiceInfo() {
		s.Health.SetServingStatus(name, healthpb.HealthCheckResponse_SERVING)
	}
	s.Health.SetServingStatus("", healthpb.HealthCheckResponse_SERVING)

	errCh := make(chan error, 1)
	go func() {
		s.Logger.Info("service listening", "addr", addr)
		errCh <- s.GRPC.Serve(lis)
	}()

	ctx, stop := signal.NotifyContext(context.Background(), syscall.SIGINT, syscall.SIGTERM)
	defer stop()

	select {
	case err := <-errCh:
		if errors.Is(err, grpc.ErrServerStopped) {
			return nil
		}
		return err
	case <-ctx.Done():
	}

	s.Shutdown()
	return nil
}

// Shutdown drains the server: health goes NOT_SERVING first so probes and
// clients stop sending new work, then in-flight RPCs are allowed to finish.
func (s *Service) Shutdown() {
	s.Logger.Info("shutdown requested, draining")
	s.Health.Shutdown()
	time.Sleep(envDuration("SHUTDOWN_DRAIN_DELAY", time.Second))

	done := make(chan struct{})
	go func() {
		s.GRPC.GracefulStop()
		close(done)
	}()
	select {
	case <-done:
		s.Logger.Info("drained cleanly")
	case <-time.After(10 * time.Second):
		s.Logger.Warn("drain timed out, forcing stop")
		s.GRPC.Stop()
	}
}

// Instance identifies this replica, which is what we want to show when
// demonstrating load balancing ("this match was served by
// matching-7d9f-x2k4p"). In order of preference:
//   - POD_NAME, set explicitly (Kubernetes downward API, docker compose)
//   - RENDER_INSTANCE_ID, Render's unique id for each instance of a service
//   - the hostname, which in Kubernetes is the pod name
func Instance() string {
	for _, key := range []string{"POD_NAME", "RENDER_INSTANCE_ID"} {
		if v := os.Getenv(key); v != "" {
			return v
		}
	}
	if h, err := os.Hostname(); err == nil {
		return h
	}
	return "unknown"
}

// Getenv returns the environment variable or a default.
func Getenv(key, def string) string {
	if v := os.Getenv(key); v != "" {
		return v
	}
	return def
}

func envDuration(key string, def time.Duration) time.Duration {
	if v := os.Getenv(key); v != "" {
		if d, err := time.ParseDuration(v); err == nil {
			return d
		}
	}
	return def
}

// unaryLogger logs every unary RPC with its method, status and latency.
// Health checks are skipped so probe traffic doesn't drown the logs.
func unaryLogger(logger *slog.Logger) grpc.UnaryServerInterceptor {
	return func(ctx context.Context, req any, info *grpc.UnaryServerInfo, handler grpc.UnaryHandler) (any, error) {
		if info.FullMethod == healthpb.Health_Check_FullMethodName {
			return handler(ctx, req)
		}
		start := time.Now()
		resp, err := handler(ctx, req)
		attrs := []any{"method", info.FullMethod, "duration_ms", time.Since(start).Milliseconds()}
		if err != nil {
			logger.Warn("rpc failed", append(attrs, "err", err)...)
		} else if os.Getenv("LOG_EVERY_RPC") == "true" {
			logger.Info("rpc ok", attrs...)
		}
		return resp, err
	}
}
