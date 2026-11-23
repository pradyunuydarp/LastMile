package driver

import (
	"context"
	"log/slog"
	"sync"

	"github.com/google/uuid"
	"google.golang.org/grpc/codes"
	"google.golang.org/grpc/status"
	"google.golang.org/protobuf/proto"

	pb "lastmile/gen/go/driver"
	"lastmile/internal/pkg/logging"
)

// Server implements the DriverServiceServer interface with an in-memory store.
//
// gRPC runs every request on its own goroutine, so the maps below are guarded
// by mu. (They weren't originally: `go test -race` caught concurrent map
// writes the first time two drivers registered at once.)
type Server struct {
	pb.UnimplementedDriverServiceServer
	logger *slog.Logger

	mu           sync.RWMutex
	drivers      map[string]*pb.Driver
	routes       map[string]*pb.Route
	driverRoutes map[string]string // driverID -> routeID
}

// NewServer creates a new Server.
// An optional logger can be provided; a JSON logger is used otherwise.
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

// RegisterDriver registers (or updates) a driver profile.
func (s *Server) RegisterDriver(ctx context.Context, req *pb.RegisterDriverRequest) (*pb.RegisterDriverResponse, error) {
	if req.Driver == nil {
		return nil, status.Errorf(codes.InvalidArgument, "driver is required")
	}

	// Store a copy: the request message belongs to the caller.
	d := proto.Clone(req.Driver).(*pb.Driver)
	if d.Id == "" {
		d.Id = uuid.New().String()
	}

	s.mu.Lock()
	s.drivers[d.Id] = d
	s.mu.Unlock()

	s.logger.Info("driver registered", "driverId", d.Id, "name", d.Name)
	return &pb.RegisterDriverResponse{Id: d.Id}, nil
}

// RegisterRoute registers or replaces the route for a driver.
func (s *Server) RegisterRoute(ctx context.Context, req *pb.RegisterRouteRequest) (*pb.RegisterRouteResponse, error) {
	if req.Route == nil {
		return nil, status.Errorf(codes.InvalidArgument, "route is required")
	}
	if req.Route.DriverId == "" {
		return nil, status.Errorf(codes.InvalidArgument, "driver_id is required")
	}

	r := proto.Clone(req.Route).(*pb.Route)

	s.mu.Lock()
	if existingID, ok := s.driverRoutes[r.DriverId]; ok {
		r.Id = existingID // one active route per driver
	} else if r.Id == "" {
		r.Id = uuid.New().String()
	}
	s.routes[r.Id] = r
	s.driverRoutes[r.DriverId] = r.Id
	s.mu.Unlock()

	s.logger.Info("route registered", "routeId", r.Id, "driverId", r.DriverId, "targetStations", r.TargetStationIds)
	return &pb.RegisterRouteResponse{Id: r.Id}, nil
}

// ListDrivers returns all registered drivers and their routes.
func (s *Server) ListDrivers(ctx context.Context, req *pb.ListDriversRequest) (*pb.ListDriversResponse, error) {
	s.mu.RLock()
	defer s.mu.RUnlock()

	drivers := make([]*pb.Driver, 0, len(s.drivers))
	for _, d := range s.drivers {
		drivers = append(drivers, proto.Clone(d).(*pb.Driver))
	}
	routes := make([]*pb.Route, 0, len(s.routes))
	for _, r := range s.routes {
		routes = append(routes, proto.Clone(r).(*pb.Route))
	}
	return &pb.ListDriversResponse{Drivers: drivers, Routes: routes}, nil
}
