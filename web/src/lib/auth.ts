import { request } from './backend';

export type GatewayUser = {
  id: string;
  email: string;
  name?: string;
  role?: number | string;
  user_metadata?: Record<string, unknown>;
};

export type GatewaySession = {
  accessToken: string;
  user: GatewayUser;
};

type SignInResponse = {
  id: string;
  email: string;
  access_token: string;
  user?: {
    id: string;
    name?: string;
    email?: string;
    role?: number | string;
  };
};

const ROLE_TO_ENUM: Record<'rider' | 'driver', number> = {
  rider: 1,
  driver: 2,
};

const normalizeRole = (value?: number | string, fallback?: 'rider' | 'driver'): 'rider' | 'driver' => {
  if (typeof value === 'number') {
    return value === 2 ? 'driver' : 'rider';
  }
  if (typeof value === 'string') {
    const normalized = value.toLowerCase();
    if (normalized.includes('driver')) {
      return 'driver';
    }
  }
  return fallback ?? 'rider';
};

export function toGatewaySession(response: SignInResponse, intent?: 'rider' | 'driver'): GatewaySession {
  const role = normalizeRole(response.user?.role, intent);
  const name = response.user?.name ?? response.email?.split('@')[0] ?? 'LastMile User';
  return {
    accessToken: response.access_token,
    user: {
      id: response.user?.id ?? response.id,
      email: response.user?.email ?? response.email,
      name,
      role,
      user_metadata: {
        role,
        full_name: name,
      },
    },
  };
}

export async function signUp(payload: {
  email: string;
  password: string;
  name?: string;
  role: 'rider' | 'driver';
}): Promise<void> {
  await request('/auth/signup', {
    method: 'POST',
    body: JSON.stringify({
      email: payload.email,
      password: payload.password,
      name: payload.name ?? payload.email.split('@')[0],
      role: ROLE_TO_ENUM[payload.role],
    }),
  });
}

export async function signIn(payload: { email: string; password: string }, intent?: 'rider' | 'driver'): Promise<GatewaySession> {
  const response = await request<SignInResponse>('/auth/signin', {
    method: 'POST',
    body: JSON.stringify(payload),
  });
  return toGatewaySession(response, intent);
}

export async function forgotPassword(email: string): Promise<void> {
  await request('/auth/forgot-password', {
    method: 'POST',
    body: JSON.stringify({ email }),
  });
}
