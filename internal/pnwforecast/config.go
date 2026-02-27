package pnwforecast

import (
	"os"
	"path/filepath"
)

var AllowedDomains = map[string]struct{}{
	"weather.gov":       {},
	"api.weather.gov":   {},
	"api.avalanche.org": {},
	"nwac.us":           {},
	"wsdot.wa.gov":      {},
	"wsdot.com":         {},
}

type FreshnessThresholds struct {
	NWSForecastSeconds   int
	NWACAvalancheSeconds int
	NWACTelemetrySeconds int
	WSDOTPassSeconds     int
}

func DefaultFreshnessThresholds() FreshnessThresholds {
	return FreshnessThresholds{
		NWSForecastSeconds:   90 * 60,
		NWACAvalancheSeconds: 6 * 60 * 60,
		NWACTelemetrySeconds: 30 * 60,
		WSDOTPassSeconds:     15 * 60,
	}
}

type SkillPaths struct {
	Root      string
	AreasPath string
	CachePath string
}

func DefaultSkillPaths() SkillPaths {
	home, _ := os.UserHomeDir()
	root := filepath.Join(home, ".openclaw", "skills", "pnw_forecast")
	return SkillPaths{
		Root:      root,
		AreasPath: filepath.Join(root, "areas.json"),
		CachePath: filepath.Join(root, "cache.json"),
	}
}
