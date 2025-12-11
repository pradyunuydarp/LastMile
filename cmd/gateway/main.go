package main

import (
	"bufio"
	"context"
	"log"
	"log/slog"
	"net"
	"net/http"
	"os"
	"os/signal"
	"strings"
	"syscall"
	"time"

	driverpb "lastmile/gen/go/driver"
	gatewaypb "lastmile/gen/go/gateway"
	locationpb "lastmile/gen/go/location"
	userpb "lastmile/gen/go/user"
	"lastmile/internal/api"
	"lastmile/internal/gateway"
	"lastmile/internal/pkg/logging"

	"google.golang.org/grpc"
	"google.golang.org/grpc/credentials/insecure"
)

func main() {
	logger := logging.New("gateway")

	locationAddr := getenv("LOCATION_ADDR", ":50054")
	conn, err := grpc.NewClient(locationAddr, grpc.WithTransportCredentials(insecure.NewCredentials()))
	if err != nil {
		logger.Warn("failed to dial location service", "err", err)
	}
	locClient := locationpb.NewLocationServiceClient(conn)

	userAddr := getenv("USER_ADDR", ":50052")
	userConn, err := grpc.NewClient(userAddr, grpc.WithTransportCredentials(insecure.NewCredentials()))
	if err != nil {
		logger.Warn("failed to dial user service", "err", err)
	}
	userClient := userpb.NewUserServiceClient(userConn)

	driverAddr := getenv("DRIVER_ADDR", ":50051")
	driverConn, err := grpc.NewClient(driverAddr, grpc.WithTransportCredentials(insecure.NewCredentials()))
	if err != nil {
		logger.Warn("failed to dial driver service", "err", err)
	}
	driverClient := driverpb.NewDriverServiceClient(driverConn)

	gw := api.NewGateway(logger.With("component", "gateway-state"), driverClient, locClient, userClient)

	hub := api.NewRealtimeHub(logger.With("component", "realtime-hub"))
	defer hub.Close()
	gw.AttachHub(hub)

	persistenceDSN := getenv("PERSISTENCE_DSN", os.Getenv("DATABASE_URL"))
	if persistenceDSN != "" {
		ctx, cancel := context.WithTimeout(context.Background(), 5*time.Second)
		store, err := api.NewPersistence(ctx, persistenceDSN, logger.With("component", "persistence"))
		cancel()
		if err != nil {
			logger.Warn("persistence disabled", "err", err)
		} else {
			defer store.Close()
			gw.AttachStore(store)
		}
	}

	grpcAddr := getenv("GATEWAY_GRPC_ADDR", ":50060")
	httpAddr := getenv("GATEWAY_HTTP_ADDR", ":8082")

	// gRPC server
	grpcServer := grpc.NewServer()
	gatewaypb.RegisterGatewayServiceServer(grpcServer, gateway.NewServer(gw, logger))

	lis, err := net.Listen("tcp", grpcAddr)
	if err != nil {
		logger.Error("failed to listen for gRPC", "err", err)
		log.Fatalf("failed to listen for gRPC: %v", err)
	}
	go func() {
		logger.Info("gateway gRPC listening", "addr", grpcAddr)
		if serveErr := grpcServer.Serve(lis); serveErr != nil {
			logger.Error("gRPC server stopped", "err", serveErr)
		}
	}()

	// HTTP compatibility for the mobile client (same handlers as before).
	httpMux := http.NewServeMux()
	httpMux.HandleFunc("/aggregates/snapshot", gw.SnapshotHandler)
	httpMux.HandleFunc("/matching/match", gw.MatchHandler)
	httpMux.HandleFunc("/drivers/requests", gw.DriverRequestsHandler)
	httpMux.HandleFunc("/drivers/requests/accept", gw.DriverAcceptHandler)
	httpMux.HandleFunc("/rides/book", gw.BookRideHandler)
	httpMux.HandleFunc("/drivers/routes", gw.DriverRouteHandler)
	httpMux.HandleFunc("/drivers/trip/start", gw.DriverTripStartHandler)
	httpMux.HandleFunc("/metro/pickups", gw.PickupPointsHandler)
	httpMux.HandleFunc("/location/stream", gw.LocationStreamHandler)
	httpMux.HandleFunc("/location/update", gw.UpdateLocationHandler)
	httpMux.HandleFunc("/notifications/token", gw.NotificationTokenHandler)
	httpMux.HandleFunc("/auth/signup", gw.SignUpHandler)
	httpMux.HandleFunc("/auth/signin", gw.SignInHandler)
	httpMux.HandleFunc("/auth/forgot-password", gw.ForgotPasswordHandler)
	httpMux.HandleFunc("/user/profile", gw.GetUserHandler)
	httpMux.HandleFunc("/trips/pickup", gw.TripPickupHandler)
	httpMux.HandleFunc("/trips/dropoff", gw.TripDropoffHandler)
	httpMux.HandleFunc("/trips/simulate", gw.SimulateTripHandler)

	restHandler := withCORS(requestLogger(logger, httpMux))
	socketHandler := hub.Handler()
	httpServer := &http.Server{
		Addr: httpAddr,
		Handler: http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
			if strings.HasPrefix(r.URL.Path, "/socket.io/") {
				socketHandler.ServeHTTP(w, r)
				return
			}
			restHandler.ServeHTTP(w, r)
		}),
	}
	go func() {
		logger.Info("gateway HTTP listening", "addr", httpAddr)
		if serveErr := httpServer.ListenAndServe(); serveErr != nil && serveErr != http.ErrServerClosed {
			logger.Error("HTTP server stopped", "err", serveErr)
		}
	}()

	// Graceful shutdown on interrupt.
	waitForShutdown(logger, grpcServer, httpServer)
}

func requestLogger(logger *slog.Logger, next http.Handler) http.Handler {
	return http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		if r.URL.Path == "/location/stream" || strings.HasPrefix(r.URL.Path, "/socket.io/") {
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

func withCORS(next http.Handler) http.Handler {
	return http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		w.Header().Set("Access-Control-Allow-Origin", "*")
		w.Header().Set("Access-Control-Allow-Credentials", "true")
		w.Header().Set("Access-Control-Allow-Headers", "Content-Type, Authorization, ngrok-skip-browser-warning")
		w.Header().Set("Access-Control-Allow-Methods", "GET, POST, PUT, PATCH, DELETE, OPTIONS")
		if r.Method == http.MethodOptions {
			w.WriteHeader(http.StatusNoContent)
			return
		}
		next.ServeHTTP(w, r)
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

func (w *statusWriter) Hijack() (net.Conn, *bufio.ReadWriter, error) {
	if hj, ok := w.ResponseWriter.(http.Hijacker); ok {
		return hj.Hijack()
	}
	return nil, nil, http.ErrNotSupported
}

type socketResponseWriter struct {
	http.ResponseWriter
	wrote bool
}

func (w *socketResponseWriter) Header() http.Header {
	return w.ResponseWriter.Header()
}

func (w *socketResponseWriter) WriteHeader(status int) {
	if w.wrote {
		return
	}
	w.wrote = true
	w.ResponseWriter.WriteHeader(status)
}

func (w *socketResponseWriter) Write(b []byte) (int, error) {
	if !w.wrote {
		w.WriteHeader(http.StatusOK)
	}
	return w.ResponseWriter.Write(b)
}

func (w *socketResponseWriter) Hijack() (net.Conn, *bufio.ReadWriter, error) {
	if hj, ok := w.ResponseWriter.(http.Hijacker); ok {
		return hj.Hijack()
	}
	return nil, nil, http.ErrNotSupported
}

func (w *socketResponseWriter) Flush() {
	if f, ok := w.ResponseWriter.(http.Flusher); ok {
		f.Flush()
	}
}

func setSocketCORS(w http.ResponseWriter) {
	w.Header().Set("Access-Control-Allow-Origin", "*")
	w.Header().Set("Access-Control-Allow-Headers", "Content-Type, Authorization, ngrok-skip-browser-warning")
	w.Header().Set("Access-Control-Allow-Methods", "GET, POST, OPTIONS")
}

func waitForShutdown(logger *slog.Logger, grpcServer *grpc.Server, httpServer *http.Server) {
	stop := make(chan os.Signal, 1)
	signal.Notify(stop, syscall.SIGINT, syscall.SIGTERM)
	<-stop

	logger.Info("shutting down gateway servers")
	grpcServer.GracefulStop()

	ctx, cancel := context.WithTimeout(context.Background(), 5*time.Second)
	defer cancel()
	_ = httpServer.Shutdown(ctx)
}

func getenv(key, defaultVal string) string {
	if val := os.Getenv(key); val != "" {
		return val
	}
	return defaultVal
}
