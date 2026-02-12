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
		addr     = flag.String("addr", ":8080", "main HTTP listen address")
		testAddr = flag.String("test-addr", ":8081", "testing HTTP listen address")
		timeZone = flag.String("tz", getenvDefault("HAA_TIMEZONE", "UTC"), "IANA timezone")
		lat      = flag.Float64("lat", getenvFloatDefault("HAA_LATITUDE", 0), "Latitude")
		lon      = flag.Float64("lon", getenvFloatDefault("HAA_LONGITUDE", 0), "Longitude")
	)
	flag.Parse()

	cfg := ingest.Config{TimeZone: *timeZone, Latitude: *lat, Longitude: *lon}

	db, err := storage.Open("data/data.sqlite")
	if err != nil {
		log.Fatalf("open db: %v", err)
	}
	defer db.Close()

	if err := storage.InitSchema(context.Background(), db); err != nil {
		log.Fatalf("init schema: %v", err)
	}

	mainSrv := api.NewServer(db, cfg)
	testSrv := api.NewTestingServer(cfg)

	errCh := make(chan error, 2)
	log.Printf("main API listening on %s", *addr)
	go func() {
		errCh <- http.ListenAndServe(*addr, mainSrv)
	}()
	log.Printf("testing API listening on %s", *testAddr)
	go func() {
		errCh <- http.ListenAndServe(*testAddr, testSrv)
	}()

	if err := <-errCh; err != nil {
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
