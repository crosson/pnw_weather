package providers

import (
	"encoding/json"
	"os"
	"path/filepath"
	"testing"
)

type fakeGetter struct {
	Map map[string]json.RawMessage
}

func (f *fakeGetter) GetJSON(endpoint string, out any) error {
	raw := f.Map[endpoint]
	return json.Unmarshal(raw, out)
}

func fixture(t *testing.T, name string) json.RawMessage {
	t.Helper()
	path := filepath.Join("..", "..", "..", "tests", "fixtures", name)
	raw, err := os.ReadFile(path)
	if err != nil {
		t.Fatalf("read fixture: %v", err)
	}
	return raw
}

func TestNWSForecastNormalization(t *testing.T) {
	f := &fakeGetter{Map: map[string]json.RawMessage{
		"https://api.weather.gov/points/47.445,-121.425":         json.RawMessage(`{"properties":{"forecast":"https://api.weather.gov/gridpoints/SEW/157,74/forecast"}}`),
		"https://api.weather.gov/gridpoints/SEW/157,74/forecast": fixture(t, "nws_forecast.json"),
	}}
	adapter := &NWSAdapter{HTTP: f}
	out, err := adapter.GetSpotForecast(47.445, -121.425)
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}
	if out["provider"] != "NWS" {
		t.Fatalf("provider = %v", out["provider"])
	}
	periods := out["periods"].([]map[string]any)
	if len(periods) == 0 || periods[0]["name"] != "Tonight" {
		t.Fatalf("unexpected periods: %#v", periods)
	}
}

func TestNWACForecastNormalization(t *testing.T) {
	f := &fakeGetter{Map: map[string]json.RawMessage{
		"https://nwac.us/api/v6/forecast/zone/SNOQUALMIE_PASS": fixture(t, "nwac_forecast.json"),
	}}
	adapter := &NWACAdapter{HTTP: f}
	out, err := adapter.GetAvalancheForecast("SNOQUALMIE_PASS")
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}
	if out["zone_name"] != "Snoqualmie Pass" {
		t.Fatalf("zone = %v", out["zone_name"])
	}
	danger := out["danger_rating"].(map[string]string)
	if danger["above_treeline"] != "High" {
		t.Fatalf("danger = %#v", danger)
	}
}

func TestWSDOTPassNormalization(t *testing.T) {
	f := &fakeGetter{Map: map[string]json.RawMessage{
		WSDOTAPIURL: fixture(t, "wsdot_passes.json"),
	}}
	adapter := &WSDOTAdapter{HTTP: f}
	out, err := adapter.GetPassConditions("snoqualmie")
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}
	if out["provider"] != "WSDOT" || out["pass_name"] != "Snoqualmie Pass" || out["status"] != "Open" {
		t.Fatalf("unexpected output: %#v", out)
	}
}
