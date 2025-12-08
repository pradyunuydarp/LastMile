package gateway

import (
	"context"
	"log/slog"

	gatewaypb "lastmile/gen/go/gateway"
	"lastmile/internal/api"

	"google.golang.org/grpc/codes"
	"google.golang.org/grpc/status"
)

// Server implements the gRPC GatewayService by composing the in-memory gateway state.
type Server struct {
	gateway *api.Gateway
	logger  *slog.Logger
	gatewaypb.UnimplementedGatewayServiceServer
}

func NewServer(gw *api.Gateway, logger *slog.Logger) *Server {
	l := slog.Default()
	if logger != nil {
		l = logger
	}

	return &Server{
		gateway: gw,
		logger:  l.With("component", "gateway-grpc"),
	}
}

func (s *Server) GetSnapshot(ctx context.Context, _ *gatewaypb.SnapshotRequest) (*gatewaypb.BackendSnapshot, error) {
	s.logger.Info("GetSnapshot")
	return s.gateway.SnapshotProto(), nil
}

func (s *Server) TriggerMatch(ctx context.Context, req *gatewaypb.TriggerMatchRequest) (*gatewaypb.GatewayTrip, error) {
	if req.GetDriverId() == "" || req.GetStationId() == "" {
		return nil, status.Errorf(codes.InvalidArgument, "driver_id and station_id are required")
	}

	trip, err := s.gateway.TriggerMatchProto(req.GetDriverId(), req.GetStationId())
	if err != nil {
		s.logger.Warn("TriggerMatch failed", "driverId", req.GetDriverId(), "stationId", req.GetStationId(), "err", err)
		return nil, status.Error(codes.FailedPrecondition, err.Error())
	}

	s.logger.Info("TriggerMatch success", "tripId", trip.GetId(), "driverId", req.GetDriverId(), "stationId", req.GetStationId())
	return trip, nil
}
