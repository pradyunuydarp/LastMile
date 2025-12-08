package main

import (
	"log"
	"net"
	"os"

	"google.golang.org/grpc"
	pb "lastmile/gen/go/rider"
	"lastmile/internal/pkg/logging"
	"lastmile/internal/rider"
)

func main() {
	logger := logging.New("rider")
	addr := getenv("RIDER_GRPC_ADDR", ":50055")

	lis, err := net.Listen("tcp", addr)
	if err != nil {
		logger.Error("failed to listen", "err", err)
		log.Fatalf("failed to listen: %v", err)
	}
	logger.Info("rider service listening", "addr", addr)

	// Create a new gRPC server
	s := grpc.NewServer()

	// Create a new rider server
	riderServer := rider.NewServer(logger.With("component", "rider-server"))

	// Register the rider server with the gRPC server
	pb.RegisterRiderServiceServer(s, riderServer)

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
