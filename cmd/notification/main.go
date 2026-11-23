// Command notification runs the LastMile notification service.
package main

import (
	"os"

	pb "lastmile/gen/go/notification"
	"lastmile/internal/notification"
	"lastmile/internal/pkg/chassis"
)

func main() {
	svc := chassis.New("notification")
	pb.RegisterNotificationServiceServer(svc.GRPC, notification.NewServer(svc.Logger))

	if err := svc.Run(chassis.Getenv("NOTIFICATION_GRPC_ADDR", ":50058")); err != nil {
		svc.Logger.Error("service stopped", "err", err)
		os.Exit(1)
	}
}
