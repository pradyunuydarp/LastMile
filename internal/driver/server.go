package driver

import (
	"context"

	"github.com/google/uuid"
	"google.golang.org/grpc/codes"
	"google.golang.org/grpc/status"

	pb "lastmile/gen/go/driver"
)

// Server implements the DriverServiceServer interface.
type Server struct {
	pb.UnimplementedDriverServiceServer
	drivers map[string]*pb.Driver
	routes  map[string]*pb.Route
}

// NewServer creates a new Server.
func NewServer() *Server {
	return &Server{
		drivers: make(map[string]*pb.Driver),
		routes:  make(map[string]*pb.Route),
	}
}

// RegisterDriver registers a new driver.
func (s *Server) RegisterDriver(ctx context.Context, req *pb.RegisterDriverRequest) (*pb.RegisterDriverResponse, error) {
	if req.Driver == nil {
		return nil, status.Errorf(codes.InvalidArgument, "driver is required")
	}

	id := uuid.New().String()
	req.Driver.Id = id
	s.drivers[id] = req.Driver

	return &pb.RegisterDriverResponse{Id: id}, nil
}

// RegisterRoute registers a new route for a driver.
func (s *Server) RegisterRoute(ctx context.Context, req *pb.RegisterRouteRequest) (*pb.RegisterRouteResponse, error) {
	if req.Route == nil {
		return nil, status.Errorf(codes.InvalidArgument, "route is required")
	}

	id := uuid.New().String()
	req.Route.Id = id
	s.routes[id] = req.Route

	return &pb.RegisterRouteResponse{Id: id}, nil
}

// UpdateLocation updates the location of a driver. This is a streaming RPC.
func (s *Server) UpdateLocation(stream pb.DriverService_UpdateLocationServer) error {
	for {
		_, err := stream.Recv()
		if err != nil {
			if err.Error() == "EOF" {
				return stream.SendAndClose(&pb.UpdateLocationResponse{Success: true})
			}
			return err
		}
	}
}
