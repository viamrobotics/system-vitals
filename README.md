# Module system-vitals

A Viam module that exposes the health of the machine `viam-server` runs on as a plain sensor:
host CPU, load, memory, disk, network and temperature, plus process-level metrics for the
`viam-server` process and the machine's Viam identity. All of it comes back through the standard
sensor `GetReadings` call, so it can be captured with data management or read from any SDK.

The `vitals-sensor` model also ships a custom control for the Viam app's control page: live
charts and readouts of the same readings, built with Svelte and LayerCake in [`vitals_ui/`](vitals_ui/).

## Building

`make module.tar.gz` compiles the Go binary, builds the control UI (Node 22 and pnpm required) and
packages `meta.json`, the binary and `vitals_ui/dist/index.html`. `./setup.sh` installs Node and
pnpm when they are missing; `viam module build local` runs both steps the way the cloud does.

### Cloud build

`meta.json` declares `setup.sh` as the build setup step, so Viam's cloud build installs Node on
its Linux builders (and `node`/`pnpm` through Homebrew on macOS) before running `make
module.tar.gz` for every platform in `build.arch`. Pushing a version tag triggers
`.github/workflows/deploy.yml`, which builds and publishes all platforms. One-time setup:

1. Set `url` in `meta.json` to this module's GitHub repository (cloud build requires it).
2. Create an org API key (`viam organization api-key create --org-id <org> --name cloud-build`)
   and store it as the `viam_key_id` and `viam_key_value` repository secrets.
3. Tag a release, for example `git tag 0.1.0 && git push origin 0.1.0`.

To build without publishing, run `viam module build start --version <semver>` from the repo, or
set `version: ""` in the workflow.

## Models

This module provides the following model(s):

- [`viam:system-vitals:vitals-sensor`](viam_system-vitals_vitals-sensor.md) - Host and `viam-server` health metrics as nested sensor readings.
