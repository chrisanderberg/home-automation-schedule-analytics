package api

import (
	"context"
	"database/sql"
	"errors"
	"fmt"
	"log"
	"net/http"
	"os"
	"path/filepath"
	"regexp"
	"runtime"

	"home-automation-analytics/aggregation/ingest"
	"home-automation-analytics/aggregation/internal/reporoot"
	"home-automation-analytics/aggregation/snapshot"
	"home-automation-analytics/aggregation/storage"
)

var slugRe = regexp.MustCompile(`^[a-z0-9]+(?:-[a-z0-9]+)*$`)

type TestingServer struct {
	cfg ingest.Config
	mux *http.ServeMux
}

// NewTestingServer wires the isolated testing API that uses per-test DB files.
func NewTestingServer(cfg ingest.Config) *TestingServer {
	s := &TestingServer{cfg: cfg, mux: http.NewServeMux()}
	s.routes()
	return s
}

func (s *TestingServer) ServeHTTP(w http.ResponseWriter, r *http.Request) {
	s.mux.ServeHTTP(w, r)
}

// routes registers the testing API endpoints, including /v1/reset.
func (s *TestingServer) routes() {
	s.mux.HandleFunc("/v1/health", s.handleHealth)
	s.mux.HandleFunc("/v1/controls", s.handleControls)
	s.mux.HandleFunc("/v1/holding-intervals", s.handleHolding)
	s.mux.HandleFunc("/v1/transitions", s.handleTransitions)
	s.mux.HandleFunc("/v1/snapshots", s.handleSnapshots)
	s.mux.HandleFunc("/v1/reset", s.handleReset)
}

func (s *TestingServer) handleHealth(w http.ResponseWriter, r *http.Request) {
	if r.Method != http.MethodGet {
		writeError(w, http.StatusMethodNotAllowed, "method not allowed")
		return
	}
	writeJSON(w, http.StatusOK, map[string]string{"status": "ok"})
}

type testingHoldingRequest struct {
	TestName    string `json:"testName"`
	ControlID   string `json:"controlId"`
	ModelID     string `json:"modelId"`
	State       int    `json:"state"`
	StartTimeMs int64  `json:"startTimeMs"`
	EndTimeMs   int64  `json:"endTimeMs"`
}

type testingTransitionRequest struct {
	TestName    string `json:"testName"`
	ControlID   string `json:"controlId"`
	ModelID     string `json:"modelId"`
	FromState   int    `json:"fromState"`
	ToState     int    `json:"toState"`
	TimestampMs int64  `json:"timestampMs"`
}

type testingSnapshotRequest struct {
	TestName     string `json:"testName"`
	SnapshotName string `json:"snapshotName"`
}

type testingResetRequest struct {
	TestName string `json:"testName"`
}

type testingControlRequest struct {
	TestName    string   `json:"testName"`
	ControlID   string   `json:"controlId"`
	ControlType string   `json:"controlType"`
	NumStates   int      `json:"numStates"`
	StateLabels []string `json:"stateLabels"`
}

func (s *TestingServer) handleControls(w http.ResponseWriter, r *http.Request) {
	if r.Method != http.MethodPost {
		writeError(w, http.StatusMethodNotAllowed, "method not allowed")
		return
	}
	var req testingControlRequest
	if err := decodeStrictJSON(r.Body, &req); err != nil {
		writeError(w, http.StatusBadRequest, "invalid json")
		return
	}
	if !isValidSlug(req.TestName) {
		writeError(w, http.StatusBadRequest, "invalid testName")
		return
	}
	if req.ControlID == "" {
		writeError(w, http.StatusBadRequest, "invalid controlId")
		return
	}
	if req.NumStates < 2 || req.NumStates > 10 {
		writeError(w, http.StatusBadRequest, "invalid numStates")
		return
	}
	if req.ControlType != string(storage.ControlTypeDiscrete) && req.ControlType != string(storage.ControlTypeSlider) {
		writeError(w, http.StatusBadRequest, "invalid controlType")
		return
	}
	if len(req.StateLabels) > 0 && len(req.StateLabels) != req.NumStates {
		writeError(w, http.StatusBadRequest, "stateLabels length must equal numStates")
		return
	}

	db, err := openTestingDB(r.Context(), req.TestName)
	if err != nil {
		log.Printf("testing controls open db failed for %q: %v", req.TestName, err)
		writeError(w, http.StatusInternalServerError, "internal server error")
		return
	}
	defer db.Close()

	control := storage.Control{
		ControlID:   req.ControlID,
		ControlType: storage.ControlType(req.ControlType),
		NumStates:   req.NumStates,
		StateLabels: req.StateLabels,
	}
	if err := storage.UpsertControl(r.Context(), db, control); err != nil {
		log.Printf("testing controls upsert failed for %q/%q: %v", req.TestName, req.ControlID, err)
		writeError(w, http.StatusInternalServerError, "internal server error")
		return
	}
	writeJSON(w, http.StatusAccepted, map[string]string{"status": "accepted"})
}

func (s *TestingServer) handleHolding(w http.ResponseWriter, r *http.Request) {
	if r.Method != http.MethodPost {
		writeError(w, http.StatusMethodNotAllowed, "method not allowed")
		return
	}
	var req testingHoldingRequest
	if err := decodeStrictJSON(r.Body, &req); err != nil {
		writeError(w, http.StatusBadRequest, "invalid json")
		return
	}
	// Testing requests must include a valid test slug to select isolated data.
	if !isValidSlug(req.TestName) {
		writeError(w, http.StatusBadRequest, "invalid testName")
		return
	}

	db, err := openTestingDB(r.Context(), req.TestName)
	if err != nil {
		log.Printf("testing holding open db failed for %q: %v", req.TestName, err)
		writeError(w, http.StatusInternalServerError, "internal server error")
		return
	}
	defer db.Close()

	input := ingest.HoldingInput{
		ControlID:   req.ControlID,
		ModelID:     req.ModelID,
		State:       req.State,
		StartTimeMs: req.StartTimeMs,
		EndTimeMs:   req.EndTimeMs,
	}
	if err := ingest.IngestHolding(r.Context(), db, s.cfg, input); err != nil {
		if ingest.IsValidationError(err) {
			writeError(w, http.StatusBadRequest, "invalid input")
			return
		}
		log.Printf("testing handleHolding ingest failed for %q: %v", req.TestName, err)
		writeError(w, http.StatusInternalServerError, "internal server error")
		return
	}
	writeJSON(w, http.StatusAccepted, map[string]string{"status": "accepted"})
}

func (s *TestingServer) handleTransitions(w http.ResponseWriter, r *http.Request) {
	if r.Method != http.MethodPost {
		writeError(w, http.StatusMethodNotAllowed, "method not allowed")
		return
	}
	var req testingTransitionRequest
	if err := decodeStrictJSON(r.Body, &req); err != nil {
		writeError(w, http.StatusBadRequest, "invalid json")
		return
	}
	if !isValidSlug(req.TestName) {
		writeError(w, http.StatusBadRequest, "invalid testName")
		return
	}

	db, err := openTestingDB(r.Context(), req.TestName)
	if err != nil {
		log.Printf("testing transitions open db failed for %q: %v", req.TestName, err)
		writeError(w, http.StatusInternalServerError, "internal server error")
		return
	}
	defer db.Close()

	input := ingest.TransitionInput{
		ControlID:   req.ControlID,
		ModelID:     req.ModelID,
		FromState:   req.FromState,
		ToState:     req.ToState,
		TimestampMs: req.TimestampMs,
	}
	if err := ingest.IngestTransition(r.Context(), db, s.cfg, input); err != nil {
		if ingest.IsValidationError(err) {
			writeError(w, http.StatusBadRequest, "invalid input")
			return
		}
		log.Printf("testing handleTransitions ingest failed for %q: %v", req.TestName, err)
		writeError(w, http.StatusInternalServerError, "internal server error")
		return
	}
	writeJSON(w, http.StatusAccepted, map[string]string{"status": "accepted"})
}

func (s *TestingServer) handleSnapshots(w http.ResponseWriter, r *http.Request) {
	if r.Method != http.MethodPost {
		writeError(w, http.StatusMethodNotAllowed, "method not allowed")
		return
	}
	var req testingSnapshotRequest
	if err := decodeStrictJSON(r.Body, &req); err != nil {
		writeError(w, http.StatusBadRequest, "invalid json")
		return
	}
	if !isValidSlug(req.TestName) {
		writeError(w, http.StatusBadRequest, "invalid testName")
		return
	}
	if !isValidSlug(req.SnapshotName) {
		writeError(w, http.StatusBadRequest, "invalid snapshotName")
		return
	}

	db, err := openTestingDB(r.Context(), req.TestName)
	if err != nil {
		log.Printf("testing snapshots open db failed for %q: %v", req.TestName, err)
		writeError(w, http.StatusInternalServerError, "internal server error")
		return
	}
	defer db.Close()

	snapshotPath, err := snapshot.ExportForTest(r.Context(), db, req.TestName, req.SnapshotName)
	if err != nil {
		log.Printf("testing snapshots export failed for %q/%q: %v", req.TestName, req.SnapshotName, err)
		writeError(w, http.StatusInternalServerError, "internal server error")
		return
	}
	writeJSON(w, http.StatusOK, map[string]string{
		"snapshotName": req.SnapshotName,
		"snapshotPath": snapshotPath,
	})
}

func (s *TestingServer) handleReset(w http.ResponseWriter, r *http.Request) {
	if r.Method != http.MethodPost {
		writeError(w, http.StatusMethodNotAllowed, "method not allowed")
		return
	}
	var req testingResetRequest
	if err := decodeStrictJSON(r.Body, &req); err != nil {
		writeError(w, http.StatusBadRequest, "invalid json")
		return
	}
	if !isValidSlug(req.TestName) {
		writeError(w, http.StatusBadRequest, "invalid testName")
		return
	}

	// Reset removes only files for the requested test dataset.
	root, err := testDataRootDir()
	if err != nil {
		log.Printf("testing reset path resolution failed for %q: %v", req.TestName, err)
		writeError(w, http.StatusInternalServerError, "internal server error")
		return
	}
	dbPath := filepath.Join(root, req.TestName+"-test-data.sqlite")
	paths := []string{dbPath, dbPath + "-wal", dbPath + "-shm"}
	removalFailed := false
	for _, path := range paths {
		err := os.Remove(path)
		if err == nil || errors.Is(err, os.ErrNotExist) {
			continue
		}
		log.Printf("reset remove failed for %s: %v", path, err)
		if _, statErr := os.Stat(path); statErr == nil {
			removalFailed = true
		}
	}
	if removalFailed {
		writeError(w, http.StatusInternalServerError, "internal server error")
		return
	}

	writeJSON(w, http.StatusOK, map[string]string{"status": "ok"})
}

// openTestingDB opens/creates a test-scoped SQLite DB and initializes schema.
func openTestingDB(ctx context.Context, testName string) (*sql.DB, error) {
	root, err := testDataRootDir()
	if err != nil {
		return nil, err
	}
	if err := os.MkdirAll(root, 0o755); err != nil {
		return nil, err
	}
	dbPath := testingDBPath(testName)
	db, err := storage.Open(dbPath)
	if err != nil {
		return nil, err
	}
	if err := storage.InitSchema(ctx, db); err != nil {
		db.Close()
		return nil, err
	}
	return db, nil
}

// testingDBPath is the canonical per-test DB filename contract.
func testingDBPath(testName string) string {
	root, err := testDataRootDir()
	if err != nil {
		// Fallback keeps callsites resilient; openTestingDB still returns explicit
		// errors from testDataRootDir before using this fallback path.
		return filepath.Join("test-data", testName+"-test-data.sqlite")
	}
	return filepath.Join(root, testName+"-test-data.sqlite")
}

func testDataRootDir() (string, error) {
	if override := os.Getenv("TEST_DATA_DIR"); override != "" {
		return filepath.Abs(override)
	}

	execPath, err := os.Executable()
	if err == nil {
		if root, ok := reporoot.Find(filepath.Dir(execPath)); ok {
			return filepath.Join(root, "test-data"), nil
		}
	}
	if _, file, _, ok := runtime.Caller(0); ok {
		if root, ok := reporoot.Find(filepath.Dir(file)); ok {
			return filepath.Join(root, "test-data"), nil
		}
	}
	wd, err := os.Getwd()
	if err != nil {
		return "", fmt.Errorf("determine working directory for test data path: %w", err)
	}
	if root, ok := reporoot.Find(wd); ok {
		return filepath.Join(root, "test-data"), nil
	}
	return "", fmt.Errorf("could not resolve test-data root")
}

// isValidSlug enforces the lowercase-hyphen slug format used in test paths.
func isValidSlug(value string) bool {
	return slugRe.MatchString(value)
}
