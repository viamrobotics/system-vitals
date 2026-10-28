<script lang="ts">
  import { bisector } from 'd3-array';
  import { getLayerCakeContext } from 'layercake';
  import { fmtSecondsAgo } from '../lib/format';
  import type { Point } from '../lib/history.svelte';
  import type { Formatter, Series } from './types';

  let { series, format }: { series: Series[]; format: Formatter } = $props();

  const k = getLayerCakeContext();
  const bisect = bisector<Point, number>((d) => d.x).center;
  const TOOLTIP_GAP = 10;
  const DOT = 8;
  const RING = 2;

  let hovered = $state<Point | undefined>(undefined);

  const onmove = (event: PointerEvent): void => {
    const data = k.data as Point[];
    if (data.length === 0) return;
    const [px] = k.pointer(event);
    const xValue = k.xScale.invert(px) as number;
    hovered = data[bisect(data, xValue)];
  };
  const onleave = (): void => {
    hovered = undefined;
  };

  const xPx = $derived(hovered ? (k.xGet(hovered) as number) : 0);
  const flip = $derived(xPx > k.width / 2);
</script>

<!-- The whole plot is the hit target, so the reader aims at a time, never at a 2px line. -->
<div
  class="absolute inset-0 cursor-crosshair"
  role="presentation"
  onpointermove={onmove}
  onpointerleave={onleave}
></div>

{#if hovered}
  <div
    class="pointer-events-none absolute top-0 bottom-0 w-px"
    style:left="{xPx}px"
    style:background="var(--vz-axis)"
    aria-hidden="true"
  ></div>
  {#each series as s (s.key)}
    {@const v = hovered[s.key]}
    {#if v !== undefined}
      <div
        class="pointer-events-none absolute rounded-full"
        style:left="{xPx - DOT / 2}px"
        style:top="{(k.yScale(v) as number) - DOT / 2}px"
        style:width="{DOT}px"
        style:height="{DOT}px"
        style:background={s.color}
        style:box-shadow="0 0 0 {RING}px var(--vz-surface)"
        aria-hidden="true"
      ></div>
    {/if}
  {/each}
  <div
    class="vz-card pointer-events-none absolute top-0 z-10 px-2 py-1 text-xs whitespace-nowrap shadow-sm"
    style:left={flip ? undefined : `${xPx + TOOLTIP_GAP}px`}
    style:right={flip ? `${k.width - xPx + TOOLTIP_GAP}px` : undefined}
    role="status"
  >
    <div class="vz-muted mb-0.5">{fmtSecondsAgo(hovered.x)}</div>
    {#each series as s (s.key)}
      {@const v = hovered[s.key]}
      <div class="flex items-center gap-2">
        <span
          class="inline-block h-0.5 w-3 rounded-sm"
          style:background={s.color}
          aria-hidden="true"
        ></span>
        <span class="vz-tabular font-semibold">{v === undefined ? '—' : format(v)}</span>
        <span class="vz-ink-2">{s.label}</span>
      </div>
    {/each}
  </div>
{/if}
