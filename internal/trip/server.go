package trip

import (
	"context"
	"log/slog"
	"sync"

	"google.golang.org/grpc/codes"
	"google.golang.org/grpc/status"
	"google.golang.org/protobuf/proto"

	pb "lastmile/gen/go/trip"
	"lastmile/internal/pkg/logging"
)

// Server implements the TripServiceServer interface with an in-memory store
// guarded by mu (gRPC handlers run concurrently).
type Server struct {
	pb.UnimplementedTripServiceServer
	logger *slog.Logger

	mu    sync.RWMutex
	trips map[string]*pb.Trip
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
	s.mu.RLock()
	trip, ok := s.trips[req.Id]
	s.mu.RUnlock()
	if !ok {
		return nil, status.Errorf(codes.NotFound, "trip not found")
	}
	return &pb.GetTripResponse{Trip: proto.Clone(trip).(*pb.Trip)}, nil
}

// UpdateTrip updates the status of a trip.
func (s *Server) UpdateTrip(ctx context.Context, req *pb.UpdateTripRequest) (*pb.UpdateTripResponse, error) {
	s.mu.Lock()
	trip, ok := s.trips[req.Id]
	if ok {
		trip.Status = req.Status
	}
	var out *pb.Trip
	if ok {
		out = proto.Clone(trip).(*pb.Trip)
	}
	s.mu.Unlock()

	if !ok {
		return nil, status.Errorf(codes.NotFound, "trip not found")
	}
	s.logger.Info("trip updated", "tripId", req.Id, "status", req.Status)
	return &pb.UpdateTripResponse{Trip: out}, nil
}
