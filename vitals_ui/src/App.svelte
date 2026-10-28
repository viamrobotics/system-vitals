<script lang="ts">
  import type { SensorClient } from '@viamrobotics/sdk';
  import {
    createResourceClient,
    createResourceQuery,
    useBridge,
  } from '@viamrobotics/post-message-sdk/svelte';
  import { Badge, Button } from '@viamrobotics/prime-core';
  import LineChart from './charts/LineChart.svelte';
  import Sparkline from './charts/Sparkline.svelte';
  import type { Series } from './charts/types';
  import Meter from './components/Meter.svelte';
  import StatTile from './components/StatTile.svelte';
  import { fmtBytes, fmtCompact, fmtDuration, fmtNumber, fmtPercent, fmtRate } from './lib/format';
  import { History } from './lib/history.svelte';
  import { flattenReadings, parseVitals, type Readings } from './lib/vitals';

  const POLL_MS = 2000;
  const MAX_SAMPLES = 91; // 3 minutes at POLL_MS
  const WINDOW_SECONDS = (MAX_SAMPLES - 1) * (POLL_MS / 1000);
  const PERCENT_MAX = 100;
  const SWAP_WARN = 50;
  const SWAP_CRITICAL = 80;

  const HOST = 'var(--vz-s1)';
  const SERVER = 'var(--vz-s2)';
  const THIRD = 'var(--vz-s3)';

  const bridge = useBridge();
  const sensor = createResourceClient<SensorClient>();
  const readings = createResourceQuery(sensor, 'getReadings', { refetchInterval: POLL_MS });

  const history = new History(MAX_SAMPLES);
  let lastData: Readings | undefined;
  let updatedAt = $state<number | undefined>(undefined);

  const vitals = $derived(readings.data ? parseVitals(readings.data) : undefined);
  const hasData = $derived(readings.data !== undefined && Object.keys(readings.data).length > 0);

  $effect(() => {
    const data = readings.data;
    if (!data || data === lastData || !vitals) return;
    lastData = data;
    history.push(vitals);
    updatedAt = Date.now();
  });

  const points = $derived(history.points);

  const cpuSeries: Series[] = [
    { key: 'cpu', label: 'Host', color: HOST },
    { key: 'serverCpu', label: 'viam-server', color: SERVER },
  ];
  const memorySeries: Series[] = [{ key: 'memory', label: 'Host memory used', color: HOST }];
  const rssSeries: Series[] = [{ key: 'serverRssMb', label: 'viam-server RSS', color: SERVER }];
  const loadSeries: Series[] = [
    { key: 'load1', label: '1 min', color: HOST },
    { key: 'load5', label: '5 min', color: SERVER },
    { key: 'load15', label: '15 min', color: THIRD },
  ];
  const netSeries: Series[] = [
    { key: 'rxRate', label: 'Received', color: HOST },
    { key: 'txRate', label: 'Sent', color: SERVER },
  ];

  const hasNetwork = $derived((vitals?.interfaces.length ?? 0) > 0);
  const hasLoad = $derived(vitals?.load.load1 !== undefined);
  const rows = $derived(readings.data ? flattenReadings(readings.data) : []);

  const hostLine = $derived.by(() => {
    if (!vitals) return '';
    const h = vitals.host;
    return [h.platform ?? h.os, h.platformVersion, h.kernelArch].filter(Boolean).join(' · ');
  });
  const percent1 = (v: number): string => fmtPercent(v, 1);
  const mb = (v: number): string => `${fmtCompact(v)} MB`;
  const rate = (v: number): string => fmtRate(v);
</script>

<main class="flex flex-col gap-3 p-3 text-sm">
  <header class="flex flex-wrap items-center justify-between gap-2">
    <div class="flex min-w-0 flex-col">
      <h1 class="truncate text-base font-semibold">
        {vitals?.host.hostname ?? sensor.name ?? 'System vitals'}
      </h1>
      {#if hostLine}
        <span class="vz-muted truncate text-xs">{hostLine}</span>
      {/if}
    </div>
    <div class="flex items-center gap-2">
      {#if vitals?.host.uptimeSeconds !== undefined}
        <span class="vz-ink-2 text-xs">up {fmtDuration(vitals.host.uptimeSeconds)}</span>
      {/if}
      {#if bridge.status === 'connected'}
        <Badge variant="success" label="Connected" />
      {:else if bridge.status === 'connecting'}
        <Badge variant="pending" label="Connecting" />
      {:else}
        <Badge variant="danger" label="Error" />
      {/if}
    </div>
  </header>

  {#if bridge.status === 'connecting'}
    <p role="status" class="vz-ink-2">Connecting to the machine…</p>
  {:else if bridge.status === 'error'}
    <p role="alert" class="text-danger-dark">
      Could not connect: {bridge.error?.message ?? 'unknown error'}
    </p>
  {:else if readings.isError}
    <p role="alert" class="text-danger-dark">
      Could not read the sensor: {readings.error?.message ?? 'unknown error'}
    </p>
  {:else if readings.isLoading}
    <p role="status" class="vz-ink-2">Loading vitals…</p>
  {:else if !hasData || !vitals}
    <p class="vz-ink-2">This sensor returned no readings.</p>
  {:else}
    <!-- Headline numbers -->
    <section
      class="grid grid-cols-2 gap-2 sm:grid-cols-3 lg:grid-cols-4"
      style:opacity={readings.isFetching ? 0.85 : 1}
      aria-label="Headline vitals"
    >
      <StatTile
        label="CPU"
        value={fmtPercent(vitals.cpu.percent)}
        sub={vitals.cpu.countLogical === undefined
          ? undefined
          : `${String(vitals.cpu.countLogical)} logical cores`}
      >
        {#snippet trend()}
          <Sparkline
            {points}
            key="cpu"
            color={HOST}
            windowSeconds={WINDOW_SECONDS}
            max={PERCENT_MAX}
          />
        {/snippet}
      </StatTile>
      <StatTile
        label="Memory used"
        value={fmtPercent(vitals.memory.usedPercent)}
        sub="{fmtBytes(vitals.memory.usedBytes)} of {fmtBytes(vitals.memory.totalBytes)}"
      >
        {#snippet trend()}
          <Sparkline
            {points}
            key="memory"
            color={HOST}
            windowSeconds={WINDOW_SECONDS}
            max={PERCENT_MAX}
          />
        {/snippet}
      </StatTile>
      {#if hasLoad}
        <StatTile
          label="Load (1 min)"
          value={fmtNumber(vitals.load.load1)}
          sub="5 min {fmtNumber(vitals.load.load5)} · 15 min {fmtNumber(vitals.load.load15)}"
        >
          {#snippet trend()}
            <Sparkline {points} key="load1" color={HOST} windowSeconds={WINDOW_SECONDS} />
          {/snippet}
        </StatTile>
      {/if}
      {#if vitals.viamServer}
        <StatTile
          label="viam-server CPU"
          value={fmtPercent(vitals.viamServer.cpuPercent, 1)}
          sub={vitals.viamServer.numThreads === undefined
            ? undefined
            : `${String(vitals.viamServer.numThreads)} threads`}
        >
          {#snippet trend()}
            <Sparkline {points} key="serverCpu" color={SERVER} windowSeconds={WINDOW_SECONDS} />
          {/snippet}
        </StatTile>
        <StatTile
          label="viam-server memory"
          value={fmtBytes(vitals.viamServer.rssBytes)}
          sub="{fmtPercent(vitals.viamServer.memoryPercent, 1)} of host memory"
        >
          {#snippet trend()}
            <Sparkline {points} key="serverRssMb" color={SERVER} windowSeconds={WINDOW_SECONDS} />
          {/snippet}
        </StatTile>
        <StatTile
          label="viam-server uptime"
          value={fmtDuration(vitals.viamServer.uptimeSeconds)}
          sub={vitals.viamServer.pid === undefined
            ? undefined
            : `pid ${String(vitals.viamServer.pid)}${vitals.viamServer.numChildren === undefined ? '' : ` · ${String(vitals.viamServer.numChildren)} module processes`}`}
        />
      {/if}
      {#if vitals.host.processCount !== undefined}
        <StatTile
          label="Processes"
          value={String(vitals.host.processCount)}
          sub={vitals.module?.rssBytes === undefined
            ? undefined
            : `this module ${fmtBytes(vitals.module.rssBytes)}`}
        />
      {/if}
    </section>

    <!-- Trends -->
    <section class="grid grid-cols-1 gap-2 lg:grid-cols-2" aria-label="Trends">
      <LineChart
        title="CPU"
        subtitle="percent"
        {points}
        series={cpuSeries}
        yDomain={[0, PERCENT_MAX]}
        format={percent1}
        axisFormat={fmtCompact}
        windowSeconds={WINDOW_SECONDS}
      />
      <LineChart
        title="Memory used"
        subtitle="percent of {fmtBytes(vitals.memory.totalBytes)}"
        {points}
        series={memorySeries}
        yDomain={[0, PERCENT_MAX]}
        format={percent1}
        axisFormat={fmtCompact}
        windowSeconds={WINDOW_SECONDS}
      />
      {#if vitals.viamServer}
        <LineChart
          title="viam-server memory"
          subtitle="resident set, MB"
          {points}
          series={rssSeries}
          format={mb}
          axisFormat={fmtCompact}
          windowSeconds={WINDOW_SECONDS}
        />
      {/if}
      {#if hasLoad}
        <LineChart
          title="Load average"
          {points}
          series={loadSeries}
          format={(v) => fmtNumber(v)}
          axisFormat={fmtCompact}
          windowSeconds={WINDOW_SECONDS}
        />
      {/if}
      {#if hasNetwork}
        <LineChart
          title="Network throughput"
          subtitle="all interfaces except loopback"
          {points}
          series={netSeries}
          format={rate}
          axisFormat={(v) => fmtBytes(v, 0)}
          windowSeconds={WINDOW_SECONDS}
        />
      {/if}
    </section>

    <!-- Capacity -->
    {#if vitals.disks.length > 0 || vitals.memory.swapTotalBytes}
      <section class="vz-card flex flex-col gap-3 p-3" aria-label="Capacity">
        <h2 class="text-sm font-semibold">Capacity</h2>
        {#each vitals.disks as d (d.path)}
          <Meter
            label="Disk {d.path}{d.fstype ? ` (${d.fstype})` : ''}"
            detail="{fmtBytes(d.usedBytes)} of {fmtBytes(d.totalBytes)}"
            percent={d.usedPercent}
          />
        {/each}
        {#if vitals.memory.swapTotalBytes}
          <Meter
            label="Swap"
            detail="{fmtBytes(vitals.memory.swapUsedBytes)} of {fmtBytes(
              vitals.memory.swapTotalBytes,
            )}"
            percent={vitals.memory.swapUsedPercent}
            warnAt={SWAP_WARN}
            criticalAt={SWAP_CRITICAL}
          />
        {/if}
      </section>
    {/if}

    <!-- Temperatures -->
    {#if vitals.temperatures.length > 0}
      <section class="vz-card flex flex-col gap-2 p-3" aria-label="Temperatures">
        <h2 class="text-sm font-semibold">Temperatures</h2>
        <ul class="grid grid-cols-2 gap-x-4 gap-y-1 text-xs sm:grid-cols-3 lg:grid-cols-4">
          {#each vitals.temperatures as t (t.key)}
            <li class="flex justify-between gap-2">
              <span class="vz-ink-2 truncate">{t.key}</span>
              <span class="vz-tabular shrink-0 font-medium">{t.celsius.toFixed(1)} °C</span>
            </li>
          {/each}
        </ul>
      </section>
    {/if}

    <!-- Identity and problems -->
    {#if vitals.machine.length > 0}
      <section class="vz-card p-3" aria-label="Machine">
        <h2 class="mb-2 text-sm font-semibold">Machine</h2>
        <dl class="grid grid-cols-[auto_1fr] gap-x-4 gap-y-1 text-xs">
          {#each vitals.machine as m (m.label)}
            <dt class="vz-ink-2">{m.label}</dt>
            <dd class="truncate font-mono">{m.value}</dd>
          {/each}
        </dl>
      </section>
    {/if}
    {#if vitals.errors.length > 0}
      <section class="vz-card p-3" aria-label="Collection warnings" role="status">
        <h2 class="mb-1 text-sm font-semibold text-warning-dark">Not collected</h2>
        <ul class="text-xs">
          {#each vitals.errors as e (e.group)}
            <li><span class="font-mono">{e.group}</span>: {e.message}</li>
          {/each}
        </ul>
      </section>
    {/if}

    <!-- Table view: every value, without hovering -->
    <details class="vz-card p-3">
      <summary class="cursor-pointer text-sm font-semibold">All readings</summary>
      <table class="mt-2 w-full text-xs">
        <thead>
          <tr class="vz-ink-2 text-left"><th class="pr-3 font-medium">Key</th><th>Value</th></tr>
        </thead>
        <tbody>
          {#each rows as r (r.path)}
            <tr>
              <td class="pr-3 align-top font-mono break-all">{r.path}</td>
              <td class="vz-tabular align-top break-all">{r.value}</td>
            </tr>
          {/each}
        </tbody>
      </table>
    </details>
  {/if}

  <footer class="flex items-center justify-between gap-2">
    <span class="vz-muted text-xs">
      {#if updatedAt !== undefined}
        Polling every {String(POLL_MS / 1000)}s · {String(history.samples.length)} samples
      {/if}
    </span>
    <Button variant="ghost-outline" onclick={() => void readings.refetch()}>Refetch</Button>
  </footer>
</main>
