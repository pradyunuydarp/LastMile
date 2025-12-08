package location

import (
	"context"
	"io"
	"log/slog"
	"math"
	"sync"

	pb "lastmile/gen/go/location"
	"lastmile/gen/go/matching"
	"lastmile/internal/pkg/logging"

	"google.golang.org/grpc"
	"google.golang.org/grpc/credentials/insecure"
)

// Server implements the LocationServiceServer interface.
type Server struct {
	pb.UnimplementedLocationServiceServer
	logger         *slog.Logger
	matchingClient matching.MatchingServiceClient

	mu            sync.RWMutex
	subscribers   map[string][]chan *pb.LocationUpdate
	lastLocations map[string]*pb.Location
}

// NewServer creates a new Server.
func NewServer(logger ...*slog.Logger) *Server {
	l := logging.New("location")
	if len(logger) > 0 && logger[0] != nil {
		l = logger[0]
	}

	return &Server{
		logger:        l,
		subscribers:   make(map[string][]chan *pb.LocationUpdate),
		lastLocations: make(map[string]*pb.Location),
	}
}

// NewServerWithMatching allows wiring proximity-triggered matches to the Matching service at targetAddr.
func NewServerWithMatching(targetAddr string, logger *slog.Logger) *Server {
	l := logger
	if l == nil {
		l = logging.New("location")
	}

	var client matching.MatchingServiceClient
	if targetAddr != "" {
		conn, err := grpc.Dial(targetAddr, grpc.WithTransportCredentials(insecure.NewCredentials()))
		if err == nil {
			client = matching.NewMatchingServiceClient(conn)
			l.Info("location matching client enabled", "target", targetAddr)
		} else {
			l.Warn("location matching client dial failed", "target", targetAddr, "err", err)
		}
	}

	return &Server{
		logger:         l,
		matchingClient: client,
		subscribers:    make(map[string][]chan *pb.LocationUpdate),
		lastLocations:  make(map[string]*pb.Location),
	}
}

func (s *Server) GetDriverLocations(ctx context.Context, req *pb.GetDriverLocationsRequest) (*pb.GetDriverLocationsResponse, error) {
	s.mu.RLock()
	defer s.mu.RUnlock()

	var locations []*pb.Location
	for _, id := range req.DriverIds {
		if loc, ok := s.lastLocations[id]; ok {
			locations = append(locations, loc)
		}
	}

	return &pb.GetDriverLocationsResponse{
		Locations: locations,
	}, nil
}

// UpdateLocation updates the location of a driver. This is a streaming RPC.
func (s *Server) UpdateLocation(stream pb.LocationService_UpdateLocationServer) error {
	logger := s.logger
	if logger == nil {
		logger = logging.New("location")
	}

	for {
		req, err := stream.Recv()
		if err == io.EOF {
			logger.Info("location stream closed")
			return stream.SendAndClose(&pb.UpdateLocationResponse{Success: true})
		}
		if err != nil {
			logger.Error("location stream failed", "err", err)
			return err
		}

		// Broadcast to subscribers
		s.broadcast(req.Location)

		// Update last location
		s.mu.Lock()
		s.lastLocations[req.Location.DriverId] = req.Location
		s.mu.Unlock()

		logger.Info("location update received",
			"driverId", req.Location.DriverId,
			"lat", req.Location.Latitude,
			"long", req.Location.Longitude)

		if stationID, ok := detectStation(req.Location.Latitude, req.Location.Longitude); ok && s.matchingClient != nil {
			logger.Info("driver in proximity, triggering match", "driverId", req.Location.DriverId, "stationId", stationID)
			_, callErr := s.matchingClient.Match(stream.Context(), &matching.MatchRequest{
				DriverId:  req.Location.DriverId,
				StationId: stationID,
			})
			if callErr != nil {
				logger.Warn("proximity match failed", "err", callErr)
			}
		}
	}
}

func (s *Server) SubscribeLocationUpdates(req *pb.SubscribeLocationRequest, stream pb.LocationService_SubscribeLocationUpdatesServer) error {
	ch := make(chan *pb.LocationUpdate, 10)
	driverID := req.DriverId

	s.mu.Lock()
	s.subscribers[driverID] = append(s.subscribers[driverID], ch)
	s.mu.Unlock()

	// Cleanup on return
	defer func() {
		s.mu.Lock()
		defer s.mu.Unlock()
		subs := s.subscribers[driverID]
		for i, sub := range subs {
			if sub == ch {
				s.subscribers[driverID] = append(subs[:i], subs[i+1:]...)
				break
			}
		}
		close(ch)
	}()

	for {
		select {
		case update := <-ch:
			if err := stream.Send(update); err != nil {
				return err
			}
		case <-stream.Context().Done():
			return stream.Context().Err()
		}
	}
}

func (s *Server) broadcast(loc *pb.Location) {
	s.mu.RLock()
	defer s.mu.RUnlock()

	update := &pb.LocationUpdate{
		DriverId:  loc.DriverId,
		Latitude:  loc.Latitude,
		Longitude: loc.Longitude,
	}

	for _, ch := range s.subscribers[loc.DriverId] {
		select {
		case ch <- update:
		default:
			// Skip if channel is full to avoid blocking
		}
	}
}

type coord struct {
	lat float64
	lon float64
}

var stationAnchors = map[string]coord{
	"station-central": {lat: 37.7936, lon: -122.3965},
	"station-mission": {lat: 37.7599, lon: -122.4148},
	"station-sunset":  {lat: 37.7530, lon: -122.4946},
}

func detectStation(lat, lon float64) (string, bool) {
	for id, anchor := range stationAnchors {
		if haversineMeters(lat, lon, anchor.lat, anchor.lon) <= 800 { // ~0.5mi
			return id, true
		}
	}
	return "", false
}

func haversineMeters(lat1, lon1, lat2, lon2 float64) float64 {
	const R = 6371000.0
	dLat := (lat2 - lat1) * math.Pi / 180
	dLon := (lon2 - lon1) * math.Pi / 180
	a := math.Sin(dLat/2)*math.Sin(dLat/2) + math.Cos(lat1*math.Pi/180)*math.Cos(lat2*math.Pi/180)*math.Sin(dLon/2)*math.Sin(dLon/2)
	c := 2 * math.Atan2(math.Sqrt(a), math.Sqrt(1-a))
	return R * c
}
