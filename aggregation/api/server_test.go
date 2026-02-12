package api

import (
	"bytes"
	"context"
	"database/sql"
	"encoding/json"
	"net/http"
	"net/http/httptest"
	"os"
	"path/filepath"
	"strings"
	"testing"
	"time"

	"home-automation-analytics/aggregation/ingest"
	"home-automation-analytics/aggregation/storage"
)

// TestHealth verifies the health endpoint is wired and returns HTTP 200 when
// the server has a valid initialized storage dependency.
func TestHealth(t *testing.T) {
	db := openTestDB(t)
	defer db.Close()

	srv := NewServer(db, ingest.Config{TimeZone: "UTC"})
	req := httptest.NewRequest(http.MethodGet, "/v1/health", nil)
	w := httptest.NewRecorder()
	srv.ServeHTTP(w, req)

	if w.Code != http.StatusOK {
		t.Fatalf("expected 200, got %d", w.Code)
	}
}

// TestHoldingEndpoint verifies valid holding payloads are accepted by the main
// API and routed through shared ingestion logic.
func TestHoldingEndpoint(t *testing.T) {
	db := openTestDB(t)
	defer db.Close()

	control := storage.Control{ControlID: "c1", ControlType: storage.ControlTypeDiscrete, NumStates: 2}
	if err := storage.UpsertControl(context.Background(), db, control); err != nil {
		t.Fatalf("upsert control: %v", err)
	}

	srv := NewServer(db, ingest.Config{TimeZone: "UTC"})

	payload := ingest.HoldingInput{
		ControlID:   "c1",
		ModelID:     "m1",
		State:       1,
		StartTimeMs: time.Date(2020, 1, 6, 0, 0, 0, 0, time.UTC).UnixMilli(),
		EndTimeMs:   time.Date(2020, 1, 6, 0, 5, 0, 0, time.UTC).UnixMilli(),
	}
	body, _ := json.Marshal(payload)

	req := httptest.NewRequest(http.MethodPost, "/v1/holding-intervals", bytes.NewReader(body))
	w := httptest.NewRecorder()
	srv.ServeHTTP(w, req)

	if w.Code != http.StatusAccepted {
		t.Fatalf("expected 202, got %d", w.Code)
	}
}

// TestTransitionEndpoint verifies valid transition payloads are accepted by the
// main API and routed through shared ingestion logic.
func TestTransitionEndpoint(t *testing.T) {
	db := openTestDB(t)
	defer db.Close()

	control := storage.Control{ControlID: "c1", ControlType: storage.ControlTypeDiscrete, NumStates: 3}
	if err := storage.UpsertControl(context.Background(), db, control); err != nil {
		t.Fatalf("upsert control: %v", err)
	}

	srv := NewServer(db, ingest.Config{TimeZone: "UTC"})

	payload := ingest.TransitionInput{
		ControlID:   "c1",
		ModelID:     "m1",
		FromState:   0,
		ToState:     2,
		TimestampMs: time.Date(2020, 1, 6, 0, 7, 0, 0, time.UTC).UnixMilli(),
	}
	body, _ := json.Marshal(payload)

	req := httptest.NewRequest(http.MethodPost, "/v1/transitions", bytes.NewReader(body))
	w := httptest.NewRecorder()
	srv.ServeHTTP(w, req)

	if w.Code != http.StatusAccepted {
		t.Fatalf("expected 202, got %d", w.Code)
	}
}

// TestSnapshotEndpoint verifies snapshot export succeeds from the main API and
// returns a created snapshot path under data/snapshots.
func TestSnapshotEndpoint(t *testing.T) {
	db := openTestDB(t)
	defer db.Close()

	srv := NewServer(db, ingest.Config{TimeZone: "UTC"})

	outputDir := t.TempDir()
	prevWD, err := os.Getwd()
	if err != nil {
		t.Fatalf("getwd: %v", err)
	}
	if err := os.Chdir(outputDir); err != nil {
		t.Fatalf("chdir: %v", err)
	}
	t.Cleanup(func() {
		_ = os.Chdir(prevWD)
	})

	req := httptest.NewRequest(http.MethodPost, "/v1/snapshots", nil)
	w := httptest.NewRecorder()
	srv.ServeHTTP(w, req)

	if w.Code != http.StatusOK {
		t.Fatalf("expected 200, got %d", w.Code)
	}

	var payload map[string]string
	if err := json.Unmarshal(w.Body.Bytes(), &payload); err != nil {
		t.Fatalf("decode response: %v", err)
	}
	snapshotPath := payload["snapshotPath"]
	if snapshotPath == "" {
		t.Fatalf("missing snapshotPath in response: %s", w.Body.String())
	}
	wantDir := filepath.Join(outputDir, "data", "snapshots")
	if !strings.HasSuffix(filepath.Dir(snapshotPath), filepath.Join("data", "snapshots")) {
		t.Fatalf("snapshot dir mismatch: got %s want suffix %s", filepath.Dir(snapshotPath), wantDir)
	}
	if _, err := os.Stat(snapshotPath); err != nil {
		t.Fatalf("snapshot file missing: %v", err)
	}
}

// TestSnapshotEndpointRejectsOutputPathOverride verifies main API clients
// cannot override runtime snapshot output paths via request payload fields.
func TestSnapshotEndpointRejectsOutputPathOverride(t *testing.T) {
	db := openTestDB(t)
	defer db.Close()

	srv := NewServer(db, ingest.Config{TimeZone: "UTC"})
	body := bytes.NewReader([]byte(`{"outputPath":"somewhere.sqlite"}`))

	req := httptest.NewRequest(http.MethodPost, "/v1/snapshots", body)
	w := httptest.NewRecorder()
	srv.ServeHTTP(w, req)

	if w.Code != http.StatusBadRequest {
		t.Fatalf("expected 400, got %d", w.Code)
	}
}

// TestMainAPIHasNoResetEndpoint verifies reset remains testing-only by
// asserting the main API exposes no /v1/reset route.
func TestMainAPIHasNoResetEndpoint(t *testing.T) {
	db := openTestDB(t)
	defer db.Close()

	srv := NewServer(db, ingest.Config{TimeZone: "UTC"})
	req := httptest.NewRequest(http.MethodPost, "/v1/reset", bytes.NewReader([]byte(`{}`)))
	w := httptest.NewRecorder()
	srv.ServeHTTP(w, req)

	if w.Code != http.StatusNotFound {
		t.Fatalf("expected 404, got %d", w.Code)
	}
}

func openTestDB(t *testing.T) *sql.DB {
	t.Helper()
	db, err := storage.Open(":memory:")
	if err != nil {
		t.Fatalf("open db: %v", err)
	}
	if err := storage.InitSchema(context.Background(), db); err != nil {
		t.Fatalf("init schema: %v", err)
	}
	return db
}
