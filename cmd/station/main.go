// Command station runs the LastMile station service.
package main

import (
	"os"

	pb "lastmile/gen/go/station"
	"lastmile/internal/pkg/chassis"
	"lastmile/internal/station"
)

func main() {
	svc := chassis.New("station")
	pb.RegisterStationServiceServer(svc.GRPC, station.NewServer(svc.Logger))

	if err := svc.Run(chassis.Getenv("STATION_GRPC_ADDR", ":50056")); err != nil {
		svc.Logger.Error("service stopped", "err", err)
		os.Exit(1)
	}
}
