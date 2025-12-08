# LastMile

## Mobile companion
- The new Expo/React Native app lives in `mobile/`.
- See `mobile/README.md` for setup plus how to point it at an HTTP gateway that fans out to the gRPC services.
- Default state uses mock data that mirrors the proto contracts so the UI is fully interactive even before wiring up real backends.
- Frontend tooling is managed with pnpm; run pnpm commands from the repo root or inside `mobile/`.

Driver workflows inside the mobile app now rely on the gateway’s `/metro/pickups`, `/drivers/routes`, and `/drivers/trip/start` HTTP surfaces so they can curate a set of pickup clusters, save available seats, and start either a live or simulated trip. Point Expo at your ngrok HTTPS URL (and opt out of the workspace-root metro heuristic) with the checked-in `mobile/.env` file:

```
EXPO_PUBLIC_API_URL=https://<your-ngrok>.ngrok-free.dev
EXPO_PUBLIC_SUPABASE_URL=...
EXPO_PUBLIC_SUPABASE_ANON_KEY=...
EXPO_NO_METRO_WORKSPACE_ROOT=1
```
