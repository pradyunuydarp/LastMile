package main

import (
	"log"
	"net"
	"os"

	"google.golang.org/grpc"
	pb "lastmile/gen/go/notification"
	"lastmile/internal/notification"
	"lastmile/internal/pkg/logging"
)

func main() {
	logger := logging.New("notification")
	addr := getenv("NOTIFICATION_GRPC_ADDR", ":50058")

	lis, err := net.Listen("tcp", addr)
	if err != nil {
		logger.Error("failed to listen", "err", err)
		log.Fatalf("failed to listen: %v", err)
	}
	logger.Info("notification service listening", "addr", addr)

	// Create a new gRPC server
	s := grpc.NewServer()

	// Create a new notification server
	notificationServer := notification.NewServer(logger.With("component", "notification-server"))

	// Register the notification server with the gRPC server
	pb.RegisterNotificationServiceServer(s, notificationServer)

	// Serve requests
	if err := s.Serve(lis); err != nil {
		logger.Error("failed to serve", "err", err)
		log.Fatalf("failed to serve: %v", err)
	}
}

func getenv(key, def string) string {
	if val := os.Getenv(key); val != "" {
		return val
	}
	return def
}
