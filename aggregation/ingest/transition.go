package ingest

import (
	"context"
	"database/sql"
	"time"

	"home-automation-analytics/aggregation/blob"
	"home-automation-analytics/aggregation/bucketing"
	"home-automation-analytics/aggregation/quarter"
	"home-automation-analytics/aggregation/storage"
)

type TransitionInput struct {
	ControlID   string
	ModelID     string
	FromState   int
	ToState     int
	TimestampMs int64
}

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

func IngestTransition(ctx context.Context, db *sql.DB, cfg Config, input TransitionInput) error {
	if err := ValidateTransition(input); err != nil {
		return err
	}

	control, err := storage.GetControl(ctx, db, input.ControlID)
	if err != nil {
		return err
	}
	if input.FromState >= control.NumStates || input.ToState >= control.NumStates {
		return ErrInvalidInput
	}

	loc, err := time.LoadLocation(cfg.TimeZone)
	if err != nil {
		return err
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
		idxUTC, err := blob.TransIndex(input.FromState, input.ToState, 0, bucketUTC, control.NumStates)
		if err != nil {
			return err
		}
		vUTC, err := b.GetU64(idxUTC)
		if err != nil {
			return err
		}
		if err := b.SetU64(idxUTC, vUTC+1); err != nil {
			return err
		}

		bucketLocal, err := bucketing.BucketAtLocal(input.TimestampMs, loc)
		if err != nil {
			return err
		}
		idxLocal, err := blob.TransIndex(input.FromState, input.ToState, 1, bucketLocal, control.NumStates)
		if err != nil {
			return err
		}
		vLocal, err := b.GetU64(idxLocal)
		if err != nil {
			return err
		}
		if err := b.SetU64(idxLocal, vLocal+1); err != nil {
			return err
		}

		copy(data, b.Data())
		return nil
	})
}
