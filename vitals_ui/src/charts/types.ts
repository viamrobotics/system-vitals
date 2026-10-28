import type { SeriesKey } from '../lib/history.svelte';

/** One plotted series: which sample field it reads, its legend label and its CSS color token. */
export interface Series {
  key: SeriesKey;
  label: string;
  /** A `var(--vz-sN)` token from app.css; identity follows the entity across every chart. */
  color: string;
}

export type Formatter = (v: number) => string;
