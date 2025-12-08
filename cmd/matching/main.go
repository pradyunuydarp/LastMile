package main

import (
	"log"
	"net"
	"os"

	"google.golang.org/grpc"
	pb "lastmile/gen/go/matching"
	"lastmile/internal/matching"
	"lastmile/internal/pkg/logging"
)

func main() {
	logger := logging.New("matching")
	addr := getenv("MATCHING_GRPC_ADDR", ":50053")

	lis, err := net.Listen("tcp", addr)
	if err != nil {
		logger.Error("failed to listen", "err", err)
		log.Fatalf("failed to listen: %v", err)
	}
	logger.Info("matching service listening", "addr", addr)

	// Create a new gRPC server
	s := grpc.NewServer()

	// Create a new matching server
	matchingServer := matching.NewServer(logger.With("component", "matching-server"))

	// Register the matching server with the gRPC server
	pb.RegisterMatchingServiceServer(s, matchingServer)

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
