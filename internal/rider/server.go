package rider

import (
	"context"
	"log/slog"
	"sync"

	"github.com/google/uuid"
	"google.golang.org/grpc/codes"
	"google.golang.org/grpc/status"
	"google.golang.org/protobuf/proto"

	pb "lastmile/gen/go/rider"
	"lastmile/internal/pkg/logging"
)

// Server implements the RiderServiceServer interface with an in-memory store
// guarded by mu (gRPC handlers run concurrently).
type Server struct {
	pb.UnimplementedRiderServiceServer
	logger *slog.Logger

	mu     sync.RWMutex
	riders map[string]*pb.Rider
	rides  map[string]*pb.Ride
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
		return nil, status.Errorf(codes.InvalidArgument, "rider is required")
	}

	r := proto.Clone(req.Rider).(*pb.Rider)
	r.Id = uuid.New().String()

	s.mu.Lock()
	s.riders[r.Id] = r
	s.mu.Unlock()

	s.logger.Info("rider registered", "riderId", r.Id, "name", r.Name)
	return &pb.RegisterRiderResponse{Id: r.Id}, nil
}

// TrackRide returns the status of a ride.
func (s *Server) TrackRide(ctx context.Context, req *pb.TrackRideRequest) (*pb.TrackRideResponse, error) {
	s.mu.RLock()
	ride, ok := s.rides[req.RideId]
	s.mu.RUnlock()
	if !ok {
		return nil, status.Errorf(codes.NotFound, "ride not found")
	}
	return &pb.TrackRideResponse{Ride: proto.Clone(ride).(*pb.Ride)}, nil
}
