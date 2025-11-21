package trip

import (
	"context"
	"testing"

	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"

	pb "lastmile/gen/go/trip"
)

func TestGetTrip(t *testing.T) {
	s := NewServer()
	trip := &pb.Trip{
		Id:       "trip-123",
		DriverId: "driver-123",
		RiderId:  "rider-456",
		Status:   "pending",
	}
	s.trips[trip.Id] = trip

	req := &pb.GetTripRequest{Id: "trip-123"}
	res, err := s.GetTrip(context.Background(), req)
	require.NoError(t, err)
	assert.Equal(t, "trip-123", res.Trip.Id)
}

func TestUpdateTrip(t *testing.T) {
	s := NewServer()
	trip := &pb.Trip{
		Id:       "trip-123",
		DriverId: "driver-123",
		RiderId:  "rider-456",
		Status:   "pending",
	}
	s.trips[trip.Id] = trip

	req := &pb.UpdateTripRequest{Id: "trip-123", Status: "accepted"}
	res, err := s.UpdateTrip(context.Background(), req)
	require.NoError(t, err)
	assert.Equal(t, "accepted", res.Trip.Status)

	// Check if the trip was actually updated
	updatedTrip, ok := s.trips["trip-123"]
	assert.True(t, ok)
	assert.Equal(t, "accepted", updatedTrip.Status)
}
