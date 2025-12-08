package main

import (
	"log"
	"net"
	"os"

	pb "lastmile/gen/go/location"
	"lastmile/internal/location"
	"lastmile/internal/pkg/logging"

	"google.golang.org/grpc"
)

func main() {
	logger := logging.New("location")
	addr := getenv("LOCATION_GRPC_ADDR", ":50054")
	matchingTarget := os.Getenv("MATCHING_ADDR")

	lis, err := net.Listen("tcp", addr)
	if err != nil {
		logger.Error("failed to listen", "err", err)
		log.Fatalf("failed to listen: %v", err)
	}
	logger.Info("location service listening", "addr", addr, "matchingTarget", matchingTarget)

	// Create a new gRPC server
	s := grpc.NewServer()

	// Create a new location server
	locationServer := location.NewServerWithMatching(matchingTarget, logger.With("component", "location-server"))

	// Register the location server with the gRPC server
	pb.RegisterLocationServiceServer(s, locationServer)

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
