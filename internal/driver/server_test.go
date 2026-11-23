package driver

import (
	"context"
	"fmt"
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

// TestConcurrentRegistration fires RegisterDriver/RegisterRoute/ListDrivers from many
// goroutines at once, the way a real gRPC server calls handlers. Run it with
// `go test -race` — before the store had a mutex this crashed with
// "fatal error: concurrent map writes" or a DATA RACE report.
func TestConcurrentRegistration(t *testing.T) {
	s := NewServer()
	ctx := context.Background()

	const workers = 50
	done := make(chan struct{})
	for i := 0; i < workers; i++ {
		go func(i int) {
			defer func() { done <- struct{}{} }()
			id := fmt.Sprintf("driver-%d", i)
			_, err := s.RegisterDriver(ctx, &pb.RegisterDriverRequest{Driver: &pb.Driver{Id: id, Name: id}})
			assert.NoError(t, err)
			_, err = s.RegisterRoute(ctx, &pb.RegisterRouteRequest{Route: &pb.Route{DriverId: id, AvailableSeats: 2}})
			assert.NoError(t, err)
			_, err = s.ListDrivers(ctx, &pb.ListDriversRequest{})
			assert.NoError(t, err)
		}(i)
	}
	for i := 0; i < workers; i++ {
		<-done
	}

	resp, err := s.ListDrivers(ctx, &pb.ListDriversRequest{})
	require.NoError(t, err)
	assert.Len(t, resp.Drivers, workers)
	assert.Len(t, resp.Routes, workers)
}
