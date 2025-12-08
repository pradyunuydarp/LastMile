package main

import (
	"log"
	"net"
	"os"

	"google.golang.org/grpc"
	pb "lastmile/gen/go/station"
	"lastmile/internal/pkg/logging"
	"lastmile/internal/station"
)

func main() {
	logger := logging.New("station")
	addr := getenv("STATION_GRPC_ADDR", ":50056")

	lis, err := net.Listen("tcp", addr)
	if err != nil {
		logger.Error("failed to listen", "err", err)
		log.Fatalf("failed to listen: %v", err)
	}
	logger.Info("station service listening", "addr", addr)

	// Create a new gRPC server
	s := grpc.NewServer()

	// Create a new station server
	stationServer := station.NewServer(logger.With("component", "station-server"))

	// Register the station server with the gRPC server
	pb.RegisterStationServiceServer(s, stationServer)

	// Serve requests
	if err := s.Serve(lis); err != nil {
		logger.Error("failed to serve", "err", err)
		log.Fatalf("failed to serve: %v", err)
	}
}

func getenv(key, def string) string {
	if val := os.Getenv(key); val != "" {
		return val
	}
	return def
}
