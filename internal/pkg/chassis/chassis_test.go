package chassis

import (
	"context"
	"fmt"
	"net"
	"os"
	"sync/atomic"
	"testing"
	"time"

	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
	"google.golang.org/grpc"
	"google.golang.org/grpc/credentials/insecure"
	healthpb "google.golang.org/grpc/health/grpc_health_v1"
	"google.golang.org/grpc/resolver"
	"google.golang.org/grpc/resolver/manual"

	pb "lastmile/gen/go/matching"
	"lastmile/internal/matching"
)

// replica starts one Matching server on a real TCP port, like one pod.
func replica(t *testing.T, name string) (addr string, stop func()) {
	t.Helper()
	lis, err := net.Listen("tcp", "127.0.0.1:0")
	require.NoError(t, err)
	srv := grpc.NewServer()
	pb.RegisterMatchingServiceServer(srv, matching.NewServer(nil, name))
	go func() { _ = srv.Serve(lis) }()
	var once atomic.Bool
	stop = func() {
		if once.CompareAndSwap(false, true) {
			srv.Stop()
		}
	}
	t.Cleanup(stop)
	return lis.Addr().String(), stop
}

func call(t *testing.T, client pb.MatchingServiceClient) (string, error) {
	t.Helper()
	ctx, cancel := context.WithTimeout(context.Background(), 2*time.Second)
	defer cancel()
	resp, err := client.RankDrivers(ctx, &pb.RankDriversRequest{
		Rider: &pb.RiderRequest{RiderId: "load-test", PickupPointId: "pickup-station-ecity"},
	})
	if err != nil {
		return "", err
	}
	return resp.ServedBy, nil
}

// TestDialSpreadsLoadAcrossReplicas is the unit-test version of "scale
// Matching from 1 to 3 pods". The manual resolver plays the role of a headless
// Kubernetes Service: it hands the client one address per pod.
func TestDialSpreadsLoadAcrossReplicas(t *testing.T) {
	r := manual.NewBuilderWithScheme("test")
	var addrs []resolver.Address
	for i := 1; i <= 3; i++ {
		addr, _ := replica(t, fmt.Sprintf("matching-pod-%d", i))
		addrs = append(addrs, resolver.Address{Addr: addr})
	}
	r.InitialState(resolver.State{Addresses: addrs})

	conn, err := Dial("test:///matching", grpc.WithResolvers(r))
	require.NoError(t, err)
	defer conn.Close()
	client := pb.NewMatchingServiceClient(conn)

	servedBy := map[string]int{}
	for i := 0; i < 300; i++ {
		pod, err := call(t, client)
		require.NoError(t, err)
		servedBy[pod]++
	}

	require.Len(t, servedBy, 3, "every replica should get traffic: %v", servedBy)
	for pod, n := range servedBy {
		assert.InDelta(t, 100, n, 10, "round robin should split evenly (%s got %d)", pod, n)
	}
}

// TestDialSurvivesAKilledReplica is "kubectl delete pod" in a unit test:
// kill one of two replicas mid-traffic and no call fails.
func TestDialSurvivesAKilledReplica(t *testing.T) {
	r := manual.NewBuilderWithScheme("test")
	addrA, stopA := replica(t, "matching-pod-a")
	addrB, _ := replica(t, "matching-pod-b")
	r.InitialState(resolver.State{Addresses: []resolver.Address{{Addr: addrA}, {Addr: addrB}}})

	conn, err := Dial("test:///matching", grpc.WithResolvers(r))
	require.NoError(t, err)
	defer conn.Close()
	client := pb.NewMatchingServiceClient(conn)

	for i := 0; i < 20; i++ {
		_, err := call(t, client)
		require.NoError(t, err)
	}

	stopA() // the pod dies

	servedBy := map[string]int{}
	for i := 0; i < 50; i++ {
		pod, err := call(t, client)
		require.NoError(t, err, "call %d failed after a replica died", i)
		servedBy[pod]++
	}
	assert.Equal(t, map[string]int{"matching-pod-b": 50}, servedBy)
}

// TestServiceHealthAndGracefulShutdown checks the chassis lifecycle: health is
// SERVING while up and NOT_SERVING once shutdown starts.
func TestServiceHealthAndGracefulShutdown(t *testing.T) {
	t.Setenv("SHUTDOWN_DRAIN_DELAY", "300ms")
	svc := New("matching")
	pb.RegisterMatchingServiceServer(svc.GRPC, matching.NewServer(nil, "x"))

	lis, err := net.Listen("tcp", "127.0.0.1:0")
	require.NoError(t, err)
	addr := lis.Addr().String()
	lis.Close()

	go func() { _ = svc.Run(addr) }()

	conn, err := Dial(addr)
	require.NoError(t, err)
	defer conn.Close()
	health := healthpb.NewHealthClient(conn)

	check := func(service string) healthpb.HealthCheckResponse_ServingStatus {
		ctx, cancel := context.WithTimeout(context.Background(), time.Second)
		defer cancel()
		resp, err := health.Check(ctx, &healthpb.HealthCheckRequest{Service: service})
		if err != nil {
			return healthpb.HealthCheckResponse_UNKNOWN
		}
		return resp.Status
	}

	require.Eventually(t, func() bool { return check("") == healthpb.HealthCheckResponse_SERVING }, 2*time.Second, 20*time.Millisecond)
	assert.Equal(t, healthpb.HealthCheckResponse_SERVING, check("matching.MatchingService"))

	go svc.Shutdown()
	require.Eventually(t, func() bool { return check("") == healthpb.HealthCheckResponse_NOT_SERVING }, time.Second, 10*time.Millisecond,
		"probes must see NOT_SERVING during the drain window")
}

// TestDefaultClientPinsOnePod shows the bug the chassis fixes. A plain gRPC
// client uses pick_first: it connects to ONE backend and sends everything
// there, so "scale Matching to 5 replicas" changes nothing for that client.
func TestDefaultClientPinsOnePod(t *testing.T) {
	r := manual.NewBuilderWithScheme("test")
	var addrs []resolver.Address
	for i := 1; i <= 3; i++ {
		addr, _ := replica(t, fmt.Sprintf("matching-pod-%d", i))
		addrs = append(addrs, resolver.Address{Addr: addr})
	}
	r.InitialState(resolver.State{Addresses: addrs})

	conn, err := grpc.NewClient("test:///matching", grpc.WithResolvers(r),
		grpc.WithTransportCredentials(insecure.NewCredentials())) // no service config
	require.NoError(t, err)
	defer conn.Close()
	client := pb.NewMatchingServiceClient(conn)

	servedBy := map[string]int{}
	for i := 0; i < 300; i++ {
		pod, err := call(t, client)
		require.NoError(t, err)
		servedBy[pod]++
	}
	assert.Len(t, servedBy, 1, "pick_first sends all 300 calls to a single pod: %v", servedBy)
}

// TestInstance checks which identity a replica reports (the UI shows it as
// "Decided by <instance>").
func TestInstance(t *testing.T) {
	host, err := os.Hostname()
	require.NoError(t, err)

	tests := []struct {
		name, podName, renderID, want string
	}{
		{name: "kubernetes or compose", podName: "matching-7d9f-x2k4p", renderID: "srv-abc-123", want: "matching-7d9f-x2k4p"},
		{name: "render", renderID: "srv-abc-123", want: "srv-abc-123"},
		{name: "plain process", want: host},
	}
	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			t.Setenv("POD_NAME", tt.podName)
			t.Setenv("RENDER_INSTANCE_ID", tt.renderID)
			assert.Equal(t, tt.want, Instance())
		})
	}
}

// TestTarget covers how the gateway finds each downstream service.
func TestTarget(t *testing.T) {
	tests := []struct {
		name             string
		addr, host, port string
		want             string
		wantErr          bool
	}{
		{name: "default for run-local", want: "localhost:50053"},
		{name: "explicit address wins", addr: "dns:///matching:50053", host: "ignored", port: "1", want: "dns:///matching:50053"},
		{name: "render discovery hostname", host: "lastmile-matching-discovery", port: "10000", want: "dns:///lastmile-matching-discovery:10000"},
		{name: "discovery host without port", host: "lastmile-matching-discovery", wantErr: true},
	}
	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			t.Setenv("MATCHING_ADDR", tt.addr)
			t.Setenv("MATCHING_DISCOVERY_HOST", tt.host)
			t.Setenv("MATCHING_PORT", tt.port)
			got, err := Target("MATCHING", "localhost:50053")
			if tt.wantErr {
				require.Error(t, err)
				return
			}
			require.NoError(t, err)
			assert.Equal(t, tt.want, got)
		})
	}
}
