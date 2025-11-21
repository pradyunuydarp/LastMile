package main

import (
	"fmt"
	"log"
	"net"

	"google.golang.org/grpc"
	"lastmile/internal/station"
	pb "lastmile/gen/go/station"
)

func main() {
	lis, err := net.Listen("tcp", ":50051")
	if err != nil {
		log.Fatalf("failed to listen: %v", err)
	}
	fmt.Println("Station service listening on port 50051")

	// Create a new gRPC server
	s := grpc.NewServer()

	// Create a new station server
	stationServer := station.NewServer()

	// Register the station server with the gRPC server
	pb.RegisterStationServiceServer(s, stationServer)

	// Serve requests
	if err := s.Serve(lis); err != nil {
		log.Fatalf("failed to serve: %v", err)
	}
}