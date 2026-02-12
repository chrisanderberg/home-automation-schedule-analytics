package storage

import (
	"context"
	"database/sql"
	"encoding/json"

	_ "modernc.org/sqlite"

	"home-automation-analytics/aggregation/blob"
)

func Open(dbPath string) (*sql.DB, error) {
	db, err := sql.Open("sqlite", dbPath)
	if err != nil {
		return nil, err
	}
	if dbPath == ":memory:" {
		// Single connection ensures the in-memory DB is shared across operations.
		db.SetMaxOpenConns(1)
	}
	return db, nil
}

// InitSchema ensures required tables exist before ingestion/query operations.
func InitSchema(ctx context.Context, db *sql.DB) error {
	_, err := db.ExecContext(ctx, Schema)
	return err
}

// UpsertControl inserts or replaces control metadata by control_id.
func UpsertControl(ctx context.Context, db *sql.DB, control Control) error {
	labels, err := encodeLabels(control.StateLabels)
	if err != nil {
		return err
	}
	_, err = db.ExecContext(
		ctx,
		`INSERT INTO controls (control_id, control_type, num_states, state_labels)
		 VALUES (?, ?, ?, ?)
		 ON CONFLICT(control_id) DO UPDATE SET
		   control_type=excluded.control_type,
		   num_states=excluded.num_states,
		   state_labels=excluded.state_labels`,
		control.ControlID,
		string(control.ControlType),
		control.NumStates,
		labels,
	)
	return err
}

// GetControl loads control metadata and returns ErrNotFound when missing.
func GetControl(ctx context.Context, db *sql.DB, controlID string) (Control, error) {
	row := db.QueryRowContext(ctx, `SELECT control_id, control_type, num_states, state_labels FROM controls WHERE control_id = ?`, controlID)
	var control Control
	var controlType string
	var labels sql.NullString
	if err := row.Scan(&control.ControlID, &controlType, &control.NumStates, &labels); err != nil {
		if err == sql.ErrNoRows {
			return Control{}, ErrNotFound
		}
		return Control{}, err
	}
	control.ControlType = ControlType(controlType)
	if labels.Valid && labels.String != "" {
		decoded, err := decodeLabels(labels.String)
		if err != nil {
			return Control{}, err
		}
		control.StateLabels = decoded
	}
	return control, nil
}

// GetOrCreateAggregate fetches an existing blob or materializes a zeroed blob
// for the requested key and state cardinality.
func GetOrCreateAggregate(ctx context.Context, db *sql.DB, key AggregateKey, numStates int) ([]byte, error) {
	row := db.QueryRowContext(
		ctx,
		`SELECT blob FROM aggregates WHERE control_id = ? AND model_id = ? AND quarter_index = ?`,
		key.ControlID,
		key.ModelID,
		key.QuarterIndex,
	)
	var blobBytes []byte
	switch err := row.Scan(&blobBytes); err {
	case nil:
		return blobBytes, nil
	case sql.ErrNoRows:
		b, err := blob.NewBlob(numStates)
		if err != nil {
			return nil, err
		}
		blobBytes = b.Data()
		_, err = db.ExecContext(
			ctx,
			`INSERT INTO aggregates (control_id, model_id, quarter_index, blob)
			 VALUES (?, ?, ?, ?)
			 ON CONFLICT(control_id, model_id, quarter_index) DO NOTHING`,
			key.ControlID,
			key.ModelID,
			key.QuarterIndex,
			blobBytes,
		)
		if err != nil {
			return nil, err
		}
		row = db.QueryRowContext(
			ctx,
			`SELECT blob FROM aggregates WHERE control_id = ? AND model_id = ? AND quarter_index = ?`,
			key.ControlID,
			key.ModelID,
			key.QuarterIndex,
		)
		if err := row.Scan(&blobBytes); err != nil {
			return nil, err
		}
		return blobBytes, nil
	default:
		return nil, err
	}
}

// UpdateAggregate performs a read-modify-write under BEGIN IMMEDIATE to avoid
// lost updates when multiple writers target the same aggregate row.
func UpdateAggregate(ctx context.Context, db *sql.DB, key AggregateKey, numStates int, update func([]byte) error) error {
	conn, err := db.Conn(ctx)
	if err != nil {
		return err
	}
	defer conn.Close()

	if _, err := conn.ExecContext(ctx, "BEGIN IMMEDIATE"); err != nil {
		return err
	}
	committed := false
	defer func() {
		if !committed {
			_, _ = conn.ExecContext(ctx, "ROLLBACK")
		}
	}()

	if err := updateAggregateWithQueryExec(
		ctx,
		conn,
		conn,
		key,
		numStates,
		update,
	); err != nil {
		return err
	}

	if _, err := conn.ExecContext(ctx, "COMMIT"); err != nil {
		return err
	}
	committed = true
	return nil
}

// UpdateAggregateTx performs aggregate read-modify-write using a caller-owned
// transaction. The caller is responsible for commit/rollback.
func UpdateAggregateTx(ctx context.Context, tx *sql.Tx, key AggregateKey, numStates int, update func([]byte) error) error {
	return updateAggregateWithQueryExec(ctx, tx, tx, key, numStates, update)
}

type queryRower interface {
	QueryRowContext(ctx context.Context, query string, args ...any) *sql.Row
}

type execContexter interface {
	ExecContext(ctx context.Context, query string, args ...any) (sql.Result, error)
}

func updateAggregateWithQueryExec(
	ctx context.Context,
	queryDB queryRower,
	execDB execContexter,
	key AggregateKey,
	numStates int,
	update func([]byte) error,
) error {
	row := queryDB.QueryRowContext(
		ctx,
		`SELECT blob FROM aggregates WHERE control_id = ? AND model_id = ? AND quarter_index = ?`,
		key.ControlID,
		key.ModelID,
		key.QuarterIndex,
	)
	var blobBytes []byte
	switch err := row.Scan(&blobBytes); err {
	case nil:
		// use existing blob
	case sql.ErrNoRows:
		b, err := blob.NewBlob(numStates)
		if err != nil {
			return err
		}
		blobBytes = b.Data()
	default:
		return err
	}

	// Work on a copy so failed updates cannot partially mutate persisted bytes.
	working := make([]byte, len(blobBytes))
	copy(working, blobBytes)
	if err := update(working); err != nil {
		return err
	}

	_, err := execDB.ExecContext(
		ctx,
		`INSERT INTO aggregates (control_id, model_id, quarter_index, blob)
		 VALUES (?, ?, ?, ?)
		 ON CONFLICT(control_id, model_id, quarter_index) DO UPDATE SET blob=excluded.blob`,
		key.ControlID,
		key.ModelID,
		key.QuarterIndex,
		working,
	)
	if err != nil {
		return err
	}
	return nil
}

// encodeLabels stores optional state labels as JSON text.
func encodeLabels(labels []string) (string, error) {
	if len(labels) == 0 {
		return "", nil
	}
	data, err := json.Marshal(labels)
	if err != nil {
		return "", err
	}
	return string(data), nil
}

// decodeLabels restores JSON-encoded optional state labels from storage.
func decodeLabels(raw string) ([]string, error) {
	if raw == "" {
		return nil, nil
	}
	var labels []string
	if err := json.Unmarshal([]byte(raw), &labels); err != nil {
		return nil, err
	}
	return labels, nil
}
