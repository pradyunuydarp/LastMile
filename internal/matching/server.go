package matching

import (
	"context"

	"github.com/google/uuid"

	pb "lastmile/gen/go/matching"
	tripb "lastmile/gen/go/trip"
)

// Server implements the MatchingServiceServer interface.
type Server struct {
	pb.UnimplementedMatchingServiceServer
}

// NewServer creates a new Server.
func NewServer() *Server {
	return &Server{}
}

// Match finds suitable riders for a driver near a station and creates trips.
func (s *Server) Match(ctx context.Context, req *pb.MatchRequest) (*pb.MatchResponse, error) {
	// In a real implementation, we would:
	// 1. Get driver details from the driver service.
	// 2. Get riders near the station from the rider service.
	// 3. Find the best matches based on destination, arrival time, etc.
	// 4. Create trips and store them in the trip service.
	// 5. Notify the driver and riders.

	// For now, we'll just return a hardcoded trip.
	trip := &tripb.Trip{
		Id:       uuid.New().String(),
		DriverId: req.DriverId,
		RiderId:  "rider-456", // Hardcoded rider
		Status:   "pending",
	}

	return &pb.MatchResponse{Trips: []*tripb.Trip{trip}}, nil
}
