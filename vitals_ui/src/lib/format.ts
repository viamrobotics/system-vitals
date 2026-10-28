const BYTE_UNITS = ['B', 'KB', 'MB', 'GB', 'TB', 'PB'];
const KIB = 1024;
const SECONDS_PER_MINUTE = 60;
const SECONDS_PER_HOUR = 3600;
const SECONDS_PER_DAY = 86_400;
const THOUSAND = 1000;
const MILLION = 1_000_000;

export const NA = '—';

const finite = (v: number | undefined): v is number => v !== undefined && Number.isFinite(v);

/** `12%`, or `12.3%` with `digits`. */
export const fmtPercent = (v: number | undefined, digits = 0): string =>
  finite(v) ? `${v.toFixed(digits)}%` : NA;

/** `1.2 GB`, binary units. */
export const fmtBytes = (v: number | undefined, digits = 1): string => {
  if (!finite(v)) return NA;
  let value = v;
  let unit = 0;
  while (value >= KIB && unit < BYTE_UNITS.length - 1) {
    value /= KIB;
    unit += 1;
  }
  return `${value.toFixed(unit === 0 ? 0 : digits)} ${BYTE_UNITS[unit]}`;
};

/** `1.2 MB/s`. */
export const fmtRate = (bytesPerSecond: number | undefined): string =>
  finite(bytesPerSecond) ? `${fmtBytes(bytesPerSecond)}/s` : NA;

/** `3d 4h`, `2h 10m`, `5m 3s`, `42s`. */
export const fmtDuration = (seconds: number | undefined): string => {
  if (!finite(seconds)) return NA;
  const s = Math.max(0, Math.floor(seconds));
  const days = Math.floor(s / SECONDS_PER_DAY);
  const hours = Math.floor((s % SECONDS_PER_DAY) / SECONDS_PER_HOUR);
  const minutes = Math.floor((s % SECONDS_PER_HOUR) / SECONDS_PER_MINUTE);
  const secs = s % SECONDS_PER_MINUTE;
  if (days > 0) return `${String(days)}d ${String(hours)}h`;
  if (hours > 0) return `${String(hours)}h ${String(minutes)}m`;
  if (minutes > 0) return `${String(minutes)}m ${String(secs)}s`;
  return `${String(secs)}s`;
};

/** Plain number with fixed digits. */
export const fmtNumber = (v: number | undefined, digits = 2): string =>
  finite(v) ? v.toFixed(digits) : NA;

/** Axis-style compact number: 0 / 1.5K / 2M. */
export const fmtCompact = (v: number): string => {
  const abs = Math.abs(v);
  if (abs >= MILLION) return `${(v / MILLION).toFixed(abs >= 10 * MILLION ? 0 : 1)}M`;
  if (abs >= THOUSAND) return `${(v / THOUSAND).toFixed(abs >= 10 * THOUSAND ? 0 : 1)}K`;
  return Number.isInteger(v) ? String(v) : v.toFixed(1);
};

/** Relative time-axis label: `now`, `−30s`, `−2m`. */
export const fmtSecondsAgo = (secondsAgo: number): string => {
  const s = Math.round(Math.abs(secondsAgo));
  if (s === 0) return 'now';
  if (s < SECONDS_PER_MINUTE) return `−${String(s)}s`;
  const m = s / SECONDS_PER_MINUTE;
  return `−${Number.isInteger(m) ? String(m) : m.toFixed(1)}m`;
};
