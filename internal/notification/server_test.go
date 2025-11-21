package notification

import (
	"context"
	"testing"

	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"

	pb "lastmile/gen/go/notification"
)

func TestSendNotification(t *testing.T) {
	s := NewServer()
	req := &pb.SendNotificationRequest{
		Notification: &pb.Notification{
			UserId:  "user-123",
			Message: "Your ride is arriving soon!",
		},
	}

	res, err := s.SendNotification(context.Background(), req)
	require.NoError(t, err)
	assert.True(t, res.Success)
}
