package providers

import (
	"fmt"
	"strconv"
	"strings"
	"time"
)

const (
	nwacCenterID          = "NWAC"
	nwacSnowObsToken      = "71ad26d7aaf410e39efe91bd414d32e1db5d"
	nwacSnowObsTimeseries = "https://api.snowobs.com/wx/v1/station/data/timeseries/"
)

var nwacZoneMap = map[string]nwacZoneConfig{
	"snoqualmie-pass": {ID: 1653, Name: "Snoqualmie Pass", Slug: "snoqualmie-pass"},
	"snoqualmie":      {ID: 1653, Name: "Snoqualmie Pass", Slug: "snoqualmie-pass"},
	"1653":            {ID: 1653, Name: "Snoqualmie Pass", Slug: "snoqualmie-pass"},

	"stevens-pass": {ID: 1649, Name: "Stevens Pass", Slug: "stevens-pass"},
	"stevens":      {ID: 1649, Name: "Stevens Pass", Slug: "stevens-pass"},
	"1649":         {ID: 1649, Name: "Stevens Pass", Slug: "stevens-pass"},

	"west-slopes-north": {ID: 1646, Name: "West Slopes North", Slug: "west-slopes-north"},
	"baker":             {ID: 1646, Name: "West Slopes North", Slug: "west-slopes-north"},
	"mt-baker":          {ID: 1646, Name: "West Slopes North", Slug: "west-slopes-north"},
	"1646":              {ID: 1646, Name: "West Slopes North", Slug: "west-slopes-north"},

	"west-slopes-south": {ID: 1648, Name: "West Slopes South", Slug: "west-slopes-south"},
	"rainier":           {ID: 1648, Name: "West Slopes South", Slug: "west-slopes-south"},
	"crystal":           {ID: 1648, Name: "West Slopes South", Slug: "west-slopes-south"},
	"rainier-crystal":   {ID: 1648, Name: "West Slopes South", Slug: "west-slopes-south"},
	"1648":              {ID: 1648, Name: "West Slopes South", Slug: "west-slopes-south"},
}

type nwacZoneConfig struct {
	ID   int
	Name string
	Slug string
}

type nwacDangerRating struct {
	AboveTreeline string `json:"above_treeline"`
	NearTreeline  string `json:"near_treeline"`
	BelowTreeline string `json:"below_treeline"`
}

type NWACAdapter struct {
	HTTP interface {
		GetJSON(endpoint string, out any) error
	}
}

type nwacForecastResponse struct {
	ID               int    `json:"id"`
	PublishedTime    string `json:"published_time"`
	ExpiresTime      string `json:"expires_time"`
	BottomLine       string `json:"bottom_line"`
	HazardDiscussion string `json:"hazard_discussion"`
	Danger           []struct {
		Lower    int    `json:"lower"`
		Middle   int    `json:"middle"`
		Upper    int    `json:"upper"`
		ValidDay string `json:"valid_day"`
	} `json:"danger"`
	ForecastAvalancheProblem []struct {
		Name        string `json:"name"`
		ProblemType struct {
			Name string `json:"name"`
		} `json:"problem_type"`
	} `json:"forecast_avalanche_problems"`
	ForecastZone []struct {
		ID   int    `json:"id"`
		Name string `json:"name"`
		URL  string `json:"url"`
	} `json:"forecast_zone"`
}

type nwacTelemetryConfig struct {
	StationID string
	PageSlug  string
	SnowObsID string
}

var nwacTelemetryStations = map[string]nwacTelemetryConfig{
	"alpental":         {StationID: "alpental", PageSlug: "alpental", SnowObsID: "1"},
	"alpental-base":    {StationID: "alpental-base", PageSlug: "alpental", SnowObsID: "1"},
	"alpental-mid":     {StationID: "alpental-mid", PageSlug: "alpental", SnowObsID: "2"},
	"alpental-middle":  {StationID: "alpental-mid", PageSlug: "alpental", SnowObsID: "2"},
	"alpental-summit":  {StationID: "alpental-summit", PageSlug: "alpental", SnowObsID: "3"},
	"snoqualmie-pass":  {StationID: "snoqualmie-pass", PageSlug: "snoqualmiepass", SnowObsID: "21"},
	"snoqualmie":       {StationID: "snoqualmie-pass", PageSlug: "snoqualmiepass", SnowObsID: "21"},
	"snoqualmiepass":   {StationID: "snoqualmiepass", PageSlug: "snoqualmiepass", SnowObsID: "21"},
	"stevens-pass":     {StationID: "stevens-pass", PageSlug: "stevenshwy2", SnowObsID: "13"},
	"stevens":          {StationID: "stevens-pass", PageSlug: "stevenshwy2", SnowObsID: "13"},
	"stevenshwy2":      {StationID: "stevenshwy2", PageSlug: "stevenshwy2", SnowObsID: "13"},
	"stevensskiarea":   {StationID: "stevensskiarea", PageSlug: "stevensskiarea", SnowObsID: "18"},
	"brookssnow":       {StationID: "brookssnow", PageSlug: "brookssnow", SnowObsID: "50"},
	"gracelakes":       {StationID: "gracelakes", PageSlug: "gracelakes", SnowObsID: "14"},
	"mt-baker":         {StationID: "mt-baker", PageSlug: "mtbakerskiarea", SnowObsID: "5"},
	"baker":            {StationID: "mt-baker", PageSlug: "mtbakerskiarea", SnowObsID: "5"},
	"mtbakerskiarea":   {StationID: "mtbakerskiarea", PageSlug: "mtbakerskiarea", SnowObsID: "5"},
	"rainier":          {StationID: "rainier", PageSlug: "paradise", SnowObsID: "35"},
	"crystal":          {StationID: "crystal", PageSlug: "crystalskiarea", SnowObsID: "28"},
	"rainier-crystal":  {StationID: "rainier-crystal", PageSlug: "crystalskiarea", SnowObsID: "28"},
	"paradise":         {StationID: "paradise", PageSlug: "paradise", SnowObsID: "35"},
	"campmuir":         {StationID: "campmuir", PageSlug: "campmuir", SnowObsID: "34"},
	"sunrise":          {StationID: "sunrise", PageSlug: "sunrise", SnowObsID: "30"},
	"crystalskiarea":   {StationID: "crystalskiarea", PageSlug: "crystalskiarea", SnowObsID: "28"},
	"crystalgrnvalley": {StationID: "crystalgrnvalley", PageSlug: "crystalgrnvalley", SnowObsID: "27"},
}

func (a *NWACAdapter) GetAvalancheForecast(zoneID string) (map[string]any, error) {
	cfg, ok := resolveNWACZone(zoneID)
	if !ok {
		return nil, fmt.Errorf("NotFound: unsupported NWAC zone '%s' (supported: snoqualmie, stevens, baker, rainier/crystal)", zoneID)
	}

	apiURL := fmt.Sprintf(
		"https://api.avalanche.org/v2/public/product?type=forecast&center_id=%s&zone_id=%d",
		nwacCenterID,
		cfg.ID,
	)

	var resp nwacForecastResponse
	if err := a.HTTP.GetJSON(apiURL, &resp); err != nil {
		return nil, err
	}

	current := selectDanger(resp.Danger)
	below := canonicalDangerFromInt(current.Lower)
	near := canonicalDangerFromInt(current.Middle)
	above := canonicalDangerFromInt(current.Upper)

	if below == "NoRating" && near == "NoRating" && above == "NoRating" {
		return nil, fmt.Errorf("ParseError: NWAC API missing danger ratings for zone_id %d", cfg.ID)
	}

	zoneName := cfg.Name
	sourceURL := fmt.Sprintf("https://nwac.us/avalanche-forecast/#/%s/", cfg.Slug)
	if len(resp.ForecastZone) > 0 {
		if strings.TrimSpace(resp.ForecastZone[0].Name) != "" {
			zoneName = resp.ForecastZone[0].Name
		}
		if strings.TrimSpace(resp.ForecastZone[0].URL) != "" {
			sourceURL = strings.ReplaceAll(resp.ForecastZone[0].URL, "http://", "https://")
		}
	}

	travel := cleanHTML(firstNonEmpty(resp.BottomLine, resp.HazardDiscussion))
	problems := normalizeProblems(resp.ForecastAvalancheProblem)
	issued := firstNonEmpty(resp.PublishedTime, time.Now().UTC().Format(time.RFC3339))

	return map[string]any{
		"provider":  "NWAC",
		"issued_at": issued,
		"source": map[string]any{
			"provider":     "NWAC",
			"url":          sourceURL,
			"api_url":      apiURL,
			"retrieved_at": time.Now().UTC().Format(time.RFC3339),
		},
		"zone_id":   cfg.Slug,
		"zone_name": zoneName,
		"danger_rating": nwacDangerRating{
			AboveTreeline: above,
			NearTreeline:  near,
			BelowTreeline: below,
		},
		"primary_problems":      problems,
		"travel_advice_summary": travel,
	}, nil
}

func resolveNWACZone(v string) (nwacZoneConfig, bool) {
	key := normalizeZoneKey(v)
	cfg, ok := nwacZoneMap[key]
	return cfg, ok
}

func normalizeZoneKey(v string) string {
	s := strings.ToLower(strings.TrimSpace(v))
	s = strings.ReplaceAll(s, "_", "-")
	s = strings.ReplaceAll(s, " ", "-")
	for strings.Contains(s, "--") {
		s = strings.ReplaceAll(s, "--", "-")
	}
	return s
}

func selectDanger(d []struct {
	Lower    int    `json:"lower"`
	Middle   int    `json:"middle"`
	Upper    int    `json:"upper"`
	ValidDay string `json:"valid_day"`
}) struct {
	Lower    int
	Middle   int
	Upper    int
	ValidDay string
} {
	for _, item := range d {
		if strings.EqualFold(item.ValidDay, "current") {
			return struct {
				Lower    int
				Middle   int
				Upper    int
				ValidDay string
			}{item.Lower, item.Middle, item.Upper, item.ValidDay}
		}
	}
	if len(d) > 0 {
		item := d[0]
		return struct {
			Lower    int
			Middle   int
			Upper    int
			ValidDay string
		}{item.Lower, item.Middle, item.Upper, item.ValidDay}
	}
	return struct {
		Lower    int
		Middle   int
		Upper    int
		ValidDay string
	}{}
}

func canonicalDangerFromInt(v int) string {
	switch v {
	case 1:
		return "Low"
	case 2:
		return "Moderate"
	case 3:
		return "Considerable"
	case 4:
		return "High"
	case 5:
		return "Extreme"
	default:
		return "NoRating"
	}
}

func normalizeProblems(items []struct {
	Name        string `json:"name"`
	ProblemType struct {
		Name string `json:"name"`
	} `json:"problem_type"`
}) []string {
	out := make([]string, 0, len(items))
	seen := map[string]struct{}{}
	for _, p := range items {
		name := strings.TrimSpace(firstNonEmpty(p.Name, p.ProblemType.Name))
		if name == "" {
			continue
		}
		if _, ok := seen[name]; ok {
			continue
		}
		seen[name] = struct{}{}
		out = append(out, name)
	}
	return out
}

func cleanHTML(v string) string {
	return normalizeHTMLText(v)
}

func (a *NWACAdapter) GetTelemetry(stationID string) (map[string]any, error) {
	cfg, err := resolveTelemetryConfig(stationID)
	if err != nil {
		return nil, err
	}

	end := time.Now().UTC()
	start := end.Add(-24 * time.Hour)
	apiURL := fmt.Sprintf(
		"%s?token=%s&source=nwac&stid=%s&start_date=%s&end_date=%s",
		nwacSnowObsTimeseries,
		nwacSnowObsToken,
		cfg.SnowObsID,
		start.Format("200601021504"),
		end.Format("200601021504"),
	)

	var resp struct {
		Station []struct {
			StationID    string `json:"stid"`
			Name         string `json:"name"`
			Observations struct {
				DateTime           []string `json:"date_time"`
				AirTemp            []any    `json:"air_temp"`
				PrecipAccumOneHour []any    `json:"precip_accum_one_hour"`
				SnowDepth24h       []any    `json:"snow_depth_24h"`
				SnowDepth          []any    `json:"snow_depth"`
				WindSpeed          []any    `json:"wind_speed"`
				WindGust           []any    `json:"wind_gust"`
				WindDirection      []any    `json:"wind_direction"`
			} `json:"observations"`
		} `json:"STATION"`
	}
	if err := a.HTTP.GetJSON(apiURL, &resp); err != nil {
		return nil, err
	}
	if len(resp.Station) == 0 {
		return nil, fmt.Errorf("ParseError: SnowObs API missing station data for stid %s", cfg.SnowObsID)
	}
	station := resp.Station[0]
	ts := lastNonEmptyString(station.Observations.DateTime)
	if ts == "" {
		ts = end.Format(time.RFC3339)
	}
	temp := lastFloat(station.Observations.AirTemp)
	snowDepth := lastFloat(station.Observations.SnowDepth)
	snow24h := lastFloat(station.Observations.SnowDepth24h)
	precip24h := sumFloats(station.Observations.PrecipAccumOneHour)
	wind := lastFloat(station.Observations.WindSpeed)
	peakGust := maxFloat(station.Observations.WindGust)
	windDirection := lastString(station.Observations.WindDirection)
	hoursObserved := len(station.Observations.DateTime)

	return map[string]any{
		"provider":  "NWAC",
		"issued_at": ts,
		"source": map[string]any{
			"provider":     "NWAC",
			"url":          fmt.Sprintf("https://nwac.us/weatherdata/%s/now/", cfg.PageSlug),
			"api_url":      apiURL,
			"retrieved_at": time.Now().UTC().Format(time.RFC3339),
		},
		"station_id":          cfg.StationID,
		"station_name":        firstNonEmpty(strings.TrimSpace(station.Name), cfg.StationID),
		"timestamp":           ts,
		"hours_observed":      hoursObserved,
		"temperature_f":       temp,
		"snow_depth_in":       snowDepth,
		"snowfall_in_24h":     snow24h,
		"precip_total_in_24h": precip24h,
		"wind_speed_mph":      wind,
		"peak_wind_gust_mph":  peakGust,
		"wind_direction":      windDirection,
	}, nil
}

func resolveTelemetryConfig(stationID string) (nwacTelemetryConfig, error) {
	base := normalizeZoneKey(stationID)
	if base == "" {
		return nwacTelemetryConfig{}, fmt.Errorf("NotFound: missing telemetry station ID")
	}
	if cfg, ok := nwacTelemetryStations[base]; ok {
		return cfg, nil
	}
	if _, err := strconv.Atoi(base); err == nil {
		return nwacTelemetryConfig{
			StationID: base,
			PageSlug:  base,
			SnowObsID: base,
		}, nil
	}
	return nwacTelemetryConfig{}, fmt.Errorf("NotFound: unsupported telemetry station '%s' (try a known NWAC station alias or SnowObs numeric station id)", stationID)
}

func lastFloat(values []any) *float64 {
	for i := len(values) - 1; i >= 0; i-- {
		switch v := values[i].(type) {
		case float64:
			out := v
			return &out
		case int:
			out := float64(v)
			return &out
		case int64:
			out := float64(v)
			return &out
		default:
			continue
		}
	}
	return nil
}

func sumFloats(values []any) *float64 {
	sum := 0.0
	count := 0
	for i := range values {
		if v := floatFromAny(values[i]); v != nil {
			sum += *v
			count++
		}
	}
	if count == 0 {
		return nil
	}
	return &sum
}

func maxFloat(values []any) *float64 {
	var max *float64
	for i := range values {
		v := floatFromAny(values[i])
		if v == nil {
			continue
		}
		if max == nil || *v > *max {
			next := *v
			max = &next
		}
	}
	return max
}

func floatFromAny(v any) *float64 {
	switch n := v.(type) {
	case float64:
		out := n
		return &out
	case int:
		out := float64(n)
		return &out
	case int64:
		out := float64(n)
		return &out
	default:
		return nil
	}
}

func lastString(values []any) string {
	for i := len(values) - 1; i >= 0; i-- {
		switch v := values[i].(type) {
		case string:
			s := strings.TrimSpace(v)
			if s != "" {
				return s
			}
		case float64:
			return strconv.FormatFloat(v, 'f', -1, 64)
		case int:
			return strconv.Itoa(v)
		case int64:
			return strconv.FormatInt(v, 10)
		default:
			continue
		}
	}
	return ""
}

func lastNonEmptyString(values []string) string {
	for i := len(values) - 1; i >= 0; i-- {
		v := strings.TrimSpace(values[i])
		if v != "" {
			return v
		}
	}
	return ""
}
