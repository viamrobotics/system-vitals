import { ResourceName, Struct, type RobotClient } from '@viamrobotics/sdk';

const SENSOR_SERVICE = 'viam.component.sensor.v1.SensorService';
const GB = 1024 * 1024 * 1024;
const MB = 1024 * 1024;
const PERCENT_MAX = 100;
const TWO_PI = Math.PI * 2;
const CYCLE = 40;

/** A smooth 0..1 wave over `CYCLE` calls, offset by `phase`, with a little noise. */
const wave = (calls: number, phase: number): number =>
  (Math.sin((TWO_PI * calls) / CYCLE + phase) + 1) / 2 + (Math.random() - 0.5) * 0.08;

/** A robot with one vitals sensor, `resourceName`, whose readings drift on every `getReadings`. */
export function createFakeRobot(resourceName: string): RobotClient {
  let calls = 0;
  let rx = 5 * GB;
  let tx = 1 * GB;
  const start = Date.now();

  const sensorService = {
    getReadings() {
      calls += 1;
      rx += (0.2 + wave(calls, 0)) * MB;
      tx += (0.05 + wave(calls, 2) * 0.4) * MB;
      const cpu = 15 + wave(calls, 0) * 50;
      const serverCpu = 2 + wave(calls, 1) * 12;
      const memUsed = 2.4 * GB + wave(calls, 3) * 0.6 * GB;
      const rss = 180 * MB + wave(calls, 1) * 60 * MB;
      const uptime = 86_400 * 3 + (Date.now() - start) / 1000;
      const readings = {
        host: {
          hostname: 'fake-rpi-5',
          os: 'linux',
          platform: 'debian',
          platform_family: 'debian',
          platform_version: '12',
          kernel_version: '6.6.31',
          kernel_arch: 'aarch64',
          uptime_seconds: uptime,
          boot_time_unix: 1_700_000_000,
          process_count: 140 + Math.round(wave(calls, 4) * 8),
        },
        cpu: { percent: cpu, count_logical: 4, count_physical: 4 },
        load: {
          load1: 0.3 + wave(calls, 0) * 1.5,
          load5: 0.4 + wave(calls, 0.6) * 0.9,
          load15: 0.5 + wave(calls, 1.2) * 0.4,
        },
        memory: {
          total_bytes: 8 * GB,
          available_bytes: 8 * GB - memUsed,
          used_bytes: memUsed,
          used_percent: (memUsed / (8 * GB)) * PERCENT_MAX,
          free_bytes: 1.1 * GB,
          swap_total_bytes: 1 * GB,
          swap_used_bytes: 0.12 * GB,
          swap_used_percent: 12,
        },
        disk: {
          '/': {
            total_bytes: 62 * GB,
            used_bytes: 41 * GB,
            free_bytes: 21 * GB,
            used_percent: 66.1,
            inodes_used_percent: 7.2,
            fstype: 'ext4',
          },
          '/data': {
            total_bytes: 240 * GB,
            used_bytes: 221 * GB,
            free_bytes: 19 * GB,
            used_percent: 92.1,
            inodes_used_percent: 3.1,
            fstype: 'ext4',
          },
        },
        network: {
          eth0: {
            bytes_recv: rx,
            bytes_sent: tx,
            packets_recv: rx / 1200,
            packets_sent: tx / 1200,
            err_in: 0,
            err_out: 0,
            drop_in: 3,
            drop_out: 0,
          },
          lo: { bytes_recv: 1e7, bytes_sent: 1e7 },
        },
        temperature: {
          cpu_thermal: 48 + wave(calls, 0) * 9,
          gpu_thermal: 44 + wave(calls, 1) * 6,
        },
        viam_server: {
          pid: 1234,
          name: 'viam-server',
          exe: '/opt/viam/bin/viam-server',
          status: 'sleep',
          cpu_percent: serverCpu,
          user_cpu_seconds: 812.4,
          system_cpu_seconds: 201.9,
          rss_bytes: rss,
          vms_bytes: 2.3 * GB,
          memory_percent: (rss / (8 * GB)) * PERCENT_MAX,
          num_threads: 31,
          num_fds: 212,
          num_children: 2,
          create_time_unix_ms: start - 3_600_000,
          uptime_seconds: 3600 + (Date.now() - start) / 1000,
        },
        module: {
          pid: 1300,
          name: 'system-vitals',
          cpu_percent: 0.4,
          rss_bytes: 28 * MB,
          num_threads: 9,
          goroutines: 14,
          go_version: 'go1.26.10',
          uptime_seconds: 3590 + (Date.now() - start) / 1000,
        },
        machine: {
          machine_id: 'a1b2c3d4-0000-4000-8000-000000000001',
          machine_part_id: 'a1b2c3d4-0000-4000-8000-000000000002',
          machine_fqdn: 'fake-rpi-5-main.abcde.viam.cloud',
          location_id: 'loc-xyz',
          primary_org_id: 'org-xyz',
          module_name: 'system-vitals',
          viam_home: '/root/.viam',
        },
      };
      return { readings: Struct.fromJson(readings).fields };
    },
  };

  // Single cast at the fake boundary: the host only calls these two methods, and a full
  // RobotClient needs a live connection.
  return {
    createServiceClient: (service: { typeName: string }) =>
      service.typeName === SENSOR_SERVICE ? sensorService : undefined,
    resourceNames: () =>
      Promise.resolve([
        new ResourceName({
          namespace: 'rdk',
          type: 'component',
          subtype: 'sensor',
          name: resourceName,
        }),
      ]),
  } as unknown as RobotClient;
}
