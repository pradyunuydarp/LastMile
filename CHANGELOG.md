# Changelog

All notable changes to this project will be documented in this file.

## [0.2.0] - 2026-11-23
- Matching is now a real, stateless service (RankDrivers / FindRiders) called by the gateway; it scales 1→5.
- Microservice chassis: gRPC health, reflection, graceful shutdown, keepalive, round_robin + retries.
- Fixed data races in every in-memory store, gateway lock/timeout bugs, trip drop-off and approval bugs.
- Location uses real Bengaluru stations and reports approach-zone entry (the matching trigger).
- Demo-mode auth (no Supabase needed), bot riders, `/system/status`, `scripts/smoke.sh`.
- Kubernetes: gRPC probes, headless Matching Service, tuned HPA, Supabase key moved to a Secret.
- docker-compose.yml, fixed web image (Node 22), Claude Code skills, class demo materials.

## [0.1.0] - 2025-11-17
- Added full Project 1 (LastMile) specification summary to `AGENTS.md` for easy reference.
- Created this changelog to track repository changes.
- Added beginner-friendly inline documentation to the Station service Go code.

## [0.1.1] - 2025-11-17
- Expanded GoDoc-style function comments across Station service/store to aid new Go readers.

## [0.1.2] - 2025-11-17
- Marked `cmd/station/tut.go` as an ignore-only tutorial file to prevent build/test conflicts with the main service entrypoint.
