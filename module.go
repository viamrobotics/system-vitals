// Package systemvitals implements a Viam sensor component that reports basic health metrics
// for the machine viam-server is running on, plus process-level metrics for viam-server itself.
package systemvitals

import (
	"context"
	"errors"
	"fmt"
	"os"
	"sync"

	"github.com/shirou/gopsutil/v3/cpu"
	"github.com/shirou/gopsutil/v3/process"
	sensor "go.viam.com/rdk/components/sensor"
	"go.viam.com/rdk/logging"
	"go.viam.com/rdk/resource"

	"systemvitals/internal/customcontrol"
)

var (
	// VitalsSensor is the model triplet for this sensor.
	VitalsSensor     = resource.NewModel("viam", "system-vitals", "vitals-sensor")
	errUnimplemented = errors.New("unimplemented")
)

func init() {
	resource.RegisterComponent(sensor.API, VitalsSensor,
		resource.Registration[sensor.Sensor, Config]{
			Constructor: newSystemVitalsVitalsSensor,
		},
	)
}

// Config holds the optional attributes for the vitals sensor. Every field has a default, so an
// empty config is valid.
type Config struct {
	// DiskPaths lists filesystem paths whose usage is reported. Defaults to ["/"].
	DiskPaths []string `json:"disk_paths,omitempty"`
	// IncludePerCPU adds a per-core CPU utilization list to the "cpu" group. Defaults to false.
	IncludePerCPU bool `json:"include_per_cpu,omitempty"`
	// IncludeNetwork controls whether per-interface network counters are reported. Defaults to true.
	IncludeNetwork *bool `json:"include_network,omitempty"`
	// ViamServerPID overrides the process inspected for the "viam_server" group. Defaults to the
	// module's parent process, which is viam-server when running as a module.
	ViamServerPID int `json:"viam_server_pid,omitempty"`
}

// Validate ensures all parts of the config are valid. This sensor has no resource dependencies.
func (cfg Config) Validate(path string) ([]string, []string, error) {
	if cfg.ViamServerPID < 0 {
		return nil, nil, fmt.Errorf("%s: viam_server_pid must not be negative, got %d", path, cfg.ViamServerPID)
	}
	for i, p := range cfg.DiskPaths {
		if p == "" {
			return nil, nil, fmt.Errorf("%s: disk_paths[%d] must not be empty", path, i)
		}
	}
	return nil, nil, nil
}

// diskPaths returns the configured disk paths or the default.
func (cfg Config) diskPaths() []string {
	if len(cfg.DiskPaths) == 0 {
		return []string{"/"}
	}
	return cfg.DiskPaths
}

// includeNetwork returns whether network counters should be reported.
func (cfg Config) includeNetwork() bool {
	return cfg.IncludeNetwork == nil || *cfg.IncludeNetwork
}

// viamServerPID returns the PID to inspect for the viam_server group.
func (cfg Config) viamServerPID() int {
	if cfg.ViamServerPID > 0 {
		return cfg.ViamServerPID
	}
	return os.Getppid()
}

type systemVitalsVitalsSensor struct {
	resource.AlwaysRebuild
	resource.Named

	name resource.Name

	logger logging.Logger
	cfg    Config

	// mu serializes Readings: CPU-percent values are deltas since the previous call, so
	// concurrent collection would corrupt them.
	mu sync.Mutex

	// proc is the viam-server process; nil if it could not be opened.
	proc *process.Process
	// self is this module's own process; nil if it could not be opened.
	self *process.Process

	cancelCtx  context.Context
	cancelFunc func()
}

func newSystemVitalsVitalsSensor(ctx context.Context, deps resource.Dependencies, rawConf resource.Config, logger logging.Logger) (sensor.Sensor, error) {
	conf, err := resource.NativeConfig[Config](rawConf)
	if err != nil {
		return nil, err
	}

	return NewVitalsSensor(ctx, deps, rawConf.ResourceName(), conf, logger)
}

// NewVitalsSensor constructs the sensor and primes the stateful CPU-percent counters so the
// first Readings call already returns meaningful utilization numbers.
func NewVitalsSensor(ctx context.Context, deps resource.Dependencies, name resource.Name, conf Config, logger logging.Logger) (sensor.Sensor, error) {
	cancelCtx, cancelFunc := context.WithCancel(context.Background())

	s := &systemVitalsVitalsSensor{
		name:       name,
		logger:     logger,
		cfg:        conf,
		cancelCtx:  cancelCtx,
		cancelFunc: cancelFunc,
	}

	pid := conf.viamServerPID()
	proc, err := process.NewProcessWithContext(ctx, int32(pid))
	if err != nil {
		logger.Warnw("could not open viam-server process; viam_server group will be unavailable", "pid", pid, "error", err)
	} else {
		s.proc = proc
	}

	self, err := process.NewProcessWithContext(ctx, int32(os.Getpid()))
	if err != nil {
		logger.Warnw("could not open own process; module group will be unavailable", "error", err)
	} else {
		s.self = self
	}

	// Prime the "since last call" counters. Results are intentionally discarded.
	_, _ = cpu.PercentWithContext(ctx, 0, false)
	if conf.IncludePerCPU {
		_, _ = cpu.PercentWithContext(ctx, 0, true)
	}
	if s.proc != nil {
		_, _ = s.proc.PercentWithContext(ctx, 0)
	}
	if s.self != nil {
		_, _ = s.self.PercentWithContext(ctx, 0)
	}

	return s, nil
}

func (s *systemVitalsVitalsSensor) Name() resource.Name {
	return s.name
}

// Readings returns the nested vitals map. Groups that fail to collect are omitted and listed under
// the "errors" key; an error is returned only if the context is done or nothing could be collected.
func (s *systemVitalsVitalsSensor) Readings(ctx context.Context, extra map[string]interface{}) (map[string]interface{}, error) {
	s.mu.Lock()
	defer s.mu.Unlock()
	return s.collect(ctx)
}

// DoCommand serves the custom control artifact declared in meta.json ({"command":
// "get-custom-control"}) and supports {"readings": true} as a convenience that returns the same
// map as Readings.
func (s *systemVitalsVitalsSensor) DoCommand(ctx context.Context, cmd map[string]interface{}) (map[string]interface{}, error) {
	if resp, handled, err := customcontrol.Handle(cmd, metaPath(), VitalsSensor.String()); handled {
		return resp, err
	}
	if v, ok := cmd["readings"]; ok {
		if b, isBool := v.(bool); !isBool || b {
			return s.Readings(ctx, nil)
		}
	}
	keys := make([]string, 0, len(cmd))
	for k := range cmd {
		keys = append(keys, k)
	}
	return nil, fmt.Errorf("command %v: %w", keys, errUnimplemented)
}

func (s *systemVitalsVitalsSensor) Status(ctx context.Context) (map[string]interface{}, error) {
	return nil, nil
}

func (s *systemVitalsVitalsSensor) Close(context.Context) error {
	s.cancelFunc()
	return nil
}
