package notification

import (
	"context"
	"log/slog"

	pb "lastmile/gen/go/notification"
	"lastmile/internal/pkg/logging"
)

// Server implements the NotificationServiceServer interface.
type Server struct {
	pb.UnimplementedNotificationServiceServer
	logger *slog.Logger
}

// NewServer creates a new Server.
func NewServer(logger ...*slog.Logger) *Server {
	l := logging.New("notification")
	if len(logger) > 0 && logger[0] != nil {
		l = logger[0]
	}

	return &Server{logger: l}
}

// SendNotification sends a notification to a user.
func (s *Server) SendNotification(ctx context.Context, req *pb.SendNotificationRequest) (*pb.SendNotificationResponse, error) {
	s.logger.Info("sending notification", "userId", req.Notification.UserId, "notificationId", req.Notification.Id)
	// In a real implementation, this would connect to a push notification service (e.g., FCM, APNS)
	// or a real-time messaging service (e.g., WebSockets, gRPC streams).
	return &pb.SendNotificationResponse{Success: true}, nil
}
