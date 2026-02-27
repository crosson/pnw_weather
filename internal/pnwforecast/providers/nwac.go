package providers

import (
	"fmt"
	"html"
	"regexp"
	"strings"
	"time"
	"unicode"
)

type NWACAdapter struct {
	HTTP interface {
		GetText(endpoint string) (string, error)
		GetJSON(endpoint string, out any) error
	}
}

func (a *NWACAdapter) GetAvalancheForecast(zoneID string) (map[string]any, error) {
	zoneSlug := zoneToSlug(zoneID)
	pageURLs := []string{
		fmt.Sprintf("https://nwac.avy-fx.org/forecasts/avalanche/%s", zoneSlug),
		fmt.Sprintf("https://nwac.us/avalanche-forecast/%s", zoneSlug),
		fmt.Sprintf("https://nwac.us/avalanche-forecast/current/%s", zoneSlug),
	}

	var rawHTML string
	var err error
	pageURL := pageURLs[0]
	for _, candidate := range pageURLs {
		rawHTML, err = a.HTTP.GetText(candidate)
		if err == nil {
			pageURL = candidate
			break
		}
	}
	if err != nil {
		return nil, err
	}

	zoneName := parseNWACZoneName(rawHTML, zoneID)
	flat := normalizeHTMLText(rawHTML)
	below := parseDanger(flat, []string{"Below Treeline"})
	near := parseDanger(flat, []string{"Treeline", "Near Treeline"})
	above := parseDanger(flat, []string{"Alpine", "Above Treeline"})
	primary := parsePrimaryProblems(flat)
	travel := parseTravelAdvice(flat)

	return map[string]any{
		"provider":  "NWAC",
		"issued_at": time.Now().UTC().Format(time.RFC3339),
		"source": map[string]any{
			"provider":     "NWAC",
			"url":          pageURL,
			"retrieved_at": time.Now().UTC().Format(time.RFC3339),
		},
		"zone_id":   zoneID,
		"zone_name": zoneName,
		"danger_rating": map[string]string{
			"below_treeline": below,
			"near_treeline":  near,
			"above_treeline": above,
		},
		"primary_problems":      primary,
		"travel_advice_summary": travel,
	}, nil
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

func parseNWACZoneName(rawHTML, zoneID string) string {
	re := regexp.MustCompile(`(?is)<title>\s*([^<]+?)\s+Avalanche Forecast\s*</title>`)
	m := re.FindStringSubmatch(rawHTML)
	if len(m) == 2 {
		return strings.TrimSpace(html.UnescapeString(m[1]))
	}
	return toTitleWords(strings.ReplaceAll(strings.ToLower(zoneID), "_", " "))
}

func parseDanger(flat string, labels []string) string {
	levels := []string{"Low", "Moderate", "Considerable", "High", "Extreme", "No Rating"}
	for _, label := range labels {
		for _, level := range levels {
			pat := regexp.MustCompile(`(?i)` + regexp.QuoteMeta(label) + `\s+` + regexp.QuoteMeta(level))
			if pat.FindStringIndex(flat) != nil {
				if level == "No Rating" {
					return "NoRating"
				}
				return level
			}
		}
	}
	return "NoRating"
}

func parsePrimaryProblems(flat string) []string {
	known := []string{
		"Wind Slab",
		"Persistent Slab",
		"Storm Slab",
		"Loose Wet",
		"Loose Dry",
		"Cornice Fall",
		"Glide Avalanche",
		"Wet Slab",
	}
	out := make([]string, 0, 2)
	for _, problem := range known {
		if strings.Contains(strings.ToLower(flat), strings.ToLower(problem)) {
			out = append(out, problem)
		}
	}
	return out
}

func parseTravelAdvice(flat string) string {
	re := regexp.MustCompile(`(?is)Travel Advice\s+(.+?)(Avalanche Problems|Weather Forecast|Recent Avalanches|$)`)
	m := re.FindStringSubmatch(flat)
	if len(m) < 2 {
		return ""
	}
	text := strings.TrimSpace(m[1])
	if len(text) > 240 {
		return strings.TrimSpace(text[:240])
	}
	return text
}

func normalizeHTMLText(raw string) string {
	withoutScripts := regexp.MustCompile(`(?is)<script[^>]*>.*?</script>`).ReplaceAllString(raw, " ")
	withoutStyles := regexp.MustCompile(`(?is)<style[^>]*>.*?</style>`).ReplaceAllString(withoutScripts, " ")
	withoutTags := regexp.MustCompile(`(?is)<[^>]+>`).ReplaceAllString(withoutStyles, " ")
	unescaped := html.UnescapeString(withoutTags)
	spaceCollapsed := regexp.MustCompile(`\s+`).ReplaceAllString(unescaped, " ")
	return strings.TrimSpace(spaceCollapsed)
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

func zoneToSlug(zoneID string) string {
	slug := strings.ToLower(strings.TrimSpace(zoneID))
	slug = strings.ReplaceAll(slug, "_", "-")
	slug = strings.ReplaceAll(slug, " ", "-")
	return slug
}
