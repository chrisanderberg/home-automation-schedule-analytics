package ingest

import (
	"errors"

	"home-automation-analytics/aggregation/bucketing"
	"home-automation-analytics/aggregation/quarter"
	"home-automation-analytics/aggregation/storage"
)

var (
	ErrInvalidInput = errors.New("invalid input")
	ErrValidation   = errors.New("validation error")
)

func IsValidationError(err error) bool {
	return errors.Is(err, ErrValidation) ||
		errors.Is(err, ErrInvalidInput) ||
		errors.Is(err, storage.ErrNotFound) ||
		errors.Is(err, quarter.ErrInvalidInterval) ||
		errors.Is(err, bucketing.ErrInvalidInterval) ||
		errors.Is(err, bucketing.ErrInvalidTimestamp) ||
		errors.Is(err, bucketing.ErrInvalidCoordinates)
}
