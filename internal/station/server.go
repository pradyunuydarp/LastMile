package station

import (
	"context"
	"log/slog"
	"sort"
	"sync"

	"github.com/google/uuid"
	"google.golang.org/grpc/codes"
	"google.golang.org/grpc/status"
	"google.golang.org/protobuf/proto"

	pb "lastmile/gen/go/station"
	"lastmile/internal/catalog"
	"lastmile/internal/pkg/logging"
)

// Server implements the StationServiceServer interface. It owns station
// metadata and starts pre-seeded with the Bengaluru catalog.
type Server struct {
	pb.UnimplementedStationServiceServer
	logger *slog.Logger

	mu       sync.RWMutex
	stations map[string]*pb.Station
}

// NewServer creates a Station server seeded from the catalog.
func NewServer(logger ...*slog.Logger) *Server {
	l := logging.New("station")
	if len(logger) > 0 && logger[0] != nil {
		l = logger[0]
	}

	s := &Server{stations: make(map[string]*pb.Station), logger: l}
	for _, c := range catalog.Stations() {
		s.stations[c.ID] = &pb.Station{
			Id:          c.ID,
			Name:        c.Name,
			NearbyAreas: c.AreaNames(),
			Latitude:    c.Latitude,
			Longitude:   c.Longitude,
		}
	}
	return s
}

// AddStation adds a new station. An ID is generated if none is given.
func (s *Server) AddStation(ctx context.Context, req *pb.AddStationRequest) (*pb.AddStationResponse, error) {
	if req.Station == nil {
		return nil, status.Errorf(codes.InvalidArgument, "station is required")
	}

	st := proto.Clone(req.Station).(*pb.Station)
	if st.Id == "" {
		st.Id = uuid.New().String()
	}

	s.mu.Lock()
	s.stations[st.Id] = st
	s.mu.Unlock()

	s.logger.Info("station added", "stationId", st.Id, "name", st.Name)
	return &pb.AddStationResponse{Id: st.Id}, nil
}

// GetStation retrieves a station by its ID.
func (s *Server) GetStation(ctx context.Context, req *pb.GetStationRequest) (*pb.GetStationResponse, error) {
	s.mu.RLock()
	st, ok := s.stations[req.Id]
	s.mu.RUnlock()
	if !ok {
		return nil, status.Errorf(codes.NotFound, "station with id '%s' not found", req.Id)
	}
	return &pb.GetStationResponse{Station: proto.Clone(st).(*pb.Station)}, nil
}

// ListStations returns every station, sorted by ID for stable output.
func (s *Server) ListStations(ctx context.Context, req *pb.ListStationsRequest) (*pb.ListStationsResponse, error) {
	s.mu.RLock()
	out := make([]*pb.Station, 0, len(s.stations))
	for _, st := range s.stations {
		out = append(out, proto.Clone(st).(*pb.Station))
	}
	s.mu.RUnlock()

	sort.Slice(out, func(i, j int) bool { return out[i].Id < out[j].Id })
	return &pb.ListStationsResponse{Stations: out}, nil
}
