package bucketing

import "time"

type BucketSpan struct {
	Bucket int
	Millis int64
}

// BucketAtUTC maps a UTC timestamp to Monday-based 5-minute week bucket index.
func BucketAtUTC(timestampMs int64) (int, error) {
	t := time.UnixMilli(timestampMs).UTC()
	return bucketFromTime(t), nil
}

// BucketAtLocal maps a timestamp into buckets using the configured local zone.
func BucketAtLocal(timestampMs int64, loc *time.Location) (int, error) {
	if loc == nil {
		return 0, ErrInvalidTimestamp
	}
	t := time.UnixMilli(timestampMs).In(loc)
	return bucketFromTime(t), nil
}

// SplitIntervalUTC splits a half-open UTC interval into per-bucket elapsed
// milliseconds preserving total real duration.
func SplitIntervalUTC(startMs, endMs int64) ([]BucketSpan, error) {
	if endMs <= startMs {
		return nil, ErrInvalidInterval
	}
	var spans []BucketSpan
	cur := startMs
	for cur < endMs {
		bucket, err := BucketAtUTC(cur)
		if err != nil {
			return nil, err
		}
		boundary := nextBoundaryUTC(cur)
		if boundary > endMs {
			boundary = endMs
		}
		millis := boundary - cur
		if millis <= 0 {
			return nil, ErrInvalidInterval
		}
		spans = append(spans, BucketSpan{Bucket: bucket, Millis: millis})
		cur = boundary
	}
	return spans, nil
}

// SplitIntervalLocal splits a half-open interval in local-time bucket
// coordinates while still measuring elapsed real milliseconds.
func SplitIntervalLocal(startMs, endMs int64, loc *time.Location) ([]BucketSpan, error) {
	if loc == nil {
		return nil, ErrInvalidTimestamp
	}
	if endMs <= startMs {
		return nil, ErrInvalidInterval
	}
	var spans []BucketSpan
	cur := startMs
	for cur < endMs {
		bucket, err := BucketAtLocal(cur, loc)
		if err != nil {
			return nil, err
		}
		boundary := nextBoundaryLocal(cur, loc)
		if boundary > endMs {
			boundary = endMs
		}
		millis := boundary - cur
		if millis <= 0 {
			return nil, ErrInvalidInterval
		}
		spans = append(spans, BucketSpan{Bucket: bucket, Millis: millis})
		cur = boundary
	}
	return spans, nil
}

// nextBoundaryUTC returns the next 5-minute UTC wall-clock boundary after ts.
func nextBoundaryUTC(timestampMs int64) int64 {
	t := time.UnixMilli(timestampMs).UTC()
	minute := (t.Minute()/5 + 1) * 5
	hour := t.Hour()
	day := t.Day()
	month := t.Month()
	year := t.Year()
	if minute >= 60 {
		minute = 0
		hour++
	}
	boundary := time.Date(year, month, day, hour, minute, 0, 0, time.UTC)
	return boundary.UnixMilli()
}

// nextBoundaryLocal returns the next local 5-minute boundary and applies a
// forward fallback when DST transitions make wall-clock construction ambiguous.
func nextBoundaryLocal(timestampMs int64, loc *time.Location) int64 {
	t := time.UnixMilli(timestampMs).In(loc)
	minute := (t.Minute()/5 + 1) * 5
	hour := t.Hour()
	day := t.Day()
	month := t.Month()
	year := t.Year()
	if minute >= 60 {
		minute = 0
		hour++
	}
	boundary := time.Date(year, month, day, hour, minute, 0, 0, loc)
	boundaryMs := boundary.UnixMilli()
	if boundaryMs <= timestampMs {
		return timestampMs + int64(5*60*1000)
	}
	return boundaryMs
}

// bucketFromTime converts a wall-clock time to Monday=0 weekly bucket index.
func bucketFromTime(t time.Time) int {
	dayIndex := (int(t.Weekday()) + 6) % 7
	bucketWithinDay := t.Hour()*12 + (t.Minute() / 5)
	return dayIndex*288 + bucketWithinDay
}
