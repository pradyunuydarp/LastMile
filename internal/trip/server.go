package trip

import (
	"context"
	"log/slog"

	"google.golang.org/grpc/codes"
	"google.golang.org/grpc/status"

	pb "lastmile/gen/go/trip"
	"lastmile/internal/pkg/logging"
)

// Server implements the TripServiceServer interface.
type Server struct {
	pb.UnimplementedTripServiceServer
	trips  map[string]*pb.Trip
	logger *slog.Logger
}

// NewServer creates a new Server.
func NewServer(logger ...*slog.Logger) *Server {
	l := logging.New("trip")
	if len(logger) > 0 && logger[0] != nil {
		l = logger[0]
	}

	return &Server{
		trips:  make(map[string]*pb.Trip),
		logger: l,
	}
}

// GetTrip retrieves a trip by its ID.
func (s *Server) GetTrip(ctx context.Context, req *pb.GetTripRequest) (*pb.GetTripResponse, error) {
	trip, ok := s.trips[req.Id]
	if !ok {
		s.logger.Warn("trip not found", "tripId", req.Id)
		return nil, status.Errorf(codes.NotFound, "trip not found")
	}
	s.logger.Info("trip fetched", "tripId", req.Id, "status", trip.Status)
	return &pb.GetTripResponse{Trip: trip}, nil
}

// UpdateTrip updates the status of a trip.
func (s *Server) UpdateTrip(ctx context.Context, req *pb.UpdateTripRequest) (*pb.UpdateTripResponse, error) {
	trip, ok := s.trips[req.Id]
	if !ok {
		s.logger.Warn("trip not found", "tripId", req.Id)
		return nil, status.Errorf(codes.NotFound, "trip not found")
	}
	trip.Status = req.Status
	s.trips[req.Id] = trip
	s.logger.Info("trip updated", "tripId", req.Id, "status", req.Status)
	return &pb.UpdateTripResponse{Trip: trip}, nil
}
