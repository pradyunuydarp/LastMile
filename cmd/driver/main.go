// Command driver runs the LastMile driver service.
package main

import (
	"os"

	pb "lastmile/gen/go/driver"
	"lastmile/internal/driver"
	"lastmile/internal/pkg/chassis"
)

func main() {
	svc := chassis.New("driver")
	pb.RegisterDriverServiceServer(svc.GRPC, driver.NewServer(svc.Logger))

	if err := svc.Run(chassis.Getenv("DRIVER_GRPC_ADDR", ":50051")); err != nil {
		svc.Logger.Error("service stopped", "err", err)
		os.Exit(1)
	}
}
