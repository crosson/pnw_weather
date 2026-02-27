package pnwforecast

import "time"

type Area struct {
	Name                    string   `json:"name"`
	Lat                     float64  `json:"lat"`
	Lon                     float64  `json:"lon"`
	ElevationFt             *int     `json:"elevation_ft,omitempty"`
	NWACZoneID              string   `json:"nwac_zone_id,omitempty"`
	NWACZoneName            string   `json:"nwac_zone_name,omitempty"`
	NWACTelemetryStationIDs []string `json:"nwac_telemetry_station_ids,omitempty"`
	WSDOTPassID             string   `json:"wsdot_pass_id,omitempty"`
	WSDOTPassName           string   `json:"wsdot_pass_name,omitempty"`
	Tags                    []string `json:"tags,omitempty"`
}

type AreasFile struct {
	Version int    `json:"version"`
	Areas   []Area `json:"areas"`
}

type Source struct {
	Provider    string `json:"provider"`
	URL         string `json:"url"`
	APIURL      string `json:"api_url,omitempty"`
	RetrievedAt string `json:"retrieved_at"`
}

type ErrorPayload struct {
	Type    string `json:"type"`
	Message string `json:"message"`
}

func ISOTimeNow() string {
	return time.Now().UTC().Format(time.RFC3339)
}
