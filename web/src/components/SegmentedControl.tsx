import type { CSSProperties } from 'react';

export type SegmentOption = { key: string; label: string };

interface SegmentedControlProps {
    options: readonly SegmentOption[];
    value: string;
    onChange: (key: string) => void;
    ariaLabel: string;
    size?: 'regular' | 'large';
    className?: string;
}

/**
 * iOS-style segmented control. Each segment is a plain <button> (named by its
 * label) with aria-pressed; the selected "thumb" slides between segments.
 */
export function SegmentedControl({ options, value, onChange, ariaLabel, size = 'regular', className = '' }: SegmentedControlProps) {
    const index = Math.max(0, options.findIndex((option) => option.key === value));
    const style = { '--n': options.length, '--i': index } as CSSProperties;
    return (
        <div role="group" aria-label={ariaLabel} className={`segmented ${size === 'large' ? 'segmented-lg' : ''} ${className}`} style={style}>
            <span aria-hidden="true" className="segmented-thumb" />
            {options.map((option) => (
                <button
                    key={option.key}
                    type="button"
                    aria-pressed={option.key === value}
                    onClick={() => onChange(option.key)}
                    className="segment"
                >
                    {option.label}
                </button>
            ))}
        </div>
    );
}
