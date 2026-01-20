# Elastic Application ServicE (EASe)

EASe is a lightweight Go web app that discovers sub-apps from a directory tree and renders them in a grouped catalog. It is designed to act as the front door for launching and proxying multiple containerized apps (Dash, Streamlit, Flask, etc.). This initial version focuses on discovery, grouping, and a simple UI.

## Features

- **Recursive discovery** of `spec.yml` files under a configurable `apps_dir`.
- **Automatic grouping** based on directory structure (nested folders become nested groups).
- **Periodic rescan** every few seconds to keep the catalog fresh.
- **Single static binary** build (templates are embedded with `go:embed`; the UI uses Tailwind via CDN for styling).

## Directory Structure

EASe expects an `apps_dir` where each app lives in its own folder with a `spec.yml`. The structure can be nested to form groups.

Example:

```
apps_dir/
├─ app1/spec.yml
├─ groupA/app2/spec.yml
├─ groupA/app3/spec.yml
├─ groupA/subgroupA2/app4/spec.yml
└─ groupB/app5/spec.yml
```

Apps are grouped by their parent folders, so `groupA/app2` and `groupA/app3` appear under **groupA**, and `groupA/subgroupA2/app4` appears under **groupA/subgroupA2**.

## spec.yml format

`spec.yml` supports a minimal schema:

```yaml
name: Sales Dashboard
description: Metrics for Q4 campaign performance.
```

`name` defaults to the app directory name if omitted. `description` is optional.

## Build

```
go build -o ease .
```

The output binary (`ease`) is the only artifact you need to run.

## Run

EASe requires an `apps_dir` path. Provide it with a flag or environment variable.

### Option A: Command-line flag

```
./ease -apps_dir /path/to/apps_dir
```

### Option B: Environment variable

```
export APPS_DIR=/path/to/apps_dir
./ease
```

### Optional flags

- `-port` (default: `8080`) — HTTP port for the UI.
- `-scan_interval` (default: `8s`) — how often EASe rescans `apps_dir`.

Example:

```
./ease -apps_dir ./apps_dir -port 9090 -scan_interval 10s
```

## Run in development

```
go run ./... -apps_dir ./apps_dir
```

Then open <http://localhost:8080> to view the catalog.

## Sample apps for screenshots

Use the helper script to create a realistic app tree for demoing the UI or capturing screenshots:

```
./scripts/create_sample_apps.sh /tmp/ease_apps
./ease -apps_dir /tmp/ease_apps
```

## Testing

Run the test suite:

```
go test ./...
```
