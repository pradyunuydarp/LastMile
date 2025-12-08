# Location Streaming and Map Integration Walkthrough

This document outlines the changes made to implement real-time location streaming from the driver to the customer using gRPC, WebSockets, and React Native Maps.

## Backend Changes

### Location Service (gRPC)
- **Proto Definition**: Updated `api/location.proto` to include `SubscribeLocationUpdates` RPC.
- **Implementation**: Updated `internal/location/server.go` to implement the streaming RPC using an in-memory pub/sub mechanism.
- **Tests**: Added `TestSubscribeLocationUpdates` in `internal/location/server_test.go` to verify the streaming logic.

### Gateway Service (HTTP/WebSocket)
- **WebSocket Handler**: Added `LocationStreamHandler` in `internal/api/gateway.go` to upgrade HTTP requests to WebSockets.
- **Proxy Logic**: The handler connects to the Location Service via gRPC `SubscribeLocationUpdates` and forwards received updates to the WebSocket client.
- **Configuration**: Updated `cmd/gateway/main.go` to initialize the Location Service client and register the `/location/stream` endpoint.

## Frontend Changes (Mobile)

### Dependencies
- Installed `react-native-maps` for map visualization.

### Services
- **Gateway Service**: Updated `mobile/src/services/gateway.ts` to include `subscribeToLocationUpdates`, which opens a WebSocket connection to the Gateway.

### Components
- **LocationMap**: Created `mobile/src/components/LocationMap.tsx`, a reusable component that displays a map and updates the driver's marker position in real-time based on the WebSocket stream.
- **DriverCard**: Integrated `LocationMap` into `mobile/src/components/DriverCard.tsx` to display live location for each driver in the list.

### Screens
- **DashboardScreen**: Integrated `LocationMap` into the dashboard to show the driver's location when a trip is highlighted.

## Build Fixes
- **Assets**: Restored valid `icon.png`, `splash.png`, and `adaptive-icon.png` to fix build failures caused by corrupted/empty asset files.
- **Configuration**: Added `react-native-maps` plugin to `mobile/app.json` to ensure native dependencies are linked correctly.

## Verification
- **Backend Tests**: All Go tests passed, including the new location streaming tests.
- **Frontend**: Code compiles and integrates the new components and services.

## Next Steps
- Run the mobile app in a simulator to visually verify the map and location updates.
- Implement actual GPS location updates from the driver app (currently simulated or manual).
## User Service & Authentication Implementation

### Backend Changes
- **Proto Definition**: Updated `api/user.proto` to include `SignUp`, `SignIn`, and `ForgotPassword` RPCs.
- **User Service**: Implemented Auth RPCs in `internal/user/server.go`. Currently uses a mock implementation but is structured to integrate with Supabase.
- **Gateway Service**: Added Auth handlers (`/auth/signup`, `/auth/signin`, `/auth/forgot-password`) in `internal/api/gateway.go` to proxy requests to the User Service.
- **Tests**: Added unit tests in `internal/user/server_test.go` to verify Auth RPCs.

### Frontend Changes (Mobile)
- **Screens**: Created `SignInScreen`, `SignUpScreen`, and `ForgotPasswordScreen` with a modern dark UI using `GlassCard` aesthetics.
- **Navigation**: Updated `AppNavigator.tsx` to include an Auth Stack and manage authentication state.
- **Gateway Service**: Updated `mobile/src/services/gateway.ts` to include methods for calling the backend Auth endpoints.
- **Dependencies**: Installed `@react-navigation/stack`.

### Database Schema
- **Schema**: Created `schema.sql` to define a `profiles` table that links to Supabase `auth.users` and stores additional user data like roles. Includes RLS policies and a trigger for automatic profile creation.

## Verification
- **Backend**: Verified Auth RPCs with `go test ./internal/user/...`.
- **Frontend**: Manual verification required (launch app, sign up, sign in).
