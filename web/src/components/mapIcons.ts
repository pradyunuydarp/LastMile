import L from 'leaflet';

// Apple-Maps-like markers built with L.divIcon. Colours come from the CSS
// system-colour variables in index.css, so they follow light/dark mode.

const svg = (body: string) =>
    `<svg xmlns="http://www.w3.org/2000/svg" viewBox="0 0 24 24" fill="none" stroke="currentColor" stroke-width="2.2" stroke-linecap="round" stroke-linejoin="round" aria-hidden="true">${body}</svg>`;

const GLYPHS = {
    car: svg('<path d="M19 17h2c.6 0 1-.4 1-1v-3c0-.9-.7-1.7-1.5-1.9C18.7 10.6 16 10 16 10s-1.3-1.4-2.2-2.3c-.5-.4-1.1-.7-1.8-.7H5c-.6 0-1.1.4-1.4.9l-1.4 2.9A3.7 3.7 0 0 0 2 12v4c0 .6.4 1 1 1h2"/><circle cx="7" cy="17" r="2"/><path d="M9 17h6"/><circle cx="17" cy="17" r="2"/>'),
    train: svg('<path d="M8 3.1V7a4 4 0 0 0 8 0V3.1"/><path d="m9 15-1-1"/><path d="m15 15 1-1"/><path d="M9 19c-2.8 0-5-2.2-5-5v-4a8 8 0 0 1 16 0v4c0 2.8-2.2 5-5 5Z"/><path d="m8 19-2 3"/><path d="m16 19 2 3"/>'),
    flag: svg('<path d="M4 15s1-1 4-1 5 2 8 2 4-1 4-1V3s-1 1-4 1-5-2-8-2-4 1-4 1z"/><line x1="4" x2="4" y1="22" y2="15"/>'),
    person: svg('<path d="M19 21v-2a4 4 0 0 0-4-4H9a4 4 0 0 0-4 4v2"/><circle cx="12" cy="7" r="4"/>'),
};

type SystemColor = 'sys-blue' | 'sys-green' | 'sys-red' | 'sys-orange' | 'sys-indigo' | 'sys-teal' | 'sys-gray';

const badge = (color: SystemColor, inner: string, size: number, extraClass = '') =>
    L.divIcon({
        className: `lm-marker ${extraClass}`.trim(),
        html: `<div class="lm-badge" style="--c: var(--${color})">${inner}</div>`,
        iconSize: [size, size],
        iconAnchor: [size / 2, size / 2],
        popupAnchor: [0, -size / 2 - 2],
    });

export const dotIcon = (color: SystemColor = 'sys-red', size = 18) =>
    L.divIcon({
        className: 'lm-marker',
        html: `<div class="lm-dot" style="--c: var(--${color})"></div>`,
        iconSize: [size, size],
        iconAnchor: [size / 2, size / 2],
        popupAnchor: [0, -size / 2 - 2],
    });

export const mapIcons = {
    /** Generic marker (used as Leaflet's default icon). */
    dot: dotIcon('sys-red'),
    /** "You are here" — the blue location puck. */
    user: L.divIcon({
        className: 'lm-marker',
        html: '<div class="lm-user"></div>',
        iconSize: [24, 24],
        iconAnchor: [12, 12],
        popupAnchor: [0, -14],
    }),
    driver: badge('sys-blue', GLYPHS.car, 32),
    /** Driver car in the trip room; glides between location updates. */
    car: badge('sys-blue', GLYPHS.car, 40, 'transition-marker'),
    station: badge('sys-green', GLYPHS.train, 30),
    rider: badge('sys-orange', GLYPHS.person, 28),
    destination: badge('sys-red', GLYPHS.flag, 32),
};

const numbered = new Map<number, L.DivIcon>();

/** Numbered stop marker (1, 2, 3…) for a planned route. */
export function stopIcon(n: number): L.DivIcon {
    let icon = numbered.get(n);
    if (!icon) {
        icon = badge('sys-blue', String(n), 26);
        numbered.set(n, icon);
    }
    return icon;
}
