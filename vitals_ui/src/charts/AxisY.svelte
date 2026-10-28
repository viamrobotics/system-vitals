<script lang="ts">
  import { getLayerCakeContext } from 'layercake';
  import type { Formatter } from './types';

  let { format }: { format: Formatter } = $props();

  const k = getLayerCakeContext();
  const TICKS = 4;
  const LABEL_GAP = 8;
  const ticks = $derived<number[]>(k.yScale.ticks(TICKS));
</script>

<g class="axis-y vz-tabular" aria-hidden="true">
  {#each ticks as tick (tick)}
    <text
      x={-LABEL_GAP}
      y={k.yScale(tick)}
      dy="0.32em"
      text-anchor="end"
      font-size="10"
      fill="var(--vz-muted)">{format(tick)}</text
    >
  {/each}
</g>
