// Command trip runs the LastMile trip service.
package main

import (
	"os"

	pb "lastmile/gen/go/trip"
	"lastmile/internal/pkg/chassis"
	"lastmile/internal/trip"
)

func main() {
	svc := chassis.New("trip")
	pb.RegisterTripServiceServer(svc.GRPC, trip.NewServer(svc.Logger))

	if err := svc.Run(chassis.Getenv("TRIP_GRPC_ADDR", ":50057")); err != nil {
		svc.Logger.Error("service stopped", "err", err)
		os.Exit(1)
	}
}
