package main

import (
	"log"
	"net"
	"os"

	"google.golang.org/grpc"
	pb "lastmile/gen/go/driver"
	"lastmile/internal/driver"
	"lastmile/internal/pkg/logging"
)

func main() {
	logger := logging.New("driver")
	addr := getenv("DRIVER_GRPC_ADDR", ":50051")

	lis, err := net.Listen("tcp", addr)
	if err != nil {
		logger.Error("failed to listen", "err", err)
		log.Fatalf("failed to listen: %v", err)
	}
	logger.Info("driver service listening", "addr", addr)

	// Create a new gRPC server
	s := grpc.NewServer()

	// Create a new driver server
	driverServer := driver.NewServer(logger.With("component", "driver-server"))

	// Register the driver server with the gRPC server
	pb.RegisterDriverServiceServer(s, driverServer)

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
