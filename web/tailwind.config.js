/** @type {import('tailwindcss').Config} */

// Apple-style design tokens. Every colour is a CSS variable defined in
// src/index.css (light values on :root, dark values under
// prefers-color-scheme: dark), so components rarely need `dark:` variants.
const rgb = (name) => `rgb(var(--${name}) / <alpha-value>)`;

export default {
    content: [
        "./index.html",
        "./src/**/*.{js,ts,jsx,tsx}",
    ],
    darkMode: 'media',
    theme: {
        extend: {
            colors: {
                // System colours (fills, dots, icons, map markers).
                'sys-blue': rgb('sys-blue'),
                'sys-green': rgb('sys-green'),
                'sys-red': rgb('sys-red'),
                'sys-orange': rgb('sys-orange'),
                'sys-yellow': rgb('sys-yellow'),
                'sys-indigo': rgb('sys-indigo'),
                'sys-teal': rgb('sys-teal'),
                'sys-gray': rgb('sys-gray'),
                // Text-safe variants (>= 4.5:1 on grouped backgrounds and tints).
                'blue-text': rgb('blue-text'),
                'green-text': rgb('green-text'),
                'red-text': rgb('red-text'),
                'orange-text': rgb('orange-text'),
                'indigo-text': rgb('indigo-text'),
                // Filled-button background (white label >= 4.5:1).
                accent: rgb('accent'),
                // Labels.
                label: rgb('label'),
                'label-2': rgb('label-2'),
                'label-3': rgb('label-3'),
                // Backgrounds.
                grouped: rgb('bg-grouped'),
                'grouped-2': rgb('bg-grouped-2'),
                'grouped-3': rgb('bg-grouped-3'),
                fill: 'var(--fill)',
                'fill-2': 'var(--fill-2)',
                separator: 'var(--separator)',
            },
            fontFamily: {
                sans: ['-apple-system', 'BlinkMacSystemFont', '"SF Pro Text"', '"SF Pro Display"', '"Helvetica Neue"', 'Inter', 'system-ui', 'sans-serif'],
                display: ['-apple-system', 'BlinkMacSystemFont', '"SF Pro Display"', '"Helvetica Neue"', '"Inter Display"', 'Inter', 'system-ui', 'sans-serif'],
                mono: ['ui-monospace', '"SF Mono"', 'SFMono-Regular', 'Menlo', '"JetBrains Mono"', 'monospace'],
            },
            // iOS Dynamic Type sizes (Large default).
            fontSize: {
                'large-title': ['34px', { lineHeight: '41px', letterSpacing: '-0.022em', fontWeight: '700' }],
                'title-1': ['28px', { lineHeight: '34px', letterSpacing: '-0.02em', fontWeight: '700' }],
                'title-2': ['22px', { lineHeight: '28px', letterSpacing: '-0.016em', fontWeight: '700' }],
                'title-3': ['20px', { lineHeight: '25px', letterSpacing: '-0.014em', fontWeight: '600' }],
                headline: ['17px', { lineHeight: '22px', letterSpacing: '-0.012em', fontWeight: '600' }],
                body: ['17px', { lineHeight: '22px', letterSpacing: '-0.012em' }],
                callout: ['16px', { lineHeight: '21px', letterSpacing: '-0.01em' }],
                subhead: ['15px', { lineHeight: '20px', letterSpacing: '-0.008em' }],
                footnote: ['13px', { lineHeight: '18px', letterSpacing: '-0.004em' }],
                caption: ['12px', { lineHeight: '16px' }],
                'caption-2': ['11px', { lineHeight: '13px', letterSpacing: '0.006em' }],
            },
            borderRadius: {
                'card': '18px',
                'control': '10px',
            },
            boxShadow: {
                float: '0 0 0 0.5px rgb(0 0 0 / 0.04), 0 4px 12px rgb(0 0 0 / 0.08), 0 16px 40px rgb(0 0 0 / 0.12)',
                thumb: '0 3px 8px rgb(0 0 0 / 0.12), 0 3px 1px rgb(0 0 0 / 0.04)',
            },
            transitionTimingFunction: {
                spring: 'cubic-bezier(0.32, 0.72, 0, 1)',
            },
        },
    },
    plugins: [],
}
