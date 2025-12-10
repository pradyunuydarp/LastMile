package user

import (
	"context"
	"log/slog"
	"strings"

	supa "github.com/nedpals/supabase-go"
	"google.golang.org/grpc/codes"
	"google.golang.org/grpc/status"

	pb "lastmile/gen/go/user"
	"lastmile/internal/pkg/logging"
)

// Server implements UserServiceServer with Supabase persistence.
type Server struct {
	pb.UnimplementedUserServiceServer
	client *supa.Client
	logger *slog.Logger
}

func NewServer(sbURL, sbKey string, logger ...*slog.Logger) *Server {
	l := logging.New("user")
	if len(logger) > 0 && logger[0] != nil {
		l = logger[0]
	}

	var client *supa.Client
	if sbURL != "" && sbKey != "" {
		client = supa.CreateClient(sbURL, sbKey)
	}

	return &Server{
		client: client,
		logger: l,
	}
}

func (s *Server) RegisterUser(ctx context.Context, req *pb.RegisterUserRequest) (*pb.RegisterUserResponse, error) {
	return nil, status.Error(codes.Unimplemented, "use SignUp instead")
}

func (s *Server) GetUser(ctx context.Context, req *pb.GetUserRequest) (*pb.GetUserResponse, error) {
	if s.client == nil {
		return nil, status.Error(codes.Unavailable, "persistence not configured")
	}

	var results []struct {
		ID       string `json:"id"`
		Email    string `json:"email"` // Note: profiles table might not have email if not synced, but we can try
		FullName string `json:"full_name"`
		Role     string `json:"role"`
	}

	// Fetch from profiles table
	err := s.client.DB.From("profiles").Select("*").Eq("id", req.Id).Execute(&results)
	if err != nil {
		s.logger.Error("failed to fetch profile", "err", err)
		return nil, status.Error(codes.Internal, "failed to fetch user")
	}

	if len(results) == 0 {
		return nil, status.Error(codes.NotFound, "user not found")
	}

	profile := results[0]
	role := pb.UserRole_USER_ROLE_RIDER
	if profile.Role == "driver" {
		role = pb.UserRole_USER_ROLE_DRIVER
	}

	return &pb.GetUserResponse{
		User: &pb.User{
			Id:    profile.ID,
			Name:  profile.FullName,
			Email: profile.Email, // This might be empty if not in profiles, but let's assume it is or we don't strictly need it for display
			Role:  role,
		},
	}, nil
}

func (s *Server) SignUp(ctx context.Context, req *pb.SignUpRequest) (*pb.SignUpResponse, error) {
	if s.client == nil {
		return nil, status.Error(codes.Unavailable, "persistence not configured")
	}

	// 1. Sign up with Supabase Auth
	user, err := s.client.Auth.SignUp(ctx, supa.UserCredentials{
		Email:    req.Email,
		Password: req.Password,
		Data: map[string]interface{}{
			"full_name": req.Name,
			"role":      roleToString(req.Role),
		},
	})
	if err != nil {
		s.logger.Error("supabase signup failed", "err", err)
		return nil, status.Error(codes.Internal, err.Error())
	}

	// 2. Sign in to get access token (since SignUp might not return it depending on config)
	authDetails, err := s.client.Auth.SignIn(ctx, supa.UserCredentials{
		Email:    req.Email,
		Password: req.Password,
	})
	if err != nil {
		s.logger.Error("supabase signin after signup failed", "err", err)
		// Fallback: return ID/Email but no token, or error out?
		// Let's return what we have, client might need to login manually
		return &pb.SignUpResponse{
			Id:    user.ID,
			Email: user.Email,
		}, nil
	}

	return &pb.SignUpResponse{
		Id:          user.ID,
		Email:       user.Email,
		AccessToken: authDetails.AccessToken,
	}, nil
}

func (s *Server) SignIn(ctx context.Context, req *pb.SignInRequest) (*pb.SignInResponse, error) {
	if s.client == nil {
		return nil, status.Error(codes.Unavailable, "persistence not configured")
	}

	// 1. Sign in with Supabase Auth
	authDetails, err := s.client.Auth.SignIn(ctx, supa.UserCredentials{
		Email:    req.Email,
		Password: req.Password,
	})
	if err != nil {
		s.logger.Error("supabase signin failed", "err", err)
		return nil, status.Error(codes.Unauthenticated, "invalid credentials")
	}

	// Look at auth metadata while we attempt to hydrate richer profile data.
	var metaRole string
	var metaName string
	if authDetails.User.UserMetadata != nil {
		if value, ok := authDetails.User.UserMetadata["role"].(string); ok {
			metaRole = strings.ToLower(strings.TrimSpace(value))
		}
		if value, ok := authDetails.User.UserMetadata["full_name"].(string); ok && value != "" {
			metaName = value
		} else if value, ok := authDetails.User.UserMetadata["name"].(string); ok && value != "" {
			metaName = value
		}
	}

	// 2. Fetch profile to get role and name
	var results []struct {
		FullName string `json:"full_name"`
		Role     string `json:"role"`
	}
	err = s.client.DB.From("profiles").Select("full_name, role").Eq("id", authDetails.User.ID).Execute(&results)

	name := metaName
	if name == "" && authDetails.User.Email != "" {
		name = authDetails.User.Email
		if local, _, ok := strings.Cut(authDetails.User.Email, "@"); ok && local != "" {
			name = local
		}
	}

	role := pb.UserRole_USER_ROLE_RIDER
	if metaRole == "driver" {
		role = pb.UserRole_USER_ROLE_DRIVER
	}

	if err == nil && len(results) > 0 {
		if results[0].FullName != "" {
			name = results[0].FullName
		}
		if strings.EqualFold(results[0].Role, "driver") {
			role = pb.UserRole_USER_ROLE_DRIVER
		} else if strings.EqualFold(results[0].Role, "rider") {
			role = pb.UserRole_USER_ROLE_RIDER
		}
	} else if err != nil {
		// Do not fail sign-in when profile lookup has transient issues, but surface the error for observability.
		s.logger.Warn("profiles lookup during signin failed", "err", err)
	}

	return &pb.SignInResponse{
		Id:          authDetails.User.ID,
		Email:       authDetails.User.Email,
		AccessToken: authDetails.AccessToken,
		User: &pb.User{
			Id:    authDetails.User.ID,
			Name:  name,
			Email: authDetails.User.Email,
			Role:  role,
		},
	}, nil
}

func (s *Server) ForgotPassword(ctx context.Context, req *pb.ForgotPasswordRequest) (*pb.ForgotPasswordResponse, error) {
	if s.client == nil {
		return nil, status.Error(codes.Unavailable, "persistence not configured")
	}

	err := s.client.Auth.ResetPasswordForEmail(ctx, req.Email, "")
	if err != nil {
		s.logger.Error("supabase reset password failed", "err", err)
		return nil, status.Error(codes.Internal, err.Error())
	}

	return &pb.ForgotPasswordResponse{Success: true}, nil
}

func roleToString(r pb.UserRole) string {
	if r == pb.UserRole_USER_ROLE_DRIVER {
		return "driver"
	}
	return "rider"
}
