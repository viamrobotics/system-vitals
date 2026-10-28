<script lang="ts">
  interface Props {
    label: string;
    detail?: string;
    percent: number | undefined;
    /** Severity thresholds on `percent`. */
    warnAt?: number;
    criticalAt?: number;
  }

  const DEFAULT_WARN = 75;
  const DEFAULT_CRITICAL = 90;
  const FULL = 100;

  let {
    label,
    detail,
    percent,
    warnAt = DEFAULT_WARN,
    criticalAt = DEFAULT_CRITICAL,
  }: Props = $props();

  const clamped = $derived(Math.min(FULL, Math.max(0, percent ?? 0)));
  const severity = $derived.by(() => {
    if (percent === undefined) return 'unknown';
    if (percent >= criticalAt) return 'critical';
    if (percent >= warnAt) return 'warning';
    return 'ok';
  });
  const fill = $derived(
    severity === 'critical'
      ? 'var(--vz-critical)'
      : severity === 'warning'
        ? 'var(--vz-warning)'
        : 'var(--vz-s1)',
  );
  const SEVERITY_LABEL: Record<string, string> = {
    critical: 'Critical',
    warning: 'High',
    ok: '',
    unknown: '',
  };
</script>

<div class="flex flex-col gap-1">
  <div class="flex items-baseline justify-between gap-3 text-xs">
    <span class="truncate font-medium">{label}</span>
    <span class="vz-tabular vz-ink-2 shrink-0">
      {#if SEVERITY_LABEL[severity]}
        <span class="mr-1" title={SEVERITY_LABEL[severity]} aria-label={SEVERITY_LABEL[severity]}
          >{severity === 'critical' ? '⚠' : '▲'}</span
        >
      {/if}
      {percent === undefined ? '—' : `${percent.toFixed(0)}%`}
      {#if detail}<span class="vz-muted"> · {detail}</span>{/if}
    </span>
  </div>
  <div
    class="h-2 w-full overflow-hidden rounded-sm"
    style:background="var(--vz-track)"
    role="meter"
    aria-label={label}
    aria-valuemin="0"
    aria-valuemax={FULL}
    aria-valuenow={percent === undefined ? undefined : Math.round(percent)}
  >
    <div class="h-full rounded-sm" style:width="{clamped}%" style:background={fill}></div>
  </div>
</div>
