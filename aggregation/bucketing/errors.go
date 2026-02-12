package bucketing

import "errors"

var (
	ErrInvalidTimestamp   = errors.New("invalid timestamp")
	ErrInvalidCoordinates = errors.New("invalid coordinates")
	ErrInvalidInterval    = errors.New("invalid interval")
	ErrInvalidBucket      = errors.New("invalid bucket")
	ErrUndefinedClock     = errors.New("clock mapping undefined")
)
