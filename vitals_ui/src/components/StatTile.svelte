<script lang="ts">
  import type { Snippet } from 'svelte';

  interface Props {
    label: string;
    value: string;
    sub?: string;
    trend?: Snippet;
  }

  let { label, value, sub, trend }: Props = $props();
</script>

<div class="vz-card flex min-w-0 flex-col gap-1 p-3">
  <span class="vz-ink-2 text-xs">{label}</span>
  <span class="truncate text-xl leading-tight font-semibold">{value}</span>
  {#if sub}<span class="vz-muted truncate text-xs">{sub}</span>{/if}
  {#if trend}{@render trend()}{/if}
</div>
