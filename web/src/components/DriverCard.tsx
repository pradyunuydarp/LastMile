import type { Driver } from '../lib/types';

interface DriverCardProps {
  driver: Driver;
}

export function DriverCard({ driver }: DriverCardProps) {
  return (
    <div className="bg-slate-900/40 border border-slate-800 rounded-2xl p-5 shadow-lg shadow-black/20">
      <div className="flex items-center justify-between">
        <div>
          <p className="text-base font-semibold text-white">{driver.name}</p>
          <p className="text-sm text-slate-400">{driver.carDetails}</p>
        </div>
        <span className="px-3 py-1 rounded-full text-xs font-semibold bg-emerald-500/10 text-emerald-300">
          {driver.status.replace('_', ' ')}
        </span>
      </div>

      <div className="grid grid-cols-2 gap-4 mt-4 text-sm text-slate-300">
        <div>
          <p className="text-slate-500 text-xs uppercase tracking-wide">Destination</p>
          <p className="font-medium text-white">{driver.route.destination}</p>
        </div>
        <div>
          <p className="text-slate-500 text-xs uppercase tracking-wide">ETA</p>
          <p className="font-medium text-white">{driver.etaMinutes} min</p>
        </div>
        <div>
          <p className="text-slate-500 text-xs uppercase tracking-wide">Seats</p>
          <p className="font-medium text-white">{driver.seatsAvailable}</p>
        </div>
        <div>
          <p className="text-slate-500 text-xs uppercase tracking-wide">Stations</p>
          <p className="font-medium text-white truncate">{driver.route.targetStationIds.join(', ')}</p>
        </div>
      </div>
    </div>
  );
}
