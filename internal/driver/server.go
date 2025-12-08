package driver

import (
	"context"
	"log/slog"

	"github.com/google/uuid"
	"google.golang.org/grpc/codes"
	"google.golang.org/grpc/status"

	pb "lastmile/gen/go/driver"
	"lastmile/internal/pkg/logging"
)

// Server implements the DriverServiceServer interface.
type Server struct {
	pb.UnimplementedDriverServiceServer
	drivers      map[string]*pb.Driver
	routes       map[string]*pb.Route
	driverRoutes map[string]string
	logger       *slog.Logger
}

// NewServer creates a new Server.
// An optional logger can be provided; slog.Default() is used otherwise.
func NewServer(logger ...*slog.Logger) *Server {
	l := logging.New("driver")
	if len(logger) > 0 && logger[0] != nil {
		l = logger[0]
	}

	return &Server{
		drivers:      make(map[string]*pb.Driver),
		routes:       make(map[string]*pb.Route),
		driverRoutes: make(map[string]string),
		logger:       l,
	}
}

// RegisterDriver registers a new driver.
func (s *Server) RegisterDriver(ctx context.Context, req *pb.RegisterDriverRequest) (*pb.RegisterDriverResponse, error) {
	if req.Driver == nil {
		s.logger.Warn("register driver: missing driver payload")
		return nil, status.Errorf(codes.InvalidArgument, "driver is required")
	}

	id := req.Driver.Id
	if id == "" {
		id = uuid.New().String()
	}
	req.Driver.Id = id
	s.drivers[id] = req.Driver
	s.logger.Info("driver registered", "driverId", id, "name", req.Driver.Name)

	return &pb.RegisterDriverResponse{Id: id}, nil
}

// RegisterRoute registers a new route for a driver.
func (s *Server) RegisterRoute(ctx context.Context, req *pb.RegisterRouteRequest) (*pb.RegisterRouteResponse, error) {
	if req.Route == nil {
		s.logger.Warn("register route: missing route payload")
		return nil, status.Errorf(codes.InvalidArgument, "route is required")
	}

	// Ensure driverId is present so we can associate the route with a user identity.
	if req.Route.DriverId == "" {
		s.logger.Warn("register route: missing driverId")
		return nil, status.Errorf(codes.InvalidArgument, "driverId is required")
	}

	if existingID, ok := s.driverRoutes[req.Route.DriverId]; ok {
		req.Route.Id = existingID
	} else if req.Route.Id == "" {
		req.Route.Id = uuid.New().String()
	}

	s.routes[req.Route.Id] = req.Route
	s.driverRoutes[req.Route.DriverId] = req.Route.Id
	s.logger.Info("route registered", "routeId", req.Route.Id, "driverId", req.Route.DriverId, "targetStations", req.Route.TargetStationIds)

	return &pb.RegisterRouteResponse{Id: req.Route.Id}, nil
}

// ListDrivers returns all registered drivers and their routes.
func (s *Server) ListDrivers(ctx context.Context, req *pb.ListDriversRequest) (*pb.ListDriversResponse, error) {
	drivers := make([]*pb.Driver, 0, len(s.drivers))
	for _, d := range s.drivers {
		drivers = append(drivers, d)
	}

	routes := make([]*pb.Route, 0, len(s.routes))
	for _, r := range s.routes {
		routes = append(routes, r)
	}

	return &pb.ListDriversResponse{
		Drivers: drivers,
		Routes:  routes,
	}, nil
}
