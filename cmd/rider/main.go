package main

import (
	"fmt"
	"log"
	"net"

	"google.golang.org/grpc"
	"lastmile/internal/rider"
	pb "lastmile/gen/go/rider"
)

func main() {
	lis, err := net.Listen("tcp", ":50051")
	if err != nil {
		log.Fatalf("failed to listen: %v", err)
	}
	fmt.Println("Rider service listening on port 50051")

	// Create a new gRPC server
	s := grpc.NewServer()

	// Create a new rider server
	riderServer := rider.NewServer()

	// Register the rider server with the gRPC server
	pb.RegisterRiderServiceServer(s, riderServer)

	// Serve requests
	if err := s.Serve(lis); err != nil {
		log.Fatalf("failed to serve: %v", err)
	}
}