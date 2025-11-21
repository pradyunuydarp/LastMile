package trip

import (
	"context"

	"google.golang.org/grpc/codes"
	"google.golang.org/grpc/status"

	pb "lastmile/gen/go/trip"
)

// Server implements the TripServiceServer interface.
type Server struct {
	pb.UnimplementedTripServiceServer
	trips map[string]*pb.Trip
}

// NewServer creates a new Server.
func NewServer() *Server {
	return &Server{
		trips: make(map[string]*pb.Trip),
	}
}

// GetTrip retrieves a trip by its ID.
func (s *Server) GetTrip(ctx context.Context, req *pb.GetTripRequest) (*pb.GetTripResponse, error) {
	trip, ok := s.trips[req.Id]
	if !ok {
		return nil, status.Errorf(codes.NotFound, "trip not found")
	}
	return &pb.GetTripResponse{Trip: trip}, nil
}

// UpdateTrip updates the status of a trip.
func (s *Server) UpdateTrip(ctx context.Context, req *pb.UpdateTripRequest) (*pb.UpdateTripResponse, error) {
	trip, ok := s.trips[req.Id]
	if !ok {
		return nil, status.Errorf(codes.NotFound, "trip not found")
	}
	trip.Status = req.Status
	s.trips[req.Id] = trip
	return &pb.UpdateTripResponse{Trip: trip}, nil
}
