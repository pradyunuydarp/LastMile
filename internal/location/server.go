package location

import (
	"io"
	"log"

	pb "lastmile/gen/go/location"
)

// Server implements the LocationServiceServer interface.
type Server struct {
	pb.UnimplementedLocationServiceServer
}

// NewServer creates a new Server.
func NewServer() *Server {
	return &Server{}
}

// UpdateLocation updates the location of a driver. This is a streaming RPC.
func (s *Server) UpdateLocation(stream pb.LocationService_UpdateLocationServer) error {
	for {
		req, err := stream.Recv()
		if err == io.EOF {
			return stream.SendAndClose(&pb.UpdateLocationResponse{Success: true})
		}
		if err != nil {
			return err
		}

		log.Printf("Received location update for driver %s: lat=%f, long=%f",
			req.Location.DriverId, req.Location.Latitude, req.Location.Longitude)
	}
}
