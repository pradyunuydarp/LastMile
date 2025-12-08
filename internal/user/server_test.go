package user

import (
	"context"
	"testing"

	pb "lastmile/gen/go/user"

	"github.com/stretchr/testify/assert"
	"google.golang.org/grpc/codes"
	"google.golang.org/grpc/status"
)

func skipIfUnavailable(t *testing.T, err error) bool {
	if err != nil && status.Code(err) == codes.Unavailable {
		t.Skipf("persistence not configured: %v", err)
		return true
	}
	return false
}

func TestSignUp(t *testing.T) {
	s := NewServer("", "")
	req := &pb.SignUpRequest{
		Email:    "test@example.com",
		Password: "password123",
		Name:     "Test User",
		Role:     pb.UserRole_USER_ROLE_RIDER,
	}

	resp, err := s.SignUp(context.Background(), req)
	if skipIfUnavailable(t, err) {
		return
	}
	assert.NoError(t, err)
	assert.NotEmpty(t, resp.Id)
	assert.Equal(t, req.Email, resp.Email)
	assert.NotEmpty(t, resp.AccessToken)
}

func TestSignIn(t *testing.T) {
	s := NewServer("", "")

	// First register a user (since we are using in-memory mock)
	signUpReq := &pb.SignUpRequest{
		Email:    "test@example.com",
		Password: "password123",
		Name:     "Test User",
		Role:     pb.UserRole_USER_ROLE_RIDER,
	}
	if _, err := s.SignUp(context.Background(), signUpReq); skipIfUnavailable(t, err) {
		return
	}

	// Now try to sign in
	req := &pb.SignInRequest{
		Email:    "test@example.com",
		Password: "password123",
	}

	resp, err := s.SignIn(context.Background(), req)
	if skipIfUnavailable(t, err) {
		return
	}
	assert.NoError(t, err)
	assert.NotEmpty(t, resp.Id)
	assert.Equal(t, req.Email, resp.Email)
	assert.NotEmpty(t, resp.AccessToken)
	assert.NotNil(t, resp.User)
	assert.Equal(t, "Test User", resp.User.Name)
}

func TestForgotPassword(t *testing.T) {
	s := NewServer("", "")
	req := &pb.ForgotPasswordRequest{
		Email: "test@example.com",
	}

	resp, err := s.ForgotPassword(context.Background(), req)
	if skipIfUnavailable(t, err) {
		return
	}
	assert.NoError(t, err)
	assert.True(t, resp.Success)
}
