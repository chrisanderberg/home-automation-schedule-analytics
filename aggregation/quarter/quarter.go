package quarter

import "time"

type QuarterSpan struct {
	QuarterIndex int
	StartMs      int64
	EndMs        int64
}

// SplitIntervalUTC splits a half-open interval at UTC quarter boundaries so
// downstream aggregation can write each span into its quarter-specific blob.
func SplitIntervalUTC(startMs, endMs int64) ([]QuarterSpan, error) {
	if endMs <= startMs {
		return nil, ErrInvalidInterval
	}
	var spans []QuarterSpan
	cur := startMs
	for cur < endMs {
		t := time.UnixMilli(cur).UTC()
		quarterIndex := quarterIndexUTC(t)
		nextBoundary := nextQuarterStartUTC(t)
		boundaryMs := nextBoundary.UnixMilli()
		if boundaryMs > endMs {
			boundaryMs = endMs
		}
		spans = append(spans, QuarterSpan{
			QuarterIndex: quarterIndex,
			StartMs:      cur,
			EndMs:        boundaryMs,
		})
		cur = boundaryMs
	}
	return spans, nil
}

// QuarterIndexUTC returns the integer quarter key defined by requirements:
// (utcYear-1970)*4 + (quarterNumber-1).
func QuarterIndexUTC(timestampMs int64) int {
	t := time.UnixMilli(timestampMs).UTC()
	return quarterIndexUTC(t)
}

func quarterIndexUTC(t time.Time) int {
	month := int(t.Month())
	quarterNumber := ((month - 1) / 3) + 1
	return (t.Year()-1970)*4 + (quarterNumber - 1)
}

// nextQuarterStartUTC computes the start instant of the quarter immediately
// after t in UTC calendar time.
func nextQuarterStartUTC(t time.Time) time.Time {
	month := int(t.Month())
	quarter := (month-1)/3 + 1
	year := t.Year()
	nextQuarter := quarter + 1
	if nextQuarter == 5 {
		nextQuarter = 1
		year++
	}
	startMonth := time.Month((nextQuarter-1)*3 + 1)
	return time.Date(year, startMonth, 1, 0, 0, 0, 0, time.UTC)
}
