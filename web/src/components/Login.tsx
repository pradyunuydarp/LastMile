import React, { useState, useEffect } from 'react';
import { useNavigate } from 'react-router-dom';
import { Car, Mail, Lock, AlertCircle } from 'lucide-react';
import { signIn, signUp } from '../lib/auth';
import { useAuth } from '../contexts/AuthContext';

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
        <div className="min-h-screen bg-slate-900 flex items-center justify-center p-4">
            <div className="bg-slate-800 p-8 rounded-2xl shadow-xl w-full max-w-md border border-slate-700">
                <div className="flex flex-col items-center mb-8">
                    <div className="w-16 h-16 bg-blue-600 rounded-full flex items-center justify-center mb-4">
                        <Car className="w-8 h-8 text-white" />
                    </div>
                    <h1 className="text-2xl font-bold text-white">LastMile</h1>
                    <p className="text-slate-400 mt-2">
                        {isSignUp ? 'Create an account' : 'Sign in to continue'}
                    </p>
                </div>

                {error && (
                    <div className="bg-red-500/10 border border-red-500/50 rounded-lg p-4 mb-6 flex items-start gap-3">
                        <AlertCircle className="w-5 h-5 text-red-500 shrink-0 mt-0.5" />
                        <p className="text-red-200 text-sm">{error}</p>
                    </div>
                )}

                <form onSubmit={handleSubmit} className="space-y-4">
                    <div className="mb-2">
                        <p className="text-xs text-slate-400 uppercase tracking-wide mb-2">Log in as</p>
                        <div className="flex bg-slate-900 p-1 rounded-lg">
                            <button
                                type="button"
                                onClick={() => setRole('rider')}
                                className={`flex-1 py-2 rounded-md text-sm font-medium transition-colors ${role === 'rider'
                                        ? 'bg-slate-700 text-white shadow-sm'
                                        : 'text-slate-400 hover:text-slate-200'
                                    }`}
                            >
                                Rider
                            </button>
                            <button
                                type="button"
                                onClick={() => setRole('driver')}
                                className={`flex-1 py-2 rounded-md text-sm font-medium transition-colors ${role === 'driver'
                                        ? 'bg-blue-600 text-white shadow-sm'
                                        : 'text-slate-400 hover:text-slate-200'
                                    }`}
                            >
                                Driver
                            </button>
                        </div>
                    </div>

                    <div>
                        <label className="block text-sm font-medium text-slate-300 mb-1">
                            Email Address
                        </label>
                        <div className="relative">
                            <Mail className="absolute left-3 top-1/2 -translate-y-1/2 w-5 h-5 text-slate-500" />
                            <input
                                type="email"
                                value={email}
                                onChange={(e) => setEmail(e.target.value)}
                                className="w-full bg-slate-900 border border-slate-700 rounded-lg py-2.5 pl-10 pr-4 text-white placeholder:text-slate-600 focus:outline-none focus:ring-2 focus:ring-blue-500"
                                placeholder="driver@example.com"
                                required
                            />
                        </div>
                    </div>

                    <div>
                        <label className="block text-sm font-medium text-slate-300 mb-1">
                            Password
                        </label>
                        <div className="relative">
                            <Lock className="absolute left-3 top-1/2 -translate-y-1/2 w-5 h-5 text-slate-500" />
                            <input
                                type="password"
                                value={password}
                                onChange={(e) => setPassword(e.target.value)}
                                className="w-full bg-slate-900 border border-slate-700 rounded-lg py-2.5 pl-10 pr-4 text-white placeholder:text-slate-600 focus:outline-none focus:ring-2 focus:ring-blue-500"
                                placeholder="••••••••"
                                required
                            />
                        </div>
                    </div>

                    <button
                        type="submit"
                        disabled={loading}
                        className="w-full bg-blue-600 hover:bg-blue-700 text-white font-semibold py-2.5 rounded-lg transition-colors disabled:opacity-50 disabled:cursor-not-allowed"
                    >
                        {loading ? 'Processing...' : isSignUp ? 'Create Account' : 'Sign In'}
                    </button>
                </form>

                <div className="mt-6 text-center">
                    <button
                        onClick={() => setIsSignUp(!isSignUp)}
                        className="text-slate-400 hover:text-white text-sm transition-colors"
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
