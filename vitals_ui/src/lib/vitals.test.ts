import { describe, expect, it } from 'vitest';
import { parseVitals } from './vitals';

describe('parseVitals machine group', () => {
  it('orders rows by a fixed list regardless of the order the readings arrive in', () => {
    const shuffled = {
      machine: {
        viam_home: '/root/.viam',
        module_name: 'system-vitals',
        zzz_custom: 'x',
        machine_part_id: 'part',
        machine_id: 'machine',
        aaa_custom: 'y',
      },
    };
    expect(parseVitals(shuffled).machine.map((m) => m.label)).toEqual([
      'Machine ID',
      'Part ID',
      'Module',
      'Viam home',
      'aaa_custom',
      'zzz_custom',
    ]);
  });

  it('omits empty values', () => {
    expect(parseVitals({ machine: { machine_id: '', location_id: 'loc' } }).machine).toEqual([
      { label: 'Location ID', value: 'loc' },
    ]);
  });
});
