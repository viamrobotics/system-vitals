import { describe, expect, it } from 'vitest';
import { History } from './history.svelte';
import { parseVitals } from './vitals';

const readings = (rx: number, cpu: number): Record<string, unknown> => ({
  cpu: { percent: cpu },
  network: { eth0: { bytes_recv: rx, bytes_sent: 0 }, lo: { bytes_recv: 999, bytes_sent: 999 } },
});

describe('History', () => {
  it('derives network rates from byte deltas and ignores loopback', () => {
    const h = new History(10);
    h.push(parseVitals(readings(1000, 10)), 1000);
    h.push(parseVitals(readings(3000, 20)), 3000);
    expect(h.samples[0]?.rxRate).toBeUndefined();
    expect(h.samples[1]?.rxRate).toBe(1000);
    expect(h.latest('cpu')).toBe(20);
  });

  it('keeps only the newest `max` samples and positions them in seconds before the latest', () => {
    const h = new History(3);
    for (let i = 0; i < 5; i += 1) h.push(parseVitals(readings(0, i)), i * 1000);
    expect(h.samples.map((s) => s.cpu)).toEqual([2, 3, 4]);
    expect(h.points.map((p) => p.x)).toEqual([-2, -1, 0]);
  });
});
