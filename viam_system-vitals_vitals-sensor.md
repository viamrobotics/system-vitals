# Model viam:system-vitals:vitals-sensor

A sensor that reports basic health metrics for the machine `viam-server` is running on, plus
process-level metrics for the `viam-server` process itself. Everything is returned through the
standard sensor `GetReadings` call, so it works with data capture, the app's test panel and any
SDK without extra configuration.

No connection back to `viam-server` is made. The `viam_server` group is read from the module's
parent process, and the `machine` group comes from the `VIAM_*` environment variables that
`viam-server` passes to modules.

## Configuration

Every attribute is optional; an empty attributes object is valid.

```json
{
  "disk_paths": ["/", "/data"],
  "include_per_cpu": false,
  "include_network": true,
  "viam_server_pid": 0
}
```

### Attributes

| Name              | Type     | Inclusion | Default        | Description |
|-------------------|----------|-----------|----------------|-------------|
| `disk_paths`      | string[] | Optional  | `["/"]`        | Filesystem paths whose usage is reported under `disk`. |
| `include_per_cpu` | bool     | Optional  | `false`        | Adds a `per_cpu_percent` list to the `cpu` group. |
| `include_network` | bool     | Optional  | `true`         | Reports per-interface byte/packet/error counters under `network`. |
| `viam_server_pid` | int      | Optional  | parent process | Overrides which process is inspected for the `viam_server` group. Useful for local testing only. |

### Example Configuration

```json
{
  "disk_paths": ["/"]
}
```

## Readings

Readings are nested by group. A group that cannot be collected on the current platform is
omitted and its error message is listed under `errors`. `temperature` and `machine` are best
effort and never produce errors.

| Group         | Contents |
|---------------|----------|
| `host`        | `hostname`, `os`, `platform`, `platform_family`, `platform_version`, `kernel_version`, `kernel_arch`, `uptime_seconds`, `boot_time_unix`, `process_count` |
| `cpu`         | `percent` (utilization since the previous reading), `count_logical`, `count_physical`, optional `per_cpu_percent` |
| `load`        | `load1`, `load5`, `load15` (not available on Windows) |
| `memory`      | `total_bytes`, `available_bytes`, `used_bytes`, `used_percent`, `free_bytes`, `swap_total_bytes`, `swap_used_bytes`, `swap_used_percent` |
| `disk`        | One entry per configured path: `total_bytes`, `used_bytes`, `free_bytes`, `used_percent`, `inodes_used_percent`, `fstype` |
| `network`     | One entry per interface: `bytes_sent`, `bytes_recv`, `packets_sent`, `packets_recv`, `err_in`, `err_out`, `drop_in`, `drop_out` |
| `temperature` | Sensor key to degrees Celsius (Linux only) |
| `viam_server` | `pid`, `name`, `exe`, `status`, `cpu_percent`, `user_cpu_seconds`, `system_cpu_seconds`, `rss_bytes`, `vms_bytes`, `memory_percent`, `num_threads`, `num_fds` (Linux), `num_children` (running module processes), `create_time_unix_ms`, `uptime_seconds` |
| `module`      | Same process fields for this module's own process, plus `goroutines` and `go_version` |
| `machine`     | `machine_id`, `machine_part_id`, `machine_fqdn`, `location_id`, `primary_org_id`, `module_name`, `module_id`, `viam_home`, `module_data_dir` (only keys with a value) |
| `errors`      | Group name to error message, only present when something failed |

### Example (trimmed)

```json
{
  "host": { "hostname": "rpi-5", "os": "linux", "platform": "debian", "kernel_arch": "aarch64", "uptime_seconds": 86400 },
  "cpu": { "percent": 12.5, "count_logical": 4, "count_physical": 4 },
  "load": { "load1": 0.42, "load5": 0.35, "load15": 0.3 },
  "memory": { "total_bytes": 8318828544, "used_percent": 23.1 },
  "disk": { "/": { "total_bytes": 62725623808, "used_percent": 41.2, "fstype": "ext4" } },
  "viam_server": { "pid": 1234, "name": "viam-server", "cpu_percent": 3.2, "rss_bytes": 210763776, "num_threads": 31, "num_children": 2, "uptime_seconds": 3600.5 },
  "module": { "pid": 1300, "rss_bytes": 28311552, "goroutines": 14, "go_version": "go1.26.10" },
  "machine": { "machine_id": "…", "machine_part_id": "…", "module_name": "system-vitals" }
}
```

## Custom control

The model ships a custom control (`vitals_ui/`, declared as `custom_control` in `meta.json`)
that the Viam app renders on the machine's control page under the **Custom** toggle. It polls
`GetReadings` every two seconds and shows headline tiles, three-minute trend charts (host and
viam-server CPU, memory, load average, network throughput), disk and swap meters, temperatures,
machine identity, and a table of every reading.

Build it with `make ui` (needs Node and pnpm) or `cd vitals_ui && pnpm install && pnpm build`.
`cd vitals_ui && pnpm dev` runs it against a fake robot, or against a real machine with the
credentials from `vitals_ui/.env.example`.

## DoCommand

| Command | Reply |
|---------|-------|
| `{"command": "get-custom-control"}` | `{"html": <contents of vitals_ui/dist/index.html>, "revision": <sha256>, "path": "vitals_ui/dist/index.html"}`. Used by the Viam app to load the control; transitional until the platform serves the declared file itself. |
| `{"readings": true}` | The same map as `GetReadings`. |

Any other command returns an "unimplemented" error.
