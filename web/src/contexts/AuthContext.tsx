import React, { createContext, useContext, useEffect, useState } from 'react';
import type { GatewaySession, GatewayUser } from '../lib/auth';

interface AuthContextType {
    session: GatewaySession | null;
    user: GatewayUser | null;
    role: 'driver' | 'rider';
    loading: boolean;
    setSession: (session: GatewaySession | null) => void;
    signOut: () => void;
}

const AuthContext = createContext<AuthContextType | undefined>(undefined);

const STORAGE_KEY = 'lastmile-web-session';
const INTENT_ROLE_KEY = 'intentRole';

const deriveRole = (user: GatewayUser | null, fallback?: string | null): 'driver' | 'rider' => {
    const meta = user?.user_metadata?.role ?? user?.role;
    const value = typeof meta === 'string'
        ? meta
        : typeof meta === 'number'
            ? (meta === 2 ? 'driver' : 'rider')
            : fallback;
    if (typeof value === 'string' && value.toLowerCase().includes('driver')) {
        return 'driver';
    }
    return 'rider';
};

export function AuthProvider({ children }: { children: React.ReactNode }) {
    const [session, setSessionState] = useState<GatewaySession | null>(null);
    const [loading, setLoading] = useState(true);
    const [role, setRole] = useState<'driver' | 'rider'>(() => {
        if (typeof window !== 'undefined') {
            const stored = window.localStorage.getItem(INTENT_ROLE_KEY);
            if (stored && stored.toLowerCase() === 'driver') {
                return 'driver';
            }
        }
        return 'rider';
    });

    useEffect(() => {
        if (typeof window === 'undefined') {
            setLoading(false);
            return;
        }
        const stored = window.localStorage.getItem(STORAGE_KEY);
        if (stored) {
            try {
                const parsed = JSON.parse(stored) as GatewaySession;
                setSessionState(parsed);
            } catch (err) {
                console.warn('Failed to parse stored session', err);
                window.localStorage.removeItem(STORAGE_KEY);
            }
        }
        setLoading(false);
    }, []);

    useEffect(() => {
        if (typeof window === 'undefined') {
            return;
        }
        if (session) {
            window.localStorage.setItem(STORAGE_KEY, JSON.stringify(session));
        } else {
            window.localStorage.removeItem(STORAGE_KEY);
        }
    }, [session]);

    useEffect(() => {
        const fallback = typeof window !== 'undefined' ? window.localStorage.getItem(INTENT_ROLE_KEY) : null;
        setRole(deriveRole(session?.user ?? null, fallback));
    }, [session]);

    const setSession = (next: GatewaySession | null) => {
        setSessionState(next);
        setLoading(false);
    };

    const signOut = () => {
        setSession(null);
        if (typeof window !== 'undefined') {
            window.localStorage.removeItem(STORAGE_KEY);
        }
    };

    const value = {
        session,
        user: session?.user ?? null,
        role,
        loading,
        setSession,
        signOut,
    };

    return <AuthContext.Provider value={value}>{children}</AuthContext.Provider>;
}

export function useAuth() {
    const context = useContext(AuthContext);
    if (context === undefined) {
        throw new Error('useAuth must be used within an AuthProvider');
    }
    return context;
}
