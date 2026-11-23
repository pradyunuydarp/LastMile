import { useSyncExternalStore } from 'react';

export type ColorScheme = 'light' | 'dark';

const QUERY = '(prefers-color-scheme: dark)';

function subscribe(onChange: () => void) {
    if (typeof window === 'undefined' || !window.matchMedia) {
        return () => {};
    }
    const media = window.matchMedia(QUERY);
    media.addEventListener('change', onChange);
    return () => media.removeEventListener('change', onChange);
}

function getSnapshot(): ColorScheme {
    return typeof window !== 'undefined' && window.matchMedia?.(QUERY).matches ? 'dark' : 'light';
}

/** Tracks the system light/dark appearance (prefers-color-scheme). */
export function useColorScheme(): ColorScheme {
    return useSyncExternalStore(subscribe, getSnapshot, () => 'light');
}
