package ingest

import (
	"context"
	"database/sql"
	"errors"
	"fmt"
	"time"

	"home-automation-analytics/aggregation/blob"
	"home-automation-analytics/aggregation/bucketing"
	"home-automation-analytics/aggregation/quarter"
	"home-automation-analytics/aggregation/storage"
)

// ValidateHolding performs basic shape checks before touching storage.
func ValidateHolding(input HoldingInput) error {
	if input.ControlID == "" || input.ModelID == "" {
		return ErrInvalidInput
	}
	if input.State < 0 {
		return ErrInvalidInput
	}
	if input.EndTimeMs <= input.StartTimeMs {
		return ErrInvalidInput
	}
	return nil
}

// IngestHolding validates input, splits by UTC quarter, then accumulates
// per-bucket elapsed milliseconds into UTC and Local holding regions.
func IngestHolding(ctx context.Context, db *sql.DB, cfg Config, input HoldingInput) error {
	if err := ValidateHolding(input); err != nil {
		return fmt.Errorf("%w: %w", ErrValidation, err)
	}

	control, loc, err := resolveControlAndLocation(ctx, db, cfg, input.ControlID)
	if err != nil {
		if errors.Is(err, storage.ErrNotFound) {
			return fmt.Errorf("%w: %w", ErrValidation, err)
		}
		return err
	}
	if input.State >= control.NumStates {
		return fmt.Errorf("%w: %w", ErrValidation, ErrInvalidInput)
	}

	quarterSpans, err := quarter.SplitIntervalUTC(input.StartTimeMs, input.EndTimeMs)
	if err != nil {
		if errors.Is(err, quarter.ErrInvalidInterval) {
			return fmt.Errorf("%w: %w", ErrValidation, err)
		}
		return err
	}

	tx, err := db.BeginTx(ctx, nil)
	if err != nil {
		return err
	}
	committed := false
	defer func() {
		if !committed {
			_ = tx.Rollback()
		}
	}()

	for _, span := range quarterSpans {
		key := storage.AggregateKey{ControlID: input.ControlID, ModelID: input.ModelID, QuarterIndex: span.QuarterIndex}
		updateErr := applyHoldingQuarter(ctx, tx, key, control.NumStates, input.State, span.StartMs, span.EndMs, loc)
		if updateErr != nil {
			if IsValidationError(updateErr) {
				return updateErr
			}
			return updateErr
		}
	}

	if err := tx.Commit(); err != nil {
		return err
	}
	committed = true
	return nil
}

func applyHoldingQuarter(ctx context.Context, tx *sql.Tx, key storage.AggregateKey, numStates int, state int, startMs int64, endMs int64, loc *time.Location) error {
	return storage.UpdateAggregateTx(ctx, tx, key, numStates, func(data []byte) error {
		b, err := blob.NewBlob(numStates)
		if err != nil {
			return err
		}
		copy(b.Data(), data)

		utcSpans, err := bucketing.SplitIntervalUTC(startMs, endMs)
		if err != nil {
			return err
		}
		if err := applyHoldingClockSpans(b, numStates, state, blob.ClockUTC, utcSpans); err != nil {
			return err
		}

		localSpans, err := bucketing.SplitIntervalLocal(startMs, endMs, loc)
		if err != nil {
			return err
		}
		if err := applyHoldingClockSpans(b, numStates, state, blob.ClockLocal, localSpans); err != nil {
			return err
		}

		copy(data, b.Data())
		return nil
	})
}

func applyHoldingClockSpans(b *blob.Blob, numStates int, state int, clock int, spans []bucketing.BucketSpan) error {
	for _, s := range spans {
		idx, err := blob.HoldIndex(state, clock, s.Bucket, numStates)
		if err != nil {
			return err
		}
		v, err := b.GetU64(idx)
		if err != nil {
			return err
		}
		if s.Millis < 0 {
			return fmt.Errorf("%w: negative holding millis for bucket %d", ErrInvalidInput, s.Bucket)
		}
		millis := uint64(s.Millis)
		if err := b.SetU64(idx, v+millis); err != nil {
			return err
		}
	}
	return nil
}
