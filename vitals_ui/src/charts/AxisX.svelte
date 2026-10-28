<script lang="ts">
  import { getLayerCakeContext } from 'layercake';
  import { fmtSecondsAgo } from '../lib/format';

  const k = getLayerCakeContext();
  const LABEL_OFFSET = 14;
  const MINUTE = 60;
  const HALF_MINUTE = 30;
  const LONG_WINDOW = 150;

  // Ticks on whole (or half) minutes back from "now", so labels read −1m, −2m rather than −1.7m.
  const ticks = $derived.by(() => {
    const [min] = k.xDomain as [number, number];
    const step = -min >= LONG_WINDOW ? MINUTE : HALF_MINUTE;
    const out: number[] = [];
    for (let t = 0; t >= min; t -= step) out.push(t);
    return out;
  });
  const anchor = (tick: number): string => (tick === 0 ? 'end' : 'middle');
</script>

<g class="axis-x vz-tabular" aria-hidden="true">
  <line
    x1="0"
    x2={k.width}
    y1={k.height}
    y2={k.height}
    stroke="var(--vz-axis)"
    stroke-width="1"
    shape-rendering="crispEdges"
  />
  {#each ticks as tick (tick)}
    <text
      x={k.xScale(tick)}
      y={k.height + LABEL_OFFSET}
      text-anchor={anchor(tick)}
      font-size="10"
      fill="var(--vz-muted)">{fmtSecondsAgo(tick)}</text
    >
  {/each}
</g>
