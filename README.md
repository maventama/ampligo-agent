# ampligo-agent

Small daemon that samples CPU, memory, disk, and load average on a host and
pushes them to Ampligo's usage ingest endpoint (`POST /api/v1/ingest/usage`)
on an interval.

## Quick install (Linux/macOS)

Generate an API key from the app's Settings tab in Ampligo (Agent API Keys
section), then run the one-liner shown there, or manually:

```
curl -sSL https://ampligo.niago.id/install-agent.sh | sudo AMPLIGO_API_KEY=amp_xxx bash
```

This downloads the right binary from [GitHub Releases](https://github.com/maventama/ampligo-agent/releases),
installs it to `/usr/local/bin`, writes `/etc/ampligo/agent.yml`, and sets it
up as a systemd service on Linux (on macOS it just installs the binary and
config - start it manually or via your own supervisor).

The sections below cover building from source instead.

## Build

```
go build -o ampligo-agent .
```

## Configure

Copy `agent.example.yml` and fill in the API key generated from the app's
Settings tab in Ampligo (Agent API Keys section):

```
api_key: "amp_xxx"
ingest_url: "https://ampligo.niago.id/api/v1/ingest/usage"
interval_seconds: 15
```

The API key is a credential separate from your Ampligo login, scoped to
pushing metrics for a single app. It never needs to leave this config file.

### Optional: MySQL/Postgres monitoring

Add a `database` block to also push connection count, queries/sec, cache hit
ratio, and replication lag:

```
database:
  type: mysql   # or postgres
  dsn: "readonly:password@tcp(127.0.0.1:3306)/mydb"
```

Use a **read-only** DB user: `PROCESS` grant for MySQL, the `pg_monitor` role
for Postgres. The DSN stays in this file - only the derived metrics get
pushed, to `db_ingest_url` (auto-derived from `ingest_url`, e.g.
`.../ingest/usage` -> `.../ingest/db-usage`).

## Run

```
./ampligo-agent --config ./agent.yml
```

Flags:
- `--config` - path to agent.yml (default `/etc/ampligo/agent.yml`)
- `--disk-path` - filesystem path to report disk usage for (default `/`)

## Install as a systemd service (Linux)

```
sudo ./install.sh
sudo $EDITOR /etc/ampligo/agent.yml   # set api_key
sudo systemctl enable --now ampligo-agent
```

## Notes

- `network_in_bytes` / `network_out_bytes` are cumulative counters since boot,
  not deltas since the last push - the server derives rates from consecutive readings.
- Failed pushes retry up to 3 times with a short backoff; there is no
  persistent offline buffer yet, so pushes made while the agent can't reach
  Ampligo are dropped rather than queued for later.
