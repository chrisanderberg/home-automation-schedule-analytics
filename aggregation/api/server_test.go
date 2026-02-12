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

func TestHealthMethodNotAllowedIncludesAllowHeader(t *testing.T) {
	db := openTestDB(t)
	defer db.Close()

	srv := NewServer(db, ingest.Config{TimeZone: "UTC"})
	req := httptest.NewRequest(http.MethodPost, "/v1/health", nil)
	w := httptest.NewRecorder()
	srv.ServeHTTP(w, req)

	if w.Code != http.StatusMethodNotAllowed {
		t.Fatalf("expected 405, got %d", w.Code)
	}
	if got := w.Header().Get("Allow"); got != http.MethodGet {
		t.Fatalf("allow header mismatch: got %q want %q", got, http.MethodGet)
	}
}

// TestControlsEndpoint verifies valid control payloads are accepted by the
// main API and persisted for subsequent ingestion validation.
func TestControlsEndpoint(t *testing.T) {
	db := openTestDB(t)
	defer db.Close()

	srv := NewServer(db, ingest.Config{TimeZone: "UTC"})
	body := bytes.NewReader([]byte(`{"controlId":"c1","controlType":"discrete","numStates":2,"stateLabels":["off","on"]}`))

	req := httptest.NewRequest(http.MethodPost, "/v1/controls", body)
	w := httptest.NewRecorder()
	srv.ServeHTTP(w, req)

	if w.Code != http.StatusAccepted {
		t.Fatalf("expected 202, got %d body=%s", w.Code, w.Body.String())
	}
	control, err := storage.GetControl(context.Background(), db, "c1")
	if err != nil {
		t.Fatalf("get control: %v", err)
	}
	if control.ControlType != storage.ControlTypeDiscrete || control.NumStates != 2 {
		t.Fatalf("control mismatch: %+v", control)
	}
}

// TestControlsEndpointAcceptsOmittedStateLabels verifies that stateLabels is
// optional; requests omitting stateLabels entirely are accepted per API contract.
func TestControlsEndpointAcceptsOmittedStateLabels(t *testing.T) {
	db := openTestDB(t)
	defer db.Close()

	srv := NewServer(db, ingest.Config{TimeZone: "UTC"})
	body := bytes.NewReader([]byte(`{"controlId":"c2","controlType":"discrete","numStates":3}`))

	req := httptest.NewRequest(http.MethodPost, "/v1/controls", body)
	w := httptest.NewRecorder()
	srv.ServeHTTP(w, req)

	if w.Code != http.StatusAccepted {
		t.Fatalf("expected 202, got %d body=%s", w.Code, w.Body.String())
	}
	control, err := storage.GetControl(context.Background(), db, "c2")
	if err != nil {
		t.Fatalf("get control: %v", err)
	}
	if control.NumStates != 3 || len(control.StateLabels) != 0 {
		t.Fatalf("control mismatch: %+v", control)
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
// returns a created snapshot path under ../data/snapshots.
func TestSnapshotEndpoint(t *testing.T) {
	db := openTestDB(t)
	defer db.Close()

	srv := NewServer(db, ingest.Config{TimeZone: "UTC"})

	outputDir := t.TempDir()
	t.Setenv("SNAPSHOT_DIR", filepath.Join(outputDir, "data", "snapshots"))

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
	gotDir := filepath.Dir(snapshotPath)
	gotDirEval, err := filepath.EvalSymlinks(gotDir)
	if err == nil {
		gotDir = gotDirEval
	}
	if !filepath.IsAbs(gotDir) {
		t.Fatalf("snapshot dir must be absolute, got %s", gotDir)
	}
	if !strings.HasSuffix(gotDir, filepath.Join("data", "snapshots")) {
		t.Fatalf("snapshot dir mismatch: got %s", gotDir)
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
