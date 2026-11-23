# TODO

1. Update driver experience (mobile + web) so logged-in drivers land on driver-focused UI that shows Riders tab only, and exposes a route-planning form.
   - Form must capture metro route metadata: pickup points (20-30 Bangalore locations with mapped stations), target stations, and available seats.
   - Allow drivers to build a list of pickup points + seat counts, persist via Driver Service, and expose "Start Trip" + "Start Simulated Trip" actions.
   - Integrate map view (e.g., react-native-maps / Google/Apple Maps) showing driver position; simulated mode should animate driver across the chosen route for demo mode.
   - As simulated/real trips progress, broadcast location updates through Location Service so matched riders can see movement on their map components.
2. Update rider experience (mobile + web) so logged-in riders see Drivers tab and a booking form instead of the command input.
   - Provide searchable dropdown of the hardcoded pickup points; enforce selection from this list.
   - On "Book" submit, call backend `/rides/book` (Gateway → Matching/Driver/Rider services) with pickup info, rider metadata, and desired destination.
   - Backend should locate drivers whose active route includes the pickup point and who haven’t passed it yet; request driver confirmation sequentially before allocating seat + creating Trip.
3. Ensure driver view lists nearby rider requests from `/drivers/requests`, with map markers for each rider location (no hardcoded trips shown).
4. Expand backend services as needed to support: driver route registration (Driver Service), rider booking (Rider Service), matching logic (Matching Service), trip lifecycle + location streaming.
   - Matching must enforce destination alignment per spec and trigger when driver nears station or pickup area; use Station metadata for “nearby” locations.
   - Rider booking should drop passengers at nearest metro station derived from location mapping.
5. Add/restore map components for both driver and rider tabs across mobile + web clients.
6. Document required env updates (.env files pointing to ngrok HTTPS gateway) and ensure `start_services.sh`-started backend cooperates with the new flows.
7. Mirror the web driver console in the iOS app:
   - Introduce a Trips tab visible only to drivers that mirrors the web trip-booking screens, including pickup/route forms, rider network view, accept buttons, and active trip state.
   - Within Trips, allow saving pickup points, seats, and route metadata to the Driver Service, then trigger Start Trip / Start Simulated Trip actions that feed location updates to matched riders.
   - Riders-only views must be hidden when a driver is signed in; rider tab should surface waiting riders for pickup points along the selected route and show their locations on the map.
8. Troubleshoot Supabase auth failures seen in the user/logs (requests targeting `postgresql://.../auth/v1/token`); confirm `SUPABASE_URL` is the HTTPS API endpoint in all service envs and not the Postgres connection string, plus refresh tokens as needed.

9. Introduce realtime trip orchestration per new requirements:
   - **DB phase:** extend `schema.sql` with idempotent tables for driver_routes, driver_trips, rider_requests, trip_events, and driver_route_pickups to persist plans, seat counts, pickup ordering, and historical trips. Ensure migrations guard with `if not exists`.
   - **Backend phase:** augment gateway (and related services if needed) with socket.io-compatible websocket server that creates per-trip "rooms", streams driver and rider locations, enforces pickup sequencing (only riders whose pickup is upcoming and not yet passed), handles seat decrement/increment when trips start/complete, and archives events when the room closes. Integrate with Driver, Rider, Matching, Trip, Location services to drive state changes.
   - **Frontend phase (web + iOS):** connect to the socket transport to show rider network to drivers (and driver network to riders), render live maps for both parties, surface seat counts + trip progress, and provide controls for accepting riders, starting trips, and concluding drop-offs. Hide rider-only data from riders' booking page; keep map/tracker in Drivers' Trips tab.
