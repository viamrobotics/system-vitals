<script lang="ts">
  import { curveMonotoneX, line } from 'd3-shape';
  import { LayerCake, Svg } from 'layercake';
  import type { Point, SeriesKey } from '../lib/history.svelte';

  interface Props {
    points: Point[];
    key: SeriesKey;
    color: string;
    windowSeconds: number;
    max?: number | null;
  }

  let { points, key, color, windowSeconds, max = null }: Props = $props();

  const MIN_POINTS = 2;
  const HEIGHT = 28;
  const PADDING = { top: 3, right: 3, bottom: 3, left: 0 };
  const ready = $derived(points.length >= MIN_POINTS);
</script>

<div class="w-full" style:height="{HEIGHT}px" aria-hidden="true">
  {#if ready}
    <LayerCake
      data={points}
      x="x"
      y={key}
      xDomain={[-windowSeconds, 0]}
      yDomain={[0, max]}
      padding={PADDING}
    >
      {#snippet children(k)}
        <Svg>
          {@const gen = line<Point>()
            .defined((d) => d[key] !== undefined)
            .x((d) => k.xGet(d) as number)
            .y((d) => k.yGet(d) as number)
            .curve(curveMonotoneX)}
          <path
            d={gen(points) ?? ''}
            fill="none"
            stroke={color}
            stroke-width="1.5"
            stroke-linejoin="round"
            stroke-linecap="round"
          />
        </Svg>
      {/snippet}
    </LayerCake>
  {/if}
</div>
