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
	s := NewServer()

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
	s := NewServer()

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
	s := NewServer()

	req := &pb.GetStationRequest{Id: "non-existent-id"}
	_, err := s.GetStation(context.Background(), req)
	require.Error(t, err)
	st, ok := status.FromError(err)
	require.True(t, ok)
	assert.Equal(t, codes.NotFound, st.Code())
}

func TestListStationsIsSeededFromCatalog(t *testing.T) {
	s := NewServer()

	res, err := s.ListStations(context.Background(), &pb.ListStationsRequest{})
	require.NoError(t, err)
	require.NotEmpty(t, res.Stations)

	var ecity *pb.Station
	for _, st := range res.Stations {
		if st.Id == "station-ecity" {
			ecity = st
		}
	}
	require.NotNil(t, ecity, "Electronic City should be seeded")
	assert.InDelta(t, 12.8456, ecity.Latitude, 0.001, "stations must carry real Bengaluru coordinates")
	assert.Contains(t, ecity.NearbyAreas, "Wipro Gate")
}
