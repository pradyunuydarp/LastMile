package main

import (
	"fmt"
	"log"
	"net"

	"google.golang.org/grpc"
	"lastmile/internal/matching"
	pb "lastmile/gen/go/matching"
)

func main() {
	lis, err := net.Listen("tcp", ":50051")
	if err != nil {
		log.Fatalf("failed to listen: %v", err)
	}
	fmt.Println("Matching service listening on port 50051")

	// Create a new gRPC server
	s := grpc.NewServer()

	// Create a new matching server
	matchingServer := matching.NewServer()

	// Register the matching server with the gRPC server
	pb.RegisterMatchingServiceServer(s, matchingServer)

	// Serve requests
	if err := s.Serve(lis); err != nil {
		log.Fatalf("failed to serve: %v", err)
	}
}