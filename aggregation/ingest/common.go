package ingest

import (
	"context"
	"database/sql"
	"fmt"
	"time"

	"home-automation-analytics/aggregation/storage"
)

// resolveControlAndLocation loads control metadata and configured timezone once
// so ingestion code can focus on quarter/bucket updates.
func resolveControlAndLocation(ctx context.Context, db *sql.DB, cfg Config, controlID string) (storage.Control, *time.Location, error) {
	control, err := storage.GetControl(ctx, db, controlID)
	if err != nil {
		return storage.Control{}, nil, fmt.Errorf("get control %q: %w", controlID, err)
	}

	loc, err := time.LoadLocation(cfg.TimeZone)
	if err != nil {
		return storage.Control{}, nil, fmt.Errorf("load location %q: %w", cfg.TimeZone, err)
	}

	return control, loc, nil
}
