package main

import (
	"fmt"
	"log"
	"net"

	"google.golang.org/grpc"
	"lastmile/internal/trip"
	pb "lastmile/gen/go/trip"
)

func main() {
	lis, err := net.Listen("tcp", ":50051")
	if err != nil {
		log.Fatalf("failed to listen: %v", err)
	}
	fmt.Println("Trip service listening on port 50051")

	// Create a new gRPC server
	s := grpc.NewServer()

	// Create a new trip server
	tripServer := trip.NewServer()

	// Register the trip server with the gRPC server
	pb.RegisterTripServiceServer(s, tripServer)

	// Serve requests
	if err := s.Serve(lis); err != nil {
		log.Fatalf("failed to serve: %v", err)
	}
}