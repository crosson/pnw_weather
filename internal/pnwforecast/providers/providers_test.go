package providers

import (
	"encoding/json"
	"fmt"
	"os"
	"path/filepath"
	"strings"
	"testing"
)

type fakeGetter struct {
	JSONMap       map[string]json.RawMessage
	JSONPrefixMap map[string]json.RawMessage
	TextMap       map[string]string
}

func (f *fakeGetter) GetJSON(endpoint string, out any) error {
	raw, ok := f.JSONMap[endpoint]
	if !ok {
		for prefix, candidate := range f.JSONPrefixMap {
			if strings.HasPrefix(endpoint, prefix) {
				raw = candidate
				ok = true
				break
			}
		}
	}
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

func TestNWACTelemetryNormalization(t *testing.T) {
	f := &fakeGetter{JSONPrefixMap: map[string]json.RawMessage{
		"https://api.snowobs.com/wx/v1/station/data/timeseries/?token=71ad26d7aaf410e39efe91bd414d32e1db5d&source=nwac&stid=1&start_date=": json.RawMessage(`{
			"STATION": [{
				"stid":"1",
				"name":"Alpental Base",
				"observations": {
					"date_time":["2026-02-27T17:00:00Z","2026-02-27T18:00:00Z"],
					"air_temp":[30,31],
					"precip_accum_one_hour":[0.2,0.1],
					"snow_depth_24h":[6,7],
					"snow_depth":[90,91],
					"wind_speed":[12,11],
					"wind_gust":[20,25],
					"wind_direction":[250,255]
				}
			}]
		}`),
	}}
	adapter := &NWACAdapter{HTTP: f}
	out, err := adapter.GetTelemetry("alpental-base")
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}
	if out["station_id"] != "alpental-base" {
		t.Fatalf("station_id = %#v", out["station_id"])
	}
	source := out["source"].(map[string]any)
	if source["url"] != "https://nwac.us/weatherdata/alpental/now/" {
		t.Fatalf("source.url = %#v", source["url"])
	}
	if !strings.HasPrefix(source["api_url"].(string), "https://api.snowobs.com/wx/v1/station/data/timeseries/") {
		t.Fatalf("source.api_url = %#v", source["api_url"])
	}
	if out["wind_direction"] != "255" {
		t.Fatalf("wind_direction = %#v", out["wind_direction"])
	}
	if out["hours_observed"] != 2 {
		t.Fatalf("hours_observed = %#v", out["hours_observed"])
	}
	snowfall := out["snowfall_in_24h"].(*float64)
	if *snowfall != 7.0 {
		t.Fatalf("snowfall_in_24h = %#v", out["snowfall_in_24h"])
	}
	precip := *(out["precip_total_in_24h"].(*float64))
	if precip < 0.299 || precip > 0.301 {
		t.Fatalf("precip_total_in_24h = %#v", out["precip_total_in_24h"])
	}
	peakGust := out["peak_wind_gust_mph"].(*float64)
	if *peakGust != 25.0 {
		t.Fatalf("peak_wind_gust_mph = %#v", out["peak_wind_gust_mph"])
	}
}

func TestNWACTelemetryNumericSnowObsStationID(t *testing.T) {
	f := &fakeGetter{JSONPrefixMap: map[string]json.RawMessage{
		"https://api.snowobs.com/wx/v1/station/data/timeseries/?token=71ad26d7aaf410e39efe91bd414d32e1db5d&source=nwac&stid=1&start_date=": json.RawMessage(`{
			"STATION": [{
				"stid":"1",
				"name":"Alpental Base",
				"observations": {
					"date_time":["2026-02-27T17:00:00Z"],
					"air_temp":[31]
				}
			}]
		}`),
	}}
	adapter := &NWACAdapter{HTTP: f}
	out, err := adapter.GetTelemetry("1")
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}
	if out["station_id"] != "1" {
		t.Fatalf("expected station_id 1, got %#v", out["station_id"])
	}
	if out["station_name"] != "Alpental Base" {
		t.Fatalf("station_name = %#v", out["station_name"])
	}
}

func TestNWACTelemetryBakerPageSlugAlias(t *testing.T) {
	f := &fakeGetter{JSONPrefixMap: map[string]json.RawMessage{
		"https://api.snowobs.com/wx/v1/station/data/timeseries/?token=71ad26d7aaf410e39efe91bd414d32e1db5d&source=nwac&stid=5&start_date=": json.RawMessage(`{
			"STATION": [{
				"stid":"5",
				"name":"Mt. Baker - Heather Meadows",
				"observations": {
					"date_time":["2026-02-27T17:00:00Z"],
					"air_temp":[29]
				}
			}]
		}`),
	}}
	adapter := &NWACAdapter{HTTP: f}
	out, err := adapter.GetTelemetry("mtbakerskiarea")
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}
	if out["station_id"] != "mtbakerskiarea" {
		t.Fatalf("station_id = %#v", out["station_id"])
	}
	source := out["source"].(map[string]any)
	if source["url"] != "https://nwac.us/weatherdata/mtbakerskiarea/now/" {
		t.Fatalf("source.url = %#v", source["url"])
	}
}

func TestNWACTelemetrySnoqualmieAndStevensAliases(t *testing.T) {
	f := &fakeGetter{JSONPrefixMap: map[string]json.RawMessage{
		"https://api.snowobs.com/wx/v1/station/data/timeseries/?token=71ad26d7aaf410e39efe91bd414d32e1db5d&source=nwac&stid=21&start_date=": json.RawMessage(`{
			"STATION": [{
				"stid":"21",
				"name":"Snoqualmie Pass",
				"observations": {"date_time":["2026-02-27T17:00:00Z"], "air_temp":[31]}
			}]
		}`),
		"https://api.snowobs.com/wx/v1/station/data/timeseries/?token=71ad26d7aaf410e39efe91bd414d32e1db5d&source=nwac&stid=13&start_date=": json.RawMessage(`{
			"STATION": [{
				"stid":"13",
				"name":"Stevens Pass - Schmidt Haus",
				"observations": {"date_time":["2026-02-27T17:00:00Z"], "air_temp":[30]}
			}]
		}`),
	}}
	adapter := &NWACAdapter{HTTP: f}

	out, err := adapter.GetTelemetry("snoqualmiepass")
	if err != nil {
		t.Fatalf("unexpected error snoqualmiepass: %v", err)
	}
	if out["station_id"] != "snoqualmiepass" {
		t.Fatalf("snoqualmiepass station_id = %#v", out["station_id"])
	}

	out, err = adapter.GetTelemetry("stevenshwy2")
	if err != nil {
		t.Fatalf("unexpected error stevenshwy2: %v", err)
	}
	if out["station_id"] != "stevenshwy2" {
		t.Fatalf("stevenshwy2 station_id = %#v", out["station_id"])
	}
}

func TestNWACTelemetryRainierCrystalAliases(t *testing.T) {
	f := &fakeGetter{JSONPrefixMap: map[string]json.RawMessage{
		"https://api.snowobs.com/wx/v1/station/data/timeseries/?token=71ad26d7aaf410e39efe91bd414d32e1db5d&source=nwac&stid=35&start_date=": json.RawMessage(`{
			"STATION": [{
				"stid":"35",
				"name":"Paradise",
				"observations": {"date_time":["2026-02-27T17:00:00Z"], "air_temp":[28]}
			}]
		}`),
		"https://api.snowobs.com/wx/v1/station/data/timeseries/?token=71ad26d7aaf410e39efe91bd414d32e1db5d&source=nwac&stid=28&start_date=": json.RawMessage(`{
			"STATION": [{
				"stid":"28",
				"name":"Crystal Mt Ski Area",
				"observations": {"date_time":["2026-02-27T17:00:00Z"], "air_temp":[29]}
			}]
		}`),
	}}
	adapter := &NWACAdapter{HTTP: f}

	out, err := adapter.GetTelemetry("paradise")
	if err != nil {
		t.Fatalf("unexpected error paradise: %v", err)
	}
	if out["station_id"] != "paradise" {
		t.Fatalf("paradise station_id = %#v", out["station_id"])
	}

	out, err = adapter.GetTelemetry("crystalskiarea")
	if err != nil {
		t.Fatalf("unexpected error crystalskiarea: %v", err)
	}
	if out["station_id"] != "crystalskiarea" {
		t.Fatalf("crystalskiarea station_id = %#v", out["station_id"])
	}
}
