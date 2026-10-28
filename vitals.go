package systemvitals

import (
	"context"
	"errors"
	"math"
	"os"
	"runtime"
	"strings"
	"time"

	"github.com/shirou/gopsutil/v3/cpu"
	"github.com/shirou/gopsutil/v3/disk"
	"github.com/shirou/gopsutil/v3/host"
	"github.com/shirou/gopsutil/v3/load"
	"github.com/shirou/gopsutil/v3/mem"
	"github.com/shirou/gopsutil/v3/net"
	"github.com/shirou/gopsutil/v3/process"
	rutils "go.viam.com/rdk/utils"
)

// Environment variables viam-server sets for modules that have no exported constant in RDK.
const (
	envModuleName = "VIAM_MODULE_NAME"
	envModuleID   = "VIAM_MODULE_ID"
	envModuleData = "VIAM_MODULE_DATA"
)

// group is a single readings section. Every value placed in the returned map must be a Go
// scalar (int/uint/float/string/bool), a map[string]interface{} or a []interface{} so that it
// survives conversion to protobuf at the gRPC boundary.
type group func(ctx context.Context) (map[string]interface{}, error)

// collect gathers every group. Failing groups are omitted and recorded under "errors".
func (s *systemVitalsVitalsSensor) collect(ctx context.Context) (map[string]interface{}, error) {
	readings := map[string]interface{}{}
	errs := map[string]interface{}{}

	groups := []struct {
		key string
		fn  group
	}{
		{"host", hostGroup},
		{"cpu", s.cpuGroup},
		{"load", loadGroup},
		{"memory", memoryGroup},
		{"viam_server", s.viamServerGroup},
		{"module", s.moduleGroup},
	}
	if s.cfg.includeNetwork() {
		groups = append(groups, struct {
			key string
			fn  group
		}{"network", networkGroup})
	}

	for _, g := range groups {
		if err := ctx.Err(); err != nil {
			return nil, err
		}
		vals, err := g.fn(ctx)
		if err != nil {
			errs[g.key] = err.Error()
			continue
		}
		readings[g.key] = vals
	}

	// Disk reports per path so one bad path does not hide the others.
	if diskVals, diskErrs := diskGroup(ctx, s.cfg.diskPaths()); len(diskVals) > 0 || len(diskErrs) > 0 {
		if len(diskVals) > 0 {
			readings["disk"] = diskVals
		}
		for k, v := range diskErrs {
			errs[k] = v
		}
	}

	// Temperature and machine identity are best effort and never produce errors.
	if temps := temperatureGroup(ctx); len(temps) > 0 {
		readings["temperature"] = temps
	}
	if machine := machineGroup(); len(machine) > 0 {
		readings["machine"] = machine
	}

	if err := ctx.Err(); err != nil {
		return nil, err
	}
	if len(readings) == 0 {
		return nil, errors.New("no vitals could be collected: " + joinErrs(errs))
	}
	if len(errs) > 0 {
		readings["errors"] = errs
	}
	return readings, nil
}

func joinErrs(errs map[string]interface{}) string {
	parts := make([]string, 0, len(errs))
	for k, v := range errs {
		parts = append(parts, k+": "+v.(string))
	}
	return strings.Join(parts, "; ")
}

func hostGroup(ctx context.Context) (map[string]interface{}, error) {
	info, err := host.InfoWithContext(ctx)
	if err != nil {
		return nil, err
	}
	return map[string]interface{}{
		"hostname":         info.Hostname,
		"os":               info.OS,
		"platform":         info.Platform,
		"platform_family":  info.PlatformFamily,
		"platform_version": info.PlatformVersion,
		"kernel_version":   info.KernelVersion,
		"kernel_arch":      info.KernelArch,
		"uptime_seconds":   info.Uptime,
		"boot_time_unix":   info.BootTime,
		"process_count":    info.Procs,
	}, nil
}

func (s *systemVitalsVitalsSensor) cpuGroup(ctx context.Context) (map[string]interface{}, error) {
	percents, err := cpu.PercentWithContext(ctx, 0, false)
	if err != nil {
		return nil, err
	}
	out := map[string]interface{}{}
	if len(percents) > 0 {
		out["percent"] = percents[0]
	}
	if n, err := cpu.CountsWithContext(ctx, true); err == nil {
		out["count_logical"] = n
	}
	if n, err := cpu.CountsWithContext(ctx, false); err == nil {
		out["count_physical"] = n
	}
	if s.cfg.IncludePerCPU {
		if per, err := cpu.PercentWithContext(ctx, 0, true); err == nil {
			list := make([]interface{}, 0, len(per))
			for _, p := range per {
				list = append(list, p)
			}
			out["per_cpu_percent"] = list
		}
	}
	return out, nil
}

func loadGroup(ctx context.Context) (map[string]interface{}, error) {
	avg, err := load.AvgWithContext(ctx)
	if err != nil {
		return nil, err
	}
	return map[string]interface{}{
		"load1":  avg.Load1,
		"load5":  avg.Load5,
		"load15": avg.Load15,
	}, nil
}

func memoryGroup(ctx context.Context) (map[string]interface{}, error) {
	vm, err := mem.VirtualMemoryWithContext(ctx)
	if err != nil {
		return nil, err
	}
	out := map[string]interface{}{
		"total_bytes":     vm.Total,
		"available_bytes": vm.Available,
		"used_bytes":      vm.Used,
		"used_percent":    vm.UsedPercent,
		"free_bytes":      vm.Free,
	}
	if swap, err := mem.SwapMemoryWithContext(ctx); err == nil {
		out["swap_total_bytes"] = swap.Total
		out["swap_used_bytes"] = swap.Used
		out["swap_used_percent"] = swap.UsedPercent
	}
	return out, nil
}

// diskGroup returns usage keyed by path plus a map of per-path errors keyed "disk:<path>".
func diskGroup(ctx context.Context, paths []string) (map[string]interface{}, map[string]interface{}) {
	out := map[string]interface{}{}
	errs := map[string]interface{}{}
	for _, p := range paths {
		usage, err := disk.UsageWithContext(ctx, p)
		if err != nil {
			errs["disk:"+p] = err.Error()
			continue
		}
		out[p] = map[string]interface{}{
			"total_bytes":         usage.Total,
			"used_bytes":          usage.Used,
			"free_bytes":          usage.Free,
			"used_percent":        usage.UsedPercent,
			"inodes_used_percent": usage.InodesUsedPercent,
			"fstype":              usage.Fstype,
		}
	}
	return out, errs
}

func networkGroup(ctx context.Context) (map[string]interface{}, error) {
	counters, err := net.IOCountersWithContext(ctx, true)
	if err != nil {
		return nil, err
	}
	out := map[string]interface{}{}
	for _, c := range counters {
		out[c.Name] = map[string]interface{}{
			"bytes_sent":   c.BytesSent,
			"bytes_recv":   c.BytesRecv,
			"packets_sent": c.PacketsSent,
			"packets_recv": c.PacketsRecv,
			"err_in":       c.Errin,
			"err_out":      c.Errout,
			"drop_in":      c.Dropin,
			"drop_out":     c.Dropout,
		}
	}
	return out, nil
}

// temperatureGroup is best effort: unsupported platforms return nil.
func temperatureGroup(ctx context.Context) map[string]interface{} {
	temps, err := host.SensorsTemperaturesWithContext(ctx)
	if err != nil {
		return nil
	}
	return usableTemperatures(temps)
}

// usableTemperatures keeps only readings a real sensor could have produced. gopsutil reports
// 0 for sensors it cannot read (on Apple Silicon every SMC key it knows is an Intel one, so all
// come back 0; on Linux a disconnected hwmon channel reads 0 too), and nothing viam-server runs
// on sits at exactly 0.000 °C. Returns nil when no reading survives so the group is omitted.
func usableTemperatures(temps []host.TemperatureStat) map[string]interface{} {
	const maxPlausibleCelsius = 200
	out := map[string]interface{}{}
	for _, t := range temps {
		if t.SensorKey == "" || t.Temperature <= 0 || t.Temperature > maxPlausibleCelsius ||
			math.IsNaN(t.Temperature) {
			continue
		}
		out[t.SensorKey] = t.Temperature
	}
	if len(out) == 0 {
		return nil
	}
	return out
}

func (s *systemVitalsVitalsSensor) viamServerGroup(ctx context.Context) (map[string]interface{}, error) {
	if s.proc == nil {
		return nil, errors.New("viam-server process is not available")
	}
	out, err := processGroup(ctx, s.proc)
	if err != nil {
		return nil, err
	}
	if exe, err := s.proc.ExeWithContext(ctx); err == nil {
		out["exe"] = exe
	}
	if children, err := s.proc.ChildrenWithContext(ctx); err == nil {
		out["num_children"] = len(children)
	}
	return out, nil
}

func (s *systemVitalsVitalsSensor) moduleGroup(ctx context.Context) (map[string]interface{}, error) {
	if s.self == nil {
		return nil, errors.New("module process is not available")
	}
	out, err := processGroup(ctx, s.self)
	if err != nil {
		return nil, err
	}
	out["goroutines"] = runtime.NumGoroutine()
	out["go_version"] = runtime.Version()
	return out, nil
}

// processGroup reports the common per-process metrics. Only the create time is required; every
// other field is skipped when its lookup fails so that one unsupported call does not hide the rest.
func processGroup(ctx context.Context, p *process.Process) (map[string]interface{}, error) {
	createMS, err := p.CreateTimeWithContext(ctx)
	if err != nil {
		return nil, err
	}
	out := map[string]interface{}{
		"pid":                 p.Pid,
		"create_time_unix_ms": createMS,
		"uptime_seconds":      float64(time.Now().UnixMilli()-createMS) / 1000.0,
	}
	if name, err := p.NameWithContext(ctx); err == nil {
		out["name"] = name
	}
	if status, err := p.StatusWithContext(ctx); err == nil && len(status) > 0 {
		out["status"] = strings.Join(status, ",")
	}
	if pct, err := p.PercentWithContext(ctx, 0); err == nil {
		out["cpu_percent"] = pct
	}
	if times, err := p.TimesWithContext(ctx); err == nil {
		out["user_cpu_seconds"] = times.User
		out["system_cpu_seconds"] = times.System
	}
	if memInfo, err := p.MemoryInfoWithContext(ctx); err == nil {
		out["rss_bytes"] = memInfo.RSS
		out["vms_bytes"] = memInfo.VMS
	}
	if memPct, err := p.MemoryPercentWithContext(ctx); err == nil {
		out["memory_percent"] = float64(memPct)
	}
	if threads, err := p.NumThreadsWithContext(ctx); err == nil {
		out["num_threads"] = threads
	}
	if fds, err := p.NumFDsWithContext(ctx); err == nil {
		out["num_fds"] = fds
	}
	return out, nil
}

// machineGroup reports identity viam-server passes to modules through the environment.
func machineGroup() map[string]interface{} {
	fields := map[string]string{
		"machine_id":      rutils.MachineIDEnvVar,
		"machine_part_id": rutils.MachinePartIDEnvVar,
		"machine_fqdn":    rutils.MachineFQDNEnvVar,
		"location_id":     rutils.LocationIDEnvVar,
		"primary_org_id":  rutils.PrimaryOrgIDEnvVar,
		"viam_home":       rutils.HomeEnvVar,
		"module_name":     envModuleName,
		"module_id":       envModuleID,
		"module_data_dir": envModuleData,
	}
	out := map[string]interface{}{}
	for key, envVar := range fields {
		if v := os.Getenv(envVar); v != "" {
			out[key] = v
		}
	}
	return out
}
