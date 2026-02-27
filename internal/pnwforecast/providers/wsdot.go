package providers

import (
	"fmt"
	"strings"
	"time"
)

const WSDOTAPIURL = "https://wsdot.wa.gov/Traffic/api/mountainpasses/mountainpassconditions"

type WSDOTAdapter struct {
	HTTP interface {
		GetJSON(endpoint string, out any) error
	}
}

type wsdotPassRecord struct {
	MountainPassID   any    `json:"MountainPassId"`
	MountainPassName string `json:"MountainPassName"`
	DateUpdated      string `json:"DateUpdated"`
	RoadCondition    string `json:"RoadCondition"`
	RestrictionOne   string `json:"RestrictionOne"`
	WeatherCondition string `json:"WeatherCondition"`
	PassIDAlt        string `json:"pass_id"`
	PassNameAlt      string `json:"pass_name"`
	LastUpdatedAlt   string `json:"last_updated"`
	StatusAlt        string `json:"status"`
	RestrictionAlt   string `json:"restrictions"`
	WeatherAlt       string `json:"weather_conditions"`
}

type wsdotResponse struct {
	Passes []wsdotPassRecord `json:"Passes"`
	Alt    []wsdotPassRecord `json:"passes"`
	Items  []wsdotPassRecord `json:"Items"`
}

func (a *WSDOTAdapter) GetPassConditions(passIDOrName string) (map[string]any, error) {
	var resp wsdotResponse
	if err := a.HTTP.GetJSON(WSDOTAPIURL, &resp); err != nil {
		return nil, err
	}
	records := resp.Passes
	if len(records) == 0 {
		records = resp.Alt
	}
	if len(records) == 0 {
		records = resp.Items
	}

	lookup := strings.ToLower(passIDOrName)
	var selected *wsdotPassRecord
	for i := range records {
		id := strings.ToLower(recordID(records[i]))
		name := strings.ToLower(firstNonEmpty(records[i].MountainPassName, records[i].PassNameAlt))
		if id == lookup || name == lookup {
			selected = &records[i]
			break
		}
	}
	if selected == nil {
		if len(records) == 0 {
			return nil, fmt.Errorf("NotFound: No WSDOT pass records available")
		}
		selected = &records[0]
	}

	passID := strings.ToLower(recordID(*selected))
	passName := firstNonEmpty(selected.MountainPassName, selected.PassNameAlt, passIDOrName)
	issuedAt := firstNonEmpty(selected.DateUpdated, selected.LastUpdatedAlt, time.Now().UTC().Format(time.RFC3339))
	status := firstNonEmpty(selected.RoadCondition, selected.StatusAlt, "Unknown")

	return map[string]any{
		"provider":  "WSDOT",
		"issued_at": issuedAt,
		"source": map[string]any{
			"provider":     "WSDOT",
			"url":          fmt.Sprintf("https://wsdot.wa.gov/travel/real-time/mountain-passes/%s", passID),
			"api_url":      WSDOTAPIURL,
			"retrieved_at": time.Now().UTC().Format(time.RFC3339),
		},
		"pass_id":            passID,
		"pass_name":          passName,
		"status":             status,
		"restrictions":       firstNonEmpty(selected.RestrictionOne, selected.RestrictionAlt),
		"weather_conditions": firstNonEmpty(selected.WeatherCondition, selected.WeatherAlt),
		"last_updated":       issuedAt,
	}, nil
}

func recordID(v wsdotPassRecord) string {
	switch id := v.MountainPassID.(type) {
	case string:
		if id != "" {
			return id
		}
	case float64:
		return fmt.Sprintf("%.0f", id)
	}
	return v.PassIDAlt
}
