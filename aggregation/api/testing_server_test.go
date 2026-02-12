package api

import (
	"bytes"
	"context"
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

func TestTestingHoldingRejectsInvalidTestName(t *testing.T) {
	srv := NewTestingServer(ingest.Config{TimeZone: "UTC"})

	payload := map[string]any{
		"testName":    "Kitchen-Test",
		"controlId":   "c1",
		"modelId":     "m1",
		"state":       0,
		"startTimeMs": time.Date(2020, 1, 6, 0, 0, 0, 0, time.UTC).UnixMilli(),
		"endTimeMs":   time.Date(2020, 1, 6, 0, 5, 0, 0, time.UTC).UnixMilli(),
	}
	body, _ := json.Marshal(payload)

	req := httptest.NewRequest(http.MethodPost, "/v1/holding-intervals", bytes.NewReader(body))
	w := httptest.NewRecorder()
	srv.ServeHTTP(w, req)

	if w.Code != http.StatusBadRequest {
		t.Fatalf("expected 400, got %d", w.Code)
	}
}

func TestTestingHoldingWritesToTestSpecificDB(t *testing.T) {
	srv := NewTestingServer(ingest.Config{TimeZone: "UTC"})

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

	seedTestControl(t, "case-one", storage.Control{ControlID: "c1", ControlType: storage.ControlTypeDiscrete, NumStates: 2})

	payload := map[string]any{
		"testName":    "case-one",
		"controlId":   "c1",
		"modelId":     "m1",
		"state":       1,
		"startTimeMs": time.Date(2020, 1, 6, 0, 0, 0, 0, time.UTC).UnixMilli(),
		"endTimeMs":   time.Date(2020, 1, 6, 0, 5, 0, 0, time.UTC).UnixMilli(),
	}
	body, _ := json.Marshal(payload)

	req := httptest.NewRequest(http.MethodPost, "/v1/holding-intervals", bytes.NewReader(body))
	w := httptest.NewRecorder()
	srv.ServeHTTP(w, req)

	if w.Code != http.StatusAccepted {
		t.Fatalf("expected 202, got %d, body=%s", w.Code, w.Body.String())
	}

	dbPath := filepath.Join("test-data", "case-one-test-data.sqlite")
	if _, err := os.Stat(dbPath); err != nil {
		t.Fatalf("expected test db to exist: %v", err)
	}

	db, err := storage.Open(dbPath)
	if err != nil {
		t.Fatalf("open db: %v", err)
	}
	defer db.Close()
	var count int
	if err := db.QueryRowContext(context.Background(), "SELECT COUNT(*) FROM aggregates").Scan(&count); err != nil {
		t.Fatalf("count aggregates: %v", err)
	}
	if count != 1 {
		t.Fatalf("expected 1 aggregate row, got %d", count)
	}
}

func TestTestingSnapshotUsesRequestedNames(t *testing.T) {
	srv := NewTestingServer(ingest.Config{TimeZone: "UTC"})

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

	seedTestControl(t, "case-one", storage.Control{ControlID: "c1", ControlType: storage.ControlTypeDiscrete, NumStates: 2})

	payload := map[string]any{"testName": "case-one", "snapshotName": "sanity-check"}
	body, _ := json.Marshal(payload)
	req := httptest.NewRequest(http.MethodPost, "/v1/snapshots", bytes.NewReader(body))
	w := httptest.NewRecorder()
	srv.ServeHTTP(w, req)
	if w.Code != http.StatusOK {
		t.Fatalf("expected 200, got %d, body=%s", w.Code, w.Body.String())
	}

	var resp map[string]string
	if err := json.Unmarshal(w.Body.Bytes(), &resp); err != nil {
		t.Fatalf("decode response: %v", err)
	}
	path := resp["snapshotPath"]
	wantSuffix := filepath.Join("test-data", "snapshots", "case-one-sanity-check-snapshot.sqlite")
	if !strings.HasSuffix(path, wantSuffix) {
		t.Fatalf("snapshot path mismatch: got %s want suffix %s", path, wantSuffix)
	}
	if _, err := os.Stat(path); err != nil {
		t.Fatalf("expected snapshot to exist: %v", err)
	}
}

func TestTestingResetRemovesOnlyRequestedTestDB(t *testing.T) {
	srv := NewTestingServer(ingest.Config{TimeZone: "UTC"})

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

	seedTestControl(t, "case-one", storage.Control{ControlID: "c1", ControlType: storage.ControlTypeDiscrete, NumStates: 2})
	seedTestControl(t, "case-two", storage.Control{ControlID: "c1", ControlType: storage.ControlTypeDiscrete, NumStates: 2})

	req := httptest.NewRequest(http.MethodPost, "/v1/reset", bytes.NewReader([]byte(`{"testName":"case-one"}`)))
	w := httptest.NewRecorder()
	srv.ServeHTTP(w, req)
	if w.Code != http.StatusOK {
		t.Fatalf("expected 200, got %d", w.Code)
	}

	if _, err := os.Stat(filepath.Join("test-data", "case-one-test-data.sqlite")); !os.IsNotExist(err) {
		t.Fatalf("expected case-one db removed, stat err=%v", err)
	}
	if _, err := os.Stat(filepath.Join("test-data", "case-two-test-data.sqlite")); err != nil {
		t.Fatalf("expected case-two db to remain: %v", err)
	}
}

func seedTestControl(t *testing.T, testName string, control storage.Control) {
	t.Helper()
	db, err := openTestingDB(context.Background(), testName)
	if err != nil {
		t.Fatalf("open test db: %v", err)
	}
	defer db.Close()
	if err := storage.UpsertControl(context.Background(), db, control); err != nil {
		t.Fatalf("upsert control: %v", err)
	}
}
