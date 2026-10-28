<script lang="ts">
  import { area, curveMonotoneX } from 'd3-shape';
  import { getLayerCakeContext } from 'layercake';
  import type { Point } from '../lib/history.svelte';
  import type { Series } from './types';

  let { series }: { series: Series } = $props();

  const k = getLayerCakeContext();
  const WASH_OPACITY = 0.1;

  const path = $derived.by(() => {
    const gen = area<Point>()
      .defined((d) => d[series.key] !== undefined)
      .x((d) => k.xGet(d) as number)
      .y0(() => k.yScale(0) as number)
      .y1((d) => k.yScale(d[series.key]) as number)
      .curve(curveMonotoneX);
    return gen(k.data as Point[]) ?? '';
  });
</script>

<path d={path} fill={series.color} fill-opacity={WASH_OPACITY} />
