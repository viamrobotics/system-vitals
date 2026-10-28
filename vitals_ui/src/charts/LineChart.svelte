<script lang="ts">
  import { Html, LayerCake, Svg } from 'layercake';
  import type { Point } from '../lib/history.svelte';
  import Area from './Area.svelte';
  import AxisX from './AxisX.svelte';
  import AxisY from './AxisY.svelte';
  import Crosshair from './Crosshair.svelte';
  import GridY from './GridY.svelte';
  import Lines from './Lines.svelte';
  import PlotClip from './PlotClip.svelte';
  import type { Formatter, Series } from './types';

  interface Props {
    title: string;
    subtitle?: string;
    points: Point[];
    series: Series[];
    /** `[min, max]`; `null` inherits that bound from the data. Defaults to `[0, null]`. */
    yDomain?: [number | null, number | null];
    format: Formatter;
    axisFormat?: Formatter;
    windowSeconds: number;
    height?: number;
  }

  let {
    title,
    subtitle,
    points,
    series,
    yDomain = [0, null],
    format,
    axisFormat = format,
    windowSeconds,
    height = 176,
  }: Props = $props();

  const PADDING = { top: 10, right: 14, bottom: 22, left: 44 };
  const uid = $props.id();
  const clipId = `plot-${uid}`;
  const MIN_POINTS = 2;
  const keys = $derived(series.map((s) => s.key));
  const ready = $derived(points.length >= MIN_POINTS);
  const single = $derived(series.length === 1 ? series[0] : undefined);
</script>

<figure class="vz-card flex flex-col gap-2 p-3">
  <figcaption class="flex flex-wrap items-baseline justify-between gap-x-3 gap-y-1">
    <div class="flex items-baseline gap-2">
      <span class="text-sm font-semibold">{title}</span>
      {#if subtitle}<span class="vz-muted text-xs">{subtitle}</span>{/if}
    </div>
    {#if series.length > 1}
      <ul class="flex flex-wrap gap-x-3 gap-y-1 text-xs" aria-label="Legend">
        {#each series as s (s.key)}
          <li class="flex items-center gap-1.5">
            <span
              class="inline-block h-0.5 w-3 rounded-sm"
              style:background={s.color}
              aria-hidden="true"
            ></span>
            <span class="vz-ink-2">{s.label}</span>
          </li>
        {/each}
      </ul>
    {/if}
  </figcaption>

  <div class="relative w-full" style:height="{height}px">
    {#if ready}
      <LayerCake
        data={points}
        x="x"
        y={keys}
        xDomain={[-windowSeconds, 0]}
        {yDomain}
        yNice
        padding={PADDING}
      >
        <Svg label={title}>
          <PlotClip id={clipId} />
          <GridY />
          <AxisY format={axisFormat} />
          <AxisX />
          {#if single}
            <Area series={single} {clipId} />
          {/if}
          <Lines {series} {clipId} />
        </Svg>
        <Html>
          <Crosshair {series} {format} />
        </Html>
      </LayerCake>
    {:else}
      <p class="vz-muted absolute inset-0 flex items-center justify-center text-xs" role="status">
        Collecting samples…
      </p>
    {/if}
  </div>
</figure>
