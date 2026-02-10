package ingest

import (
	"context"
	"testing"
	"time"

	"home-automation-analytics/aggregation/blob"
	"home-automation-analytics/aggregation/bucketing"
	"home-automation-analytics/aggregation/quarter"
	"home-automation-analytics/aggregation/storage"
)

func TestHoldingIngestSingleBucketUTC(t *testing.T) {
	ctx := context.Background()
	db, err := storage.Open(":memory:")
	if err != nil {
		t.Fatalf("open db: %v", err)
	}
	defer db.Close()
	if err := storage.InitSchema(ctx, db); err != nil {
		t.Fatalf("init schema: %v", err)
	}

	control := storage.Control{ControlID: "c1", ControlType: storage.ControlTypeDiscrete, NumStates: 2}
	if err := storage.UpsertControl(ctx, db, control); err != nil {
		t.Fatalf("upsert control: %v", err)
	}

	cfg := Config{TimeZone: "UTC", Latitude: 0, Longitude: 0}

	start := time.Date(2020, 1, 6, 0, 0, 0, 0, time.UTC)
	end := time.Date(2020, 1, 6, 0, 5, 0, 0, time.UTC)
	input := HoldingInput{
		ControlID:   "c1",
		ModelID:     "m1",
		State:       1,
		StartTimeMs: start.UnixMilli(),
		EndTimeMs:   end.UnixMilli(),
	}

	if err := IngestHolding(ctx, db, cfg, input); err != nil {
		t.Fatalf("ingest holding: %v", err)
	}

	key := storage.AggregateKey{
		ControlID:    "c1",
		ModelID:      "m1",
		QuarterIndex: quarter.QuarterIndexUTC(start.UnixMilli()),
	}
	blobBytes, err := storage.GetOrCreateAggregate(ctx, db, key, 2)
	if err != nil {
		t.Fatalf("get aggregate: %v", err)
	}
	b, err := blob.NewBlob(2)
	if err != nil {
		t.Fatalf("new blob: %v", err)
	}
	copy(b.Data(), blobBytes)

	bucket, err := bucketing.BucketAtUTC(start.UnixMilli())
	if err != nil {
		t.Fatalf("bucketAtUTC: %v", err)
	}

	idxUTC, err := blob.HoldIndex(1, 0, bucket, 2)
	if err != nil {
		t.Fatalf("hold index utc: %v", err)
	}
	idxLocal, err := blob.HoldIndex(1, 1, bucket, 2)
	if err != nil {
		t.Fatalf("hold index local: %v", err)
	}
	vUTC, err := b.GetU64(idxUTC)
	if err != nil {
		t.Fatalf("get utc value: %v", err)
	}
	vLocal, err := b.GetU64(idxLocal)
	if err != nil {
		t.Fatalf("get local value: %v", err)
	}
	if vUTC != 5*60*1000 {
		t.Fatalf("utc holding mismatch: got %d want %d", vUTC, 5*60*1000)
	}
	if vLocal != 5*60*1000 {
		t.Fatalf("local holding mismatch: got %d want %d", vLocal, 5*60*1000)
	}

	idxOtherState, err := blob.HoldIndex(0, 0, bucket, 2)
	if err != nil {
		t.Fatalf("hold index other state: %v", err)
	}
	vOther, err := b.GetU64(idxOtherState)
	if err != nil {
		t.Fatalf("get other state value: %v", err)
	}
	if vOther != 0 {
		t.Fatalf("unexpected other state value: %d", vOther)
	}

	idxOtherBucket, err := blob.HoldIndex(1, 0, (bucket+1)%bucketingBucketsPerWeek(), 2)
	if err != nil {
		t.Fatalf("hold index other bucket: %v", err)
	}
	vOtherBucket, err := b.GetU64(idxOtherBucket)
	if err != nil {
		t.Fatalf("get other bucket value: %v", err)
	}
	if vOtherBucket != 0 {
		t.Fatalf("unexpected other bucket value: %d", vOtherBucket)
	}
}

func bucketingBucketsPerWeek() int {
	return 7 * 288
}
