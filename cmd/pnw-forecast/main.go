package main

import (
	"encoding/json"
	"flag"
	"fmt"
	"os"
	"path/filepath"

	"github.com/crosson/pnw_weather/internal/pnwforecast"
)

func main() {
	root := flag.String("root", "", "Override data root path")
	noCache := flag.Bool("no-cache", false, "Bypass cache reads/writes for this run")
	flag.Parse()

	if flag.NArg() < 1 {
		fatalf("missing command")
	}
	cmd := flag.Arg(0)
	args := flag.Args()[1:]

	paths := pnwforecast.DefaultSkillPaths()
	if *root != "" {
		abs, err := filepath.Abs(*root)
		if err != nil {
			fatal(err)
		}
		paths = pnwforecast.SkillPaths{
			Root:      abs,
			AreasPath: filepath.Join(abs, "areas.json"),
			CachePath: filepath.Join(abs, "cache.json"),
		}
	}

	svc, err := pnwforecast.NewService(paths, 15*60)
	if err != nil {
		fatal(err)
	}
	svc.NoCache = *noCache

	switch cmd {
	case "area":
		handleArea(svc, args)
	case "weather":
		handleWeather(svc, args)
	case "weather-7day", "7day":
		handleWeather7Day(svc, args)
	case "avalanche":
		handleAvalanche(svc, args)
	case "telemetry":
		handleTelemetry(svc, args)
	case "pass":
		handlePass(svc, args)
	case "digest":
		handleDigest(svc, args)
	default:
		fatalf("unknown command: %s", cmd)
	}
}

func handleArea(svc *pnwforecast.Service, args []string) {
	if len(args) < 1 {
		fatalf("area subcommand required")
	}
	sub := args[0]
	switch sub {
	case "save":
		if len(args) < 2 {
			fatalf("area save requires a name")
		}
		name := args[1]
		fs := flag.NewFlagSet("area save", flag.ExitOnError)
		lat := fs.Float64("lat", 0, "Latitude")
		lon := fs.Float64("lon", 0, "Longitude")
		elevation := fs.Int("elevation-ft", 0, "Elevation (ft)")
		nwacZoneID := fs.String("nwac-zone-id", "", "NWAC zone ID")
		nwacZoneName := fs.String("nwac-zone-name", "", "NWAC zone name")
		wsdotPassID := fs.String("wsdot-pass-id", "", "WSDOT pass ID")
		wsdotPassName := fs.String("wsdot-pass-name", "", "WSDOT pass name")
		stationIDs := multiValue{}
		tags := multiValue{}
		fs.Var(&stationIDs, "station-id", "NWAC telemetry station ID (repeatable)")
		fs.Var(&tags, "tag", "Area tag (repeatable)")
		if err := fs.Parse(args[2:]); err != nil {
			fatal(err)
		}
		if *lat == 0 && *lon == 0 {
			fatalf("--lat and --lon are required")
		}
		var elevationPtr *int
		if *elevation != 0 {
			elevationPtr = elevation
		}
		area, err := svc.SaveArea(name, pnwforecast.Area{
			Lat:                     *lat,
			Lon:                     *lon,
			ElevationFt:             elevationPtr,
			NWACZoneID:              *nwacZoneID,
			NWACZoneName:            *nwacZoneName,
			NWACTelemetryStationIDs: stationIDs,
			WSDOTPassID:             *wsdotPassID,
			WSDOTPassName:           *wsdotPassName,
			Tags:                    tags,
		})
		if err != nil {
			fatal(err)
		}
		emit(area)
	case "list":
		areas, err := svc.ListAreas()
		if err != nil {
			fatal(err)
		}
		emit(areas)
	case "delete":
		if len(args) < 2 {
			fatalf("area delete requires a name")
		}
		ok, err := svc.DeleteArea(args[1])
		if err != nil {
			fatal(err)
		}
		emit(map[string]bool{"deleted": ok})
	default:
		fatalf("unknown area subcommand: %s", sub)
	}
}

func handleWeather(svc *pnwforecast.Service, args []string) {
	fs := flag.NewFlagSet("weather", flag.ExitOnError)
	area := fs.String("area", "", "Area name")
	lat := fs.Float64("lat", 0, "Latitude")
	lon := fs.Float64("lon", 0, "Longitude")
	elevation := fs.Int("elevation-ft", 0, "Elevation (ft)")
	datetime := fs.String("datetime", "", "Datetime")
	if err := fs.Parse(args); err != nil {
		fatal(err)
	}
	var elevationPtr *int
	if *elevation != 0 {
		elevationPtr = elevation
	}
	out, err := svc.GetSpotForecast(*area, *lat, *lon, elevationPtr, *datetime)
	if err != nil {
		fatal(err)
	}
	emit(out)
}

func handleWeather7Day(svc *pnwforecast.Service, args []string) {
	fs := flag.NewFlagSet("weather-7day", flag.ExitOnError)
	area := fs.String("area", "", "Area name")
	lat := fs.Float64("lat", 0, "Latitude")
	lon := fs.Float64("lon", 0, "Longitude")
	elevation := fs.Int("elevation-ft", 0, "Elevation (ft)")
	datetime := fs.String("datetime", "", "Datetime")
	if err := fs.Parse(args); err != nil {
		fatal(err)
	}
	var elevationPtr *int
	if *elevation != 0 {
		elevationPtr = elevation
	}
	out, err := svc.GetSpotForecast7Day(*area, *lat, *lon, elevationPtr, *datetime)
	if err != nil {
		fatal(err)
	}
	emit(out)
}

func handleAvalanche(svc *pnwforecast.Service, args []string) {
	fs := flag.NewFlagSet("avalanche", flag.ExitOnError)
	area := fs.String("area", "", "Area name")
	zone := fs.String("zone-id", "", "NWAC zone ID")
	if err := fs.Parse(args); err != nil {
		fatal(err)
	}
	out, err := svc.GetAvalancheForecast(*area, *zone)
	if err != nil {
		fatal(err)
	}
	emit(out)
}

func handleTelemetry(svc *pnwforecast.Service, args []string) {
	fs := flag.NewFlagSet("telemetry", flag.ExitOnError)
	area := fs.String("area", "", "Area name")
	station := fs.String("station-id", "", "NWAC station ID")
	zone := fs.String("zone-id", "", "NWAC zone ID")
	if err := fs.Parse(args); err != nil {
		fatal(err)
	}
	out, err := svc.GetTelemetry(*area, *station, *zone)
	if err != nil {
		fatal(err)
	}
	emit(out)
}

func handlePass(svc *pnwforecast.Service, args []string) {
	fs := flag.NewFlagSet("pass", flag.ExitOnError)
	area := fs.String("area", "", "Area name")
	passID := fs.String("pass-id", "", "WSDOT pass ID")
	passName := fs.String("pass-name", "", "WSDOT pass name")
	if err := fs.Parse(args); err != nil {
		fatal(err)
	}
	out, err := svc.GetPassConditions(*area, *passID, *passName)
	if err != nil {
		fatal(err)
	}
	emit(out)
}

func handleDigest(svc *pnwforecast.Service, args []string) {
	fs := flag.NewFlagSet("digest", flag.ExitOnError)
	date := fs.String("date", "", "Date")
	areas := multiValue{}
	fs.Var(&areas, "area", "Area name (repeatable)")
	if err := fs.Parse(args); err != nil {
		fatal(err)
	}
	out, err := svc.GetDailyDigest(*date, areas)
	if err != nil {
		fatal(err)
	}
	emit(out)
}

type multiValue []string

func (m *multiValue) String() string {
	return fmt.Sprintf("%v", []string(*m))
}

func (m *multiValue) Set(v string) error {
	*m = append(*m, v)
	return nil
}

func emit(v any) {
	enc := json.NewEncoder(os.Stdout)
	enc.SetIndent("", "  ")
	if err := enc.Encode(v); err != nil {
		fatal(err)
	}
}

func fatal(err error) {
	fmt.Fprintln(os.Stderr, err)
	os.Exit(1)
}

func fatalf(format string, args ...any) {
	fatal(fmt.Errorf(format, args...))
}
