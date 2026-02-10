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

func IngestHolding(ctx context.Context, db *sql.DB, cfg Config, input HoldingInput) error {
	if err := ValidateHolding(input); err != nil {
		return err
	}

	control, err := storage.GetControl(ctx, db, input.ControlID)
	if err != nil {
		return err
	}
	if input.State >= control.NumStates {
		return ErrInvalidInput
	}

	loc, err := time.LoadLocation(cfg.TimeZone)
	if err != nil {
		return err
	}

	quarterSpans, err := quarter.SplitIntervalUTC(input.StartTimeMs, input.EndTimeMs)
	if err != nil {
		return err
	}

	for _, span := range quarterSpans {
		key := storage.AggregateKey{ControlID: input.ControlID, ModelID: input.ModelID, QuarterIndex: span.QuarterIndex}
		updateErr := storage.UpdateAggregate(ctx, db, key, control.NumStates, func(data []byte) error {
			b, err := blob.NewBlob(control.NumStates)
			if err != nil {
				return err
			}
			copy(b.Data(), data)

			utcSpans, err := bucketing.SplitIntervalUTC(span.StartMs, span.EndMs)
			if err != nil {
				return err
			}
			for _, s := range utcSpans {
				idx, err := blob.HoldIndex(input.State, 0, s.Bucket, control.NumStates)
				if err != nil {
					return err
				}
				v, err := b.GetU64(idx)
				if err != nil {
					return err
				}
				if err := b.SetU64(idx, v+uint64(s.Millis)); err != nil {
					return err
				}
			}

			localSpans, err := bucketing.SplitIntervalLocal(span.StartMs, span.EndMs, loc)
			if err != nil {
				return err
			}
			for _, s := range localSpans {
				idx, err := blob.HoldIndex(input.State, 1, s.Bucket, control.NumStates)
				if err != nil {
					return err
				}
				v, err := b.GetU64(idx)
				if err != nil {
					return err
				}
				if err := b.SetU64(idx, v+uint64(s.Millis)); err != nil {
					return err
				}
			}

			copy(data, b.Data())
			return nil
		})
		if updateErr != nil {
			return updateErr
		}
	}

	return nil
}
