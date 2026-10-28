package main

import (
	"context"
	"encoding/json"
	"fmt"
	"os"
	"systemvitals"
	"time"

	sensor "go.viam.com/rdk/components/sensor"
	"go.viam.com/rdk/logging"
	"go.viam.com/rdk/resource"
)

func main() {
	err := realMain()
	if err != nil {
		panic(err)
	}
}

func realMain() error {
	ctx := context.Background()
	logger := logging.NewLogger("cli")

	deps := resource.Dependencies{}

	// Outside viam-server the parent process is just the shell; inspect it anyway so the
	// viam_server group is populated and the full readings shape can be seen.
	cfg := systemvitals.Config{ViamServerPID: os.Getppid()}

	thing, err := systemvitals.NewVitalsSensor(ctx, deps, sensor.Named("foo"), cfg, logger)
	if err != nil {
		return err
	}
	defer thing.Close(ctx)

	// Let the "since last call" CPU counters accumulate a measurable window.
	time.Sleep(500 * time.Millisecond)

	readings, err := thing.Readings(ctx, nil)
	if err != nil {
		return err
	}
	out, err := json.MarshalIndent(readings, "", "  ")
	if err != nil {
		return err
	}
	fmt.Println(string(out))
	return nil
}
