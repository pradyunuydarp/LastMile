package station

import (
	"context"

	"github.com/google/uuid"
	"google.golang.org/grpc/codes"
	"google.golang.org/grpc/status"

	pb "lastmile/gen/go/station"
)

// Server implements the StationServiceServer interface.
type Server struct {
	pb.UnimplementedStationServiceServer
	stations map[string]*pb.Station
}

// NewServer creates a new Server.
func NewServer() *Server {
	return &Server{
		stations: make(map[string]*pb.Station),
	}
}

// AddStation adds a new station.
func (s *Server) AddStation(ctx context.Context, req *pb.AddStationRequest) (*pb.AddStationResponse, error) {
	if req.Station == nil {
		return nil, status.Errorf(codes.InvalidArgument, "station is required")
	}

	id := uuid.New().String()
	req.Station.Id = id
	s.stations[id] = req.Station

	return &pb.AddStationResponse{Id: id}, nil
}

// GetStation retrieves a station by its ID.
func (s *Server) GetStation(ctx context.Context, req *pb.GetStationRequest) (*pb.GetStationResponse, error) {
	station, ok := s.stations[req.Id]
	if !ok {
		return nil, status.Errorf(codes.NotFound, "station with id '%s' not found", req.Id)
	}

	return &pb.GetStationResponse{Station: station}, nil
}
