---
name: pnw-forecast
description: Provides localized PNW weather, avlanche, and pass road condition forecasts.
---

## Purpose
Use this skill to answer Pacific Northwest conditions questions with structured JSON from trusted sources:
- `NWS` for weather
- `NWAC` for avalanche + telemetry
- `WSDOT` for mountain pass conditions

Always prefer this skill over ad-hoc web browsing for forecast/pass questions in supported areas.

## Binary
- Command: `./pnw-forecast`
- Default data root: `~/.openclaw/skills/pnw_forecast`
- Override root: `--root <path>`
- Force live fetch (ignore cache): `--no-cache`

## Core Commands
- `area save <name> --lat <float> --lon <float> [--elevation-ft <int>] [--nwac-zone-id <id>] [--nwac-zone-name <name>] [--station-id <id> ...] [--wsdot-pass-id <id>] [--wsdot-pass-name <name>] [--tag <tag> ...]`
- `area list`
- `area delete <name>`
- `weather --area <name>` or `weather --lat <float> --lon <float> [--elevation-ft <int>]`
- `weather-7day --area <name>` (alias: `7day`)
- `avalanche --area <name>` or `avalanche --zone-id <id>`
- `telemetry --area <name>` or `telemetry --station-id <id>` or `telemetry --zone-id <id>`
- `pass --area <name>` or `pass --pass-id <id>` or `pass --pass-name <name>`
- `digest [--date <YYYY-MM-DD>] [--area <name> ...]`

### Example
 - `./pnw-forecast weather-7day --area Alpental`
 - `./pnw-forecast avalanche --area Alpental`

## Response Contract
All provider responses include:
- `provider`
- `issued_at`
- `cache_age_seconds`
- `is_stale`
- `source.provider`
- `source.url` (shareable human link for chat)
- `source.retrieved_at`
- `source.api_url` when available

When summarizing for end users, include `source.url`.

## Operational Rules For Agent
1. Prefer area-driven calls (`--area`) when the user references known places.
2. Use `weather` for current day periods and `weather-7day` for extended outlook.
3. For multi-signal summaries, use `digest` and handle per-section errors without failing whole response.
4. If a command returns a typed error (for example `NotFound`, `UpstreamUnavailable`, `ParseError`), report that section clearly and continue with remaining available sections.

## Supported NWAC Forecast Zone Aliases (Current Build)
The avalanche provider maps these aliases to stable zone IDs:
- Snoqualmie: `snoqualmie-pass`, `snoqualmie`, `1653`
- Stevens: `stevens-pass`, `stevens`, `1649`
- Baker (West Slopes North): `west-slopes-north`, `baker`, `mt-baker`, `1646`
- Rainier/Crystal (West Slopes South): `west-slopes-south`, `rainier`, `crystal`, `rainier-crystal`, `1648`

If user asks for an unsupported NWAC zone, ask them to pick one of the supported zones or add a mapped area entry first.

## Example Calls
```bash
./pnw-forecast area save Alpental \
  --lat 47.445 --lon -121.425 \
  --nwac-zone-id snoqualmie-pass \
  --station-id alpental-base \
  --wsdot-pass-id snoqualmie
```

```bash
./pnw-forecast --no-cache weather --area Alpental
```

```bash
./pnw-forecast --no-cache avalanche --area Alpental
```

```bash
./pnw-forecast digest --area Alpental
```

## Notes
- Output is JSON only.
- JSON object key order is not guaranteed; treat keys semantically, not by display order.
