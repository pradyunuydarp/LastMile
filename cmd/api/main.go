package main

import (
	"bufio"
	"log"
	"log/slog"
	"net"
	"net/http"
	"os"
	"time"

	"lastmile/internal/api"
	"lastmile/internal/pkg/logging"

	driverpb "lastmile/gen/go/driver"
	locationpb "lastmile/gen/go/location"
	userpb "lastmile/gen/go/user"

	"google.golang.org/grpc"
	"google.golang.org/grpc/credentials/insecure"
)

func main() {
	logger := logging.New("api-gateway")
	// Dial Location Service
	locAddr := os.Getenv("LOCATION_GRPC_ADDR")
	if locAddr == "" {
		locAddr = "localhost:50051"
	}
	locConn, err := grpc.Dial(locAddr, grpc.WithTransportCredentials(insecure.NewCredentials()))
	if err != nil {
		logger.Error("failed to dial location service", "err", err)
		log.Fatalf("failed to dial location service: %v", err)
	}
	defer locConn.Close()
	locClient := locationpb.NewLocationServiceClient(locConn)

	// Dial User Service
	userAddr := os.Getenv("USER_GRPC_ADDR")
	if userAddr == "" {
		userAddr = "localhost:50053"
	}
	userConn, err := grpc.Dial(userAddr, grpc.WithTransportCredentials(insecure.NewCredentials()))
	if err != nil {
		logger.Error("failed to dial user service", "err", err)
		log.Fatalf("failed to dial user service: %v", err)
	}
	defer userConn.Close()
	userClient := userpb.NewUserServiceClient(userConn)

	// Dial Driver Service
	driverAddr := os.Getenv("DRIVER_GRPC_ADDR")
	if driverAddr == "" {
		driverAddr = "localhost:50051"
	}
	driverConn, err := grpc.Dial(driverAddr, grpc.WithTransportCredentials(insecure.NewCredentials()))
	if err != nil {
		logger.Error("failed to dial driver service", "err", err)
		log.Fatalf("failed to dial driver service: %v", err)
	}
	defer driverConn.Close()
	driverClient := driverpb.NewDriverServiceClient(driverConn)

	gateway := api.NewGateway(logger, driverClient, locClient, userClient)

	mux := http.NewServeMux()
	mux.HandleFunc("/aggregates/snapshot", gateway.SnapshotHandler)
	mux.HandleFunc("/matching/match", gateway.MatchHandler)
	mux.HandleFunc("/drivers/requests", gateway.DriverRequestsHandler)
	mux.HandleFunc("/rides/book", gateway.BookRideHandler)
	mux.HandleFunc("/auth/signup", gateway.SignUpHandler)
	mux.HandleFunc("/auth/signin", gateway.SignInHandler)
	mux.HandleFunc("/auth/forgot-password", gateway.ForgotPasswordHandler)
	mux.HandleFunc("/user/profile", gateway.GetUserHandler)
	mux.HandleFunc("/location/stream", gateway.LocationStreamHandler)

	addr := os.Getenv("API_HTTP_ADDR")
	if addr == "" {
		addr = ":8080"
	}

	logger.Info("http gateway listening", "addr", addr)
	if err := http.ListenAndServe(addr, requestLogger(logger, mux)); err != nil {
		logger.Error("http gateway stopped", "err", err)
		log.Fatalf("http gateway stopped: %v", err)
	}
}

func requestLogger(logger *slog.Logger, next http.Handler) http.Handler {
	return http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		// Skip logging/wrapping for WebSocket endpoints to avoid Hijacker issues
		if r.URL.Path == "/location/stream" {
			next.ServeHTTP(w, r)
			return
		}

		start := time.Now()
		ww := &statusWriter{ResponseWriter: w, status: http.StatusOK}
		next.ServeHTTP(ww, r)
		logger.Info("http request",
			"method", r.Method,
			"path", r.URL.Path,
			"status", ww.status,
			"duration_ms", time.Since(start).Milliseconds())
	})
}

type statusWriter struct {
	http.ResponseWriter
	status int
}

func (w *statusWriter) WriteHeader(status int) {
	w.status = status
	w.ResponseWriter.WriteHeader(status)
}

// Hijack implements http.Hijacker to allow WebSocket upgrades.
func (w *statusWriter) Hijack() (net.Conn, *bufio.ReadWriter, error) {
	if hijacker, ok := w.ResponseWriter.(http.Hijacker); ok {
		return hijacker.Hijack()
	}
	return nil, nil, http.ErrNotSupported
}
