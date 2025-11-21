package main

import (
	"fmt"
	"log"
	"net"

	"google.golang.org/grpc"
	"lastmile/internal/location"
	pb "lastmile/gen/go/location"
)

func main() {
	lis, err := net.Listen("tcp", ":50051")
	if err != nil {
		log.Fatalf("failed to listen: %v", err)
	}
	fmt.Println("Location service listening on port 50051")

	// Create a new gRPC server
	s := grpc.NewServer()

	// Create a new location server
	locationServer := location.NewServer()

	// Register the location server with the gRPC server
	pb.RegisterLocationServiceServer(s, locationServer)

	// Serve requests
	if err := s.Serve(lis); err != nil {
		log.Fatalf("failed to serve: %v", err)
	}
}