package api

import (
	"context"
	"database/sql"
	"encoding/json"
	"io"
	"net/http"
	"os"
	"path/filepath"
	"regexp"

	"home-automation-analytics/aggregation/ingest"
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
		writeError(w, http.StatusBadRequest, err.Error())
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
		writeError(w, http.StatusBadRequest, err.Error())
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
		writeError(w, http.StatusBadRequest, err.Error())
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
		writeError(w, http.StatusBadRequest, err.Error())
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
		writeError(w, http.StatusBadRequest, err.Error())
		return
	}
	defer db.Close()

	path, err := snapshot.ExportForTest(r.Context(), db, req.TestName, req.SnapshotName)
	if err != nil {
		writeError(w, http.StatusBadRequest, err.Error())
		return
	}
	writeJSON(w, http.StatusOK, map[string]string{"snapshotPath": path})
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
	dbPath := testingDBPath(req.TestName)
	_ = os.Remove(dbPath)
	_ = os.Remove(dbPath + "-wal")
	_ = os.Remove(dbPath + "-shm")

	writeJSON(w, http.StatusOK, map[string]string{"status": "ok"})
}

// openTestingDB opens/creates a test-scoped SQLite DB and initializes schema.
func openTestingDB(ctx context.Context, testName string) (*sql.DB, error) {
	if err := os.MkdirAll("test-data", 0o755); err != nil {
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
	return filepath.Join("test-data", testName+"-test-data.sqlite")
}

// isValidSlug enforces the lowercase-hyphen slug format used in test paths.
func isValidSlug(value string) bool {
	return slugRe.MatchString(value)
}

// decodeStrictJSON rejects unknown fields and trailing JSON tokens.
func decodeStrictJSON(r io.Reader, out any) error {
	dec := json.NewDecoder(r)
	dec.DisallowUnknownFields()
	if err := dec.Decode(out); err != nil {
		return err
	}
	if err := dec.Decode(&struct{}{}); err != io.EOF {
		return io.ErrUnexpectedEOF
	}
	return nil
}
