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
ARG SERVICE
RUN CGO_ENABLED=0 GOOS=linux go build -o /bin/${SERVICE} ./cmd/${SERVICE}


# Stage 2: Create the final minimal image
FROM gcr.io/distroless/static-debian11

ARG SERVICE
COPY --from=builder /bin/${SERVICE} /service
ENTRYPOINT ["/service"]
