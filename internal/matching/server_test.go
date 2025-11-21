package matching

import (
	"context"
	"testing"

	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"

	pb "lastmile/gen/go/matching"
)

func TestMatch(t *testing.T) {
	s := NewServer()
	req := &pb.MatchRequest{
		DriverId:  "driver-123",
		StationId: "station-1",
	}

	res, err := s.Match(context.Background(), req)
	require.NoError(t, err)
	assert.NotEmpty(t, res.Trips)
	assert.Equal(t, "driver-123", res.Trips[0].DriverId)
}
