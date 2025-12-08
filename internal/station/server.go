package station

import (
	"context"
	"log/slog"

	"github.com/google/uuid"
	"google.golang.org/grpc/codes"
	"google.golang.org/grpc/status"

	pb "lastmile/gen/go/station"
	"lastmile/internal/pkg/logging"
)

// Server implements the StationServiceServer interface.
type Server struct {
	pb.UnimplementedStationServiceServer
	stations map[string]*pb.Station
	logger   *slog.Logger
}

// NewServer creates a new Server.
func NewServer(logger ...*slog.Logger) *Server {
	l := logging.New("station")
	if len(logger) > 0 && logger[0] != nil {
		l = logger[0]
	}

	return &Server{
		stations: make(map[string]*pb.Station),
		logger:   l,
	}
}

// AddStation adds a new station.
func (s *Server) AddStation(ctx context.Context, req *pb.AddStationRequest) (*pb.AddStationResponse, error) {
	logger := s.logger
	if logger == nil {
		logger = logging.New("station")
	}

	if req.Station == nil {
		logger.Warn("add station: missing station payload")
		return nil, status.Errorf(codes.InvalidArgument, "station is required")
	}

	id := uuid.New().String()
	req.Station.Id = id
	s.stations[id] = req.Station
	logger.Info("station added", "stationId", id, "name", req.Station.Name)

	return &pb.AddStationResponse{Id: id}, nil
}

// GetStation retrieves a station by its ID.
func (s *Server) GetStation(ctx context.Context, req *pb.GetStationRequest) (*pb.GetStationResponse, error) {
	logger := s.logger
	if logger == nil {
		logger = logging.New("station")
	}

	station, ok := s.stations[req.Id]
	if !ok {
		logger.Warn("station not found", "stationId", req.Id)
		return nil, status.Errorf(codes.NotFound, "station with id '%s' not found", req.Id)
	}

	logger.Info("station fetched", "stationId", req.Id)
	return &pb.GetStationResponse{Station: station}, nil
}
