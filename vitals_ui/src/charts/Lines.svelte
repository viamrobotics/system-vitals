<script lang="ts">
  import { curveMonotoneX, line } from 'd3-shape';
  import { getLayerCakeContext } from 'layercake';
  import type { Point } from '../lib/history.svelte';
  import type { Series } from './types';

  let { series, clipId }: { series: Series[]; clipId?: string } = $props();

  const k = getLayerCakeContext();
  const STROKE = 2;
  const DOT_RADIUS = 4;
  const RING = 2;

  const pathFor = (s: Series): string => {
    const gen = line<Point>()
      .defined((d) => d[s.key] !== undefined)
      .x((d) => k.xGet(d) as number)
      .y((d) => k.yScale(d[s.key]) as number)
      .curve(curveMonotoneX);
    return gen(k.data as Point[]) ?? '';
  };

  const endOf = (s: Series): Point | undefined => {
    const data = k.data as Point[];
    for (let i = data.length - 1; i >= 0; i -= 1) {
      const d = data[i];
      if (d !== undefined && d[s.key] !== undefined) return d;
    }
    return undefined;
  };
</script>

<g class="lines" clip-path={clipId ? `url(#${clipId})` : undefined}>
  {#each series as s (s.key)}
    <path
      d={pathFor(s)}
      fill="none"
      stroke={s.color}
      stroke-width={STROKE}
      stroke-linejoin="round"
      stroke-linecap="round"
    />
  {/each}
  {#each series as s (s.key)}
    {@const end = endOf(s)}
    {#if end}
      <circle
        cx={k.xGet(end)}
        cy={k.yScale(end[s.key])}
        r={DOT_RADIUS}
        fill={s.color}
        stroke="var(--vz-surface)"
        stroke-width={RING}
      />
    {/if}
  {/each}
</g>
