<script lang="ts">
  import { getLayerCakeContext } from 'layercake';

  /** A clip rectangle for the plot area, padded so end-dots on the right edge keep their ring. */
  let { id }: { id: string } = $props();

  const k = getLayerCakeContext();
  const BLEED = 6;
</script>

<clipPath {id}>
  <rect x="0" y={-BLEED} width={k.width + BLEED} height={k.height + BLEED * 2} />
</clipPath>
