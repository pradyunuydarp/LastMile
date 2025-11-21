package rider

import (
	"context"

	"github.com/google/uuid"
	"google.golang.org/grpc/codes"
	"google.golang.org/grpc/status"

	pb "lastmile/gen/go/rider"
)

// Server implements the RiderServiceServer interface.
type Server struct {
	pb.UnimplementedRiderServiceServer
	riders map[string]*pb.Rider
	rides  map[string]*pb.Ride
}

// NewServer creates a new Server.
func NewServer() *Server {
	return &Server{
		riders: make(map[string]*pb.Rider),
		rides:  make(map[string]*pb.Ride),
	}
}

// RegisterRider registers a new rider.
func (s *Server) RegisterRider(ctx context.Context, req *pb.RegisterRiderRequest) (*pb.RegisterRiderResponse, error) {
	if req.Rider == nil {
		return nil, status.Errorf(codes.InvalidArgument, "rider is required")
	}

	id := uuid.New().String()
	req.Rider.Id = id
	s.riders[id] = req.Rider

	return &pb.RegisterRiderResponse{Id: id}, nil
}

// TrackRide tracks the status of a ride.
func (s *Server) TrackRide(ctx context.Context, req *pb.TrackRideRequest) (*pb.TrackRideResponse, error) {
	ride, ok := s.rides[req.RideId]
	if !ok {
		return nil, status.Errorf(codes.NotFound, "ride not found")
	}

	return &pb.TrackRideResponse{Ride: ride}, nil
}
