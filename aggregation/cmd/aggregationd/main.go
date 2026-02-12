package main

import (
	"context"
	"errors"
	"flag"
	"fmt"
	"log"
	"net/http"
	"os"
	"os/signal"
	"syscall"
	"time"

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

	db, err := storage.Open("../data/data.sqlite")
	if err != nil {
		log.Fatalf("open db: %v", err)
	}
	defer db.Close()

	if err := storage.InitSchema(context.Background(), db); err != nil {
		log.Fatalf("init schema: %v", err)
	}

	mainSrv := api.NewServer(db, cfg)
	testSrv := api.NewTestingServer(cfg)
	mainHTTP := &http.Server{Addr: *addr, Handler: mainSrv}
	testHTTP := &http.Server{Addr: *testAddr, Handler: testSrv}

	runCtx, stop := signal.NotifyContext(context.Background(), os.Interrupt, syscall.SIGTERM)
	defer stop()

	// Run both APIs concurrently and coordinate shutdown on signal or failure.
	errCh := make(chan error, 2)
	log.Printf("main API listening on %s", *addr)
	go func() {
		if err := mainHTTP.ListenAndServe(); err != nil && !errors.Is(err, http.ErrServerClosed) {
			errCh <- fmt.Errorf("main api: %w", err)
		}
	}()
	log.Printf("testing API listening on %s", *testAddr)
	go func() {
		if err := testHTTP.ListenAndServe(); err != nil && !errors.Is(err, http.ErrServerClosed) {
			errCh <- fmt.Errorf("testing api: %w", err)
		}
	}()

	var runErr error
	select {
	case <-runCtx.Done():
		log.Printf("shutdown signal received")
	case err := <-errCh:
		runErr = err
		log.Printf("server error: %v", err)
	}

	shutdownCtx, cancel := context.WithTimeout(context.Background(), 10*time.Second)
	defer cancel()
	if err := shutdownServer(shutdownCtx, mainHTTP, "main"); err != nil {
		log.Printf("main shutdown error: %v", err)
	}
	if err := shutdownServer(shutdownCtx, testHTTP, "testing"); err != nil {
		log.Printf("testing shutdown error: %v", err)
	}

	if runErr != nil {
		log.Fatalf("listen: %v", runErr)
	}
}

// getenvDefault returns env value when set, otherwise def.
func getenvDefault(key, def string) string {
	if val := os.Getenv(key); val != "" {
		return val
	}
	return def
}

// getenvFloatDefault parses a float env var and falls back on parse failure.
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

func shutdownServer(ctx context.Context, srv *http.Server, name string) error {
	if srv == nil {
		return nil
	}
	if err := srv.Shutdown(ctx); err != nil {
		return fmt.Errorf("%s server shutdown: %w", name, err)
	}
	return nil
}
