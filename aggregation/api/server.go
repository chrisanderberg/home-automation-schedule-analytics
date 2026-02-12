package api

import (
	"context"
	"database/sql"
	"encoding/json"
	"io"
	"log"
	"net/http"

	"home-automation-analytics/aggregation/ingest"
	"home-automation-analytics/aggregation/snapshot"
	"home-automation-analytics/aggregation/storage"
)

type Server struct {
	db  *sql.DB
	cfg ingest.Config
	mux *http.ServeMux
}

// NewServer wires the main API surface on the shared ingestion/storage stack.
func NewServer(db *sql.DB, cfg ingest.Config) *Server {
	s := &Server{db: db, cfg: cfg, mux: http.NewServeMux()}
	s.routes()
	return s
}

func (s *Server) ServeHTTP(w http.ResponseWriter, r *http.Request) {
	s.mux.ServeHTTP(w, r)
}

// routes registers all main API endpoints.
func (s *Server) routes() {
	s.mux.HandleFunc("/v1/health", s.handleHealth)
	s.mux.HandleFunc("/v1/controls", s.handleControls)
	s.mux.HandleFunc("/v1/holding-intervals", s.handleHolding)
	s.mux.HandleFunc("/v1/transitions", s.handleTransitions)
	s.mux.HandleFunc("/v1/snapshots", s.handleSnapshots)
}

func (s *Server) handleHealth(w http.ResponseWriter, r *http.Request) {
	if r.Method != http.MethodGet {
		writeError(w, http.StatusMethodNotAllowed, "method not allowed")
		return
	}
	writeJSON(w, http.StatusOK, map[string]string{"status": "ok"})
}

type controlRequest struct {
	ControlID   string   `json:"controlId"`
	ControlType string   `json:"controlType"`
	NumStates   int      `json:"numStates"`
	StateLabels []string `json:"stateLabels"`
}

// handleControls upserts control metadata required by ingestion endpoints.
func (s *Server) handleControls(w http.ResponseWriter, r *http.Request) {
	if r.Method != http.MethodPost {
		writeError(w, http.StatusMethodNotAllowed, "method not allowed")
		return
	}
	var req controlRequest
	if err := decodeStrictJSON(r.Body, &req); err != nil {
		writeError(w, http.StatusBadRequest, "invalid json")
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

	control := storage.Control{
		ControlID:   req.ControlID,
		ControlType: storage.ControlType(req.ControlType),
		NumStates:   req.NumStates,
		StateLabels: req.StateLabels,
	}
	if err := storage.UpsertControl(r.Context(), s.db, control); err != nil {
		log.Printf("handleControls upsert failed: %v", err)
		writeError(w, http.StatusInternalServerError, "internal server error")
		return
	}
	writeJSON(w, http.StatusAccepted, map[string]string{"status": "accepted"})
}

// handleHolding decodes a main API holding request and forwards to ingestion.
func (s *Server) handleHolding(w http.ResponseWriter, r *http.Request) {
	if r.Method != http.MethodPost {
		writeError(w, http.StatusMethodNotAllowed, "method not allowed")
		return
	}
	var input ingest.HoldingInput
	if err := decodeStrictJSON(r.Body, &input); err != nil {
		writeError(w, http.StatusBadRequest, "invalid json")
		return
	}
	ctx := r.Context()
	if err := ingest.IngestHolding(ctx, s.db, s.cfg, input); err != nil {
		if ingest.IsValidationError(err) {
			writeError(w, http.StatusBadRequest, "invalid input")
			return
		}
		log.Printf("handleHolding ingest failed: %v", err)
		writeError(w, http.StatusInternalServerError, "internal server error")
		return
	}
	writeJSON(w, http.StatusAccepted, map[string]string{"status": "accepted"})
}

// handleTransitions decodes a main API transition request and forwards to ingestion.
func (s *Server) handleTransitions(w http.ResponseWriter, r *http.Request) {
	if r.Method != http.MethodPost {
		writeError(w, http.StatusMethodNotAllowed, "method not allowed")
		return
	}
	var input ingest.TransitionInput
	if err := decodeStrictJSON(r.Body, &input); err != nil {
		writeError(w, http.StatusBadRequest, "invalid json")
		return
	}
	ctx := r.Context()
	if err := ingest.IngestTransition(ctx, s.db, s.cfg, input); err != nil {
		writeError(w, http.StatusBadRequest, err.Error())
		return
	}
	writeJSON(w, http.StatusAccepted, map[string]string{"status": "accepted"})
}

// handleSnapshots exports a snapshot from the fixed runtime location contract.
func (s *Server) handleSnapshots(w http.ResponseWriter, r *http.Request) {
	if r.Method != http.MethodPost {
		writeError(w, http.StatusMethodNotAllowed, "method not allowed")
		return
	}
	if r.Body != nil {
		var req struct{}
		decoder := json.NewDecoder(r.Body)
		decoder.DisallowUnknownFields()
		if err := decoder.Decode(&req); err != nil && err != io.EOF {
			writeError(w, http.StatusBadRequest, "invalid json")
			return
		}
	}
	path, err := snapshot.Export(r.Context(), s.db)
	if err != nil {
		log.Printf("handleSnapshots export failed: %v", err)
		writeError(w, http.StatusInternalServerError, "internal server error")
		return
	}
	writeJSON(w, http.StatusOK, map[string]string{"snapshotPath": path})
}

// writeJSON is the shared response encoder for success and error payloads.
func writeJSON(w http.ResponseWriter, status int, body any) {
	w.Header().Set("Content-Type", "application/json")
	w.WriteHeader(status)
	_ = json.NewEncoder(w).Encode(body)
}

func writeError(w http.ResponseWriter, status int, message string) {
	writeJSON(w, status, map[string]string{"error": message})
}

func decodeStrictJSON(r io.Reader, v any) error {
	decoder := json.NewDecoder(r)
	decoder.DisallowUnknownFields()
	return decoder.Decode(v)
}

// WithContext injects a fixed context into a handler; primarily used by tests.
func WithContext(ctx context.Context, h http.Handler) http.Handler {
	return http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		h.ServeHTTP(w, r.WithContext(ctx))
	})
}
