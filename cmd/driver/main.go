package main

import (
	"fmt"
	"log"
	"net"

	"google.golang.org/grpc"
	"lastmile/internal/driver"
	pb "lastmile/gen/go/driver"
)

func main() {
	lis, err := net.Listen("tcp", ":50051")
	if err != nil {
		log.Fatalf("failed to listen: %v", err)
	}
	fmt.Println("Driver service listening on port 50051")

	// Create a new gRPC server
	s := grpc.NewServer()

	// Create a new driver server
	driverServer := driver.NewServer()

	// Register the driver server with the gRPC server
	pb.RegisterDriverServiceServer(s, driverServer)

	// Serve requests
	if err := s.Serve(lis); err != nil {
		log.Fatalf("failed to serve: %v", err)
	}
}