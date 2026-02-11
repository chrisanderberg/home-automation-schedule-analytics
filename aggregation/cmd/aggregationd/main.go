package main

import (
	"context"
	"flag"
	"fmt"
	"log"
	"net/http"
	"os"

	"home-automation-analytics/aggregation/api"
	"home-automation-analytics/aggregation/ingest"
	"home-automation-analytics/aggregation/storage"
)

func main() {
	var (
		addr     = flag.String("addr", ":8080", "HTTP listen address")
		dbPath   = flag.String("db", "aggregation.sqlite", "SQLite DB path")
		timeZone = flag.String("tz", getenvDefault("HAA_TIMEZONE", "UTC"), "IANA timezone")
		lat      = flag.Float64("lat", getenvFloatDefault("HAA_LATITUDE", 0), "Latitude")
		lon      = flag.Float64("lon", getenvFloatDefault("HAA_LONGITUDE", 0), "Longitude")
	)
	flag.Parse()

	cfg := ingest.Config{TimeZone: *timeZone, Latitude: *lat, Longitude: *lon}

	db, err := storage.Open(*dbPath)
	if err != nil {
		log.Fatalf("open db: %v", err)
	}
	defer db.Close()

	if err := storage.InitSchema(context.Background(), db); err != nil {
		log.Fatalf("init schema: %v", err)
	}

	srv := api.NewServer(db, cfg)

	log.Printf("listening on %s", *addr)
	if err := http.ListenAndServe(*addr, srv); err != nil {
		log.Fatalf("listen: %v", err)
	}
}

func getenvDefault(key, def string) string {
	if val := os.Getenv(key); val != "" {
		return val
	}
	return def
}

func getenvFloatDefault(key string, def float64) float64 {
	if val := os.Getenv(key); val != "" {
		var parsed float64
		_, err := fmt.Sscanf(val, "%f", &parsed)
		if err == nil {
			return parsed
		}
	}
	return def
}
