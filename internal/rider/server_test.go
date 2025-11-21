package rider

import (
	"context"
	"testing"
	"time"

	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
	"google.golang.org/protobuf/types/known/timestamppb"

	pb "lastmile/gen/go/rider"
)

func TestRegisterRider(t *testing.T) {
	s := NewServer()
	req := &pb.RegisterRiderRequest{
		Rider: &pb.Rider{
			Name:        "Jane Doe",
			Destination: "123 Main St",
			ArrivalTime: timestamppb.New(time.Now()),
		},
	}

	res, err := s.RegisterRider(context.Background(), req)
	require.NoError(t, err)
	assert.NotEmpty(t, res.Id)

	// Check if the rider was actually added
	rider, ok := s.riders[res.Id]
	assert.True(t, ok)
	assert.Equal(t, "Jane Doe", rider.Name)
}

func TestTrackRide(t *testing.T) {
	s := NewServer()
	ride := &pb.Ride{
		Id:       "ride-123",
		RiderId:  "rider-123",
		DriverId: "driver-123",
		Status:   "en-route",
	}
	s.rides[ride.Id] = ride

	req := &pb.TrackRideRequest{RideId: "ride-123"}
	res, err := s.TrackRide(context.Background(), req)
	require.NoError(t, err)
	assert.Equal(t, "ride-123", res.Ride.Id)
	assert.Equal(t, "en-route", res.Ride.Status)
}
