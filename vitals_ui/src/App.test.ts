import { render, screen } from '@testing-library/svelte';
import { beforeEach, describe, expect, it, vi } from 'vitest';

const state = vi.hoisted(() => ({
  bridge: { status: 'connected' as string, error: undefined as { message: string } | undefined },
  query: {
    data: undefined as Record<string, unknown> | undefined,
    error: undefined as { message: string } | undefined,
    isLoading: false,
    isFetching: false,
    isError: false,
    refetch: () => Promise.resolve(),
  },
}));

vi.mock('@viamrobotics/post-message-sdk/svelte', () => ({
  useBridge: () => ({ ...state.bridge, resourceName: 'vitals' }),
  createResourceClient: () => ({ current: {}, name: 'vitals' }),
  createResourceQuery: () => state.query,
}));

import App from './App.svelte';

const GB = 1024 * 1024 * 1024;

const sampleReadings = (): Record<string, unknown> => ({
  host: {
    hostname: 'rpi-5',
    os: 'linux',
    platform: 'debian',
    platform_version: '12',
    kernel_arch: 'aarch64',
    uptime_seconds: 90_000,
    process_count: 143,
  },
  cpu: { percent: 37.4, count_logical: 4, count_physical: 4 },
  load: { load1: 0.42, load5: 0.35, load15: 0.3 },
  memory: {
    total_bytes: 8 * GB,
    used_bytes: 2 * GB,
    used_percent: 25,
    swap_total_bytes: GB,
    swap_used_bytes: 0.1 * GB,
    swap_used_percent: 10,
  },
  disk: { '/': { total_bytes: 64 * GB, used_bytes: 32 * GB, used_percent: 50, fstype: 'ext4' } },
  network: { eth0: { bytes_recv: 1000, bytes_sent: 500 }, lo: { bytes_recv: 9, bytes_sent: 9 } },
  viam_server: {
    pid: 1234,
    cpu_percent: 3.2,
    rss_bytes: 0.2 * GB,
    memory_percent: 2.5,
    num_threads: 31,
    num_children: 2,
    uptime_seconds: 3600,
  },
  module: { pid: 1300, rss_bytes: 0.03 * GB },
  machine: { machine_id: 'abc-123', module_name: 'system-vitals' },
  errors: { temperature: 'not supported' },
});

describe('App', () => {
  beforeEach(() => {
    state.bridge.status = 'connected';
    state.bridge.error = undefined;
    state.query.data = undefined;
    state.query.error = undefined;
    state.query.isLoading = false;
    state.query.isFetching = false;
    state.query.isError = false;
  });

  it('keeps the dashboard mounted while a refetch or reconnect is in flight', () => {
    state.query.data = sampleReadings();
    state.query.isFetching = true;
    state.query.isLoading = true;
    state.bridge.status = 'connecting';
    render(App);
    expect(screen.queryByText('Loading vitals…')).toBeNull();
    expect(screen.queryByText('Connecting to the machine…')).toBeNull();
    expect(screen.getByRole('region', { name: 'Headline vitals' })).toBeTruthy();
    expect(screen.getByRole('status', { name: 'Updating' })).toBeTruthy();
  });

  it('keeps the last readings and shows a warning when a poll fails', () => {
    state.query.data = sampleReadings();
    state.query.isError = true;
    state.query.error = { message: 'timeout' };
    render(App);
    expect(screen.getByRole('region', { name: 'Headline vitals' })).toBeTruthy();
    expect(screen.getByRole('alert').textContent).toContain('timeout');
  });

  it('shows the loading placeholder only before the first readings arrive', () => {
    state.query.isLoading = true;
    render(App);
    expect(screen.getByText('Loading vitals…')).toBeTruthy();
  });

  it('renders headline tiles, capacity and identity from the readings', () => {
    state.query.data = sampleReadings();
    render(App);
    expect(screen.getByRole('heading', { level: 1 }).textContent).toContain('rpi-5');
    expect(screen.getByText('CPU', { selector: 'span.vz-ink-2' })).toBeTruthy();
    expect(screen.getByText('37%')).toBeTruthy();
    expect(screen.getByText('25%')).toBeTruthy();
    expect(screen.getByText('viam-server uptime')).toBeTruthy();
    expect(screen.getByText('1h 0m')).toBeTruthy();
    expect(screen.getByRole('meter', { name: 'Disk / (ext4)' })).toBeTruthy();
    expect(screen.getByText('abc-123', { selector: 'dd' })).toBeTruthy();
    expect(screen.getByRole('status', { name: 'Collection warnings' }).textContent).toContain(
      'not supported',
    );
    // Charts wait for a second sample before plotting.
    expect(screen.getAllByText('Collecting samples…').length).toBeGreaterThan(0);
  });

  it('lists every reading in the table view', () => {
    state.query.data = sampleReadings();
    render(App);
    expect(screen.getByText('viam_server.num_threads')).toBeTruthy();
    expect(screen.getByText('31')).toBeTruthy();
  });

  it('renders the empty state when the sensor returns nothing', () => {
    state.query.data = {};
    render(App);
    expect(screen.getByText('This sensor returned no readings.')).toBeTruthy();
  });

  it('renders the error state when the query fails', () => {
    state.query.isError = true;
    state.query.error = { message: 'boom' };
    render(App);
    expect(screen.getByRole('alert').textContent).toContain('boom');
  });
});
