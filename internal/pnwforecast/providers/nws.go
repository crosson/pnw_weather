package providers

import (
	"fmt"
	"strings"
	"time"
)

type NWSAdapter struct {
	HTTP interface {
		GetJSON(endpoint string, out any) error
	}
}

type nwsPointResponse struct {
	Properties struct {
		Forecast string `json:"forecast"`
	} `json:"properties"`
}

type nwsForecastResponse struct {
	Properties struct {
		Updated string `json:"updated"`
		Periods []struct {
			Name                     string `json:"name"`
			StartTime                string `json:"startTime"`
			EndTime                  string `json:"endTime"`
			Temperature              *int   `json:"temperature"`
			WindDirection            string `json:"windDirection"`
			WindSpeed                string `json:"windSpeed"`
			ShortForecast            string `json:"shortForecast"`
			ProbabilityOfPrecipation struct {
				Value *int `json:"value"`
			} `json:"probabilityOfPrecipitation"`
		} `json:"periods"`
	} `json:"properties"`
}

func (a *NWSAdapter) GetSpotForecast(lat, lon float64) (map[string]any, error) {
	pointURL := fmt.Sprintf("https://api.weather.gov/points/%v,%v", lat, lon)
	var point nwsPointResponse
	if err := a.HTTP.GetJSON(pointURL, &point); err != nil {
		return nil, err
	}
	if point.Properties.Forecast == "" {
		return nil, fmt.Errorf("ParseError: NWS point response missing forecast URL")
	}

	var forecast nwsForecastResponse
	if err := a.HTTP.GetJSON(point.Properties.Forecast, &forecast); err != nil {
		return nil, err
	}

	periods := make([]map[string]any, 0, len(forecast.Properties.Periods))
	for _, p := range forecast.Properties.Periods {
		periods = append(periods, map[string]any{
			"name":                       p.Name,
			"start":                      p.StartTime,
			"end":                        p.EndTime,
			"temperature_f":              p.Temperature,
			"wind":                       strings.TrimSpace(fmt.Sprintf("%s %s", p.WindDirection, p.WindSpeed)),
			"precip_probability_percent": p.ProbabilityOfPrecipation.Value,
			"short_forecast":             p.ShortForecast,
		})
	}

	issuedAt := forecast.Properties.Updated
	if issuedAt == "" {
		issuedAt = time.Now().UTC().Format(time.RFC3339)
	}

	return map[string]any{
		"provider":  "NWS",
		"issued_at": issuedAt,
		"source": map[string]any{
			"provider":     "NWS",
			"url":          fmt.Sprintf("https://forecast.weather.gov/MapClick.php?lat=%v&lon=%v", lat, lon),
			"api_url":      point.Properties.Forecast,
			"retrieved_at": time.Now().UTC().Format(time.RFC3339),
		},
		"periods": periods,
	}, nil
}
