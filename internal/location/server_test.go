package location

import (
	"context"
	"net"
	"testing"
	"time"

	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
	"google.golang.org/grpc"
	"google.golang.org/grpc/credentials/insecure"
	"google.golang.org/grpc/test/bufconn"

	pb "lastmile/gen/go/location"
)

const bufSize = 1024 * 1024

var lis *bufconn.Listener

func init() {
	lis = bufconn.Listen(bufSize)
	s := grpc.NewServer()
	// Use NewServer to ensure map is initialized
	pb.RegisterLocationServiceServer(s, NewServer())
	go func() {
		if err := s.Serve(lis); err != nil {
			panic(err)
		}
	}()
}

func bufDialer(context.Context, string) (net.Conn, error) {
	return lis.Dial()
}

func TestUpdateLocation(t *testing.T) {
	ctx := context.Background()
	conn, err := grpc.DialContext(ctx, "bufnet", grpc.WithContextDialer(bufDialer), grpc.WithTransportCredentials(insecure.NewCredentials()))
	require.NoError(t, err)
	defer conn.Close()

	client := pb.NewLocationServiceClient(conn)

	stream, err := client.UpdateLocation(ctx)
	require.NoError(t, err)

	locations := []*pb.Location{
		{DriverId: "driver-1", Latitude: 34.0522, Longitude: -118.2437},
		{DriverId: "driver-1", Latitude: 34.0523, Longitude: -118.2438},
		{DriverId: "driver-2", Latitude: 34.0524, Longitude: -118.2439},
	}

	for _, loc := range locations {
		req := &pb.UpdateLocationRequest{Location: loc}
		err := stream.Send(req)
		require.NoError(t, err)
	}

	resp, err := stream.CloseAndRecv()
	require.NoError(t, err)
	assert.True(t, resp.Success)
}

func TestSubscribeLocationUpdates(t *testing.T) {
	ctx := context.Background()
	conn, err := grpc.DialContext(ctx, "bufnet", grpc.WithContextDialer(bufDialer), grpc.WithTransportCredentials(insecure.NewCredentials()))
	require.NoError(t, err)
	defer conn.Close()

	client := pb.NewLocationServiceClient(conn)

	// Start subscriber
	subStream, err := client.SubscribeLocationUpdates(ctx, &pb.SubscribeLocationRequest{DriverId: "driver-1"})
	require.NoError(t, err)

	// Give it a moment to subscribe
	time.Sleep(100 * time.Millisecond)

	// Start publisher
	pubStream, err := client.UpdateLocation(ctx)
	require.NoError(t, err)

	expectedLoc := &pb.Location{DriverId: "driver-1", Latitude: 37.7749, Longitude: -122.4194}
	err = pubStream.Send(&pb.UpdateLocationRequest{Location: expectedLoc})
	require.NoError(t, err)

	// Receive update
	update, err := subStream.Recv()
	require.NoError(t, err)
	assert.Equal(t, expectedLoc.DriverId, update.DriverId)
	assert.Equal(t, expectedLoc.Latitude, update.Latitude)
	assert.Equal(t, expectedLoc.Longitude, update.Longitude)

	// Send another update for a different driver (should not be received)
	otherLoc := &pb.Location{DriverId: "driver-2", Latitude: 37.7750, Longitude: -122.4195}
	err = pubStream.Send(&pb.UpdateLocationRequest{Location: otherLoc})
	require.NoError(t, err)

	// Send another update for the same driver
	expectedLoc2 := &pb.Location{DriverId: "driver-1", Latitude: 37.7751, Longitude: -122.4196}
	err = pubStream.Send(&pb.UpdateLocationRequest{Location: expectedLoc2})
	require.NoError(t, err)

	update2, err := subStream.Recv()
	require.NoError(t, err)
	assert.Equal(t, expectedLoc2.Latitude, update2.Latitude)

	pubStream.CloseAndRecv()
}
