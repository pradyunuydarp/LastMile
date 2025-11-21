package notification

import (
	"context"
	"log"

	pb "lastmile/gen/go/notification"
)

// Server implements the NotificationServiceServer interface.
type Server struct {
	pb.UnimplementedNotificationServiceServer
}

// NewServer creates a new Server.
func NewServer() *Server {
	return &Server{}
}

// SendNotification sends a notification to a user.
func (s *Server) SendNotification(ctx context.Context, req *pb.SendNotificationRequest) (*pb.SendNotificationResponse, error) {
	log.Printf("Sending notification to user %s: %s", req.Notification.UserId, req.Notification.Message)
	// In a real implementation, this would connect to a push notification service (e.g., FCM, APNS)
	// or a real-time messaging service (e.g., WebSockets, gRPC streams).
	return &pb.SendNotificationResponse{Success: true}, nil
}
