package chassis

import (
	"google.golang.org/grpc"
	"google.golang.org/grpc/credentials/insecure"
)

// clientServiceConfig turns on two things for every outgoing call:
//
//  1. round_robin load balancing across *all* addresses the target resolves
//     to. Pair this with a headless Kubernetes Service (clusterIP: None) so
//     DNS returns one address per pod. With a normal ClusterIP Service, DNS
//     returns a single virtual IP, gRPC opens one HTTP/2 connection to it,
//     and every request lands on whichever pod kube-proxy picked first.
//
//  2. automatic retries on UNAVAILABLE. When a replica is killed mid-demo,
//     the call is transparently retried on a healthy one.
const clientServiceConfig = `{
  "loadBalancingConfig": [{"round_robin": {}}],
  "methodConfig": [{
    "name": [{}],
    "retryPolicy": {
      "maxAttempts": 4,
      "initialBackoff": "0.1s",
      "maxBackoff": "1s",
      "backoffMultiplier": 2,
      "retryableStatusCodes": ["UNAVAILABLE"]
    }
  }]
}`

// Dial opens a lazily-connecting client connection to target.
//
// Use a "dns:///host:port" target (the default scheme for grpc.NewClient)
// so the resolver can return multiple backends. The connection does not
// fail if the server is down at startup; calls fail (and retry) instead,
// which lets services start in any order.
func Dial(target string, extra ...grpc.DialOption) (*grpc.ClientConn, error) {
	opts := append([]grpc.DialOption{
		grpc.WithTransportCredentials(insecure.NewCredentials()),
		grpc.WithDefaultServiceConfig(clientServiceConfig),
	}, extra...)
	return grpc.NewClient(target, opts...)
}
