# OpenClaw Skill: PNW Weather / Avalanche / Pass Conditions

## 1. Overview

This project implements an OpenClaw skill that aggregates Pacific Northwest weather, avalanche, and mountain pass conditions from authoritative public sources.

The agent should call this skill and receive all required information without performing general web searches. 

Authoritative sources:

* National Weather Service (NWS): [https://weather.gov](https://weather.gov) and [https://api.weather.gov](https://api.weather.gov)
* Northwest Avalanche Center (NWAC): [https://nwac.us](https://nwac.us)
* NWAC Telemetry Data: [https://nwac.us/weatherdata/](https://nwac.us/weatherdata/)
* Washington State Department of Transportation (WSDOT): [https://wsdot.wa.gov](https://wsdot.wa.gov)

No additional data sources are permitted.

## 2. Goals

* Provide spot weather forecasts via NWS.
* Provide avalanche forecasts and telemetry from NWAC.
* Provide mountain pass conditions from WSDOT.
* Normalize all provider responses into a stable, consistent clean and concise schema.
* Allow users to define and persist named Areas of Interest.
  - JSON file schema that saves lat/lon for NWS usage, NWAC telemetry station IDs and avalanche zone IDs, and WSDOT pass IDs
* Provide a daily digest combining all configured data for selected areas.
* Implement caching and partial-failure tolerance.
* CLI interface for OpenClaw agent to use.

## 3. Non-Goals

* No UI or frontend.
* No scraping of unofficial sites.
* No third-party commercial weather APIs.
* No user authentication.
* No historical trend analysis (initial version).
* No push notifications (initial version).

## 4. Architecture

High-level flow:

Agent
-> Skill API
-> Provider Adapters
-> weather.gov (NWS)
-> nwac.us
-> wsdot.wa.gov
-> Normalization Layer
-> Cache Layer
-> Area Storage Layer

Design principles:

* Provider adapters must be isolated.
* Normalized models must be provider-agnostic.
* Partial provider failures must not fail the entire request.
* All outputs must include provider name, timestamp, cache age, and a source link.

## 5. Core Concepts

### 5.1 Area of Interest

An Area ties together:

* Coordinates (required for NWS)
* Optional NWAC zone ID
* Optional WSDOT pass ID
* Optional metadata tags

Example schema:

{
"name": "Alpental",
"lat": 47.445,
"lon": -121.425,
"elevation_ft": 3140,
"nwac_zone_id": "SNOQUALMIE_PASS",
"nwac_zone_name": "Snoqualmie Pass",
"nwac_telemetry_station_ids": ["alpental-base"],
"wsdot_pass_id": "snoqualmie",
"wsdot_pass_name": "Snoqualmie Pass",
"tags": ["ski"]
}

Coordinates are mandatory. All other fields are optional.

ID fields are canonical for storage and lookup. Name fields are optional display helpers.

## 6. Public Skill API

### 6.0 Common Response Metadata

All normalized provider responses must include:

{
"provider": "NWS|NWAC|WSDOT",
"issued_at": "ISO-8601",
"cache_age_seconds": 0,
"is_stale": false,
"source": {
"provider": "NWS|NWAC|WSDOT",
"url": "https://human-readable-page",
"api_url": "https://api-endpoint-used",
"retrieved_at": "ISO-8601"
}
}

Rules:

* `source.url` is a user-facing link the agent can share in chat.
* `source.api_url` is optional when the provider has no stable API URL for that record.
* `is_stale=true` when data age exceeds freshness threshold for that data type.

### 6.1 Area Management

save_area(name, definition)
Creates or updates an area.

list_areas()
Returns all saved areas.

delete_area(name)
Deletes a saved area.

### 6.2 Weather

get_spot_forecast(area_name | {lat, lon, elevation?}, datetime?)

Returns normalized NWS forecast.

Normalized response example:

{
"provider": "NWS",
"issued_at": "ISO-8601",
"cache_age_seconds": 32,
"is_stale": false,
"source": {
"provider": "NWS",
"url": "https://forecast.weather.gov/MapClick.php?lat=47.445&lon=-121.425",
"api_url": "https://api.weather.gov/gridpoints/SEW/157,74/forecast",
"retrieved_at": "ISO-8601"
},
"periods": [
{
"name": "Tonight",
"start": "ISO-8601",
"end": "ISO-8601",
"temperature_f": 28,
"wind": "NW 10 mph",
"precip_probability_percent": 60,
"short_forecast": "Snow showers likely"
}
]
}

### 6.3 Avalanche

get_avalanche_forecast(area_name | nwac_zone_id)

Normalized response example:

{
"provider": "NWAC",
"issued_at": "ISO-8601",
"cache_age_seconds": 104,
"is_stale": false,
"source": {
"provider": "NWAC",
"url": "https://nwac.us/avalanche-forecast/#/snoqualmie-pass",
"api_url": "https://nwac.us/api/v6/forecast/zone/SNOQUALMIE_PASS",
"retrieved_at": "ISO-8601"
},
"zone_id": "SNOQUALMIE_PASS",
"zone_name": "Snoqualmie Pass",
"danger_rating": {
"below_treeline": "Moderate",
"near_treeline": "Considerable",
"above_treeline": "High"
},
"primary_problems": [
"Wind Slabs",
"Persistent Slabs"
],
"travel_advice_summary": "Avoid steep wind-loaded slopes."
}

### 6.4 Telemetry

get_telemetry(area_name | station_id | nwac_zone_id)

Normalized response example:

{
"provider": "NWAC",
"issued_at": "ISO-8601",
"cache_age_seconds": 75,
"is_stale": false,
"source": {
"provider": "NWAC",
"url": "https://nwac.us/weatherdata/alpental-base/",
"api_url": "https://nwac.us/api/v6/station/alpental-base",
"retrieved_at": "ISO-8601"
},
"station_id": "alpental-base",
"station_name": "Alpental Base",
"timestamp": "ISO-8601",
"temperature_f": 30,
"snow_depth_in": 78,
"wind_speed_mph": 15,
"wind_direction": "NW"
}

### 6.5 Pass Conditions

get_pass_conditions(area_name | wsdot_pass_id | pass_name)

Normalized response example:

{
"provider": "WSDOT",
"issued_at": "ISO-8601",
"cache_age_seconds": 58,
"is_stale": false,
"source": {
"provider": "WSDOT",
"url": "https://wsdot.com/travel/real-time/mountainpasses/snoqualmie",
"api_url": "https://wsdot.wa.gov/Traffic/api/mountainpasses/mountainpassconditions",
"retrieved_at": "ISO-8601"
},
"pass_id": "snoqualmie",
"pass_name": "Snoqualmie Pass",
"status": "Open",
"restrictions": "Chains required for some vehicles",
"weather_conditions": "Snowing",
"last_updated": "ISO-8601"
}

### 6.6 Daily Digest

get_daily_digest(date?, areas? = all_saved)

Returns, per area:

* Spot forecast
* Avalanche forecast (if configured)
* Telemetry summary (if configured)
* Pass conditions (if configured)

Digest response example:

{
"generated_at": "ISO-8601",
"areas": [
{
"name": "Alpental",
"weather": { "data": { ... }, "error": null },
"avalanche": { "data": { ... }, "error": null },
"telemetry": { "data": { ... }, "error": null },
"pass_conditions": { "data": { ... }, "error": null }
}
]
}

## 7. Caching

* Default TTL: 15 minutes (configurable).
* Cache key must include provider, endpoint, and request parameters.
* Cache must persist between process restarts.
* Include cache age in response metadata.
* Freshness thresholds (for `is_stale`):
  * NWS forecast: 90 minutes
  * NWAC avalanche forecast: 6 hours
  * NWAC telemetry: 30 minutes
  * WSDOT pass conditions: 15 minutes

## 8. Resilience and Error Handling

* If one provider fails, return partial results.
* Include structured error field per provider.
* Only raise a fatal error if all providers fail.
* During the summer some stations/services will not load data or have partial data, we should account for this.
* Null data is valid when upstream marks a metric unavailable (for example, `snow_depth_in: null`).
* Distinguish "no data" from hard failure in the error model.

Example error structure:

{
"provider": "NWAC",
"error": {
"type": "UpstreamUnavailable",
"message": "Timeout contacting nwac.us"
}
}

Error type enum (initial):

* `UpstreamUnavailable`
* `Timeout`
* `NotFound`
* `NoDataSeasonal`
* `RateLimited`
* `ParseError`
* `Unknown`

Domain enum guidance (initial):

* Avalanche danger ratings: `Low`, `Moderate`, `Considerable`, `High`, `Extreme`, `NoRating`
* Pass status: `Open`, `Closed`, `Restricted`, `Unknown`

## 9. Time and Locale

* Default timezone: America/Los_Angeles.
* All timestamps in ISO-8601.
* Forecast presentation respects Pacific Time.

## 10. Storage

* Areas stored locally.
* Default path: ~/.openclaw/skills/pnw_forecast/areas.json
* Must include version field for future migrations.

Example storage format:

{
"version": 1,
"areas": [ ... ]
}

## 11. Security Constraints

* Only allow outbound requests to:

  * weather.gov
  * api.weather.gov
  * nwac.us
  * wsdot.wa.gov
* No dynamic domain resolution.
* No arbitrary URL fetching.

## 12. Testing Requirements

* Unit tests for each provider adapter using fixtures.
* Tests for normalization logic.
* Tests for digest assembly.
* Include recorded fixture responses for:

  * One NWS spot forecast
  * One NWAC forecast
  * One WSDOT pass condition

## 13. Future Extensions (Out of Scope for v1)

* Alerting system
* SMS or push notifications
* Historical trend comparison
* Multi-state support
* Additional avalanche centers
