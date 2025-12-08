package station

import (
	"context"
	"testing"

	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
	"google.golang.org/grpc/codes"
	"google.golang.org/grpc/status"

	pb "lastmile/gen/go/station"
)

func TestAddStation(t *testing.T) {
	s := &Server{
		stations: make(map[string]*pb.Station),
	}

	req := &pb.AddStationRequest{
		Station: &pb.Station{
			Name:        "Central Station",
			NearbyAreas: []string{"Downtown", "Midtown"},
		},
	}

	res, err := s.AddStation(context.Background(), req)
	require.NoError(t, err)
	assert.NotEmpty(t, res.Id)

	// Check if the station was actually added
	station, ok := s.stations[res.Id]
	assert.True(t, ok)
	assert.Equal(t, "Central Station", station.Name)
}

func TestGetStation(t *testing.T) {
	s := &Server{
		stations: make(map[string]*pb.Station),
	}

	// Add a station first
	addedStation := &pb.Station{
		Id:          "test-station-id",
		Name:        "Test Station",
		NearbyAreas: []string{"Test Area"},
	}
	s.stations[addedStation.Id] = addedStation

	req := &pb.GetStationRequest{Id: "test-station-id"}
	res, err := s.GetStation(context.Background(), req)
	require.NoError(t, err)
	assert.Equal(t, addedStation.Id, res.Station.Id)
	assert.Equal(t, addedStation.Name, res.Station.Name)
}

func TestGetStation_NotFound(t *testing.T) {
	s := &Server{
		stations: make(map[string]*pb.Station),
	}

	req := &pb.GetStationRequest{Id: "non-existent-id"}
	_, err := s.GetStation(context.Background(), req)
	require.Error(t, err)
	st, ok := status.FromError(err)
	require.True(t, ok)
	assert.Equal(t, codes.NotFound, st.Code())
}
