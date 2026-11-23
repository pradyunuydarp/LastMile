import type { Driver } from '../lib/types';

interface DriverCardProps {
  driver: Driver;
}

const initials = (name: string) =>
  name
    .replace(/\(.*?\)/g, '')
    .split(/[^\p{L}\p{N}]+/u)
    .filter(Boolean)
    .slice(0, 2)
    .map((part) => part.charAt(0).toUpperCase())
    .join('') || '?';

const statusPill = (status: string) => {
  const value = status.toLowerCase();
  if (value.includes('offline') || value.includes('inactive')) return 'pill-gray';
  if (value.includes('trip') || value.includes('busy') || value.includes('pickup')) return 'pill-orange';
  return 'pill-green';
};

/** One driver, rendered as a row of an inset grouped list. */
export function DriverCard({ driver }: DriverCardProps) {
  return (
    <div className="list-row items-start py-3.5">
      <span
        aria-hidden="true"
        className="mt-0.5 flex h-10 w-10 shrink-0 items-center justify-center rounded-full bg-sys-blue text-footnote font-semibold text-white"
      >
        {initials(driver.name)}
      </span>
      <div className="min-w-0 flex-1">
        <div className="flex items-start justify-between gap-3">
          <div className="min-w-0">
            <p className="truncate text-callout font-semibold text-label">{driver.name}</p>
            <p className="truncate text-footnote text-label-2">{driver.carDetails}</p>
          </div>
          <span className={`pill ${statusPill(driver.status)}`}>
            <span className="pill-dot" aria-hidden="true" />
            {driver.status.replace('_', ' ')}
          </span>
        </div>

        <dl className="mt-2.5 grid grid-cols-3 gap-x-4 gap-y-2 text-footnote">
          <div className="min-w-0">
            <dt className="text-label-2">Destination</dt>
            <dd className="truncate font-medium text-label">{driver.route.destination}</dd>
          </div>
          <div>
            <dt className="text-label-2">ETA</dt>
            <dd className="tabular font-medium text-label">{driver.etaMinutes} min</dd>
          </div>
          <div>
            <dt className="text-label-2">Seats</dt>
            <dd className="tabular font-medium text-label">{driver.seatsAvailable}</dd>
          </div>
          <div className="col-span-3 min-w-0">
            <dt className="sr-only">Stations</dt>
            <dd className="truncate text-label-2">{driver.route.targetStationIds.join(' · ')}</dd>
          </div>
        </dl>
      </div>
    </div>
  );
}
