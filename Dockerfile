# One Dockerfile for every Go service:
#   docker build --build-arg SERVICE=matching -t lastmile-matching .

# ---- build ----------------------------------------------------------------
FROM golang:1.25-alpine AS builder
WORKDIR /src

# Download modules first so this layer is cached across code changes.
COPY go.mod go.sum ./
RUN go mod download

COPY cmd ./cmd
COPY internal ./internal
COPY gen ./gen

ARG SERVICE
RUN test -n "$SERVICE" || (echo "pass --build-arg SERVICE=<name>" && false)
RUN CGO_ENABLED=0 GOOS=linux go build -trimpath -ldflags="-s -w" -o /out/service ./cmd/${SERVICE}

# ---- run ------------------------------------------------------------------
# distroless/static: no shell, no package manager, just our binary.
FROM gcr.io/distroless/static-debian12:nonroot
COPY --from=builder /out/service /service
USER nonroot:nonroot
ENTRYPOINT ["/service"]
