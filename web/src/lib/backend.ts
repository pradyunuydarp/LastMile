import type {
  BackendSnapshot,
  BookRidePayload,
  BookRideResponse,
  DriverRequestsResponse,
  DriverRoutePayload,
  DriverRouteResponse,
  PickupPoint,
  Trip,
} from './types';
import { pickupCatalog } from './pickupCatalog';

const configuredBaseUrl = import.meta.env.VITE_GATEWAY_URL ?? '/api';
const baseUrl = configuredBaseUrl.replace(/\/$/, '');
const isNgrok = baseUrl.includes('ngrok');

function decoratePath(path: string): string {
  if (!isNgrok) {
    return path;
  }

  if (path.includes('ngrok-skip-browser-warning')) {
    return path;
  }

  const separator = path.includes('?') ? '&' : '?';
  return `${path}${separator}ngrok-skip-browser-warning=true`;
}

export async function request<T>(path: string, init?: RequestInit): Promise<T> {
  const headers = new Headers(init?.headers ?? {});
  headers.set('Content-Type', 'application/json');
  if (isNgrok) {
    headers.set('ngrok-skip-browser-warning', 'true');
  }

  const response = await fetch(`${baseUrl}${decoratePath(path)}`, {
    ...init,
    headers,
  });

  const raw = await response.text();
  if (!response.ok) {
    throw new Error(raw || `Request failed (${response.status})`);
  }

  try {
    return JSON.parse(raw) as T;
  } catch (err) {
    throw new Error(`Invalid JSON from ${path}: ${raw.substring(0, 120)}`);
  }
}

export async function fetchSnapshot(): Promise<BackendSnapshot> {
  return request<BackendSnapshot>('/aggregates/snapshot');
}

export async function fetchDriverRequests(driverId: string): Promise<DriverRequestsResponse> {
  return request<DriverRequestsResponse>(`/drivers/requests?driverId=${driverId}`);
}

export async function bookRide(payload: BookRidePayload): Promise<BookRideResponse> {
  return request<BookRideResponse>('/rides/book', {
    method: 'POST',
    body: JSON.stringify(payload),
  });
}

export async function fetchPickupPoints(): Promise<PickupPoint[]> {
  try {
    const data = await request<{ pickupPoints: PickupPoint[] }>('/metro/pickups');
    return data.pickupPoints;
  } catch (error) {
    console.warn('pickup fetch failed, using fallback catalog', error);
    return pickupCatalog;
  }
}

export async function saveDriverRoute(payload: DriverRoutePayload): Promise<DriverRouteResponse> {
  return request<DriverRouteResponse>('/drivers/routes', {
    method: 'POST',
    body: JSON.stringify(payload),
  });
}

export async function startDriverTrip(driverId: string, simulate?: boolean): Promise<DriverRouteResponse> {
  return request<DriverRouteResponse>('/drivers/trip/start', {
    method: 'POST',
    body: JSON.stringify({ driverId, simulate }),
  });
}

export async function acceptDriverRequest(payload: { driverId: string; riderId: string }): Promise<Trip> {
  return request<Trip>('/drivers/requests/accept', {
    method: 'POST',
    body: JSON.stringify(payload),
  });
}
