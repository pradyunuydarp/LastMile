package main

import (
	"log"
	"net"
	"os"

	"google.golang.org/grpc"
	pb "lastmile/gen/go/trip"
	"lastmile/internal/pkg/logging"
	"lastmile/internal/trip"
)

func main() {
	logger := logging.New("trip")
	addr := getenv("TRIP_GRPC_ADDR", ":50057")

	lis, err := net.Listen("tcp", addr)
	if err != nil {
		logger.Error("failed to listen", "err", err)
		log.Fatalf("failed to listen: %v", err)
	}
	logger.Info("trip service listening", "addr", addr)

	// Create a new gRPC server
	s := grpc.NewServer()

	// Create a new trip server
	tripServer := trip.NewServer(logger.With("component", "trip-server"))

	// Register the trip server with the gRPC server
	pb.RegisterTripServiceServer(s, tripServer)

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
