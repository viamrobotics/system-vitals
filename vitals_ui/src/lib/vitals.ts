/** Typed view of the `viam:system-vitals:vitals-sensor` readings map. Every field is optional,
 * because any group can be missing on a given platform. */

export type Readings = Record<string, unknown>;
type Obj = Record<string, unknown>;

const obj = (v: unknown): Obj | undefined =>
  v !== null && typeof v === 'object' && !Array.isArray(v) ? (v as Obj) : undefined;
const num = (v: unknown): number | undefined =>
  typeof v === 'number' && Number.isFinite(v) ? v : undefined;
const str = (v: unknown): string | undefined => (typeof v === 'string' && v !== '' ? v : undefined);

export interface DiskUsage {
  path: string;
  totalBytes?: number;
  usedBytes?: number;
  freeBytes?: number;
  usedPercent?: number;
  fstype?: string;
}

export interface InterfaceCounters {
  name: string;
  bytesRecv?: number;
  bytesSent?: number;
  errIn?: number;
  errOut?: number;
  dropIn?: number;
  dropOut?: number;
}

export interface ProcessVitals {
  pid?: number;
  name?: string;
  status?: string;
  cpuPercent?: number;
  rssBytes?: number;
  memoryPercent?: number;
  numThreads?: number;
  numFds?: number;
  numChildren?: number;
  uptimeSeconds?: number;
}

export interface Vitals {
  host: {
    hostname?: string;
    os?: string;
    platform?: string;
    platformVersion?: string;
    kernelArch?: string;
    uptimeSeconds?: number;
    processCount?: number;
  };
  cpu: { percent?: number; countLogical?: number; countPhysical?: number };
  load: { load1?: number; load5?: number; load15?: number };
  memory: {
    totalBytes?: number;
    usedBytes?: number;
    availableBytes?: number;
    usedPercent?: number;
    swapTotalBytes?: number;
    swapUsedBytes?: number;
    swapUsedPercent?: number;
  };
  disks: DiskUsage[];
  interfaces: InterfaceCounters[];
  temperatures: { key: string; celsius: number }[];
  viamServer?: ProcessVitals;
  module?: ProcessVitals;
  machine: { label: string; value: string }[];
  errors: { group: string; message: string }[];
}

const LOOPBACK = /^lo\d*$/;

const parseProcess = (p: Obj | undefined): ProcessVitals | undefined =>
  p && {
    pid: num(p.pid),
    name: str(p.name),
    status: str(p.status),
    cpuPercent: num(p.cpu_percent),
    rssBytes: num(p.rss_bytes),
    memoryPercent: num(p.memory_percent),
    numThreads: num(p.num_threads),
    numFds: num(p.num_fds),
    numChildren: num(p.num_children),
    uptimeSeconds: num(p.uptime_seconds),
  };

const MACHINE_LABELS: Record<string, string> = {
  machine_fqdn: 'FQDN',
  machine_id: 'Machine ID',
  machine_part_id: 'Part ID',
  location_id: 'Location ID',
  primary_org_id: 'Org ID',
  module_name: 'Module',
  module_id: 'Module ID',
  viam_home: 'Viam home',
  module_data_dir: 'Module data',
};

export function parseVitals(r: Readings): Vitals {
  const host = obj(r.host) ?? {};
  const cpu = obj(r.cpu) ?? {};
  const load = obj(r.load) ?? {};
  const memory = obj(r.memory) ?? {};

  const disks = Object.entries(obj(r.disk) ?? {})
    .map(([path, v]): DiskUsage => {
      const d = obj(v) ?? {};
      return {
        path,
        totalBytes: num(d.total_bytes),
        usedBytes: num(d.used_bytes),
        freeBytes: num(d.free_bytes),
        usedPercent: num(d.used_percent),
        fstype: str(d.fstype),
      };
    })
    .sort((a, b) => a.path.localeCompare(b.path));

  const interfaces = Object.entries(obj(r.network) ?? {})
    .map(([name, v]): InterfaceCounters => {
      const n = obj(v) ?? {};
      return {
        name,
        bytesRecv: num(n.bytes_recv),
        bytesSent: num(n.bytes_sent),
        errIn: num(n.err_in),
        errOut: num(n.err_out),
        dropIn: num(n.drop_in),
        dropOut: num(n.drop_out),
      };
    })
    .sort((a, b) => a.name.localeCompare(b.name));

  const temperatures = Object.entries(obj(r.temperature) ?? {})
    .flatMap(([key, v]) => {
      const celsius = num(v);
      return celsius === undefined ? [] : [{ key, celsius }];
    })
    .sort((a, b) => b.celsius - a.celsius);

  const machine = Object.entries(obj(r.machine) ?? {}).flatMap(([key, v]) => {
    const value = str(v);
    return value === undefined ? [] : [{ label: MACHINE_LABELS[key] ?? key, value }];
  });

  const errors = Object.entries(obj(r.errors) ?? {}).map(([group, v]) => ({
    group,
    message: typeof v === 'string' ? v : JSON.stringify(v),
  }));

  return {
    host: {
      hostname: str(host.hostname),
      os: str(host.os),
      platform: str(host.platform),
      platformVersion: str(host.platform_version),
      kernelArch: str(host.kernel_arch),
      uptimeSeconds: num(host.uptime_seconds),
      processCount: num(host.process_count),
    },
    cpu: {
      percent: num(cpu.percent),
      countLogical: num(cpu.count_logical),
      countPhysical: num(cpu.count_physical),
    },
    load: { load1: num(load.load1), load5: num(load.load5), load15: num(load.load15) },
    memory: {
      totalBytes: num(memory.total_bytes),
      usedBytes: num(memory.used_bytes),
      availableBytes: num(memory.available_bytes),
      usedPercent: num(memory.used_percent),
      swapTotalBytes: num(memory.swap_total_bytes),
      swapUsedBytes: num(memory.swap_used_bytes),
      swapUsedPercent: num(memory.swap_used_percent),
    },
    disks,
    interfaces,
    temperatures,
    viamServer: parseProcess(obj(r.viam_server)),
    module: parseProcess(obj(r.module)),
    machine,
    errors,
  };
}

/** Sum of bytes over every non-loopback interface, for throughput deltas. */
export const totalTraffic = (ifaces: InterfaceCounters[]): { rx: number; tx: number } =>
  ifaces
    .filter((i) => !LOOPBACK.test(i.name))
    .reduce((acc, i) => ({ rx: acc.rx + (i.bytesRecv ?? 0), tx: acc.tx + (i.bytesSent ?? 0) }), {
      rx: 0,
      tx: 0,
    });

/** Flattens the readings into `path → value` rows for the table view. */
export function flattenReadings(r: Readings, prefix = ''): { path: string; value: string }[] {
  return Object.entries(r)
    .sort(([a], [b]) => a.localeCompare(b))
    .flatMap(([key, v]) => {
      const path = prefix ? `${prefix}.${key}` : key;
      const nested = obj(v);
      if (nested) return flattenReadings(nested, path);
      return [{ path, value: typeof v === 'string' ? v : JSON.stringify(v) }];
    });
}
