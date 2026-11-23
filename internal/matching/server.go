// Package matching implements the Matching service: a stateless gRPC service
// that applies LastMile's matching rules (see rules.go).
package matching

import (
	"context"
	"log/slog"
	"strings"
	"time"

	"google.golang.org/grpc/codes"
	"google.golang.org/grpc/status"

	pb "lastmile/gen/go/matching"
	"lastmile/internal/pkg/logging"
)

// Server is the gRPC adapter around the pure rules in rules.go.
type Server struct {
	pb.UnimplementedMatchingServiceServer
	logger   *slog.Logger
	instance string
	now      func() time.Time
}

// NewServer creates a Matching server. instance names this replica (the pod
// name in Kubernetes) and is echoed back in every response as served_by.
func NewServer(logger *slog.Logger, instance string) *Server {
	if logger == nil {
		logger = logging.New("matching")
	}
	return &Server{logger: logger, instance: instance, now: time.Now}
}

// RankDrivers answers "a rider just booked: who can take them?".
func (s *Server) RankDrivers(ctx context.Context, req *pb.RankDriversRequest) (*pb.RankDriversResponse, error) {
	if req.GetRider() == nil || req.Rider.GetPickupPointId() == "" {
		return nil, status.Error(codes.InvalidArgument, "rider with pickup_point_id is required")
	}
	candidates, rejected := RankDrivers(req.Rider, req.Drivers, s.now())

	if !isSynthetic(req.Rider.GetRiderId()) {
		s.logger.Info("ranked drivers",
			"riderId", req.Rider.GetRiderId(),
			"pickup", req.Rider.GetPickupPointId(),
			"considered", len(req.Drivers),
			"eligible", len(candidates))
	}
	return &pb.RankDriversResponse{Candidates: candidates, Rejections: rejected, ServedBy: s.instance}, nil
}

// FindRiders answers "a driver is nearly at the station: who should they pick up?".
func (s *Server) FindRiders(ctx context.Context, req *pb.FindRidersRequest) (*pb.FindRidersResponse, error) {
	if req.GetDriver() == nil || req.Driver.GetDriverId() == "" {
		return nil, status.Error(codes.InvalidArgument, "driver with driver_id is required")
	}
	riders, rejected := FindRiders(req.Driver, req.Riders, s.now())
	if !isSynthetic(req.Driver.GetDriverId()) && len(riders) > 0 {
		s.logger.Info("found riders",
			"driverId", req.Driver.GetDriverId(),
			"considered", len(req.Riders),
			"eligible", len(riders))
	}
	return &pb.FindRidersResponse{Riders: riders, Rejections: rejected, ServedBy: s.instance}, nil
}

// isSynthetic marks load-generator traffic so we don't log thousands of
// lines per second during the autoscaling demo.
func isSynthetic(id string) bool {
	return strings.HasPrefix(id, "load-")
}
