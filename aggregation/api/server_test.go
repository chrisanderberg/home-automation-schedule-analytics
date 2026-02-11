package api

import (
	"bytes"
	"context"
	"encoding/json"
	"net/http"
	"net/http/httptest"
	"database/sql"
	"testing"
	"time"

	"home-automation-analytics/aggregation/ingest"
	"home-automation-analytics/aggregation/storage"
)

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

func TestSnapshotEndpoint(t *testing.T) {
	db := openTestDB(t)
	defer db.Close()

	srv := NewServer(db, ingest.Config{TimeZone: "UTC"})

	outputDir := t.TempDir()
	payload := map[string]string{"outputPath": outputDir + "/snapshot.sqlite"}
	body, _ := json.Marshal(payload)

	req := httptest.NewRequest(http.MethodPost, "/v1/snapshots", bytes.NewReader(body))
	w := httptest.NewRecorder()
	srv.ServeHTTP(w, req)

	if w.Code != http.StatusOK {
		t.Fatalf("expected 200, got %d", w.Code)
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
