package pnwforecast

import (
	"encoding/json"
	"os"
	"path/filepath"
	"testing"

	"github.com/crosson/pnw_weather/internal/pnwforecast/providers"
)

type fakeGetter struct {
	Map map[string]json.RawMessage
}

func (f *fakeGetter) GetJSON(endpoint string, out any) error {
	raw, ok := f.Map[endpoint]
	if !ok {
		return NewSkillError("UpstreamUnavailable", "missing fixture for "+endpoint)
	}
	return json.Unmarshal(raw, out)
}

func TestDailyDigestPartialFailure(t *testing.T) {
	tmp := t.TempDir()
	paths := SkillPaths{
		Root:      tmp,
		AreasPath: filepath.Join(tmp, "areas.json"),
		CachePath: filepath.Join(tmp, "cache.json"),
	}
	svc, err := NewService(paths, 900)
	if err != nil {
		t.Fatalf("new service: %v", err)
	}

	wsdotFixture, err := os.ReadFile(filepath.Join("..", "..", "tests", "fixtures", "wsdot_passes.json"))
	if err != nil {
		t.Fatalf("fixture: %v", err)
	}
	nwsFixture, err := os.ReadFile(filepath.Join("..", "..", "tests", "fixtures", "nws_forecast.json"))
	if err != nil {
		t.Fatalf("fixture: %v", err)
	}

	getter := &fakeGetter{Map: map[string]json.RawMessage{
		"https://api.weather.gov/points/47.445,-121.425":         json.RawMessage(`{"properties":{"forecast":"https://api.weather.gov/gridpoints/SEW/157,74/forecast"}}`),
		"https://api.weather.gov/gridpoints/SEW/157,74/forecast": json.RawMessage(nwsFixture),
		providers.WSDOTAPIURL:                                    json.RawMessage(wsdotFixture),
		"https://nwac.us/api/v6/forecast/zone/SNOQUALMIE_PASS":   json.RawMessage(`{"issued_at":"2026-02-26T20:00:00Z","zone_name":"Snoqualmie Pass","danger_rating":{"below_treeline":"Moderate","near_treeline":"Considerable","above_treeline":"High"}}`),
		// telemetry intentionally missing to force partial failure
	}}

	svc.NWS = &providers.NWSAdapter{HTTP: getter}
	svc.NWAC = &providers.NWACAdapter{HTTP: getter}
	svc.WSDOT = &providers.WSDOTAdapter{HTTP: getter}

	_, err = svc.SaveArea("Alpental", Area{
		Lat:                     47.445,
		Lon:                     -121.425,
		NWACZoneID:              "SNOQUALMIE_PASS",
		NWACTelemetryStationIDs: []string{"alpental-base"},
		WSDOTPassID:             "snoqualmie",
	})
	if err != nil {
		t.Fatalf("save area: %v", err)
	}

	digest, err := svc.GetDailyDigest("", nil)
	if err != nil {
		t.Fatalf("digest: %v", err)
	}

	areas := digest["areas"].([]map[string]any)
	first := areas[0]
	tele := first["telemetry"].(map[string]any)
	if tele["data"] != nil {
		t.Fatalf("expected telemetry data nil")
	}
	errPayload := tele["error"].(ErrorPayload)
	if errPayload.Type != "UpstreamUnavailable" {
		t.Fatalf("expected UpstreamUnavailable, got %s", errPayload.Type)
	}
}
