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

// TestTestingHoldingRejectsInvalidTestName verifies testing API slug validation
// rejects non-lowercase-hyphen test names before ingest logic runs.
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

// TestTestingControlsUpsertWritesControl verifies testing-only control upsert
// persists control metadata into the test-scoped database.
func TestTestingControlsUpsertWritesControl(t *testing.T) {
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
	setTestDataDirForTest(t, outputDir)

	payload := map[string]any{
		"testName":    "case-one",
		"controlId":   "c1",
		"controlType": "discrete",
		"numStates":   2,
		"stateLabels": []string{"off", "on"},
	}
	body, _ := json.Marshal(payload)

	req := httptest.NewRequest(http.MethodPost, "/v1/controls", bytes.NewReader(body))
	w := httptest.NewRecorder()
	srv.ServeHTTP(w, req)

	if w.Code != http.StatusAccepted {
		t.Fatalf("expected 202, got %d, body=%s", w.Code, w.Body.String())
	}

	dbPath := testingDBPath("case-one")
	db, err := storage.Open(dbPath)
	if err != nil {
		t.Fatalf("open db: %v", err)
	}
	defer db.Close()

	control, err := storage.GetControl(context.Background(), db, "c1")
	if err != nil {
		t.Fatalf("get control: %v", err)
	}
	if control.ControlType != storage.ControlTypeDiscrete || control.NumStates != 2 {
		t.Fatalf("control mismatch: %+v", control)
	}
}

// TestTestingHoldingWritesToTestSpecificDB verifies testing ingestion writes to
// the per-test database path and appends aggregate data in that isolated DB.
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
	setTestDataDirForTest(t, outputDir)

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

	dbPath := testingDBPath("case-one")
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

// TestTestingSnapshotUsesRequestedNames verifies testing snapshot export uses
// requested test and snapshot slugs in the required output filename.
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
	setTestDataDirForTest(t, outputDir)

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
	if resp["snapshotName"] != "sanity-check" {
		t.Fatalf("snapshot name mismatch: got %q want %q", resp["snapshotName"], "sanity-check")
	}
	wantSuffix := filepath.Join("test-data", "snapshots", "case-one-sanity-check-snapshot.sqlite")
	snapshotPath := resp["snapshotPath"]
	if !strings.HasSuffix(snapshotPath, wantSuffix) {
		t.Fatalf("snapshot path mismatch: got %s want suffix %s (outputDir=%s)", snapshotPath, wantSuffix, outputDir)
	}
	if _, err := os.Stat(snapshotPath); err != nil {
		t.Fatalf("expected snapshot to exist: %v", err)
	}
}

// TestTestingSnapshotContainsSeededControl verifies the testing API flow
// can ingest data and export a snapshot that still contains control metadata.
// This scenario intentionally mirrors analytics/tests/test_testing_api_asset_flow.py.
func TestTestingSnapshotContainsSeededControl(t *testing.T) {
	srv := NewTestingServer(ingest.Config{TimeZone: "UTC"})
	testName := "asset-flow"
	snapshotName := "contains-control"

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
	setTestDataDirForTest(t, outputDir)

	seedTestControl(t, testName, storage.Control{ControlID: "c1", ControlType: storage.ControlTypeDiscrete, NumStates: 2})

	holdingPayload := map[string]any{
		"testName":    testName,
		"controlId":   "c1",
		"modelId":     "m1",
		"state":       1,
		"startTimeMs": time.Date(2020, 1, 6, 0, 0, 0, 0, time.UTC).UnixMilli(),
		"endTimeMs":   time.Date(2020, 1, 6, 0, 5, 0, 0, time.UTC).UnixMilli(),
	}
	holdingBody, _ := json.Marshal(holdingPayload)
	holdingReq := httptest.NewRequest(http.MethodPost, "/v1/holding-intervals", bytes.NewReader(holdingBody))
	holdingRes := httptest.NewRecorder()
	srv.ServeHTTP(holdingRes, holdingReq)
	if holdingRes.Code != http.StatusAccepted {
		t.Fatalf("expected holding 202, got %d, body=%s", holdingRes.Code, holdingRes.Body.String())
	}

	snapshotPayload := map[string]any{"testName": testName, "snapshotName": snapshotName}
	snapshotBody, _ := json.Marshal(snapshotPayload)
	snapshotReq := httptest.NewRequest(http.MethodPost, "/v1/snapshots", bytes.NewReader(snapshotBody))
	snapshotRes := httptest.NewRecorder()
	srv.ServeHTTP(snapshotRes, snapshotReq)
	if snapshotRes.Code != http.StatusOK {
		t.Fatalf("expected snapshot 200, got %d, body=%s", snapshotRes.Code, snapshotRes.Body.String())
	}

	snapshotPath := filepath.Join(outputDir, "test-data", "snapshots", testName+"-"+snapshotName+"-snapshot.sqlite")
	db, err := storage.Open(snapshotPath)
	if err != nil {
		t.Fatalf("open snapshot db: %v", err)
	}
	defer db.Close()

	var controlsCount int
	if err := db.QueryRowContext(context.Background(), "SELECT COUNT(*) FROM controls").Scan(&controlsCount); err != nil {
		t.Fatalf("count controls: %v", err)
	}
	if controlsCount < 1 {
		t.Fatalf("expected snapshot to contain at least one control row, got %d", controlsCount)
	}

	var aggregatesCount int
	if err := db.QueryRowContext(context.Background(), "SELECT COUNT(*) FROM aggregates").Scan(&aggregatesCount); err != nil {
		t.Fatalf("count aggregates: %v", err)
	}
	if aggregatesCount < 1 {
		t.Fatalf("expected snapshot to contain at least one aggregate row, got %d", aggregatesCount)
	}
}

// TestTestingResetRemovesOnlyRequestedTestDB verifies testing reset deletes only
// the requested test database and leaves other test datasets intact.
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
	setTestDataDirForTest(t, outputDir)

	seedTestControl(t, "case-one", storage.Control{ControlID: "c1", ControlType: storage.ControlTypeDiscrete, NumStates: 2})
	seedTestControl(t, "case-two", storage.Control{ControlID: "c1", ControlType: storage.ControlTypeDiscrete, NumStates: 2})

	req := httptest.NewRequest(http.MethodPost, "/v1/reset", bytes.NewReader([]byte(`{"testName":"case-one"}`)))
	w := httptest.NewRecorder()
	srv.ServeHTTP(w, req)
	if w.Code != http.StatusOK {
		t.Fatalf("expected 200, got %d", w.Code)
	}

	caseOnePath := filepath.Join(outputDir, "test-data", "case-one-test-data.sqlite")
	caseTwoPath := filepath.Join(outputDir, "test-data", "case-two-test-data.sqlite")
	if _, err := os.Stat(caseOnePath); !os.IsNotExist(err) {
		t.Fatalf("expected case-one db removed, stat err=%v", err)
	}
	if _, err := os.Stat(caseTwoPath); err != nil {
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

func setTestDataDirForTest(t *testing.T, outputDir string) {
	t.Helper()
	t.Setenv("TEST_DATA_DIR", filepath.Join(outputDir, "test-data"))
}
