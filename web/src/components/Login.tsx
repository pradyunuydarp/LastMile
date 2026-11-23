import React, { useState, useEffect } from 'react';
import { useNavigate } from 'react-router-dom';
import { Car, Mail, Lock, AlertCircle, ChevronRight } from 'lucide-react';
import { signIn, signUp } from '../lib/auth';
import { useAuth } from '../contexts/AuthContext';
import { SegmentedControl } from './SegmentedControl';

// Seeded by the User service when it runs without Supabase (demo mode).
const DEMO_ACCOUNTS: { label: string; email: string; role: 'rider' | 'driver' }[] = [
    { label: 'Asha · driver', email: 'driver@lastmile.demo', role: 'driver' },
    { label: 'Ravi · driver', email: 'driver2@lastmile.demo', role: 'driver' },
    { label: 'Priya · rider', email: 'rider@lastmile.demo', role: 'rider' },
    { label: 'Kabir · rider', email: 'rider2@lastmile.demo', role: 'rider' },
];

const ROLE_OPTIONS = [
    { key: 'rider', label: 'Rider' },
    { key: 'driver', label: 'Driver' },
] as const;

export function Login() {
    const [email, setEmail] = useState('');
    const [password, setPassword] = useState('');
    const [isSignUp, setIsSignUp] = useState(false);
    const [role, setRole] = useState<'rider' | 'driver'>(() => {
        if (typeof window !== 'undefined') {
            const stored = window.localStorage.getItem('intentRole');
            if (stored && stored.toLowerCase() === 'driver') {
                return 'driver';
            }
        }
        return 'rider';
    });
    const [error, setError] = useState<string | null>(null);
    const [loading, setLoading] = useState(false);
    const navigate = useNavigate();
    const { setSession } = useAuth();

    useEffect(() => {
        if (typeof window !== 'undefined') {
            window.localStorage.setItem('intentRole', role);
        }
    }, [role]);

    const handleSubmit = async (e: React.FormEvent) => {
        e.preventDefault();
        setError(null);
        setLoading(true);

        try {
            if (isSignUp) {
                await signUp({ email, password, role });
            }
            const session = await signIn({ email, password }, role);
            setSession(session);
            if (typeof window !== 'undefined') {
                window.localStorage.setItem('intentRole', role);
            }
            navigate('/dashboard');
        } catch (err) {
            setError((err as Error).message);
        } finally {
            setLoading(false);
        }
    };

    return (
        <div className="flex min-h-screen items-start justify-center bg-grouped px-4 py-10 sm:items-center sm:py-12">
            <div className="w-full max-w-[400px] animate-in">
                <div className="mb-8 flex flex-col items-center text-center">
                    <div className="mb-5 flex h-[72px] w-[72px] items-center justify-center rounded-[18px] bg-accent text-white shadow-float">
                        <Car className="h-9 w-9" strokeWidth={2} aria-hidden="true" />
                    </div>
                    <h1 className="text-large-title text-label">LastMile</h1>
                    <p className="mt-1.5 text-callout text-label-2">
                        {isSignUp ? 'Create an account to get started.' : 'Rides from the metro, matched as your driver arrives.'}
                    </p>
                </div>

                {error && (
                    <div role="alert" className="notice notice-red mb-4 animate-pop">
                        <AlertCircle className="mt-0.5 h-[18px] w-[18px] shrink-0" aria-hidden="true" />
                        <p>{error}</p>
                    </div>
                )}

                <form onSubmit={handleSubmit} className="space-y-4">
                    <div>
                        <p className="section-label mb-2">{isSignUp ? 'Sign up as' : 'Log in as'}</p>
                        <SegmentedControl
                            ariaLabel="Account type"
                            options={ROLE_OPTIONS}
                            value={role}
                            onChange={(key) => setRole(key as 'rider' | 'driver')}
                            size="large"
                        />
                    </div>

                    <div
                        className="list-plain transition-shadow duration-200 focus-within:shadow-[0_0_0_1px_rgb(var(--sys-blue)),0_0_0_4px_rgb(var(--sys-blue)/0.25)]"
                        style={{ ['--row-inset' as string]: '48px' }}
                    >
                        <label className="list-row cursor-text py-0">
                            <Mail className="h-[18px] w-[18px] shrink-0 text-label-2" aria-hidden="true" />
                            <span className="sr-only">Email address</span>
                            <input
                                type="email"
                                value={email}
                                onChange={(e) => setEmail(e.target.value)}
                                className="min-h-[50px] w-full bg-transparent text-body text-label outline-none"
                                placeholder="Email address"
                                autoComplete="email"
                                required
                            />
                        </label>
                        <label className="list-row cursor-text py-0">
                            <Lock className="h-[18px] w-[18px] shrink-0 text-label-2" aria-hidden="true" />
                            <span className="sr-only">Password</span>
                            <input
                                type="password"
                                value={password}
                                onChange={(e) => setPassword(e.target.value)}
                                className="min-h-[50px] w-full bg-transparent text-body text-label outline-none"
                                placeholder="Password"
                                autoComplete={isSignUp ? 'new-password' : 'current-password'}
                                required
                            />
                        </label>
                    </div>

                    <button type="submit" disabled={loading} className="btn btn-filled btn-lg w-full">
                        {loading ? <span className="spinner !border-white/30 !border-t-white" aria-hidden="true" /> : null}
                        {loading ? 'Processing...' : isSignUp ? 'Create Account' : 'Sign In'}
                    </button>
                </form>

                {!isSignUp && (
                    <div className="mt-8">
                        <div className="mb-2 flex items-baseline justify-between px-1">
                            <p className="section-label px-0">Demo accounts</p>
                            <p className="text-footnote text-label-2">Password: lastmile</p>
                        </div>
                        <div className="list-plain" style={{ ['--row-inset' as string]: '64px' }}>
                            {DEMO_ACCOUNTS.map((account) => {
                                const selected = email === account.email;
                                return (
                                    <button
                                        key={account.email}
                                        type="button"
                                        onClick={() => {
                                            setEmail(account.email);
                                            setPassword('lastmile');
                                            setRole(account.role);
                                        }}
                                        className="list-row w-full text-left transition-colors duration-200 hover:bg-fill active:bg-fill-2"
                                    >
                                        <span
                                            aria-hidden="true"
                                            className={`flex h-9 w-9 shrink-0 items-center justify-center rounded-full text-subhead font-semibold text-white ${account.role === 'driver' ? 'bg-sys-blue' : 'bg-sys-indigo'}`}
                                        >
                                            {account.label.charAt(0)}
                                        </span>
                                        <span className="min-w-0 flex-1">
                                            <span className="block text-callout font-semibold text-label">{account.label}</span>
                                            <span className="block truncate text-footnote text-label-2">{account.email}</span>
                                        </span>
                                        {selected ? (
                                            <span className="pill pill-blue">Selected</span>
                                        ) : (
                                            <ChevronRight className="h-[18px] w-[18px] shrink-0 text-label-3" aria-hidden="true" />
                                        )}
                                    </button>
                                );
                            })}
                        </div>
                    </div>
                )}

                <div className="mt-6 text-center">
                    <button
                        onClick={() => setIsSignUp(!isSignUp)}
                        className="btn btn-plain btn-sm"
                    >
                        {isSignUp
                            ? 'Already have an account? Sign in'
                            : "Don't have an account? Sign up"}
                    </button>
                </div>
            </div>
        </div>
    );
}
