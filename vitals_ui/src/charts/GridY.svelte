<script lang="ts">
  import { getLayerCakeContext } from 'layercake';

  const k = getLayerCakeContext();
  const TICKS = 4;
  const ticks = $derived<number[]>(k.yScale.ticks(TICKS));
</script>

<g class="grid-y" aria-hidden="true">
  {#each ticks as tick (tick)}
    <line
      x1="0"
      x2={k.width}
      y1={k.yScale(tick)}
      y2={k.yScale(tick)}
      stroke="var(--vz-grid)"
      stroke-width="1"
      shape-rendering="crispEdges"
    />
  {/each}
</g>
