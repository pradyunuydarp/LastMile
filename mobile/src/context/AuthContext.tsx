import React, { createContext, useContext, useState, useEffect } from 'react';
import { supabase } from '../services/supabase';

type AppRole = 'driver' | 'rider';

interface User {
    id: string;
    email: string;
    name?: string;
    role?: string | number;
    // allow passthrough metadata from Supabase or custom auth sources
    user_metadata?: Record<string, unknown>;
}

interface AuthContextType {
    session: any | null;
    user: User | null;
    role: AppRole | undefined;
    signIn: (token: string, user: User) => void;
    signOut: () => void;
}

const AuthContext = createContext<AuthContextType>({
    session: null,
    user: null,
    role: undefined,
    signIn: () => { },
    signOut: () => { },
});

const deriveRole = (input: unknown): AppRole | undefined => {
    if (typeof input === 'number') {
        if (input === 2) {
            return 'driver';
        }
        if (input === 1) {
            return 'rider';
        }
    }
    if (typeof input === 'string') {
        const normalized = input.toLowerCase();
        if (normalized.includes('driver')) {
            return 'driver';
        }
        if (normalized.includes('rider')) {
            return 'rider';
        }
    }
    return undefined;
};

export const AuthProvider: React.FC<{ children: React.ReactNode }> = ({ children }) => {
    const [session, setSession] = useState<any>(null);
    const [user, setUser] = useState<User | null>(null);
    const [role, setRole] = useState<AppRole | undefined>(undefined);

    useEffect(() => {
        // Check for existing Supabase session (if any)
        supabase.auth.getSession().then(({ data: { session } }) => {
            if (session) {
                setSession(session);
                // Note: Supabase session user might not match our custom backend user structure exactly
                // but for now we rely on manual signIn for full user data
            }
        });
    }, []);

    const signIn = (token: string, userData: User) => {
        const normalizedRole =
            deriveRole(userData?.role) ??
            deriveRole((userData?.user_metadata as any)?.role);

        setSession({ access_token: token });
        setUser({
            ...userData,
            role: normalizedRole ?? userData?.role,
        });
        setRole(normalizedRole);
    };

    const signOut = async () => {
        await supabase.auth.signOut();
        setSession(null);
        setUser(null);
        setRole(undefined);
    };

    return (
        <AuthContext.Provider value={{ session, user, role, signIn, signOut }}>
            {children}
        </AuthContext.Provider>
    );
};

export const useAuth = () => useContext(AuthContext);
