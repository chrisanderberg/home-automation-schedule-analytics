package ingest

import (
	"context"
	"database/sql"

	"home-automation-analytics/aggregation/blob"
	"home-automation-analytics/aggregation/bucketing"
	"home-automation-analytics/aggregation/quarter"
	"home-automation-analytics/aggregation/storage"
)

// TransitionInput represents one user-initiated directed state change.
type TransitionInput struct {
	ControlID   string
	ModelID     string
	FromState   int
	ToState     int
	TimestampMs int64
}

// ValidateTransition performs basic shape checks before touching storage.
func ValidateTransition(input TransitionInput) error {
	if input.ControlID == "" || input.ModelID == "" {
		return ErrInvalidInput
	}
	if input.FromState < 0 || input.ToState < 0 {
		return ErrInvalidInput
	}
	if input.FromState == input.ToState {
		return ErrInvalidInput
	}
	return nil
}

// IngestTransition validates input, resolves control metadata, then increments
// directed transition counters for UTC and Local clocks in the quarter row.
func IngestTransition(ctx context.Context, db *sql.DB, cfg Config, input TransitionInput) error {
	if err := ValidateTransition(input); err != nil {
		return err
	}

	control, loc, err := resolveControlAndLocation(ctx, db, cfg, input.ControlID)
	if err != nil {
		return err
	}
	if input.FromState >= control.NumStates || input.ToState >= control.NumStates {
		return ErrInvalidInput
	}

	quarterIndex := quarter.QuarterIndexUTC(input.TimestampMs)
	key := storage.AggregateKey{ControlID: input.ControlID, ModelID: input.ModelID, QuarterIndex: quarterIndex}

	return storage.UpdateAggregate(ctx, db, key, control.NumStates, func(data []byte) error {
		b, err := blob.NewBlob(control.NumStates)
		if err != nil {
			return err
		}
		copy(b.Data(), data)

		bucketUTC, err := bucketing.BucketAtUTC(input.TimestampMs)
		if err != nil {
			return err
		}
		if err := incrementTransitionCount(b, input.FromState, input.ToState, control.NumStates, blob.ClockUTC, bucketUTC); err != nil {
			return err
		}

		bucketLocal, err := bucketing.BucketAtLocal(input.TimestampMs, loc)
		if err != nil {
			return err
		}
		if err := incrementTransitionCount(b, input.FromState, input.ToState, control.NumStates, blob.ClockLocal, bucketLocal); err != nil {
			return err
		}

		copy(data, b.Data())
		return nil
	})
}

func incrementTransitionCount(b *blob.Blob, fromState int, toState int, numStates int, clock int, bucket int) error {
	idx, err := blob.TransIndex(fromState, toState, clock, bucket, numStates)
	if err != nil {
		return err
	}
	v, err := b.GetU64(idx)
	if err != nil {
		return err
	}
	if err := b.SetU64(idx, v+1); err != nil {
		return err
	}
	return nil
}
