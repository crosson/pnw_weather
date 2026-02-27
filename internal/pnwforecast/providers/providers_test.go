package providers

import (
	"encoding/json"
	"fmt"
	"os"
	"path/filepath"
	"testing"
)

type fakeGetter struct {
	JSONMap map[string]json.RawMessage
	TextMap map[string]string
}

func (f *fakeGetter) GetJSON(endpoint string, out any) error {
	raw, ok := f.JSONMap[endpoint]
	if !ok {
		return fmt.Errorf("missing JSON fixture for %s", endpoint)
	}
	return json.Unmarshal(raw, out)
}

func (f *fakeGetter) GetText(endpoint string) (string, error) {
	raw, ok := f.TextMap[endpoint]
	if !ok {
		return "", fmt.Errorf("missing text fixture for %s", endpoint)
	}
	return raw, nil
}

func fixtureJSON(t *testing.T, name string) json.RawMessage {
	t.Helper()
	path := filepath.Join("..", "..", "..", "tests", "fixtures", name)
	raw, err := os.ReadFile(path)
	if err != nil {
		t.Fatalf("read fixture: %v", err)
	}
	return raw
}

func fixtureText(t *testing.T, name string) string {
	t.Helper()
	path := filepath.Join("..", "..", "..", "tests", "fixtures", name)
	raw, err := os.ReadFile(path)
	if err != nil {
		t.Fatalf("read fixture: %v", err)
	}
	return string(raw)
}

func TestNWSForecastNormalization(t *testing.T) {
	f := &fakeGetter{JSONMap: map[string]json.RawMessage{
		"https://api.weather.gov/points/47.445,-121.425":         json.RawMessage(`{"properties":{"forecast":"https://api.weather.gov/gridpoints/SEW/157,74/forecast"}}`),
		"https://api.weather.gov/gridpoints/SEW/157,74/forecast": fixtureJSON(t, "nws_forecast.json"),
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
	f := &fakeGetter{JSONMap: map[string]json.RawMessage{
		"https://api.avalanche.org/v2/public/product?type=forecast&center_id=NWAC&zone_id=1653": fixtureJSON(t, "nwac_forecast.json"),
	}}
	adapter := &NWACAdapter{HTTP: f}
	out, err := adapter.GetAvalancheForecast("SNOQUALMIE_PASS")
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}
	if out["zone_name"] != "Snoqualmie Pass" {
		t.Fatalf("zone = %v", out["zone_name"])
	}
	danger := out["danger_rating"].(nwacDangerRating)
	if danger.AboveTreeline != "Considerable" || danger.NearTreeline != "Considerable" || danger.BelowTreeline != "Moderate" {
		t.Fatalf("danger = %#v", danger)
	}
}

func TestWSDOTPassNormalization(t *testing.T) {
	f := &fakeGetter{
		JSONMap: map[string]json.RawMessage{},
		TextMap: map[string]string{
			"https://wsdot.com/travel/real-time/mountainpasses/snoqualmie": fixtureText(t, "wsdot_pass_page.html"),
		},
	}
	adapter := &WSDOTAdapter{HTTP: f}
	out, err := adapter.GetPassConditions("snoqualmie")
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}
	if out["provider"] != "WSDOT" || out["pass_name"] != "Snoqualmie Pass" {
		t.Fatalf("unexpected output: %#v", out)
	}
}
