import { totalTraffic, type Vitals } from './vitals';

/** One polled sample. Missing metrics stay `undefined` so lines break instead of dropping to 0. */
export interface Sample {
  t: number;
  cpu?: number;
  memory?: number;
  load1?: number;
  load5?: number;
  load15?: number;
  serverCpu?: number;
  serverRssMb?: number;
  rxRate?: number;
  txRate?: number;
}

/** A sample positioned on the chart's time axis, in seconds before the newest sample. */
export type Point = Sample & { x: number };

export type SeriesKey = Exclude<keyof Sample, 't'>;

const BYTES_PER_MB = 1024 * 1024;
const MS_PER_SECOND = 1000;

/** Rolling window of samples; the latest is at the end. Samples older than `windowSeconds`
 * are dropped as well as any beyond `max`, so nothing plots left of the chart's x-domain even
 * when polls arrive more slowly than planned. */
export class History {
  samples = $state<Sample[]>([]);
  readonly max: number;
  readonly windowSeconds: number;
  #lastTraffic: { t: number; rx: number; tx: number } | undefined;

  constructor(max: number, windowSeconds: number) {
    this.max = max;
    this.windowSeconds = windowSeconds;
  }

  push(v: Vitals, t = Date.now()): void {
    const traffic = totalTraffic(v.interfaces);
    let rxRate: number | undefined;
    let txRate: number | undefined;
    if (v.interfaces.length > 0) {
      const prev = this.#lastTraffic;
      const dt = prev ? (t - prev.t) / MS_PER_SECOND : 0;
      if (prev && dt > 0) {
        rxRate = Math.max(0, (traffic.rx - prev.rx) / dt);
        txRate = Math.max(0, (traffic.tx - prev.tx) / dt);
      }
      this.#lastTraffic = { t, ...traffic };
    }

    const sample: Sample = {
      t,
      cpu: v.cpu.percent,
      memory: v.memory.usedPercent,
      load1: v.load.load1,
      load5: v.load.load5,
      load15: v.load.load15,
      serverCpu: v.viamServer?.cpuPercent,
      serverRssMb:
        v.viamServer?.rssBytes === undefined ? undefined : v.viamServer.rssBytes / BYTES_PER_MB,
      rxRate,
      txRate,
    };
    const oldest = t - this.windowSeconds * MS_PER_SECOND;
    const next = [...this.samples.filter((s) => s.t >= oldest), sample];
    this.samples = next.length > this.max ? next.slice(next.length - this.max) : next;
  }

  /** Samples re-keyed to seconds before the newest one, for a fixed `[−window, 0]` x-domain. */
  get points(): Point[] {
    const last = this.samples.at(-1);
    if (!last) return [];
    return this.samples.map((s) => ({ ...s, x: (s.t - last.t) / MS_PER_SECOND }));
  }

  /** The newest value of one series, if any. */
  latest(key: SeriesKey): number | undefined {
    return this.samples.at(-1)?.[key];
  }
}
