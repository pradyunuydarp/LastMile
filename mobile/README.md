# LastMile Mobile (React Native)

A lightweight React Native (Expo) companion that visualizes the existing LastMile gRPC backend: driver arrivals, riders waiting at stations, and live trip creation. The UI leans on iOS-inspired liquid-glass panels so dispatchers (or riders) can glance at system vitals.

## Highlights
- Dashboard that surfaces key metrics (pending matches, seats, rider wait) plus the next actionable trip.
- Driver and rider directories backed by the same `driver`, `rider`, and `matching` domain models defined in `api/*.proto`.
- Manual “Match now” control wired to the backend gateway so you can trigger a gRPC `Match` flow or fall back to the provided mock snapshot.
- Shared glassmorphism primitives (`GlassCard`, `LiquidButton`) for iOS-like styling.

## Stack versions (Nov 2025)
- **Expo SDK 54 / React Native 0.81 / React 19.1** for the iOS-like liquid UI and modern RN APIs.
- **React Navigation 7** powers the bottom tabs with the latest theming capabilities.
- **TypeScript 5.9 + ESLint 8** (the React Native community config is still limited to ESLint 8 until eslint-plugin-react-native ships ESLint 9 support).

## Project layout
```
mobile/
  App.tsx                # Expo entry point
  app.json               # Expo project metadata
  package.json           # Dependencies + scripts
  src/
    App.tsx              # Root providers + navigation container
    navigation/          # Bottom tab navigator
    screens/             # Dashboard, Drivers, Riders
    components/          # Glass UI primitives & cards
    services/            # Backend context + HTTP gateway + mock data
    theme/               # Palette + spacing tokens
    types/               # Shared TypeScript definitions mirroring protos
```

## Run it locally
```bash
cd mobile
pnpm install
pnpm start    # then press i for iOS simulator or scan the QR code
```

### Configure backend access
The Go services speak gRPC. The Expo app talks to them via the HTTP gateway spun up by `cmd/gateway`, which now exposes:

- `GET /aggregates/snapshot` → aggregated `BackendSnapshot` (drivers, riders, trips, metrics).
- `GET /metro/pickups` → curated pickup clusters (20+ Bengaluru landmarks tied to metro stations).
- `POST /drivers/routes` → configure the stations + pickup list + seat count for the signed-in driver.
- `POST /drivers/trip/start` → kick off a live or simulated trip so riders can follow the map.
- `POST /rides/book` → rider “book” command that asks nearby drivers sequentially.

The repo now ships `mobile/.env` so you can point Expo at your ngrok HTTPS gateway (and opt out of the workspace-root metro setting Expo 54 enables by default):

```
EXPO_PUBLIC_API_URL=https://tumular-unheated-briella.ngrok-free.dev
EXPO_PUBLIC_SUPABASE_URL=...
EXPO_PUBLIC_SUPABASE_ANON_KEY=...
EXPO_NO_METRO_WORKSPACE_ROOT=1
```

Update the `EXPO_PUBLIC_API_URL` entry to whatever ngrok URL you cut for the running gateway, then run `pnpm start` and Expo will inject the values automatically. If the variable is unset, the UI falls back to deterministic mock data from `src/services/mockData.ts` so designers can still demo the glass UI.

## Next steps
1. Replace the mock snapshot with real gRPC data by standing up a grpc-gateway service that composes `DriverService`, `RiderService`, and `MatchingService`.
2. Add authentication (e.g., hitting the `User` service) and rider/driver-facing flows.
3. Wire push notifications (Notification service) so new matches surface instantly on device.
