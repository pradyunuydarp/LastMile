# Stage 1: Build the Go binaries
FROM golang:1.25-alpine AS builder

WORKDIR /app

# Copy the Go modules files
COPY go.mod go.sum ./
# Download Go modules
RUN go mod download

# Copy the entire project
COPY . .

# Build all the services
RUN CGO_ENABLED=0 GOOS=linux go build -o /bin/station ./cmd/station
RUN CGO_ENABLED=0 GOOS=linux go build -o /bin/driver ./cmd/driver
RUN CGO_ENABLED=0 GOOS=linux go build -o /bin/rider ./cmd/rider
RUN CGO_ENABLED=0 GOOS=linux go build -o /bin/location ./cmd/location
RUN CGO_ENABLED=0 GOOS=linux go build -o /bin/matching ./cmd/matching
RUN CGO_ENABLED=0 GOOS=linux go build -o /bin/trip ./cmd/trip
RUN CGO_ENABLED=0 GOOS=linux go build -o /bin/notification ./cmd/notification
RUN CGO_ENABLED=0 GOOS=linux go build -o /bin/user ./cmd/user

# Stage 2: Create the final minimal image
FROM gcr.io/distroless/static-debian11

ARG SERVICE
COPY --from=builder /bin/$SERVICE /
ENTRYPOINT ["/"]
