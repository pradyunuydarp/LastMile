package user

import (
	"context"
	"testing"

	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
	"google.golang.org/grpc/codes"
	"google.golang.org/grpc/status"

	pb "lastmile/gen/go/user"
)

// These tests run against the in-memory backend (no Supabase configured).
// Before demo mode existed, every test here was silently skipped.

func TestSignUpThenSignIn(t *testing.T) {
	s := NewServer("", "")
	require.True(t, s.DemoMode())
	ctx := context.Background()

	up, err := s.SignUp(ctx, &pb.SignUpRequest{
		Email: "Test@Example.com", Password: "password123", Name: "Test User", Role: pb.UserRole_USER_ROLE_DRIVER,
	})
	require.NoError(t, err)
	assert.NotEmpty(t, up.Id)
	assert.NotEmpty(t, up.AccessToken)

	in, err := s.SignIn(ctx, &pb.SignInRequest{Email: "test@example.com", Password: "password123"})
	require.NoError(t, err)
	assert.Equal(t, up.Id, in.Id, "email lookup is case-insensitive")
	assert.Equal(t, "Test User", in.User.Name)
	assert.Equal(t, pb.UserRole_USER_ROLE_DRIVER, in.User.Role)

	got, err := s.GetUser(ctx, &pb.GetUserRequest{Id: up.Id})
	require.NoError(t, err)
	assert.Equal(t, "Test User", got.User.Name)
}

func TestSeededDemoAccounts(t *testing.T) {
	s := NewServer("", "")
	for _, a := range DemoAccounts {
		t.Run(a.Email, func(t *testing.T) {
			resp, err := s.SignIn(context.Background(), &pb.SignInRequest{Email: a.Email, Password: DemoPassword})
			require.NoError(t, err)
			assert.Equal(t, a.ID, resp.User.Id)
			assert.Equal(t, a.Role, resp.User.Role)
		})
	}
}

func TestAuthErrors(t *testing.T) {
	s := NewServer("", "")
	ctx := context.Background()

	tests := []struct {
		name string
		call func() error
		want codes.Code
	}{
		{"wrong password", func() error {
			_, err := s.SignIn(ctx, &pb.SignInRequest{Email: "rider@lastmile.demo", Password: "nope"})
			return err
		}, codes.Unauthenticated},
		{"unknown email", func() error {
			_, err := s.SignIn(ctx, &pb.SignInRequest{Email: "ghost@lastmile.demo", Password: DemoPassword})
			return err
		}, codes.Unauthenticated},
		{"duplicate sign-up", func() error {
			_, err := s.SignUp(ctx, &pb.SignUpRequest{Email: "rider@lastmile.demo", Password: "password123"})
			return err
		}, codes.AlreadyExists},
		{"short password", func() error {
			_, err := s.SignUp(ctx, &pb.SignUpRequest{Email: "new@lastmile.demo", Password: "123"})
			return err
		}, codes.InvalidArgument},
		{"unknown user id", func() error {
			_, err := s.GetUser(ctx, &pb.GetUserRequest{Id: "nobody"})
			return err
		}, codes.NotFound},
	}
	for _, tc := range tests {
		t.Run(tc.name, func(t *testing.T) {
			assert.Equal(t, tc.want, status.Code(tc.call()))
		})
	}
}

func TestForgotPasswordInDemoMode(t *testing.T) {
	s := NewServer("", "")
	resp, err := s.ForgotPassword(context.Background(), &pb.ForgotPasswordRequest{Email: "rider@lastmile.demo"})
	require.NoError(t, err)
	assert.True(t, resp.Success)
}
