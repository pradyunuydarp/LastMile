// Command stack runs the whole LastMile backend in ONE container: the eight
// microservices plus the gateway, each as its own child process talking gRPC
// over localhost. It exists for hosts that give you a single free web
// service (Render's free plan can't receive private-network traffic, so the
// services can't be split across free instances; see docs/deploy.md).
//
// The code and the gRPC contracts are exactly the ones Kubernetes runs. What
// you give up is the point of microservices in production: the services
// can't be scaled or restarted independently, and if one process dies the
// launcher stops the container so the platform restarts all of it.
//
// Environment:
//
//	PORT             public HTTP port for the gateway (Render sets it)
//	STACK_BASE_PORT  gRPC ports are BASE+1..BASE+8 (default 50050, i.e. the
//	                 usual 50051..50058) and BASE+10 for the gateway
//	STACK_BIN_DIR    where the service binaries are (default: next to stack)
//
// Everything else (PERSISTENCE_DSN, SUPABASE_URL, ...) is passed through.
package main

import (
	"context"
	"errors"
	"fmt"
	"log/slog"
	"net"
	"os"
	"os/exec"
	"os/signal"
	"path/filepath"
	"strconv"
	"strings"
	"syscall"
	"time"

	"lastmile/internal/pkg/chassis"
	"lastmile/internal/pkg/logging"
)

// service is one child process. offset is added to STACK_BASE_PORT.
type service struct {
	name    string
	addrEnv string // the variable the service reads its listen address from
	offset  int
}

// backends start first, in this order; the gateway starts once they listen.
var backends = []service{
	{"station", "STATION_GRPC_ADDR", 6},
	{"driver", "DRIVER_GRPC_ADDR", 1},
	{"user", "USER_GRPC_ADDR", 2},
	{"matching", "MATCHING_GRPC_ADDR", 3},
	{"location", "LOCATION_GRPC_ADDR", 4},
	{"rider", "RIDER_GRPC_ADDR", 5},
	{"trip", "TRIP_GRPC_ADDR", 7},
	{"notification", "NOTIFICATION_GRPC_ADDR", 8},
}

const (
	gatewayGRPCOffset = 10
	readyTimeout      = 30 * time.Second
	stopTimeout       = 20 * time.Second // Render waits 30s before SIGKILL
)

type exit struct {
	name string
	err  error
}

func main() {
	logger := logging.New("stack").With("instance", chassis.Instance())
	if err := run(logger); err != nil {
		logger.Error("stack stopped", "err", err)
		os.Exit(1)
	}
}

func run(logger *slog.Logger) error {
	base, err := strconv.Atoi(chassis.Getenv("STACK_BASE_PORT", "50050"))
	if err != nil || base < 1 || base > 65000 {
		return fmt.Errorf("invalid STACK_BASE_PORT %q", os.Getenv("STACK_BASE_PORT"))
	}
	binDir, err := binaryDir()
	if err != nil {
		return err
	}
	addr := func(offset int) string { return net.JoinHostPort("127.0.0.1", strconv.Itoa(base+offset)) }

	ctx, stop := signal.NotifyContext(context.Background(), syscall.SIGINT, syscall.SIGTERM)
	defer stop()

	exited := make(chan exit, len(backends)+1)
	var running []*exec.Cmd
	start := func(name string, env ...string) error {
		cmd := exec.Command(filepath.Join(binDir, name))
		// Later entries win, so these override anything inherited.
		cmd.Env = append(os.Environ(), append(env, "POD_NAME="+name+"-"+chassis.Instance())...)
		cmd.Stdout, cmd.Stderr = os.Stdout, os.Stderr
		if err := cmd.Start(); err != nil {
			return fmt.Errorf("start %s: %w", name, err)
		}
		running = append(running, cmd)
		go func() { exited <- exit{name, cmd.Wait()} }()
		logger.Info("started", "child", name, "pid", cmd.Process.Pid)
		return nil
	}

	gatewayEnv := []string{"GATEWAY_GRPC_ADDR=" + addr(gatewayGRPCOffset)}
	for _, svc := range backends {
		env := []string{svc.addrEnv + "=" + addr(svc.offset)}
		if svc.name == "location" {
			env = append(env, "STATION_ADDR="+addr(6))
		}
		if err := start(svc.name, env...); err != nil {
			shutdown(logger, running, exited, 0)
			return err
		}
		gatewayEnv = append(gatewayEnv, strings.ToUpper(svc.name)+"_ADDR="+addr(svc.offset))
	}

	// Start the gateway only when every backend accepts connections, so its
	// first calls (loading stations) don't race the backends' startup.
	for _, svc := range backends {
		if err := waitListening(ctx, addr(svc.offset), exited); err != nil {
			shutdown(logger, running, exited, 0)
			return fmt.Errorf("%s did not start: %w", svc.name, err)
		}
	}
	if err := start("gateway", gatewayEnv...); err != nil {
		shutdown(logger, running, exited, 0)
		return err
	}
	logger.Info("all services running", "http_port", chassis.Getenv("PORT", "8082"))

	select {
	case <-ctx.Done():
		logger.Info("shutdown requested, stopping children")
		shutdown(logger, running, exited, 0)
		return nil
	case e := <-exited:
		// One service died: stop the rest and exit non-zero so the platform
		// restarts the container (there is no per-service restart here).
		logger.Error("child exited", "child", e.name, "err", e.err)
		shutdown(logger, running, exited, 1)
		return fmt.Errorf("%s exited", e.name)
	}
}

// shutdown sends SIGTERM to every child (they drain gracefully, see
// chassis.Service.Shutdown) and waits for their exits on the exited channel,
// minus the `reaped` ones the caller already received. Children still alive
// after stopTimeout are killed.
func shutdown(logger *slog.Logger, cmds []*exec.Cmd, exited <-chan exit, reaped int) {
	for _, cmd := range cmds {
		_ = cmd.Process.Signal(syscall.SIGTERM) // fails harmlessly if it already exited
	}
	alive := len(cmds) - reaped
	deadline := time.After(stopTimeout)
	for alive > 0 {
		select {
		case e := <-exited:
			logger.Info("stopped", "child", e.name)
			alive--
		case <-deadline:
			logger.Warn("children did not stop in time, killing them", "remaining", alive)
			for _, cmd := range cmds {
				_ = cmd.Process.Kill()
			}
			return
		}
	}
}

// waitListening polls addr until it accepts a TCP connection, a child exits,
// ctx is cancelled, or readyTimeout passes.
func waitListening(ctx context.Context, addr string, exited chan exit) error {
	ctx, cancel := context.WithTimeout(ctx, readyTimeout)
	defer cancel()
	var dialer net.Dialer
	for {
		dialCtx, dialCancel := context.WithTimeout(ctx, time.Second)
		conn, err := dialer.DialContext(dialCtx, "tcp", addr)
		dialCancel()
		if err == nil {
			return conn.Close()
		}
		select {
		case e := <-exited:
			exited <- e // let the caller's shutdown account for it
			return fmt.Errorf("%s exited during startup: %v", e.name, e.err)
		case <-ctx.Done():
			return errors.Join(ctx.Err(), err)
		case <-time.After(200 * time.Millisecond):
		}
	}
}

func binaryDir() (string, error) {
	if dir := os.Getenv("STACK_BIN_DIR"); dir != "" {
		return dir, nil
	}
	self, err := os.Executable()
	if err != nil {
		return "", err
	}
	return filepath.Dir(self), nil
}
