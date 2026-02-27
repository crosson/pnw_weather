package pnwforecast

import (
	"encoding/json"
	"fmt"
	"strings"
	"time"

	"github.com/crosson/pnw_weather/internal/pnwforecast/providers"
)

type Service struct {
	Storage    *AreaStorage
	Cache      *JSONCache
	Thresholds FreshnessThresholds
	NWS        *providers.NWSAdapter
	NWAC       *providers.NWACAdapter
	WSDOT      *providers.WSDOTAdapter
}

func NewService(paths SkillPaths, cacheTTLSeconds int) (*Service, error) {
	storage, err := NewAreaStorage(paths.AreasPath)
	if err != nil {
		return nil, err
	}
	cache, err := NewJSONCache(paths.CachePath, cacheTTLSeconds)
	if err != nil {
		return nil, err
	}
	http := NewSafeHTTP()
	return &Service{
		Storage:    storage,
		Cache:      cache,
		Thresholds: DefaultFreshnessThresholds(),
		NWS:        &providers.NWSAdapter{HTTP: http},
		NWAC:       &providers.NWACAdapter{HTTP: http},
		WSDOT:      &providers.WSDOTAdapter{HTTP: http},
	}, nil
}

func (s *Service) SaveArea(name string, definition Area) (Area, error) {
	definition.Name = name
	if definition.Lat == 0 && definition.Lon == 0 {
		return Area{}, NewSkillError("NotFound", "lat and lon are required")
	}
	if err := s.Storage.SaveArea(definition); err != nil {
		return Area{}, err
	}
	return definition, nil
}

func (s *Service) ListAreas() ([]Area, error) {
	return s.Storage.ListAreas()
}

func (s *Service) DeleteArea(name string) (bool, error) {
	return s.Storage.DeleteArea(name)
}

func (s *Service) withCache(provider, endpoint string, params any, staleAfter int, fetch func() (any, error)) (map[string]any, error) {
	key, err := s.Cache.MakeKey(provider, endpoint, params)
	if err != nil {
		return nil, err
	}
	if cachedRaw, age, ok, err := s.Cache.Get(key); err != nil {
		return nil, err
	} else if ok {
		payload := map[string]any{}
		if err := json.Unmarshal(cachedRaw, &payload); err == nil {
			payload["cache_age_seconds"] = age
			issuedAt, _ := payload["issued_at"].(string)
			payload["is_stale"] = isOlderThan(issuedAt, staleAfter)
			return payload, nil
		}
	}

	value, err := fetch()
	if err != nil {
		return nil, err
	}

	raw, err := json.Marshal(value)
	if err != nil {
		return nil, err
	}
	payload := map[string]any{}
	if err := json.Unmarshal(raw, &payload); err != nil {
		return nil, err
	}
	issuedAt, _ := payload["issued_at"].(string)
	payload["cache_age_seconds"] = 0
	payload["is_stale"] = isOlderThan(issuedAt, staleAfter)

	if err := s.Cache.Put(key, payload); err != nil {
		return nil, err
	}
	return payload, nil
}

func (s *Service) resolveArea(areaName string, lat, lon float64, elevation *int) (Area, error) {
	if strings.TrimSpace(areaName) != "" {
		area, err := s.Storage.GetArea(areaName)
		if err != nil {
			return Area{}, err
		}
		if area == nil {
			return Area{}, NewSkillError("NotFound", fmt.Sprintf("Unknown area: %s", areaName))
		}
		return *area, nil
	}
	if lat == 0 && lon == 0 {
		return Area{}, NewSkillError("NotFound", "Either area_name or lat/lon is required")
	}
	return Area{Name: "ad-hoc", Lat: lat, Lon: lon, ElevationFt: elevation}, nil
}

func (s *Service) GetSpotForecast(areaName string, lat, lon float64, elevation *int, datetime string) (map[string]any, error) {
	area, err := s.resolveArea(areaName, lat, lon, elevation)
	if err != nil {
		return nil, err
	}
	params := map[string]any{"lat": area.Lat, "lon": area.Lon, "datetime": datetime}
	full, err := s.withCache("NWS", "spot_forecast", params, s.Thresholds.NWSForecastSeconds, func() (any, error) {
		return s.NWS.GetSpotForecast(area.Lat, area.Lon)
	})
	if err != nil {
		return nil, err
	}
	return withPeriodLimit(full, 2), nil
}

func (s *Service) GetSpotForecast7Day(areaName string, lat, lon float64, elevation *int, datetime string) (map[string]any, error) {
	area, err := s.resolveArea(areaName, lat, lon, elevation)
	if err != nil {
		return nil, err
	}
	params := map[string]any{"lat": area.Lat, "lon": area.Lon, "datetime": datetime}
	full, err := s.withCache("NWS", "spot_forecast", params, s.Thresholds.NWSForecastSeconds, func() (any, error) {
		return s.NWS.GetSpotForecast(area.Lat, area.Lon)
	})
	if err != nil {
		return nil, err
	}
	return withPeriodLimit(full, 14), nil
}

func (s *Service) GetAvalancheForecast(areaName, zoneID string) (map[string]any, error) {
	effectiveZoneID := zoneID
	if effectiveZoneID == "" && areaName != "" {
		area, err := s.Storage.GetArea(areaName)
		if err != nil {
			return nil, err
		}
		if area == nil {
			return nil, NewSkillError("NotFound", fmt.Sprintf("Unknown area: %s", areaName))
		}
		effectiveZoneID = area.NWACZoneID
	}
	if effectiveZoneID == "" {
		return nil, NewSkillError("NotFound", "Missing NWAC zone ID")
	}

	return s.withCache("NWAC", "avalanche_forecast", map[string]any{"zone_id": effectiveZoneID}, s.Thresholds.NWACAvalancheSeconds, func() (any, error) {
		return s.NWAC.GetAvalancheForecast(effectiveZoneID)
	})
}

func (s *Service) GetTelemetry(areaName, stationID, zoneID string) (map[string]any, error) {
	effective := stationID
	if effective == "" && areaName != "" {
		area, err := s.Storage.GetArea(areaName)
		if err != nil {
			return nil, err
		}
		if area == nil {
			return nil, NewSkillError("NotFound", fmt.Sprintf("Unknown area: %s", areaName))
		}
		if len(area.NWACTelemetryStationIDs) > 0 {
			effective = area.NWACTelemetryStationIDs[0]
		} else if zoneID != "" {
			effective = strings.ToLower(strings.ReplaceAll(zoneID, "_", "-"))
		}
	}
	if effective == "" && zoneID != "" {
		effective = strings.ToLower(strings.ReplaceAll(zoneID, "_", "-"))
	}
	if effective == "" {
		return nil, NewSkillError("NotFound", "Missing telemetry station ID")
	}

	return s.withCache("NWAC", "telemetry", map[string]any{"station_id": effective}, s.Thresholds.NWACTelemetrySeconds, func() (any, error) {
		return s.NWAC.GetTelemetry(effective)
	})
}

func (s *Service) GetPassConditions(areaName, passID, passName string) (map[string]any, error) {
	lookup := passID
	if lookup == "" {
		lookup = passName
	}
	if lookup == "" && areaName != "" {
		area, err := s.Storage.GetArea(areaName)
		if err != nil {
			return nil, err
		}
		if area == nil {
			return nil, NewSkillError("NotFound", fmt.Sprintf("Unknown area: %s", areaName))
		}
		if area.WSDOTPassID != "" {
			lookup = area.WSDOTPassID
		} else {
			lookup = area.WSDOTPassName
		}
	}
	if lookup == "" {
		return nil, NewSkillError("NotFound", "Missing WSDOT pass ID or pass name")
	}
	return s.withCache("WSDOT", "pass_conditions", map[string]any{"pass_lookup": lookup}, s.Thresholds.WSDOTPassSeconds, func() (any, error) {
		return s.WSDOT.GetPassConditions(lookup)
	})
}

func (s *Service) GetDailyDigest(date string, areas []string) (map[string]any, error) {
	selected, err := s.Storage.ListAreas()
	if err != nil {
		return nil, err
	}
	if len(areas) > 0 {
		wanted := map[string]struct{}{}
		for _, a := range areas {
			wanted[strings.ToLower(a)] = struct{}{}
		}
		filtered := make([]Area, 0, len(areas))
		for _, a := range selected {
			if _, ok := wanted[strings.ToLower(a.Name)]; ok {
				filtered = append(filtered, a)
			}
		}
		selected = filtered
	}

	out := make([]map[string]any, 0, len(selected))
	for _, area := range selected {
		item := map[string]any{
			"name":            area.Name,
			"weather":         envelope(func() (map[string]any, error) { return s.GetSpotForecast(area.Name, 0, 0, nil, date) }),
			"avalanche":       envelope(func() (map[string]any, error) { return s.GetAvalancheForecast(area.Name, "") }),
			"telemetry":       envelope(func() (map[string]any, error) { return s.GetTelemetry(area.Name, "", "") }),
			"pass_conditions": envelope(func() (map[string]any, error) { return s.GetPassConditions(area.Name, "", "") }),
		}
		out = append(out, item)
	}
	return map[string]any{
		"generated_at": ISOTimeNow(),
		"areas":        out,
	}, nil
}

func envelope(call func() (map[string]any, error)) map[string]any {
	data, err := call()
	if err == nil {
		return map[string]any{"data": data, "error": nil}
	}
	if se, ok := err.(*SkillError); ok {
		return map[string]any{"data": nil, "error": se.Payload()}
	}
	return map[string]any{"data": nil, "error": ErrorPayload{Type: "Unknown", Message: err.Error()}}
}

func isOlderThan(iso string, seconds int) bool {
	if iso == "" {
		return false
	}
	t, err := time.Parse(time.RFC3339, iso)
	if err != nil {
		return false
	}
	return int(time.Since(t).Seconds()) > seconds
}

func withPeriodLimit(payload map[string]any, max int) map[string]any {
	periodAny, ok := payload["periods"]
	if !ok {
		return payload
	}
	periods, ok := periodAny.([]any)
	if !ok {
		return payload
	}
	if len(periods) <= max {
		return payload
	}
	limited := periods[:max]
	out := make(map[string]any, len(payload))
	for k, v := range payload {
		out[k] = v
	}
	out["periods"] = limited
	return out
}
