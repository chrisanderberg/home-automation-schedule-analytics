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
	"strconv"
	"syscall"
	"time"

	"home-automation-analytics/aggregation/api"
	"home-automation-analytics/aggregation/ingest"
	"home-automation-analytics/aggregation/storage"
)

func main() {
	if err := run(); err != nil {
		log.Printf("aggregationd failed: %v", err)
		os.Exit(1)
	}
}

func run() error {
	var (
		addr     = flag.String("addr", ":8080", "main HTTP listen address")
		testAddr = flag.String("test-addr", ":8081", "testing HTTP listen address")
		dbPath   = flag.String("db-path", getenvDefault("HAA_DB_PATH", "../data/data.sqlite"), "SQLite DB path")
		timeZone = flag.String("tz", getenvDefault("HAA_TIMEZONE", "UTC"), "IANA timezone")
		lat      = flag.Float64("lat", getenvFloatDefault("HAA_LATITUDE", 0), "Latitude")
		lon      = flag.Float64("lon", getenvFloatDefault("HAA_LONGITUDE", 0), "Longitude")
	)
	flag.Parse()

	cfg := ingest.Config{TimeZone: *timeZone, Latitude: *lat, Longitude: *lon}

	db, err := storage.Open(*dbPath)
	if err != nil {
		return fmt.Errorf("open db: %w", err)
	}
	defer db.Close()

	if err := storage.InitSchema(context.Background(), db); err != nil {
		return fmt.Errorf("init schema: %w", err)
	}

	mainSrv := api.NewServer(db, cfg)
	testSrv := api.NewTestingServer(cfg)
	mainHTTP := &http.Server{
		Addr:         *addr,
		Handler:      mainSrv,
		ReadTimeout:  10 * time.Second,
		WriteTimeout: 30 * time.Second,
		IdleTimeout:  60 * time.Second,
	}
	testHTTP := &http.Server{
		Addr:         *testAddr,
		Handler:      testSrv,
		ReadTimeout:  10 * time.Second,
		WriteTimeout: 30 * time.Second,
		IdleTimeout:  60 * time.Second,
	}

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
	select {
	case err := <-errCh:
		log.Printf("additional server error: %v", err)
		if runErr == nil {
			runErr = err
		}
	default:
	}

	shutdownCtx, cancel := context.WithTimeout(context.Background(), 10*time.Second)
	defer cancel()
	if err := shutdownServer(shutdownCtx, mainHTTP, "main"); err != nil {
		log.Printf("main shutdown error: %v", err)
	}

	testShutdownCtx, testCancel := context.WithTimeout(context.Background(), 10*time.Second)
	defer testCancel()
	if err := shutdownServer(testShutdownCtx, testHTTP, "testing"); err != nil {
		log.Printf("testing shutdown error: %v", err)
	}

	if runErr != nil {
		return fmt.Errorf("listen: %w", runErr)
	}
	return nil
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
		parsed, err := strconv.ParseFloat(val, 64)
		if err == nil {
			return parsed
		}
		log.Printf("invalid float value for %s=%q, using default %v", key, val, def)
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
