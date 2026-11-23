// Package location implements the Location service: it ingests live driver
// GPS updates, fans them out to subscribers, and detects when a driver is
// approaching a metro station (the trigger for matching).
package location

import (
	"context"
	"io"
	"log/slog"
	"math"
	"sync"

	"google.golang.org/grpc/codes"
	"google.golang.org/grpc/status"

	pb "lastmile/gen/go/location"
	"lastmile/internal/catalog"
	"lastmile/internal/pkg/geo"
	"lastmile/internal/pkg/logging"
)

// Anchor is a station position used for proximity detection.
type Anchor struct {
	ID        string
	Latitude  float64
	Longitude float64
}

// Server implements the LocationServiceServer interface.
type Server struct {
	pb.UnimplementedLocationServiceServer
	logger *slog.Logger
	radius float64

	mu            sync.RWMutex
	anchors       []Anchor
	subscribers   map[string][]chan *pb.LocationUpdate
	lastLocations map[string]*pb.Location
	insideZone    map[string]string // driverID -> station zone they are inside
}

// NewServer creates a Location server using the built-in station catalog.
// Call SetAnchors to replace the stations (e.g. with data from the Station
// service).
func NewServer(logger ...*slog.Logger) *Server {
	l := logging.New("location")
	if len(logger) > 0 && logger[0] != nil {
		l = logger[0]
	}

	anchors := make([]Anchor, 0)
	for _, s := range catalog.Stations() {
		anchors = append(anchors, Anchor{ID: s.ID, Latitude: s.Latitude, Longitude: s.Longitude})
	}

	return &Server{
		logger:        l,
		radius:        catalog.ApproachRadiusMeters,
		anchors:       anchors,
		subscribers:   make(map[string][]chan *pb.LocationUpdate),
		lastLocations: make(map[string]*pb.Location),
		insideZone:    make(map[string]string),
	}
}

// SetAnchors replaces the stations used for proximity detection.
func (s *Server) SetAnchors(anchors []Anchor) {
	if len(anchors) == 0 {
		return
	}
	s.mu.Lock()
	s.anchors = append([]Anchor(nil), anchors...)
	s.mu.Unlock()
}

// GetDriverLocations returns the last known location of each requested driver.
func (s *Server) GetDriverLocations(ctx context.Context, req *pb.GetDriverLocationsRequest) (*pb.GetDriverLocationsResponse, error) {
	s.mu.RLock()
	defer s.mu.RUnlock()

	var locations []*pb.Location
	for _, id := range req.DriverIds {
		if loc, ok := s.lastLocations[id]; ok {
			locations = append(locations, &pb.Location{DriverId: loc.DriverId, Latitude: loc.Latitude, Longitude: loc.Longitude})
		}
	}
	return &pb.GetDriverLocationsResponse{Locations: locations}, nil
}

// UpdateLocation ingests a client stream of location updates. The response
// carries the proximity result for the last update in the stream.
func (s *Server) UpdateLocation(stream pb.LocationService_UpdateLocationServer) error {
	resp := &pb.UpdateLocationResponse{Success: true}
	for {
		req, err := stream.Recv()
		if err == io.EOF {
			return stream.SendAndClose(resp)
		}
		if err != nil {
			return err
		}
		loc := req.GetLocation()
		if loc == nil || loc.DriverId == "" {
			return status.Error(codes.InvalidArgument, "location with driver_id is required")
		}

		s.broadcast(loc)
		resp = s.record(loc)
	}
}

// record stores the location and runs proximity detection.
func (s *Server) record(loc *pb.Location) *pb.UpdateLocationResponse {
	s.mu.Lock()
	defer s.mu.Unlock()

	s.lastLocations[loc.DriverId] = &pb.Location{DriverId: loc.DriverId, Latitude: loc.Latitude, Longitude: loc.Longitude}

	nearest, dist := "", math.Inf(1)
	for _, a := range s.anchors {
		d := geo.HaversineMeters(loc.Latitude, loc.Longitude, a.Latitude, a.Longitude)
		if d <= s.radius && d < dist {
			nearest, dist = a.ID, d
		}
	}

	resp := &pb.UpdateLocationResponse{Success: true}
	previous := s.insideZone[loc.DriverId]
	if nearest == "" {
		delete(s.insideZone, loc.DriverId)
		return resp
	}

	s.insideZone[loc.DriverId] = nearest
	resp.NearbyStationId = nearest
	resp.DistanceToStationMeters = dist
	resp.EnteredZone = previous != nearest
	if resp.EnteredZone {
		s.logger.Info("driver entered station zone",
			"driverId", loc.DriverId, "stationId", nearest, "distanceMeters", int(dist))
	}
	return resp
}

// SubscribeLocationUpdates streams every future update for one driver.
func (s *Server) SubscribeLocationUpdates(req *pb.SubscribeLocationRequest, stream pb.LocationService_SubscribeLocationUpdatesServer) error {
	ch := make(chan *pb.LocationUpdate, 10)
	driverID := req.DriverId

	s.mu.Lock()
	s.subscribers[driverID] = append(s.subscribers[driverID], ch)
	s.mu.Unlock()

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
		if len(s.subscribers[driverID]) == 0 {
			delete(s.subscribers, driverID)
		}
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

	for _, ch := range s.subscribers[loc.DriverId] {
		update := &pb.LocationUpdate{DriverId: loc.DriverId, Latitude: loc.Latitude, Longitude: loc.Longitude}
		select {
		case ch <- update:
		default: // slow subscriber: drop rather than block the publisher
		}
	}
}
