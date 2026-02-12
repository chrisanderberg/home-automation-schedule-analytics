package bucketing

import (
	"math"
	"time"
)

func BucketAtMeanSolar(timestampMs int64, latitude, longitude float64) (int, error) {
	// Mean solar time is a longitude-only offset from UTC.
	offsetMinutes := longitude * 4
	adj := time.UnixMilli(timestampMs).UTC().Add(time.Duration(offsetMinutes) * time.Minute)
	return bucketFromTime(adj), nil
}

func BucketAtApparentSolar(timestampMs int64, latitude, longitude float64) (int, error) {
	// Apparent solar adds the equation-of-time seasonal correction.
	offsetMinutes := longitude*4 + equationOfTimeMinutes(time.UnixMilli(timestampMs).UTC())
	adj := time.UnixMilli(timestampMs).UTC().Add(time.Duration(offsetMinutes) * time.Minute)
	return bucketFromTime(adj), nil
}

// BucketAtUnequalHours maps solar minutes into 12 equal daylight and 12 equal
// night hours, yielding a synthetic 24-hour day used for experimental bucketing.
func BucketAtUnequalHours(timestampMs int64, latitude, longitude float64) (int, error) {
	if latitude > 90 || latitude < -90 || longitude > 180 || longitude < -180 {
		return 0, ErrInvalidTimestamp
	}

	eqTime := equationOfTimeMinutes(time.UnixMilli(timestampMs).UTC())
	offsetMinutes := longitude*4 + eqTime

	adj := time.UnixMilli(timestampMs).UTC().Add(time.Duration(offsetMinutes) * time.Minute)
	dayTime := time.Date(adj.Year(), adj.Month(), adj.Day(), 0, 0, 0, 0, time.UTC)
	solarMinutes := (adj.Sub(dayTime)).Minutes()
	if solarMinutes < 0 {
		solarMinutes += 1440
	}

	sunrise, sunset, err := sunriseSunsetSolarMinutes(dayTime, latitude)
	if err != nil {
		return 0, err
	}
	if solarMinutes < 0 || solarMinutes >= 1440 {
		return 0, ErrInvalidTimestamp
	}

	dayLength := sunset - sunrise
	nightLength := 1440 - dayLength
	if dayLength <= 0 || nightLength <= 0 {
		return 0, ErrUndefinedClock
	}

	var pseudoMinutes float64
	if solarMinutes >= sunrise && solarMinutes < sunset {
		dayFraction := (solarMinutes - sunrise) / dayLength
		pseudoMinutes = 360 + dayFraction*720
	} else {
		var nightFraction float64
		if solarMinutes >= sunset {
			nightFraction = (solarMinutes - sunset) / nightLength
		} else {
			nightFraction = (solarMinutes + 1440 - sunset) / nightLength
		}
		pseudoMinutes = 1080 + nightFraction*720
		if pseudoMinutes >= 1440 {
			pseudoMinutes -= 1440
		}
	}

	bucketWithinDay := int(pseudoMinutes) / 5
	dayIndex := (int(adj.Weekday()) + 6) % 7
	return dayIndex*288 + bucketWithinDay, nil
}

// SplitIntervalMeanSolar splits with a fixed longitude-derived offset.
func SplitIntervalMeanSolar(startMs, endMs int64, latitude, longitude float64) ([]BucketSpan, error) {
	return splitIntervalWithOffset(startMs, endMs, func(ts int64) float64 {
		return longitude * 4
	})
}

// SplitIntervalApparentSolar splits with longitude plus equation-of-time
// offset sampled at each boundary step.
func SplitIntervalApparentSolar(startMs, endMs int64, latitude, longitude float64) ([]BucketSpan, error) {
	return splitIntervalWithOffset(startMs, endMs, func(ts int64) float64 {
		return longitude*4 + equationOfTimeMinutes(time.UnixMilli(ts).UTC())
	})
}

// SplitIntervalUnequalHours splits an interval in unequal-hour bucket space and
// returns ErrUndefinedClock when sunrise/sunset are undefined for that date.
func SplitIntervalUnequalHours(startMs, endMs int64, latitude, longitude float64) ([]BucketSpan, error) {
	if endMs <= startMs {
		return nil, ErrInvalidInterval
	}
	var spans []BucketSpan
	cur := startMs
	for cur < endMs {
		bucket, err := BucketAtUnequalHours(cur, latitude, longitude)
		if err != nil {
			if err == ErrUndefinedClock {
				return nil, ErrUndefinedClock
			}
			return nil, err
		}
		boundary, err := nextUnequalBoundary(cur, latitude, longitude)
		if err != nil {
			return nil, err
		}
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

func unequalHoursBoundary(t time.Time, latitude, longitude float64) (time.Time, error) {
	// Boundary based on next 5-minute bucket in unequal-hours clock.
	bucket, err := BucketAtUnequalHours(t.UnixMilli(), latitude, longitude)
	if err != nil {
		return time.Time{}, err
	}
	nextBucketStart := t.Add(5 * time.Minute)
	for i := 0; i < 400; i++ { // safety cap
		b, err := BucketAtUnequalHours(nextBucketStart.UnixMilli(), latitude, longitude)
		if err != nil {
			return time.Time{}, err
		}
		if b != bucket {
			return nextBucketStart, nil
		}
		nextBucketStart = nextBucketStart.Add(1 * time.Minute)
	}
	return time.Time{}, ErrInvalidInterval
}

func nextUnequalBoundary(timestampMs int64, latitude, longitude float64) (int64, error) {
	cur := time.UnixMilli(timestampMs).UTC()
	boundary, err := unequalHoursBoundary(cur, latitude, longitude)
	if err != nil {
		return 0, err
	}
	if boundary.UnixMilli() <= timestampMs {
		return timestampMs + int64(5*60*1000), nil
	}
	return boundary.UnixMilli(), nil
}

// splitIntervalWithOffset handles clocks expressible as UTC plus a minute
// offset function and then maps boundaries back to UTC elapsed time.
func splitIntervalWithOffset(startMs, endMs int64, offsetMinutes func(int64) float64) ([]BucketSpan, error) {
	if endMs <= startMs {
		return nil, ErrInvalidInterval
	}
	var spans []BucketSpan
	cur := startMs
	for cur < endMs {
		offset := offsetMinutes(cur)
		adj := time.UnixMilli(cur).UTC().Add(time.Duration(offset) * time.Minute)
		bucket := bucketFromTime(adj)
		adjBoundary := nextBoundaryUTC(adj.UnixMilli())
		boundaryUTC := time.UnixMilli(adjBoundary).Add(-time.Duration(offset) * time.Minute).UnixMilli()
		if boundaryUTC <= cur {
			boundaryUTC = cur + int64(5*60*1000)
		}
		if boundaryUTC > endMs {
			boundaryUTC = endMs
		}
		millis := boundaryUTC - cur
		if millis <= 0 {
			return nil, ErrInvalidInterval
		}
		spans = append(spans, BucketSpan{Bucket: bucket, Millis: millis})
		cur = boundaryUTC
	}
	return spans, nil
}

func equationOfTimeMinutes(day time.Time) float64 {
	gamma := fractionalYear(day)
	return 229.18 * (0.000075 + 0.001868*math.Cos(gamma) - 0.032077*math.Sin(gamma) - 0.014615*math.Cos(2*gamma) - 0.040849*math.Sin(2*gamma))
}

func solarDeclination(day time.Time) float64 {
	gamma := fractionalYear(day)
	return 0.006918 - 0.399912*math.Cos(gamma) + 0.070257*math.Sin(gamma) - 0.006758*math.Cos(2*gamma) + 0.000907*math.Sin(2*gamma) - 0.002697*math.Cos(3*gamma) + 0.00148*math.Sin(3*gamma)
}

func fractionalYear(day time.Time) float64 {
	yday := day.YearDay()
	return 2 * math.Pi / 365 * (float64(yday-1) + (float64(day.Hour())-12)/24)
}

// sunriseSunsetSolarMinutes estimates sunrise/sunset in local solar minutes
// from midnight and reports ErrUndefinedClock at polar day/night extremes.
func sunriseSunsetSolarMinutes(day time.Time, latitude float64) (float64, float64, error) {
	decl := solarDeclination(day)
	latRad := latitude * math.Pi / 180
	solarZenith := 90.833 * math.Pi / 180

	cosH := (math.Cos(solarZenith)/(math.Cos(latRad)*math.Cos(decl)) - math.Tan(latRad)*math.Tan(decl))
	if cosH > 1 || cosH < -1 {
		return 0, 0, ErrUndefinedClock
	}
	H := math.Acos(cosH) // radians
	Hdeg := H * 180 / math.Pi
	sunrise := 720 - 4*Hdeg
	sunset := 720 + 4*Hdeg
	return sunrise, sunset, nil
}
