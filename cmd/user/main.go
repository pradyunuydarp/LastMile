package main

import (
	"log"
	"net"
	"os"

	pb "lastmile/gen/go/user"
	"lastmile/internal/pkg/logging"
	"lastmile/internal/user"

	"github.com/joho/godotenv"
	"google.golang.org/grpc"
)

func main() {
	_ = godotenv.Load(".env.development")
	logger := logging.New("user")
	addr := os.Getenv("USER_GRPC_ADDR")
	if addr == "" {
		addr = ":50052"
	}

	lis, err := net.Listen("tcp", addr)
	if err != nil {
		logger.Error("failed to listen", "err", err)
		log.Fatalf("failed to listen: %v", err)
	}
	logger.Info("user service listening", "addr", addr)

	supabaseURL := os.Getenv("SUPABASE_URL")
	supabaseKey := os.Getenv("SUPABASE_KEY")
	if supabaseURL == "" || supabaseKey == "" {
		logger.Warn("SUPABASE_URL or SUPABASE_KEY not set, persistence will be disabled")
	}

	s := grpc.NewServer()
	pb.RegisterUserServiceServer(s, user.NewServer(supabaseURL, supabaseKey, logger.With("component", "user-server")))

	if err := s.Serve(lis); err != nil {
		logger.Error("failed to serve", "err", err)
		log.Fatalf("failed to serve: %v", err)
	}
}
