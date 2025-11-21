package driver

import (
	"context"
	"testing"

	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"

	pb "lastmile/gen/go/driver"
)

func TestRegisterDriver(t *testing.T) {
	s := NewServer()
	req := &pb.RegisterDriverRequest{
		Driver: &pb.Driver{
			Name:       "John Doe",
			CarDetails: "Toyota Camry",
		},
	}

	res, err := s.RegisterDriver(context.Background(), req)
	require.NoError(t, err)
	assert.NotEmpty(t, res.Id)

	// Check if the driver was actually added
	driver, ok := s.drivers[res.Id]
	assert.True(t, ok)
	assert.Equal(t, "John Doe", driver.Name)
}

func TestRegisterRoute(t *testing.T) {
	s := NewServer()
	req := &pb.RegisterRouteRequest{
		Route: &pb.Route{
			DriverId:         "driver-123",
			TargetStationIds: []string{"station-1", "station-2"},
			AvailableSeats:   3,
		},
	}

	res, err := s.RegisterRoute(context.Background(), req)
	require.NoError(t, err)
	assert.NotEmpty(t, res.Id)

	// Check if the route was actually added
	route, ok := s.routes[res.Id]
	assert.True(t, ok)
	assert.Equal(t, "driver-123", route.DriverId)
}
