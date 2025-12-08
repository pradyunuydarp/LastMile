import {
  BackendSnapshot,
  BookRidePayload,
  BookRideResponse,
  DriverRequestsResponse,
  DriverRoutePayload,
  DriverRouteResponse,
  PickupPoint,
  Trip,
} from '../types';
import { createMockTrip, mockSnapshot } from './mockData';
import { pickupCatalog } from './pickupCatalog';

const configuredBaseUrl = process.env.EXPO_PUBLIC_API_URL ?? 'http://localhost:8082';
const baseUrl = configuredBaseUrl.replace(/\/$/, '');

const defaultHeaders: Record<string, string> = {
  'Content-Type': 'application/json',
  'ngrok-skip-browser-warning': 'true',
};

const mergeHeaders = (overrides?: HeadersInit): Record<string, string> => {
  const headers: Record<string, string> = { ...defaultHeaders };
  if (!overrides) {
    return headers;
  }

  if (typeof Headers !== 'undefined' && overrides instanceof Headers) {
    overrides.forEach((value, key) => {
      headers[key] = value;
    });
    return headers;
  }

  if (Array.isArray(overrides)) {
    overrides.forEach(([key, value]) => {
      headers[key] = value;
    });
    return headers;
  }

  return {
    ...headers,
    ...(overrides as Record<string, string>),
  };
};

async function request<T>(path: string, init?: RequestInit): Promise<T> {
  const response = await fetch(`${baseUrl}${path}`, {
    ...init,
    headers: mergeHeaders(init?.headers ?? undefined),
  });

  if (!response.ok) {
    throw new Error(`Request failed with status ${response.status}`);
  }

  const raw = await response.text();
  if (!raw) {
    return undefined as T;
  }
  return JSON.parse(raw) as T;
}

export class BackendGateway {
  async fetchSnapshot(): Promise<BackendSnapshot> {
    try {
      return await request<BackendSnapshot>('/aggregates/snapshot');
    } catch (error) {
      console.warn(`[mobile] Snapshot request failed at ${baseUrl}, using mock data`, error);
      return mockSnapshot;
    }
  }

  async triggerMatch(driverId: string, stationId: string): Promise<Trip> {
    try {
      return await request<Trip>('/matching/match', {
        method: 'POST',
        body: JSON.stringify({ driverId, stationId }),
      });
    } catch (error) {
      console.warn(`[mobile] Match request failed at ${baseUrl}, using mock trip`, error);
      return createMockTrip(driverId, stationId);
    }
  }

  async fetchDriverRequests(driverId: string): Promise<DriverRequestsResponse> {
    if (!driverId) {
      throw new Error('driverId is required');
    }
    return request<DriverRequestsResponse>(`/drivers/requests?driverId=${driverId}`);
  }

  async bookRide(payload: BookRidePayload): Promise<BookRideResponse> {
    return request<BookRideResponse>('/rides/book', {
      method: 'POST',
      body: JSON.stringify(payload),
    });
  }

  async fetchPickupPoints(): Promise<PickupPoint[]> {
    try {
      const response = await request<{ pickupPoints: PickupPoint[] }>('/metro/pickups');
      return response.pickupPoints;
    } catch (error) {
      console.warn('[mobile] pickup fetch failed, using fallback catalog', error);
      return pickupCatalog;
    }
  }

  async saveDriverRoute(payload: DriverRoutePayload): Promise<DriverRouteResponse> {
    return request<DriverRouteResponse>('/drivers/routes', {
      method: 'POST',
      body: JSON.stringify(payload),
    });
  }

  async startDriverTrip(driverId: string, simulate?: boolean): Promise<DriverRouteResponse> {
    return request<DriverRouteResponse>('/drivers/trip/start', {
      method: 'POST',
      body: JSON.stringify({ driverId, simulate }),
    });
  }

  async acceptDriverRequest(driverId: string, riderId: string): Promise<Trip> {
    return request<Trip>('/drivers/requests/accept', {
      method: 'POST',
      body: JSON.stringify({ driverId, riderId }),
    });
  }

  async registerPushToken(userId: string, token: string): Promise<void> {
    await request('/notifications/token', {
      method: 'POST',
      body: JSON.stringify({ userId, token }),
    });
  }

  subscribeToLocationUpdates(driverId: string, onUpdate: (update: any) => void): () => void {
    // Replace http/https with ws/wss
    const wsProtocol = baseUrl.startsWith('https') ? 'wss' : 'ws';
    const wsUrl = baseUrl.replace(/^http(s)?/, 'ws$1');
    const socket = new WebSocket(
      `${wsUrl}/location/stream?driverId=${encodeURIComponent(driverId)}&ngrok-skip-browser-warning=true`,
    );

    socket.onmessage = (event) => {
      try {
        const update = JSON.parse(event.data);
        onUpdate(update);
      } catch (e) {
        console.warn('Failed to parse location update', e);
      }
    };

    socket.onerror = (e) => {
      console.warn('WebSocket error', e);
    };

    return () => {
      socket.close();
    };
  }

  async signUp(data: any): Promise<any> {
    const response = await fetch(`${baseUrl}/auth/signup`, {
      method: 'POST',
      headers: mergeHeaders(),
      body: JSON.stringify(data),
    });
    if (!response.ok) {
      const text = await response.text();
      throw new Error(text || 'Sign up failed');
    }
    return response.json();
  }

  async signIn(data: any): Promise<any> {
    const response = await fetch(`${baseUrl}/auth/signin`, {
      method: 'POST',
      headers: mergeHeaders(),
      body: JSON.stringify(data),
    });
    if (!response.ok) {
      const text = await response.text();
      throw new Error(text || 'Sign in failed');
    }
    return response.json();
  }

  async forgotPassword(email: string): Promise<any> {
    const response = await fetch(`${baseUrl}/auth/forgot-password`, {
      method: 'POST',
      headers: mergeHeaders(),
      body: JSON.stringify({ email }),
    });
    if (!response.ok) {
      const text = await response.text();
      throw new Error(text || 'Password reset failed');
    }
    return response.json();
  }
  async getUser(id: string): Promise<any> {
    return await request<any>(`/user/profile?id=${id}`);
  }
}

export const gateway = new BackendGateway();
