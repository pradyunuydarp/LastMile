package location

import (
	"context"
	"testing"

	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
	"google.golang.org/grpc"
	"google.golang.org/grpc/test/bufconn"

	pb "lastmile/gen/go/location"
	"net"
)

const bufSize = 1024 * 1024

var lis *bufconn.Listener

func init() {
	lis = bufconn.Listen(bufSize)
	s := grpc.NewServer()
	pb.RegisterLocationServiceServer(s, &Server{})
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
	conn, err := grpc.DialContext(ctx, "bufnet", grpc.WithContextDialer(bufDialer), grpc.WithInsecure())
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
