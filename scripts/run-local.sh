#!/bin/bash

# Function to kill all background processes on exit
cleanup() {
    echo "Stopping all services..."
    kill $(jobs -p)
}
trap cleanup EXIT

echo "Starting LastMile Services..."

# Start core services
go run ./cmd/location &
echo "Location Service started (PID $!)"

go run ./cmd/driver &
echo "Driver Service started (PID $!)"

go run ./cmd/rider &
echo "Rider Service started (PID $!)"

go run ./cmd/station &
echo "Station Service started (PID $!)"

go run ./cmd/trip &
echo "Trip Service started (PID $!)"

go run ./cmd/matching &
echo "Matching Service started (PID $!)"

go run ./cmd/user &
echo "User Service started (PID $!)"

# Wait a bit for services to initialize
sleep 2

# Start Gateway (depends on others)
go run ./cmd/gateway &
echo "Gateway Service started (PID $!)"

# Wait for all background processes
wait
