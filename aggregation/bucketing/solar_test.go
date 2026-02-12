package bucketing

import (
	"errors"
	"testing"
	"time"
)

// TestSolarBucketsDefined verifies all three solar clocks return valid bucket
// indices when the solar mapping is defined for a normal location/date.
func TestSolarBucketsDefined(t *testing.T) {
	// San Francisco, 2020-06-01T12:00:00Z
	lat := 37.7749
	lon := -122.4194
	timestamp := time.Date(2020, 6, 1, 12, 0, 0, 0, time.UTC).UnixMilli()

	checks := []struct {
		name string
		fn   func(int64, float64, float64) (int, error)
	}{
		{"mean", BucketAtMeanSolar},
		{"apparent", BucketAtApparentSolar},
		{"unequal", BucketAtUnequalHours},
	}

	for _, tc := range checks {
		t.Run(tc.name, func(t *testing.T) {
			bucket, err := tc.fn(timestamp, lat, lon)
			if err != nil {
				t.Fatalf("bucket error: %v", err)
			}
			if bucket < 0 || bucket >= 7*288 {
				t.Fatalf("bucket out of range: %d", bucket)
			}
		})
	}
}

// TestUnequalHoursUndefinedOnly verifies unequal-hours bucketing reports an
// undefined-clock error when sunrise/sunset do not exist while other solar
// clocks remain computable for the same timestamp and location.
func TestUnequalHoursUndefinedOnly(t *testing.T) {
	// High latitude in winter to trigger no sunrise/sunset for unequal hours.
	lat := 78.2232 // Longyearbyen
	lon := 15.6469
	timestamp := time.Date(2020, 12, 21, 12, 0, 0, 0, time.UTC).UnixMilli()

	bucket, err := BucketAtMeanSolar(timestamp, lat, lon)
	if err != nil {
		t.Fatalf("mean solar error: %v", err)
	}
	if bucket < 0 || bucket >= 7*288 {
		t.Fatalf("mean solar bucket out of range: %d", bucket)
	}

	bucket, err = BucketAtApparentSolar(timestamp, lat, lon)
	if err != nil {
		t.Fatalf("apparent solar error: %v", err)
	}
	if bucket < 0 || bucket >= 7*288 {
		t.Fatalf("apparent solar bucket out of range: %d", bucket)
	}

	_, err = BucketAtUnequalHours(timestamp, lat, lon)
	if err == nil {
		t.Fatalf("expected unequal hours to be undefined")
	}
	if !errors.Is(err, ErrUndefinedClock) {
		t.Fatalf("unexpected error: %v", err)
	}
}

// TestSolarCoordinateValidation verifies all solar clock variants reject
// out-of-range latitude/longitude input consistently.
func TestSolarCoordinateValidation(t *testing.T) {
	timestamp := time.Date(2020, 6, 1, 12, 0, 0, 0, time.UTC).UnixMilli()

	bucketChecks := []struct {
		name string
		fn   func(int64, float64, float64) (int, error)
	}{
		{"mean", BucketAtMeanSolar},
		{"apparent", BucketAtApparentSolar},
		{"unequal", BucketAtUnequalHours},
	}
	for _, tc := range bucketChecks {
		t.Run("bucket_"+tc.name, func(t *testing.T) {
			_, err := tc.fn(timestamp, 95, 0)
			if !errors.Is(err, ErrInvalidCoordinates) {
				t.Fatalf("expected ErrInvalidCoordinates, got %v", err)
			}
		})
	}

	splitChecks := []struct {
		name string
		fn   func(int64, int64, float64, float64) ([]BucketSpan, error)
	}{
		{"mean", SplitIntervalMeanSolar},
		{"apparent", SplitIntervalApparentSolar},
		{"unequal", SplitIntervalUnequalHours},
	}
	for _, tc := range splitChecks {
		t.Run("split_"+tc.name, func(t *testing.T) {
			_, err := tc.fn(timestamp, timestamp+60_000, 0, 200)
			if !errors.Is(err, ErrInvalidCoordinates) {
				t.Fatalf("expected ErrInvalidCoordinates, got %v", err)
			}
		})
	}
}
