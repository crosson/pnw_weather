# pnw_weather

OpenClaw skill implementation for Pacific Northwest weather, avalanche, telemetry, and mountain pass conditions, now implemented as a Go CLI binary.

## What is implemented

- Area management with local storage (`areas.json`, versioned)
- Provider adapters for NWS, NWAC, and WSDOT
- Normalized response schema with source links and cache metadata
- Persistent JSON cache with TTL
- Partial-failure tolerant daily digest envelopes (`data` + `error`)
- CLI for all public APIs in `SPEC.md`

## Build and run

```bash
source ~/.profile
go build ./cmd/pnw-forecast
```

Run with data path defaults (`~/.openclaw/skills/pnw_forecast`):

```bash
./pnw-forecast area list
```

Use a local data root while developing:

```bash
./pnw-forecast --root ./.local-data area list
```

## Examples

Save an area:

```bash
./pnw-forecast area save Alpental \
  --lat 47.445 --lon -121.425 \
  --nwac-zone-id SNOQUALMIE_PASS \
  --station-id alpental-base \
  --wsdot-pass-id snoqualmie
```

Get weather by area:

```bash
./pnw-forecast weather --area Alpental
```

Get digest:

```bash
./pnw-forecast digest
```

## Tests

```bash
source ~/.profile
GOCACHE=/tmp/go-build go test ./...
```
