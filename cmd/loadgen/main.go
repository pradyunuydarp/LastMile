package main

import (
	"context"
	"log"
	"os"
	"strconv"
	"sync"
	"time"

	"google.golang.org/grpc"
	"google.golang.org/grpc/credentials/insecure"

	pb "lastmile/gen/go/matching"
)

func main() {
	addr := os.Getenv("MATCHING_ADDR")
	if addr == "" {
		addr = "matching.lastmile.svc.cluster.local:50053"
	}

	concurrencyStr := os.Getenv("CONCURRENCY")
	concurrency := 10
	if concurrencyStr != "" {
		if val, err := strconv.Atoi(concurrencyStr); err == nil {
			concurrency = val
		}
	}

	log.Printf("Starting load generation against %s with %d workers", addr, concurrency)

	conn, err := grpc.Dial(addr,
		grpc.WithTransportCredentials(insecure.NewCredentials()),
		grpc.WithDefaultServiceConfig(`{"loadBalancingPolicy":"round_robin"}`),
	)
	if err != nil {
		log.Fatalf("did not connect: %v", err)
	}
	defer conn.Close()

	client := pb.NewMatchingServiceClient(conn)

	var wg sync.WaitGroup
	wg.Add(concurrency)

	for i := 0; i < concurrency; i++ {
		go func(workerID int) {
			defer wg.Done()
			for {
				ctx, cancel := context.WithTimeout(context.Background(), time.Second)
				_, err := client.Match(ctx, &pb.MatchRequest{
					DriverId:  "driver-load-test",
					StationId: "station-load-test",
				})
				if err != nil {
					log.Printf("Worker %d: Error calling Match: %v", workerID, err)
					// Backoff slightly on error
					time.Sleep(100 * time.Millisecond)
				}
				cancel()
				// No sleep here to maximize load, or small sleep to control rate
				// time.Sleep(10 * time.Millisecond)
			}
		}(i)
	}

	wg.Wait()
}
