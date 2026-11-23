// Command matching runs the LastMile Matching service.
//
// Matching is stateless, so it is the service we scale from 1 to 5 replicas
// in the Kubernetes demo. Every response says which replica served it.
package main

import (
	"os"

	pb "lastmile/gen/go/matching"
	"lastmile/internal/matching"
	"lastmile/internal/pkg/chassis"
)

func main() {
	svc := chassis.New("matching")
	pb.RegisterMatchingServiceServer(svc.GRPC, matching.NewServer(svc.Logger, chassis.Instance()))

	if err := svc.Run(chassis.Getenv("MATCHING_GRPC_ADDR", ":50053")); err != nil {
		svc.Logger.Error("service stopped", "err", err)
		os.Exit(1)
	}
}
