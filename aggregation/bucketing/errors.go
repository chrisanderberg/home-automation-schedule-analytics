package bucketing

import "errors"

var (
	ErrInvalidTimestamp = errors.New("invalid timestamp")
	ErrInvalidInterval  = errors.New("invalid interval")
	ErrInvalidBucket    = errors.New("invalid bucket")
	ErrUndefinedClock   = errors.New("clock mapping undefined")
)
