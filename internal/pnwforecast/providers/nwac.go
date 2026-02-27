package providers

import (
	"fmt"
	"strings"
	"time"
	"unicode"
)

type NWACAdapter struct {
	HTTP interface {
		GetJSON(endpoint string, out any) error
	}
}

type nwacForecastResponse struct {
	IssuedAt           string            `json:"issued_at"`
	IssuedAtAlt        string            `json:"issuedAt"`
	ZoneName           string            `json:"zone_name"`
	ZoneNameAlt        string            `json:"zoneName"`
	DangerRating       map[string]string `json:"danger_rating"`
	DangerRatingAlt    map[string]string `json:"dangerRating"`
	PrimaryProblems    []string          `json:"primary_problems"`
	PrimaryProblemsAlt []string          `json:"primaryProblems"`
	TravelAdvice       string            `json:"travel_advice_summary"`
	TravelAdviceAlt    string            `json:"travelAdviceSummary"`
}

type nwacTelemetryResponse struct {
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

func (a *NWACAdapter) GetAvalancheForecast(zoneID string) (map[string]any, error) {
	apiURL := fmt.Sprintf("https://nwac.us/api/v6/forecast/zone/%s", zoneID)
	var resp nwacForecastResponse
	if err := a.HTTP.GetJSON(apiURL, &resp); err != nil {
		return nil, err
	}
	issuedAt := firstNonEmpty(resp.IssuedAt, resp.IssuedAtAlt, time.Now().UTC().Format(time.RFC3339))
	zoneName := firstNonEmpty(resp.ZoneName, resp.ZoneNameAlt, toTitleWords(strings.ReplaceAll(strings.ToLower(zoneID), "_", " ")))
	danger := resp.DangerRating
	if len(danger) == 0 {
		danger = resp.DangerRatingAlt
	}
	if danger == nil {
		danger = map[string]string{}
	}

	primary := resp.PrimaryProblems
	if len(primary) == 0 {
		primary = resp.PrimaryProblemsAlt
	}
	travel := firstNonEmpty(resp.TravelAdvice, resp.TravelAdviceAlt)

	return map[string]any{
		"provider":  "NWAC",
		"issued_at": issuedAt,
		"source": map[string]any{
			"provider":     "NWAC",
			"url":          fmt.Sprintf("https://nwac.us/avalanche-forecast/#/%s", strings.ToLower(strings.ReplaceAll(zoneID, "_", "-"))),
			"api_url":      apiURL,
			"retrieved_at": time.Now().UTC().Format(time.RFC3339),
		},
		"zone_id":   zoneID,
		"zone_name": zoneName,
		"danger_rating": map[string]string{
			"below_treeline": firstNonEmpty(danger["below_treeline"], danger["belowTreeline"], "NoRating"),
			"near_treeline":  firstNonEmpty(danger["near_treeline"], danger["nearTreeline"], "NoRating"),
			"above_treeline": firstNonEmpty(danger["above_treeline"], danger["aboveTreeline"], "NoRating"),
		},
		"primary_problems":      primary,
		"travel_advice_summary": travel,
	}, nil
}

func (a *NWACAdapter) GetTelemetry(stationID string) (map[string]any, error) {
	apiURL := fmt.Sprintf("https://nwac.us/api/v6/station/%s", stationID)
	var resp nwacTelemetryResponse
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

func firstNonEmpty(values ...string) string {
	for _, v := range values {
		if strings.TrimSpace(v) != "" {
			return v
		}
	}
	return ""
}

func toTitleWords(v string) string {
	parts := strings.Fields(v)
	for i := range parts {
		runes := []rune(parts[i])
		if len(runes) == 0 {
			continue
		}
		runes[0] = unicode.ToUpper(runes[0])
		parts[i] = string(runes)
	}
	return strings.Join(parts, " ")
}
