// Command user runs the LastMile User service (profiles + authentication).
package main

import (
	"os"

	"github.com/joho/godotenv"

	pb "lastmile/gen/go/user"
	"lastmile/internal/pkg/chassis"
	"lastmile/internal/user"
)

func main() {
	_ = godotenv.Load(".env.development") // optional local overrides
	svc := chassis.New("user")

	server := user.NewServer(os.Getenv("SUPABASE_URL"), os.Getenv("SUPABASE_KEY"), svc.Logger)
	if server.DemoMode() {
		svc.Logger.Warn("SUPABASE_URL/SUPABASE_KEY not set: using in-memory demo accounts",
			"accounts", "driver@lastmile.demo, rider@lastmile.demo (+driver2, rider2)",
			"password", user.DemoPassword)
	}
	pb.RegisterUserServiceServer(svc.GRPC, server)

	if err := svc.Run(chassis.Getenv("USER_GRPC_ADDR", ":50052")); err != nil {
		svc.Logger.Error("service stopped", "err", err)
		os.Exit(1)
	}
}
