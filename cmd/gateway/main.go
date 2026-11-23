// Command gateway runs LastMile's API gateway: HTTP + socket.io for the web
// and mobile apps on one side, gRPC to the microservices on the other.
package main

import (
	"bufio"
	"context"
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
	matchingpb "lastmile/gen/go/matching"
	stationpb "lastmile/gen/go/station"
	userpb "lastmile/gen/go/user"
	"lastmile/internal/api"
	"lastmile/internal/gateway"
	"lastmile/internal/pkg/chassis"
	"lastmile/internal/pkg/logging"

	"google.golang.org/grpc"
)

func main() {
	logger := logging.New("gateway").With("instance", chassis.Instance())

	// Every downstream is dialed through the chassis: lazy connect (services
	// may start in any order), round_robin across replicas, and retries on
	// UNAVAILABLE. Targets default to localhost for `make run-local`.
	targets := map[string]string{
		"driver":       getenv("DRIVER_ADDR", "localhost:50051"),
		"user":         getenv("USER_ADDR", "localhost:50052"),
		"matching":     getenv("MATCHING_ADDR", "localhost:50053"),
		"location":     getenv("LOCATION_ADDR", "localhost:50054"),
		"rider":        getenv("RIDER_ADDR", "localhost:50055"),
		"station":      getenv("STATION_ADDR", "localhost:50056"),
		"trip":         getenv("TRIP_ADDR", "localhost:50057"),
		"notification": getenv("NOTIFICATION_ADDR", "localhost:50058"),
	}
	conns := make(map[string]*grpc.ClientConn, len(targets))
	for name, target := range targets {
		conn, err := chassis.Dial(target)
		if err != nil {
			logger.Error("invalid service address", "service", name, "target", target, "err", err)
			os.Exit(1)
		}
		defer conn.Close()
		conns[name] = conn
	}

	gw := api.NewGateway(logger.With("component", "gateway-state"),
		driverpb.NewDriverServiceClient(conns["driver"]),
		locationpb.NewLocationServiceClient(conns["location"]),
		userpb.NewUserServiceClient(conns["user"]))
	gw.AttachMatching(matchingpb.NewMatchingServiceClient(conns["matching"]))
	gw.AttachStatusChecker(api.NewStatusChecker(chassis.Instance(), conns))
	go gw.LoadStations(stationpb.NewStationServiceClient(conns["station"]))

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

	// gRPC edge (GatewayService) for non-browser clients.
	grpcServer := grpc.NewServer()
	gatewaypb.RegisterGatewayServiceServer(grpcServer, gateway.NewServer(gw, logger))

	lis, err := net.Listen("tcp", grpcAddr)
	if err != nil {
		logger.Error("failed to listen for gRPC", "err", err)
		os.Exit(1)
	}
	go func() {
		logger.Info("gateway gRPC listening", "addr", grpcAddr)
		if serveErr := grpcServer.Serve(lis); serveErr != nil {
			logger.Error("gRPC server stopped", "err", serveErr)
		}
	}()

	// HTTP + socket.io for the web and mobile clients.
	httpMux := http.NewServeMux()
	httpMux.HandleFunc("/healthz", gw.HealthzHandler)
	httpMux.HandleFunc("/system/status", gw.SystemStatusHandler)
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
		Addr:              httpAddr,
		ReadHeaderTimeout: 10 * time.Second,
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

	waitForShutdown(logger, grpcServer, httpServer)
}

func requestLogger(logger *slog.Logger, next http.Handler) http.Handler {
	return http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		if r.URL.Path == "/location/stream" || r.URL.Path == "/healthz" || strings.HasPrefix(r.URL.Path, "/socket.io/") {
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
