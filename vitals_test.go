package systemvitals

import (
	"context"
	"os"
	"path/filepath"
	"runtime"
	"strings"
	"testing"
	"time"

	sensor "go.viam.com/rdk/components/sensor"
	"go.viam.com/rdk/logging"
	"go.viam.com/rdk/protoutils"
)

func TestValidate(t *testing.T) {
	if _, _, err := (Config{}).Validate("components.0"); err != nil {
		t.Fatalf("empty config should validate, got %v", err)
	}
	if _, _, err := (Config{ViamServerPID: -1}).Validate("components.0"); err == nil {
		t.Fatal("negative viam_server_pid should fail validation")
	}
	if _, _, err := (Config{DiskPaths: []string{"/", ""}}).Validate("components.0"); err == nil {
		t.Fatal("empty disk path should fail validation")
	}
}

func newTestSensor(t *testing.T, cfg Config) sensor.Sensor {
	t.Helper()
	ctx := context.Background()
	if cfg.ViamServerPID == 0 {
		cfg.ViamServerPID = os.Getpid()
	}
	s, err := NewVitalsSensor(ctx, nil, sensor.Named("vitals"), cfg, logging.NewTestLogger(t))
	if err != nil {
		t.Fatalf("NewVitalsSensor: %v", err)
	}
	t.Cleanup(func() { _ = s.Close(ctx) })
	// Give the "since last call" CPU counters a window to measure.
	time.Sleep(50 * time.Millisecond)
	return s
}

func groupOf(t *testing.T, readings map[string]interface{}, key string) map[string]interface{} {
	t.Helper()
	v, ok := readings[key]
	if !ok {
		t.Fatalf("readings missing %q group; errors=%v", key, readings["errors"])
	}
	m, ok := v.(map[string]interface{})
	if !ok {
		t.Fatalf("readings[%q] is %T, want map[string]interface{}", key, v)
	}
	return m
}

func TestReadingsShape(t *testing.T) {
	s := newTestSensor(t, Config{})
	readings, err := s.Readings(context.Background(), nil)
	if err != nil {
		t.Fatalf("Readings: %v", err)
	}

	for _, key := range []string{"host", "cpu", "memory", "disk", "viam_server", "module"} {
		groupOf(t, readings, key)
	}
	if errs, ok := readings["errors"].(map[string]interface{}); ok {
		for _, key := range []string{"host", "cpu", "memory", "disk:/", "viam_server", "module"} {
			if msg, bad := errs[key]; bad {
				t.Errorf("unexpected error for %q: %v", key, msg)
			}
		}
	}

	cpuPct, ok := groupOf(t, readings, "cpu")["percent"].(float64)
	if !ok {
		t.Fatal("cpu.percent missing or not float64")
	}
	if cpuPct < 0 || cpuPct > 100*float64(runtime.NumCPU()) {
		t.Errorf("cpu.percent out of range: %v", cpuPct)
	}

	total, ok := groupOf(t, readings, "memory")["total_bytes"].(uint64)
	if !ok || total == 0 {
		t.Errorf("memory.total_bytes should be a positive uint64, got %v", groupOf(t, readings, "memory")["total_bytes"])
	}

	if pid, _ := groupOf(t, readings, "viam_server")["pid"].(int32); int(pid) != os.Getpid() {
		t.Errorf("viam_server.pid = %v, want %d", pid, os.Getpid())
	}
	if _, ok := groupOf(t, readings, "module")["go_version"].(string); !ok {
		t.Error("module.go_version missing")
	}

	// Everything must survive conversion to protobuf.
	if _, err := protoutils.ReadingGoToProto(readings); err != nil {
		t.Fatalf("readings are not proto-convertible: %v", err)
	}
}

func TestReadingsOptions(t *testing.T) {
	off := false
	s := newTestSensor(t, Config{
		IncludeNetwork: &off,
		IncludePerCPU:  true,
		DiskPaths:      []string{"/", "/definitely/not/a/real/mount"},
	})
	readings, err := s.Readings(context.Background(), nil)
	if err != nil {
		t.Fatalf("Readings: %v", err)
	}
	if _, ok := readings["network"]; ok {
		t.Error("network group should be omitted when include_network is false")
	}
	if _, ok := groupOf(t, readings, "cpu")["per_cpu_percent"].([]interface{}); !ok {
		t.Error("cpu.per_cpu_percent should be a []interface{} when include_per_cpu is true")
	}
	diskGroup := groupOf(t, readings, "disk")
	if _, ok := diskGroup["/"]; !ok {
		t.Error("disk should still report \"/\" when another path fails")
	}
	errs, ok := readings["errors"].(map[string]interface{})
	if !ok {
		t.Fatal("errors group missing for bogus disk path")
	}
	if _, ok := errs["disk:/definitely/not/a/real/mount"]; !ok {
		t.Errorf("expected disk error entry, got %v", errs)
	}
	if _, err := protoutils.ReadingGoToProto(readings); err != nil {
		t.Fatalf("readings are not proto-convertible: %v", err)
	}
}

func TestDoCommand(t *testing.T) {
	s := newTestSensor(t, Config{})
	ctx := context.Background()
	out, err := s.DoCommand(ctx, map[string]interface{}{"readings": true})
	if err != nil {
		t.Fatalf("DoCommand readings: %v", err)
	}
	if _, ok := out["host"]; !ok {
		t.Error("DoCommand readings should return the readings map")
	}
	if _, err := s.DoCommand(ctx, map[string]interface{}{"bogus": 1}); err == nil {
		t.Error("unknown command should return an error")
	}
}

func TestDoCommandCustomControl(t *testing.T) {
	root := t.TempDir()
	meta := `{"models":[{"model":"viam:system-vitals:vitals-sensor","custom_control":"vitals_ui/dist/index.html"}]}`
	if err := os.WriteFile(filepath.Join(root, "meta.json"), []byte(meta), 0o600); err != nil {
		t.Fatal(err)
	}
	dist := filepath.Join(root, "vitals_ui", "dist")
	if err := os.MkdirAll(dist, 0o755); err != nil {
		t.Fatal(err)
	}
	const html = "<!doctype html><title>vitals</title>"
	if err := os.WriteFile(filepath.Join(dist, "index.html"), []byte(html), 0o600); err != nil {
		t.Fatal(err)
	}
	t.Setenv("VIAM_MODULE_ROOT", root)
	// metaPath is memoized; this test must observe the temp root, so reset the memo through a
	// fresh lookup helper rather than the package-level once.
	metaPath = func() string { return filepath.Join(root, "meta.json") }

	s := newTestSensor(t, Config{})
	out, err := s.DoCommand(context.Background(), map[string]interface{}{"command": "get-custom-control"})
	if err != nil {
		t.Fatalf("get-custom-control: %v", err)
	}
	if got, _ := out["html"].(string); got != html {
		t.Errorf("html = %q, want %q", got, html)
	}
	if rev, _ := out["revision"].(string); len(rev) != 64 || strings.Trim(rev, "0123456789abcdef") != "" {
		t.Errorf("revision should be a hex sha256, got %q", rev)
	}
	if p, _ := out["path"].(string); p != "vitals_ui/dist/index.html" {
		t.Errorf("path = %q", p)
	}
}
