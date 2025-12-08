package rider

import (
	"context"
	"log/slog"

	"github.com/google/uuid"
	"google.golang.org/grpc/codes"
	"google.golang.org/grpc/status"

	pb "lastmile/gen/go/rider"
	"lastmile/internal/pkg/logging"
)

// Server implements the RiderServiceServer interface.
type Server struct {
	pb.UnimplementedRiderServiceServer
	riders map[string]*pb.Rider
	rides  map[string]*pb.Ride
	logger *slog.Logger
}

// NewServer creates a new Server.
func NewServer(logger ...*slog.Logger) *Server {
	l := logging.New("rider")
	if len(logger) > 0 && logger[0] != nil {
		l = logger[0]
	}

	return &Server{
		riders: make(map[string]*pb.Rider),
		rides:  make(map[string]*pb.Ride),
		logger: l,
	}
}

// RegisterRider registers a new rider.
func (s *Server) RegisterRider(ctx context.Context, req *pb.RegisterRiderRequest) (*pb.RegisterRiderResponse, error) {
	if req.Rider == nil {
		s.logger.Warn("register rider: missing rider payload")
		return nil, status.Errorf(codes.InvalidArgument, "rider is required")
	}

	id := uuid.New().String()
	req.Rider.Id = id
	s.riders[id] = req.Rider
	s.logger.Info("rider registered", "riderId", id, "name", req.Rider.Name)

	return &pb.RegisterRiderResponse{Id: id}, nil
}

// TrackRide tracks the status of a ride.
func (s *Server) TrackRide(ctx context.Context, req *pb.TrackRideRequest) (*pb.TrackRideResponse, error) {
	ride, ok := s.rides[req.RideId]
	if !ok {
		s.logger.Warn("ride not found", "rideId", req.RideId)
		return nil, status.Errorf(codes.NotFound, "ride not found")
	}

	s.logger.Info("ride status requested", "rideId", req.RideId, "status", ride.Status)
	return &pb.TrackRideResponse{Ride: ride}, nil
}
