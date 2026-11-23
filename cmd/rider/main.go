// Command rider runs the LastMile rider service.
package main

import (
	"os"

	pb "lastmile/gen/go/rider"
	"lastmile/internal/pkg/chassis"
	"lastmile/internal/rider"
)

func main() {
	svc := chassis.New("rider")
	pb.RegisterRiderServiceServer(svc.GRPC, rider.NewServer(svc.Logger))

	if err := svc.Run(chassis.Getenv("RIDER_GRPC_ADDR", ":50055")); err != nil {
		svc.Logger.Error("service stopped", "err", err)
		os.Exit(1)
	}
}
