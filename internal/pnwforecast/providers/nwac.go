package providers

import (
	"fmt"
	"strings"
	"time"
)

const (
	nwacCenterID = "NWAC"
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
		"danger_rating": map[string]string{
			"below_treeline": below,
			"near_treeline":  near,
			"above_treeline": above,
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
	apiURL := fmt.Sprintf("https://nwac.us/api/v6/station/%s", stationID)
	var resp struct {
		StationName     string   `json:"station_name"`
		StationNameAlt  string   `json:"stationName"`
		Timestamp       string   `json:"timestamp"`
		ObservedAt      string   `json:"observed_at"`
		ObservedAtAlt   string   `json:"observedAt"`
		TemperatureF    *float64 `json:"temperature_f"`
		TemperatureFAlt *float64 `json:"temperatureF"`
		SnowDepthIn     *float64 `json:"snow_depth_in"`
		SnowDepthInAlt  *float64 `json:"snowDepthIn"`
		WindSpeedMPH    *float64 `json:"wind_speed_mph"`
		WindSpeedMPHAlt *float64 `json:"windSpeedMph"`
		WindDirection   string   `json:"wind_direction"`
		WindDirection2  string   `json:"windDirection"`
	}
	if err := a.HTTP.GetJSON(apiURL, &resp); err != nil {
		return nil, err
	}

	ts := firstNonEmpty(resp.Timestamp, resp.ObservedAt, resp.ObservedAtAlt, time.Now().UTC().Format(time.RFC3339))
	temp := resp.TemperatureF
	if temp == nil {
		temp = resp.TemperatureFAlt
	}
	snow := resp.SnowDepthIn
	if snow == nil {
		snow = resp.SnowDepthInAlt
	}
	wind := resp.WindSpeedMPH
	if wind == nil {
		wind = resp.WindSpeedMPHAlt
	}

	return map[string]any{
		"provider":  "NWAC",
		"issued_at": ts,
		"source": map[string]any{
			"provider":     "NWAC",
			"url":          fmt.Sprintf("https://nwac.us/weatherdata/%s/", stationID),
			"api_url":      apiURL,
			"retrieved_at": time.Now().UTC().Format(time.RFC3339),
		},
		"station_id":     stationID,
		"station_name":   firstNonEmpty(resp.StationName, resp.StationNameAlt, stationID),
		"timestamp":      ts,
		"temperature_f":  temp,
		"snow_depth_in":  snow,
		"wind_speed_mph": wind,
		"wind_direction": firstNonEmpty(resp.WindDirection, resp.WindDirection2),
	}, nil
}
